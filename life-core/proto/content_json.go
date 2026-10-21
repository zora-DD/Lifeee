package proto

import "encoding/json"

// MarshalJSON 固定统一内容摘要的前端字段命名。
func (x *ContentSummary) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TargetID   string `json:"targetId"`
		TargetType string `json:"targetType"`
		Title      string `json:"title"`
		Category   string `json:"category"`
		Status     string `json:"status"`
		Priority   string `json:"priority"`
		DueAt      string `json:"dueAt"`
		Icon       string `json:"icon"`
		Visited    bool   `json:"visited"`
		VisitCount uint32 `json:"visitCount"`
		TotalSpent string `json:"totalSpent"`
		CoverURL   string `json:"coverUrl"`
		CreatedAt  string `json:"createdAt"`
	}{
		x.TargetId, x.TargetType, x.Title, x.Category, x.Status, x.Priority,
		x.DueAt, x.Icon, x.Visited, x.VisitCount, x.TotalSpent, x.CoverUrl, x.CreatedAt,
	})
}

func (x *ListContributionsResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*ContentSummary, 0)
	}
	return json.Marshal(struct {
		Items    []*ContentSummary `json:"items"`
		Page     int32             `json:"page"`
		PageSize int32             `json:"pageSize"`
		Total    int64             `json:"total"`
		HasMore  bool              `json:"has_more"`
	}{items, x.Page, x.PageSize, x.Total, x.HasMore})
}

func (x *FavoriteItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        string          `json:"id"`
		Target    *ContentSummary `json:"target"`
		CreatedAt string          `json:"createdAt"`
	}{x.Id, x.Target, x.CreatedAt})
}

func (x *ListFavoritesResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*FavoriteItem, 0)
	}
	return json.Marshal(struct {
		Items    []*FavoriteItem `json:"items"`
		Page     int32           `json:"page"`
		PageSize int32           `json:"pageSize"`
		Total    int64           `json:"total"`
		HasMore  bool            `json:"has_more"`
	}{items, x.Page, x.PageSize, x.Total, x.HasMore})
}

func (x *FavoriteStatusResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TargetType string `json:"targetType"`
		TargetID   string `json:"targetId"`
		Favorited  bool   `json:"favorited"`
	}{x.TargetType, x.TargetId, x.Favorited})
}
