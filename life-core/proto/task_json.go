package proto

import "encoding/json"

// MarshalJSON 固定任务列表接口的 JSON 契约，并确保空列表返回 []。
func (x *ListTasksResponse) MarshalJSON() ([]byte, error) {
	items := x.Items
	if items == nil {
		items = make([]*TaskItem, 0)
	}
	return json.Marshal(struct {
		Items        []*TaskItem `json:"items"`
		Page         int32       `json:"page"`
		PageSize     int32       `json:"pageSize"`
		Total        int64       `json:"total"`
		HasMore      bool        `json:"has_more"`
		OverallTotal int64       `json:"overallTotal"`
		TodoCount    int64       `json:"todoCount"`
		DoneCount    int64       `json:"doneCount"`
	}{
		Items:        items,
		Page:         x.Page,
		PageSize:     x.PageSize,
		Total:        x.Total,
		HasMore:      x.HasMore,
		OverallTotal: x.OverallTotal,
		TodoCount:    x.TodoCount,
		DoneCount:    x.DoneCount,
	})
}

// MarshalJSON 固定任务条目的 JSON 字段命名。
func (x *TaskItem) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          string       `json:"id"`
		Title       string       `json:"title"`
		Description string       `json:"description"`
		Status      string       `json:"status"`
		Priority    string       `json:"priority"`
		Category    string       `json:"category"`
		DueAt       string       `json:"dueAt"`
		CreatedBy   string       `json:"createdBy"`
		CreatedAt   string       `json:"createdAt"`
		UpdatedAt   string       `json:"updatedAt"`
		Creator     *TaskCreator `json:"creator"`
	}{
		ID:          x.Id,
		Title:       x.Title,
		Description: x.Description,
		Status:      x.Status,
		Priority:    x.Priority,
		Category:    x.Category,
		DueAt:       x.DueAt,
		CreatedBy:   x.CreatedBy,
		CreatedAt:   x.CreatedAt,
		UpdatedAt:   x.UpdatedAt,
		Creator:     x.Creator,
	})
}

func (x *TaskCreator) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID          string `json:"id"`
		Nickname    string `json:"nickname"`
		AvatarEmoji string `json:"avatarEmoji"`
		AvatarColor string `json:"avatarColor"`
	}{x.Id, x.Nickname, x.AvatarEmoji, x.AvatarColor})
}

// MarshalJSON 固定语音任务草稿的前端字段命名。
func (x *RecognizeTaskVoiceResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		RawText         string `json:"rawText"`
		Title           string `json:"title"`
		Description     string `json:"description"`
		Category        string `json:"category"`
		AudioDurationMS int64  `json:"audioDurationMs"`
		ASRRequestID    string `json:"asrRequestId"`
	}{x.RawText, x.Title, x.Description, x.Category, x.AudioDurationMs, x.AsrRequestId})
}
