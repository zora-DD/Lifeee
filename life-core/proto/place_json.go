package proto

import "encoding/json"

// MarshalJSON 固定地点列表的前端 JSON 契约，并确保空列表返回 []。
func (x *ListPlacesResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*PlaceItem, 0)
	}
	return json.Marshal(struct {
		Items         []*PlaceItem `json:"items"`
		Page          int32        `json:"page"`
		PageSize      int32        `json:"pageSize"`
		Total         int64        `json:"total"`
		HasMore       bool         `json:"has_more"`
		OverallTotal  int64        `json:"overallTotal"`
		VisitedCount  int64        `json:"visitedCount"`
		WishlistCount int64        `json:"wishlistCount"`
	}{items, x.Page, x.PageSize, x.Total, x.HasMore, x.OverallTotal, x.VisitedCount, x.WishlistCount})
}

// MarshalJSON 固定地点条目的字段命名。
func (x *PlaceItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID             string `json:"id"`
		Name           string `json:"name"`
		Category       string `json:"category"`
		Icon           string `json:"icon"`
		Visited        bool   `json:"visited"`
		VisitCount     uint32 `json:"visitCount"`
		LastVisitedAt  string `json:"lastVisitedAt"`
		Feature        string `json:"feature"`
		Note           string `json:"note"`
		Address        string `json:"address"`
		BestTime       string `json:"bestTime"`
		CoverURL       string `json:"coverUrl"`
		CoverObjectKey string `json:"coverObjectKey"`
		Latitude       string `json:"latitude"`
		Longitude      string `json:"longitude"`
		CreatedBy      string `json:"createdBy"`
		CreatedAt      string `json:"createdAt"`
		UpdatedAt      string `json:"updatedAt"`
		GuideURL       string `json:"guideUrl"`
		DistanceKM     uint32 `json:"distanceKm"`
		DriveMinutes   uint32 `json:"driveDurationMinutes"`
	}{
		x.Id, x.Name, x.Category, x.Icon, x.Visited, x.VisitCount, x.LastVisitedAt,
		x.Feature, x.Note, x.Address, x.BestTime, x.CoverUrl, x.CoverObjectKey,
		x.Latitude, x.Longitude, x.CreatedBy, x.CreatedAt, x.UpdatedAt, x.GuideUrl,
		x.DistanceKm, x.DriveDurationMinutes,
	})
}
