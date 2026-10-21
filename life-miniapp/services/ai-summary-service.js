const { request } = require('./request')

const API_PATHS = {
  taskSummary: '/v1/life-core/task/ai_summary',
  lifeSummary: '/v1/life-core/report/life_summary',
}

function normalizeTaskSummary(result = {}) {
  const focusTasks = result.focus_tasks ?? result.focusTasks
  return {
    ...result,
    date: result.date || '',
    todoCount: result.todo_count ?? result.todoCount ?? 0,
    doneTodayCount: result.done_today_count ?? result.doneTodayCount ?? 0,
    overdueCount: result.overdue_count ?? result.overdueCount ?? 0,
    dueTodayCount: result.due_today_count ?? result.dueTodayCount ?? 0,
    upcomingCount: result.upcoming_count ?? result.upcomingCount ?? 0,
    highPriorityCount: result.high_priority_count ?? result.highPriorityCount ?? 0,
    focusTasks: Array.isArray(focusTasks) ? focusTasks.map((item) => ({
      ...item,
      dueAt: item.due_at ?? item.dueAt ?? '',
    })) : [],
    suggestions: Array.isArray(result.suggestions) ? result.suggestions : [],
    cached: Boolean(result.cached),
    aiGenerated: Boolean(result.ai_generated ?? result.aiGenerated),
    generatedAt: result.generated_at ?? result.generatedAt ?? '',
  }
}

function normalizeLifeSummary(result = {}) {
  return {
    ...result,
    periodLabel: result.period_label ?? result.periodLabel ?? '',
    taskCreatedCount: result.task_created_count ?? result.taskCreatedCount ?? 0,
    taskDoneCount: result.task_done_count ?? result.taskDoneCount ?? 0,
    activityCount: result.activity_count ?? result.activityCount ?? 0,
    activityMinutes: result.activity_minutes ?? result.activityMinutes ?? 0,
    billCount: result.bill_count ?? result.billCount ?? 0,
    expenseTotal: result.expense_total ?? result.expenseTotal ?? '0.00',
    restaurantVisitCount: result.restaurant_visit_count ?? result.restaurantVisitCount ?? 0,
    visitedPlaceCount: result.visited_place_count ?? result.visitedPlaceCount ?? 0,
    highlights: Array.isArray(result.highlights) ? result.highlights : [],
    suggestions: Array.isArray(result.suggestions) ? result.suggestions : [],
    cached: Boolean(result.cached),
    aiGenerated: Boolean(result.ai_generated ?? result.aiGenerated),
    generatedAt: result.generated_at ?? result.generatedAt ?? '',
  }
}

const AISummaryService = {
  taskSummary({ refresh = false } = {}) {
    return request({ path: API_PATHS.taskSummary, method: 'POST', data: { refresh } })
      .then(normalizeTaskSummary)
  },

  lifeSummary({ scope = 'month', year, month, refresh = false } = {}) {
    return request({
      path: API_PATHS.lifeSummary,
      method: 'POST',
      data: { scope, year, month, refresh },
    }).then(normalizeLifeSummary)
  },
}

module.exports = { API_PATHS, AISummaryService }
