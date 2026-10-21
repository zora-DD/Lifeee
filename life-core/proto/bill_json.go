package proto

import "encoding/json"

func (x *BillCreator) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          string `json:"id"`
		Nickname    string `json:"nickname"`
		AvatarEmoji string `json:"avatarEmoji"`
		AvatarColor string `json:"avatarColor"`
	}{x.Id, x.Nickname, x.AvatarEmoji, x.AvatarColor})
}

func (x *BillItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID         string       `json:"id"`
		Type       string       `json:"type"`
		Title      string       `json:"title"`
		Amount     string       `json:"amount"`
		Category   string       `json:"category"`
		OccurredAt string       `json:"occurredAt"`
		Note       string       `json:"note"`
		InputType  string       `json:"inputType"`
		RawText    string       `json:"rawText"`
		CreatedBy  string       `json:"createdBy"`
		Creator    *BillCreator `json:"creator"`
		CreatedAt  string       `json:"createdAt"`
		UpdatedAt  string       `json:"updatedAt"`
	}{x.Id, x.Type, x.Title, x.Amount, x.Category, x.OccurredAt, x.Note, x.InputType, x.RawText, x.CreatedBy, x.Creator, x.CreatedAt, x.UpdatedAt})
}

func (x *ListBillsResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*BillItem, 0)
	}
	creators := x.Creators
	if creators == nil {
		creators = make([]*BillCreator, 0)
	}
	return json.Marshal(struct {
		Items    []*BillItem    `json:"items"`
		Creators []*BillCreator `json:"creators"`
		Page     int32          `json:"page"`
		PageSize int32          `json:"pageSize"`
		Total    int64          `json:"total"`
		HasMore  bool           `json:"has_more"`
	}{items, creators, x.Page, x.PageSize, x.Total, x.HasMore})
}

func (x *BillCategoryStat) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Category string `json:"category"`
		Amount   string `json:"amount"`
		Count    int32  `json:"count"`
	}{x.Category, x.Amount, x.Count})
}

func (x *BillStatsResponse) MarshalJSON() ([]byte, error) {
	categories := x.Categories
	if categories == nil {
		categories = make([]*BillCategoryStat, 0)
	}
	monthly := x.Monthly
	if monthly == nil {
		monthly = make([]*BillMonthlyStat, 0)
	}
	users := x.Users
	if users == nil {
		users = make([]*BillUserStat, 0)
	}
	return json.Marshal(struct {
		ExpenseTotal string              `json:"expenseTotal"`
		IncomeTotal  string              `json:"incomeTotal"`
		BillCount    int32               `json:"billCount"`
		Categories   []*BillCategoryStat `json:"categories"`
		Monthly      []*BillMonthlyStat  `json:"monthly"`
		Users        []*BillUserStat     `json:"users"`
		Scope        string              `json:"scope"`
		Year         int32               `json:"year"`
		Month        int32               `json:"month"`
	}{x.ExpenseTotal, x.IncomeTotal, x.BillCount, categories, monthly, users, x.Scope, x.Year, x.Month})
}

func (x *BillMonthlyStat) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Month         int32  `json:"month"`
		Label         string `json:"label"`
		ExpenseAmount string `json:"expenseAmount"`
		IncomeAmount  string `json:"incomeAmount"`
		Count         int32  `json:"count"`
	}{x.Month, x.Label, x.ExpenseAmount, x.IncomeAmount, x.Count})
}

func (x *BillUserStat) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		User          *BillCreator `json:"user"`
		ExpenseAmount string       `json:"expenseAmount"`
		IncomeAmount  string       `json:"incomeAmount"`
		Count         int32        `json:"count"`
	}{x.User, x.ExpenseAmount, x.IncomeAmount, x.Count})
}

func (x *ActivityLogDraft) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Content         string `json:"content"`
		OccurredAt      string `json:"occurredAt"`
		DurationMinutes int32  `json:"durationMinutes"`
	}{x.Content, x.OccurredAt, x.DurationMinutes})
}

func (x *BillDraft) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Type       string `json:"type"`
		Title      string `json:"title"`
		Amount     string `json:"amount"`
		Category   string `json:"category"`
		OccurredAt string `json:"occurredAt"`
		Note       string `json:"note"`
	}{x.Type, x.Title, x.Amount, x.Category, x.OccurredAt, x.Note})
}

func (x *QuickRecordDraftResponse) MarshalJSON() ([]byte, error) {
	bills := x.Bills
	if bills == nil {
		bills = make([]*BillDraft, 0)
	}
	return json.Marshal(struct {
		Intent          string            `json:"intent"`
		RawText         string            `json:"rawText"`
		Activity        *ActivityLogDraft `json:"activity"`
		Bills           []*BillDraft      `json:"bills"`
		AudioDurationMS int64             `json:"audioDurationMs"`
		ASRRequestID    string            `json:"asrRequestId"`
	}{x.Intent, x.RawText, x.Activity, bills, x.AudioDurationMs, x.AsrRequestId})
}
