package proto

import (
	"google.golang.org/protobuf/encoding/protojson"
	gproto "google.golang.org/protobuf/proto"
)

// unmarshalRequestJSON 使用 Proto JSON 规则解析请求，兼容 lowerCamelCase 和 snake_case 字段名。
// Hollow 当前生成的 POST Handler 使用 encoding/json；为请求消息实现该方法后，
// dueAt、pageSize、restaurantId 等字段不会在绑定时丢失。
func unmarshalRequestJSON(data []byte, message gproto.Message) error {
	return protojson.UnmarshalOptions{DiscardUnknown: false}.Unmarshal(data, message)
}

func (x *WechatLoginRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *GetCurrentUserRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *GetAccessStatusRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListAccessRequestsRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ModifyAccessRequestRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListContributionsRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListFavoritesRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *GetFavoriteStatusRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ModifyFavoriteRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListTasksRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *GetTaskRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *CreateTaskRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *UpdateTaskRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *UpdateTaskStatusRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *DeleteTaskRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *RecognizeTaskVoiceRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListActivityLogsRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *GetActivityLogRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *CreateActivityLogRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *UpdateActivityLogRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *DeleteActivityLogRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ActivityLogStatsRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *RecognizeActivityVoiceRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListBillsRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *GetBillRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *CreateBillRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *UpdateBillRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *DeleteBillRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *BillStatsRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ParseQuickRecordRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *RecognizeQuickRecordVoiceRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListRestaurantsRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *GetRestaurantRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *CreateRestaurantRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *UpdateRestaurantRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *DeleteRestaurantRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *RecommendRestaurantRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListRestaurantConsumptionsRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *CreateRestaurantConsumptionRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *UpdateRestaurantConsumptionRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *DeleteRestaurantConsumptionRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListPlacesRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *GetPlaceRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *CreatePlaceRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *UpdatePlaceRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *DeletePlaceRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *ListMemoriesRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
func (x *CreateMemoryRequest) UnmarshalJSON(data []byte) error {
	return unmarshalRequestJSON(data, x)
}
