package proto

import "encoding/json"

// MarshalJSON 固定用户信息的前端 JSON 契约。
func (x *UserItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID           string `json:"id"`
		Nickname     string `json:"nickname"`
		AvatarURL    string `json:"avatarUrl"`
		AvatarEmoji  string `json:"avatarEmoji"`
		AvatarColor  string `json:"avatarColor"`
		Signature    string `json:"signature"`
		AccessStatus string `json:"accessStatus"`
		Role         string `json:"role"`
		CreatedAt    string `json:"createdAt"`
	}{
		ID: x.Id, Nickname: x.Nickname, AvatarURL: x.AvatarUrl,
		AvatarEmoji: x.AvatarEmoji, AvatarColor: x.AvatarColor, Signature: x.Signature,
		AccessStatus: x.AccessStatus, Role: x.Role, CreatedAt: x.CreatedAt,
	})
}

// MarshalJSON 固定审批列表分页字段，并确保空列表返回 []。
func (x *ListAccessRequestsResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*UserItem, 0)
	}
	return json.Marshal(struct {
		Items    []*UserItem `json:"items"`
		Page     int32       `json:"page"`
		PageSize int32       `json:"pageSize"`
		Total    int64       `json:"total"`
		HasMore  bool        `json:"has_more"`
	}{
		Items: items, Page: x.Page, PageSize: x.PageSize, Total: x.Total, HasMore: x.HasMore,
	})
}
