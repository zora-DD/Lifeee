const { request } = require('./request')

const API_PATHS = {
  list: '/v1/life-core/task/routine/list',
  create: '/v1/life-core/task/routine/create',
  update: '/v1/life-core/task/routine/modify',
  remove: '/v1/life-core/task/routine/delete',
  checkin: '/v1/life-core/task/routine/checkin',
}

function normalizeRoutine(item = {}) {
  const normalizeDay = (day = {}) => ({
    ...day,
    completed: Boolean(day.completed),
    scheduled: Boolean(day.scheduled ?? true),
    checkedAt: day.checkedAt || day.checked_at || '',
  })
  return {
    ...item,
    id: String(item.id || ''),
    timeOfDay: item.timeOfDay || item.time_of_day || '09:00',
    enabled: Boolean(item.enabled),
    checkedToday: Boolean(item.checkedToday ?? item.checked_today),
    checkedAt: item.checkedAt || item.checked_at || '',
    createdAt: item.createdAt || item.created_at || '',
    updatedAt: item.updatedAt || item.updated_at || '',
    currentStreak: Number(item.currentStreak ?? item.current_streak) || 0,
    totalCheckins: Number(item.totalCheckins ?? item.total_checkins) || 0,
    last7DayRate: Number(item.lastSevenDayRate ?? item.last_seven_day_rate) || 0,
    repeatUnit: item.repeatUnit || item.repeat_unit || 'day',
    repeatInterval: Number(item.repeatInterval ?? item.repeat_interval) || 1,
    startDate: item.startDate || item.start_date || '',
    scheduledToday: Boolean(item.scheduledToday ?? item.scheduled_today ?? true),
    history: Array.isArray(item.history) ? item.history.map(normalizeDay) : [],
    monthHistory: Array.isArray(item.monthHistory ?? item.month_history)
      ? (item.monthHistory ?? item.month_history).map(normalizeDay)
      : [],
  }
}

function toRoutineInput(input = {}) {
  return {
    title: input.title,
    time_of_day: input.timeOfDay,
    repeat_unit: input.repeatUnit,
    repeat_interval: input.repeatInterval,
    start_date: input.startDate,
    enabled: input.enabled,
  }
}

const DailyRoutineService = {
  list({ date, includeDisabled = false } = {}) {
    return request({
      path: API_PATHS.list,
      method: 'POST',
      data: { date, include_disabled: includeDisabled },
    }).then((result) => ({
      date: result?.date || date || '',
      items: Array.isArray(result?.items) ? result.items.map(normalizeRoutine) : [],
    }))
  },

  create(input) {
    return request({ path: API_PATHS.create, method: 'POST', data: toRoutineInput(input) }).then(normalizeRoutine)
  },

  update(id, input) {
    return request({ path: API_PATHS.update, method: 'POST', data: { id, ...toRoutineInput(input) } }).then(normalizeRoutine)
  },

  remove(id) {
    return request({ path: API_PATHS.remove, method: 'POST', data: { id } })
  },

  checkin(id, date, completed) {
    return request({
      path: API_PATHS.checkin,
      method: 'POST',
      data: { id, date, completed },
    }).then(normalizeRoutine)
  },
}

module.exports = { API_PATHS, DailyRoutineService, normalizeRoutine, toRoutineInput }
