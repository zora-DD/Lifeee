package service

// Recognition 表示语音识别结果。
type Recognition struct {
	Text            string
	AudioDurationMS int64
	RequestID       string
}

// ActivityDraft 表示由大模型整理、但尚未保存的生活记录草稿。
type ActivityDraft struct {
	Content         string `json:"content"`
	OccurredAt      string `json:"occurred_at"`
	DurationMinutes int32  `json:"duration_minutes"`
}

// TaskDraft 表示由大模型整理、但尚未保存的任务草稿。
type TaskDraft struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	Category    string `json:"category"`
}

// BillDraft 表示从一句话中拆出的单条账单草稿。
type BillDraft struct {
	Type       string `json:"type"`
	Title      string `json:"title"`
	Amount     string `json:"amount"`
	Category   string `json:"category"`
	OccurredAt string `json:"occurred_at"`
	Note       string `json:"note"`
}

// QuickRecordDraft 表示统一“快速记”入口的意图识别结果。
type QuickRecordDraft struct {
	Intent   string         `json:"intent"`
	Activity *ActivityDraft `json:"activity"`
	Bills    []BillDraft    `json:"bills"`
}

// TaskSummaryFact 是传给大模型的任务事实，所有数量均由服务端计算。
type TaskSummaryFact struct {
	Date              string          `json:"date"`
	TodoCount         int             `json:"todo_count"`
	DoneTodayCount    int             `json:"done_today_count"`
	OverdueCount      int             `json:"overdue_count"`
	DueTodayCount     int             `json:"due_today_count"`
	UpcomingCount     int             `json:"upcoming_count"`
	HighPriorityCount int             `json:"high_priority_count"`
	FocusTasks        []TaskFocusFact `json:"focus_tasks"`
}

type TaskFocusFact struct {
	Title    string `json:"title"`
	Priority string `json:"priority"`
	DueAt    string `json:"due_at"`
	Reason   string `json:"reason"`
}

// LifeSummaryFact 是个人生活报告的结构化事实。
type LifeSummaryFact struct {
	PeriodLabel          string `json:"period_label"`
	TaskCreatedCount     int    `json:"task_created_count"`
	TaskDoneCount        int    `json:"task_done_count"`
	ActivityCount        int    `json:"activity_count"`
	ActivityMinutes      int    `json:"activity_minutes"`
	BillCount            int    `json:"bill_count"`
	ExpenseTotal         string `json:"expense_total"`
	RestaurantVisitCount int    `json:"restaurant_visit_count"`
	VisitedPlaceCount    int    `json:"visited_place_count"`
}

// SummaryDraft 是大模型生成的简短总结与建议。
type SummaryDraft struct {
	Summary     string   `json:"summary"`
	Highlights  []string `json:"highlights"`
	Suggestions []string `json:"suggestions"`
}
