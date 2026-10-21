package proto

import "encoding/json"

// MarshalJSON 固定时光列表 JSON 契约，并确保空数组返回 []。
func (x *ListMemoriesResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*MemoryItem, 0)
	}
	years := x.Years
	if years == nil {
		years = make([]int32, 0)
	}
	return json.Marshal(struct {
		Items    []*MemoryItem `json:"items"`
		Page     int32         `json:"page"`
		PageSize int32         `json:"pageSize"`
		Total    int64         `json:"total"`
		HasMore  bool          `json:"has_more"`
		Years    []int32       `json:"years"`
	}{items, x.Page, x.PageSize, x.Total, x.HasMore, years})
}

// MarshalJSON 固定时光记录字段命名。
func (x *MemoryItem) MarshalJSON() ([]byte, error) {
	images := x.Images
	if images == nil {
		images = make([]*MemoryPhotoItem, 0)
	}
	return json.Marshal(struct {
		ID            string             `json:"id"`
		Date          string             `json:"date"`
		DatePrecision string             `json:"datePrecision"`
		Place         string             `json:"place"`
		Title         string             `json:"title"`
		Description   string             `json:"description"`
		Images        []*MemoryPhotoItem `json:"images"`
		CreatedBy     string             `json:"createdBy"`
		CreatedAt     string             `json:"createdAt"`
		UpdatedAt     string             `json:"updatedAt"`
	}{x.Id, x.Date, x.DatePrecision, x.Place, x.Title, x.Description, images, x.CreatedBy, x.CreatedAt, x.UpdatedAt})
}

// MarshalJSON 固定时光图片字段命名。
func (x *MemoryPhotoItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID        string `json:"id"`
		MediaID   string `json:"mediaId"`
		ObjectKey string `json:"objectKey"`
		URL       string `json:"url"`
		Width     uint32 `json:"width"`
		Height    uint32 `json:"height"`
		AltText   string `json:"altText"`
		SortOrder uint32 `json:"sortOrder"`
	}{x.Id, x.MediaId, x.ObjectKey, x.Url, x.Width, x.Height, x.AltText, x.SortOrder})
}
