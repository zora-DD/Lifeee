package proto

import "encoding/json"

// MarshalJSON 固定吃喝地点列表的前端 JSON 契约，并确保空列表返回 []。
func (x *ListRestaurantsResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*RestaurantItem, 0)
	}
	return json.Marshal(struct {
		Items           []*RestaurantItem `json:"items"`
		Page            int32             `json:"page"`
		PageSize        int32             `json:"pageSize"`
		Total           int64             `json:"total"`
		HasMore         bool              `json:"has_more"`
		OverallTotal    int64             `json:"overallTotal"`
		TotalVisitCount int64             `json:"totalVisitCount"`
		TotalSpent      string            `json:"totalSpent"`
	}{items, x.Page, x.PageSize, x.Total, x.HasMore, x.OverallTotal, x.TotalVisitCount, x.TotalSpent})
}

// MarshalJSON 固定吃喝地点条目的字段命名。
func (x *RestaurantItem) MarshalJSON() ([]byte, error) {
	consumptions := x.Consumptions
	if consumptions == nil {
		consumptions = make([]*RestaurantConsumptionItem, 0)
	}
	return json.Marshal(struct {
		ID             string                       `json:"id"`
		Name           string                       `json:"name"`
		Category       string                       `json:"category"`
		Cuisine        string                       `json:"cuisine"`
		Preference     string                       `json:"preference"`
		Address        string                       `json:"address"`
		BusinessHours  string                       `json:"businessHours"`
		Features       string                       `json:"features"`
		CoverURL       string                       `json:"coverUrl"`
		CoverObjectKey string                       `json:"coverObjectKey"`
		VisitCount     uint32                       `json:"visitCount"`
		TotalSpent     string                       `json:"totalSpent"`
		LastVisitedAt  string                       `json:"lastVisitedAt"`
		LastAmount     string                       `json:"lastAmount"`
		LastMeal       string                       `json:"lastMeal"`
		LastNote       string                       `json:"lastNote"`
		CreatedBy      string                       `json:"createdBy"`
		CreatedAt      string                       `json:"createdAt"`
		UpdatedAt      string                       `json:"updatedAt"`
		Consumptions   []*RestaurantConsumptionItem `json:"consumptions"`
	}{
		x.Id, x.Name, x.Category, x.Cuisine, x.Preference, x.Address,
		x.BusinessHours, x.Features, x.CoverUrl, x.CoverObjectKey, x.VisitCount,
		x.TotalSpent, x.LastVisitedAt, x.LastAmount, x.LastMeal, x.LastNote,
		x.CreatedBy, x.CreatedAt, x.UpdatedAt, consumptions,
	})
}

// MarshalJSON 固定消费记录列表的前端 JSON 契约。
func (x *ListRestaurantConsumptionsResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*RestaurantConsumptionItem, 0)
	}
	return json.Marshal(struct {
		Items    []*RestaurantConsumptionItem `json:"items"`
		Page     int32                        `json:"page"`
		PageSize int32                        `json:"pageSize"`
		Total    int64                        `json:"total"`
		HasMore  bool                         `json:"has_more"`
	}{items, x.Page, x.PageSize, x.Total, x.HasMore})
}

// MarshalJSON 固定消费记录字段命名。
func (x *RestaurantConsumptionItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           string `json:"id"`
		RestaurantID string `json:"restaurantId"`
		ConsumedBy   string `json:"consumedBy"`
		ConsumedAt   string `json:"consumedAt"`
		Amount       string `json:"amount"`
		Meal         string `json:"meal"`
		Note         string `json:"note"`
		CreatedAt    string `json:"createdAt"`
		UpdatedAt    string `json:"updatedAt"`
	}{x.Id, x.RestaurantId, x.ConsumedBy, x.ConsumedAt, x.Amount, x.Meal, x.Note, x.CreatedAt, x.UpdatedAt})
}
