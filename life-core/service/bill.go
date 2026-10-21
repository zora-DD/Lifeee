package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/vaynedu/hollow/pkg/hecode"
	"gorm.io/gorm"

	authctx "life-core/auth"
	"life-core/dao"
	"life-core/model"
	"life-core/proto"
)

const (
	defaultBillPageSize = 20
	maxBillPageSize     = 100
	defaultBillCreator  = uint64(1)
)

var billCategories = map[string]struct{}{
	"餐饮": {}, "交通": {}, "购物": {}, "生活缴费": {}, "宠物": {},
	"娱乐": {}, "医疗": {}, "住房": {}, "收入": {}, "其他": {},
	"日常采购": {}, "智能设备": {}, "家居用品": {}, "转账": {},
}

type BillRepository interface {
	Create(context.Context, *gorm.DB, *model.Bill) error
	GetByID(context.Context, *gorm.DB, uint64) (*dao.BillWithCreator, error)
	Update(context.Context, *gorm.DB, *model.Bill) error
	SoftDelete(context.Context, *gorm.DB, uint64) error
	List(context.Context, *gorm.DB, dao.BillListQuery) ([]dao.BillWithCreator, int64, error)
	ListCreators(context.Context, *gorm.DB) ([]dao.BillCreator, error)
	Aggregate(context.Context, *gorm.DB, uint64, time.Time, time.Time) ([]dao.BillCategoryAggregate, error)
	AggregateMonthly(context.Context, *gorm.DB, uint64, time.Time, time.Time) ([]dao.BillMonthlyAggregate, error)
	AggregateUsers(context.Context, *gorm.DB, time.Time, time.Time) ([]dao.BillUserAggregate, error)
}

// BillService 负责公开账单的参数校验、读写和汇总。
type BillService struct {
	db         *gorm.DB
	repository BillRepository
	filter     dao.BillListQuery
	ready      bool
}

func NewBillService(db *gorm.DB, repository BillRepository) *BillService {
	return &BillService{db: db, repository: repository}
}

func (s *BillService) InitAndCheckListParam(_ context.Context, req *proto.ListBillsRequest) error {
	if req == nil {
		return fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	typeValue := strings.TrimSpace(req.GetType())
	if typeValue != "" && typeValue != model.BillTypeExpense && typeValue != model.BillTypeIncome {
		return fmt.Errorf("%w: unsupported bill type", hecode.ErrInvalidParam)
	}
	category := strings.TrimSpace(req.GetCategory())
	if category != "" {
		if _, ok := billCategories[category]; !ok {
			return fmt.Errorf("%w: unsupported bill category", hecode.ErrInvalidParam)
		}
	}
	from, err := parseOptionalRFC3339(req.GetFrom(), "from")
	if err != nil {
		return err
	}
	to, err := parseOptionalRFC3339(req.GetTo(), "to")
	if err != nil {
		return err
	}
	if from != nil && to != nil && !from.Before(*to) {
		return fmt.Errorf("%w: from must be earlier than to", hecode.ErrInvalidParam)
	}
	page := int(req.GetPage())
	if page == 0 {
		page = 1
	}
	pageSize := int(req.GetPageSize())
	if pageSize == 0 {
		pageSize = defaultBillPageSize
	}
	if page < 1 || pageSize < 1 || pageSize > maxBillPageSize {
		return fmt.Errorf("%w: invalid pagination", hecode.ErrInvalidParam)
	}
	s.filter = dao.BillListQuery{
		Type: typeValue, Category: category, Keyword: strings.TrimSpace(req.GetKeyword()),
		CreatedBy: req.GetCreatedBy(), From: from, To: to, Page: page, PageSize: pageSize,
	}
	s.ready = true
	return nil
}

func (s *BillService) List(ctx context.Context, _ *proto.ListBillsRequest) (*proto.ListBillsResponse, error) {
	if s == nil || !s.ready {
		return nil, fmt.Errorf("%w: InitAndCheckListParam must be called first", hecode.ErrService)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	rows, total, err := s.repository.List(ctx, s.db, s.filter)
	if err != nil {
		return nil, fmt.Errorf("%w: list bills: %v", hecode.ErrDBQuery, err)
	}
	creators, err := s.repository.ListCreators(ctx, s.db)
	if err != nil {
		return nil, fmt.Errorf("%w: list bill creators: %v", hecode.ErrDBQuery, err)
	}
	items := make([]*proto.BillItem, 0, len(rows))
	for i := range rows {
		items = append(items, billToProto(&rows[i]))
	}
	creatorItems := make([]*proto.BillCreator, 0, len(creators))
	for _, creator := range creators {
		creatorItems = append(creatorItems, &proto.BillCreator{
			Id: strconv.FormatUint(creator.ID, 10), Nickname: creator.Nickname,
			AvatarEmoji: creator.AvatarEmoji, AvatarColor: creator.AvatarColor,
		})
	}
	return &proto.ListBillsResponse{
		Items: items, Creators: creatorItems, Page: int32(s.filter.Page), PageSize: int32(s.filter.PageSize),
		Total: total, HasMore: int64(s.filter.Page)*int64(s.filter.PageSize) < total,
	}, nil
}

func (s *BillService) Get(ctx context.Context, req *proto.GetBillRequest) (*proto.BillItem, error) {
	id, err := parseBillID(req.GetId())
	if err != nil {
		return nil, err
	}
	row, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return billToProto(row), nil
}

func (s *BillService) Create(ctx context.Context, req *proto.CreateBillRequest) (*proto.BillItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	row, err := buildBill(req.GetType(), req.GetTitle(), req.GetAmount(), req.GetCategory(), req.GetOccurredAt(), req.GetNote(), req.GetInputType(), req.GetRawText())
	if err != nil {
		return nil, err
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row.CreatedBy = authctx.UserIDOrDefault(ctx, defaultBillCreator)
	if err := s.repository.Create(ctx, s.db, row); err != nil {
		return nil, fmt.Errorf("%w: create bill: %v", hecode.ErrDBInsert, err)
	}
	created, err := s.getByID(ctx, row.ID)
	if err != nil {
		return nil, err
	}
	return billToProto(created), nil
}

func (s *BillService) Update(ctx context.Context, req *proto.UpdateBillRequest) (*proto.BillItem, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseBillID(req.GetId())
	if err != nil {
		return nil, err
	}
	input, err := buildBill(req.GetType(), req.GetTitle(), req.GetAmount(), req.GetCategory(), req.GetOccurredAt(), req.GetNote(), req.GetInputType(), req.GetRawText())
	if err != nil {
		return nil, err
	}
	current, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireOwnerOrAdmin(ctx, current.CreatedBy); err != nil {
		return nil, err
	}
	input.ID = current.ID
	if err := s.repository.Update(ctx, s.db, input); err != nil {
		return nil, fmt.Errorf("%w: update bill: %v", hecode.ErrDBUpdate, err)
	}
	updated, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return billToProto(updated), nil
}

func (s *BillService) Delete(ctx context.Context, req *proto.DeleteBillRequest) (*proto.DeleteBillResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	id, err := parseBillID(req.GetId())
	if err != nil {
		return nil, err
	}
	current, err := s.getByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := requireOwnerOrAdmin(ctx, current.CreatedBy); err != nil {
		return nil, err
	}
	if err := s.repository.SoftDelete(ctx, s.db, id); err != nil {
		return nil, fmt.Errorf("%w: delete bill: %v", hecode.ErrDBDelete, err)
	}
	return &proto.DeleteBillResponse{Deleted: true}, nil
}

func (s *BillService) Stats(ctx context.Context, req *proto.BillStatsRequest) (*proto.BillStatsResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: nil request", hecode.ErrInvalidParam)
	}
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	now := time.Now()
	scope := strings.TrimSpace(req.GetScope())
	if scope == "" {
		scope = "month"
	}
	if scope != "month" && scope != "year" {
		return nil, fmt.Errorf("%w: unsupported bill stats scope", hecode.ErrInvalidParam)
	}
	year, month := int(req.GetYear()), int(req.GetMonth())
	if year == 0 {
		year = now.Year()
	}
	if month == 0 && scope == "month" {
		month = int(now.Month())
	}
	if year < 2000 || year > 2100 || (scope == "month" && (month < 1 || month > 12)) {
		return nil, fmt.Errorf("%w: invalid year or month", hecode.ErrInvalidParam)
	}
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.Local)
	to := from.AddDate(1, 0, 0)
	if scope == "month" {
		from = time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local)
		to = from.AddDate(0, 1, 0)
	}
	rows, err := s.repository.Aggregate(ctx, s.db, req.GetCreatedBy(), from, to)
	if err != nil {
		return nil, fmt.Errorf("%w: aggregate bills: %v", hecode.ErrDBQuery, err)
	}
	var expenseCents, incomeCents int64
	categories := make([]*proto.BillCategoryStat, 0)
	var billCount int32
	for _, row := range rows {
		cents, parseErr := billMoneyToCents(row.Amount)
		if parseErr != nil {
			return nil, fmt.Errorf("%w: invalid aggregate amount: %v", hecode.ErrDBQuery, parseErr)
		}
		billCount += int32(row.Count)
		if row.Type == model.BillTypeIncome {
			incomeCents += cents
			continue
		}
		expenseCents += cents
		categories = append(categories, &proto.BillCategoryStat{Category: row.Category, Amount: billCentsToMoney(cents), Count: int32(row.Count)})
	}
	response := &proto.BillStatsResponse{
		ExpenseTotal: billCentsToMoney(expenseCents), IncomeTotal: billCentsToMoney(incomeCents),
		BillCount: billCount, Categories: categories, Scope: scope, Year: int32(year), Month: int32(month),
		Monthly: make([]*proto.BillMonthlyStat, 0), Users: make([]*proto.BillUserStat, 0),
	}
	if req.GetCreatedBy() == 0 {
		userRows, userErr := s.repository.AggregateUsers(ctx, s.db, from, to)
		if userErr != nil {
			return nil, fmt.Errorf("%w: aggregate user bills: %v", hecode.ErrDBQuery, userErr)
		}
		response.Users = buildBillUserStats(userRows)
	}
	if scope != "year" {
		return response, nil
	}
	monthlyRows, err := s.repository.AggregateMonthly(ctx, s.db, req.GetCreatedBy(), from, to)
	if err != nil {
		return nil, fmt.Errorf("%w: aggregate monthly bills: %v", hecode.ErrDBQuery, err)
	}
	response.Monthly = buildBillMonthlyStats(monthlyRows)
	return response, nil
}

func buildBillMonthlyStats(rows []dao.BillMonthlyAggregate) []*proto.BillMonthlyStat {
	result := make([]*proto.BillMonthlyStat, 12)
	for month := 1; month <= 12; month++ {
		result[month-1] = &proto.BillMonthlyStat{Month: int32(month), Label: fmt.Sprintf("%d月", month), ExpenseAmount: "0.00", IncomeAmount: "0.00"}
	}
	for _, row := range rows {
		if row.Month < 1 || row.Month > 12 {
			continue
		}
		item := result[row.Month-1]
		item.Count += int32(row.Count)
		if row.Type == model.BillTypeIncome {
			item.IncomeAmount = normalizeAggregateMoney(row.Amount)
		} else {
			item.ExpenseAmount = normalizeAggregateMoney(row.Amount)
		}
	}
	return result
}

func buildBillUserStats(rows []dao.BillUserAggregate) []*proto.BillUserStat {
	items := make(map[uint64]*proto.BillUserStat)
	order := make([]uint64, 0)
	for _, row := range rows {
		item := items[row.CreatedBy]
		if item == nil {
			item = &proto.BillUserStat{
				User:          &proto.BillCreator{Id: strconv.FormatUint(row.CreatedBy, 10), Nickname: row.Nickname, AvatarEmoji: row.AvatarEmoji, AvatarColor: row.AvatarColor},
				ExpenseAmount: "0.00", IncomeAmount: "0.00",
			}
			items[row.CreatedBy] = item
			order = append(order, row.CreatedBy)
		}
		if row.Type == model.BillTypeIncome {
			item.IncomeAmount = normalizeAggregateMoney(row.Amount)
		} else {
			item.ExpenseAmount = normalizeAggregateMoney(row.Amount)
			item.Count += int32(row.Count)
		}
	}
	result := make([]*proto.BillUserStat, 0, len(order))
	for _, id := range order {
		result = append(result, items[id])
	}
	return result
}

func normalizeAggregateMoney(value string) string {
	cents, err := billMoneyToCents(value)
	if err != nil {
		return "0.00"
	}
	return billCentsToMoney(cents)
}

func (s *BillService) checkDependencies() error {
	if s == nil || s.db == nil || s.repository == nil {
		return fmt.Errorf("%w: bill service dependencies are not initialized", hecode.ErrService)
	}
	return nil
}

func (s *BillService) getByID(ctx context.Context, id uint64) (*dao.BillWithCreator, error) {
	if err := s.checkDependencies(); err != nil {
		return nil, err
	}
	row, err := s.repository.GetByID(ctx, s.db, id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("%w: bill %d", hecode.ErrNotFound, id)
	}
	if err != nil {
		return nil, fmt.Errorf("%w: get bill: %v", hecode.ErrDBQuery, err)
	}
	return row, nil
}

func parseBillID(value string) (uint64, error) {
	id, err := strconv.ParseUint(strings.TrimSpace(value), 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("%w: invalid bill id", hecode.ErrInvalidParam)
	}
	return id, nil
}

func buildBill(typeValue, title, amount, category, occurredAt, note, inputType, rawText string) (*model.Bill, error) {
	typeValue = strings.TrimSpace(typeValue)
	if typeValue == "" {
		typeValue = model.BillTypeExpense
	}
	if typeValue != model.BillTypeExpense && typeValue != model.BillTypeIncome {
		return nil, fmt.Errorf("%w: unsupported bill type", hecode.ErrInvalidParam)
	}
	title = strings.TrimSpace(title)
	if title == "" || utf8.RuneCountInString(title) > 120 {
		return nil, fmt.Errorf("%w: title is required and must not exceed 120 characters", hecode.ErrInvalidParam)
	}
	normalizedAmount, err := normalizeMoney(amount)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", hecode.ErrInvalidParam, err)
	}
	category = strings.TrimSpace(category)
	if _, ok := billCategories[category]; !ok {
		return nil, fmt.Errorf("%w: unsupported bill category", hecode.ErrInvalidParam)
	}
	occurred, err := time.Parse(time.RFC3339, strings.TrimSpace(occurredAt))
	if err != nil {
		return nil, fmt.Errorf("%w: occurredAt must be RFC3339", hecode.ErrInvalidParam)
	}
	inputType = strings.TrimSpace(inputType)
	if inputType == "" {
		inputType = model.BillInputTypeManual
	}
	if inputType != model.BillInputTypeManual && inputType != model.BillInputTypeText && inputType != model.BillInputTypeVoice {
		return nil, fmt.Errorf("%w: unsupported input type", hecode.ErrInvalidParam)
	}
	note, rawText = strings.TrimSpace(note), strings.TrimSpace(rawText)
	if utf8.RuneCountInString(note) > 1000 || utf8.RuneCountInString(rawText) > 2000 {
		return nil, fmt.Errorf("%w: note or rawText is too long", hecode.ErrInvalidParam)
	}
	row := &model.Bill{Type: typeValue, Title: title, Amount: normalizedAmount, Category: category, OccurredAt: occurred, InputType: inputType}
	if note != "" {
		row.Note = &note
	}
	if rawText != "" {
		row.RawText = &rawText
	}
	return row, nil
}

func normalizeMoney(value string) (string, error) {
	value = strings.TrimSpace(value)
	parts := strings.Split(value, ".")
	if len(parts) > 2 || len(parts[0]) == 0 || len(parts[0]) > 10 {
		return "", errors.New("amount must be a positive number with at most two decimal places")
	}
	for _, char := range parts[0] {
		if char < '0' || char > '9' {
			return "", errors.New("amount must be a positive number with at most two decimal places")
		}
	}
	fraction := ""
	if len(parts) == 2 {
		fraction = parts[1]
		if len(fraction) == 0 || len(fraction) > 2 {
			return "", errors.New("amount must be a positive number with at most two decimal places")
		}
		for _, char := range fraction {
			if char < '0' || char > '9' {
				return "", errors.New("amount must be a positive number with at most two decimal places")
			}
		}
	}
	for len(fraction) < 2 {
		fraction += "0"
	}
	whole := strings.TrimLeft(parts[0], "0")
	if whole == "" {
		whole = "0"
	}
	if whole == "0" && fraction == "00" {
		return "", errors.New("amount must be greater than zero")
	}
	return whole + "." + fraction, nil
}

func billMoneyToCents(value string) (int64, error) {
	normalized, err := normalizeMoney(value)
	if err != nil {
		if strings.TrimSpace(value) == "0" || strings.TrimSpace(value) == "0.00" {
			return 0, nil
		}
		return 0, err
	}
	parts := strings.Split(normalized, ".")
	whole, _ := strconv.ParseInt(parts[0], 10, 64)
	fraction, _ := strconv.ParseInt(parts[1], 10, 64)
	return whole*100 + fraction, nil
}

func billCentsToMoney(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func billToProto(row *dao.BillWithCreator) *proto.BillItem {
	if row == nil {
		return nil
	}
	item := &proto.BillItem{
		Id: strconv.FormatUint(row.ID, 10), Type: row.Type, Title: row.Title, Amount: row.Amount,
		Category: row.Category, OccurredAt: formatTime(row.OccurredAt), InputType: row.InputType,
		CreatedBy: strconv.FormatUint(row.CreatedBy, 10), CreatedAt: formatTime(row.CreatedAt), UpdatedAt: formatTime(row.UpdatedAt),
		Creator: &proto.BillCreator{Id: strconv.FormatUint(row.CreatedBy, 10), Nickname: row.CreatorNickname, AvatarEmoji: row.CreatorEmoji, AvatarColor: row.CreatorColor},
	}
	if row.Note != nil {
		item.Note = *row.Note
	}
	if row.RawText != nil {
		item.RawText = *row.RawText
	}
	return item
}
