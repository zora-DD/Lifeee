package proto

import "encoding/json"

// MarshalJSON 固定生活记录列表的前端 JSON 契约。
func (x *ListActivityLogsResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*ActivityLogItem, 0)
	}
	return json.Marshal(struct {
		Items    []*ActivityLogItem `json:"items"`
		Page     int32              `json:"page"`
		PageSize int32              `json:"pageSize"`
		Total    int64              `json:"total"`
		HasMore  bool               `json:"has_more"`
	}{items, x.Page, x.PageSize, x.Total, x.HasMore})
}

// MarshalJSON 固定生活记录条目的字段命名。
func (x *ActivityLogItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID              string              `json:"id"`
		Content         string              `json:"content"`
		OccurredAt      string              `json:"occurredAt"`
		DurationMinutes int32               `json:"durationMinutes"`
		InputType       string              `json:"inputType"`
		AudioURL        string              `json:"audioUrl"`
		CreatedBy       string              `json:"createdBy"`
		CreatedAt       string              `json:"createdAt"`
		UpdatedAt       string              `json:"updatedAt"`
		Creator         *ActivityLogCreator `json:"creator"`
	}{x.Id, x.Content, x.OccurredAt, x.DurationMinutes, x.InputType, x.AudioUrl, x.CreatedBy, x.CreatedAt, x.UpdatedAt, x.Creator})
}

func (x *ActivityLogCreator) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          string `json:"id"`
		Nickname    string `json:"nickname"`
		AvatarEmoji string `json:"avatarEmoji"`
		AvatarColor string `json:"avatarColor"`
	}{x.Id, x.Nickname, x.AvatarEmoji, x.AvatarColor})
}

// MarshalJSON 固定生活记录统计的字段命名，并保留空数组。
func (x *ActivityLogStatsResponse) MarshalJSON() ([]byte, error) {
	daily := x.DailyBreakdown
	if daily == nil {
		daily = make([]*ActivityLogDailyBreakdown, 0)
	}
	monthly := x.MonthlyBreakdown
	if monthly == nil {
		monthly = make([]*ActivityLogMonthlyBreakdown, 0)
	}
	return json.Marshal(struct {
		RecordCount         int32                          `json:"recordCount"`
		ActiveDays          int32                          `json:"activeDays"`
		TotalMinutes        int32                          `json:"totalMinutes"`
		DurationRecordCount int32                          `json:"durationRecordCount"`
		AverageMinutes      int32                          `json:"averageMinutes"`
		LongestMinutes      int32                          `json:"longestMinutes"`
		DailyBreakdown      []*ActivityLogDailyBreakdown   `json:"dailyBreakdown"`
		MonthlyBreakdown    []*ActivityLogMonthlyBreakdown `json:"monthlyBreakdown"`
	}{
		x.RecordCount,
		x.ActiveDays,
		x.TotalMinutes,
		x.DurationRecordCount,
		x.AverageMinutes,
		x.LongestMinutes,
		daily,
		monthly,
	})
}

func (x *ActivityLogDailyBreakdown) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Key         string `json:"key"`
		Label       string `json:"label"`
		RecordCount int32  `json:"recordCount"`
		Minutes     int32  `json:"minutes"`
	}{x.Key, x.Label, x.RecordCount, x.Minutes})
}

func (x *ActivityLogMonthlyBreakdown) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Month       int32  `json:"month"`
		Label       string `json:"label"`
		RecordCount int32  `json:"recordCount"`
		Minutes     int32  `json:"minutes"`
	}{x.Month, x.Label, x.RecordCount, x.Minutes})
}

// MarshalJSON 固定语音识别草稿的前端字段命名。
func (x *RecognizeActivityVoiceResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RawText         string `json:"rawText"`
		Content         string `json:"content"`
		OccurredAt      string `json:"occurredAt"`
		DurationMinutes int32  `json:"durationMinutes"`
		AudioDurationMS int64  `json:"audioDurationMs"`
		ASRRequestID    string `json:"asrRequestId"`
	}{x.RawText, x.Content, x.OccurredAt, x.DurationMinutes, x.AudioDurationMs, x.AsrRequestId})
}
