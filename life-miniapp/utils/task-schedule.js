const { formatDateKey, formatLocalRFC3339 } = require('./task-calendar')
const { getWeekdayLabel } = require('./date')

function startOfLocalDay(value) {
  const date = new Date(value)
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function addLocalDays(value, count) {
  const date = startOfLocalDay(value)
  date.setDate(date.getDate() + count)
  return date
}

function getTodayRange(now = new Date()) {
  const start = startOfLocalDay(now)
  const end = addLocalDays(start, 1)
  return {
    dueFrom: formatLocalRFC3339(start),
    dueTo: formatLocalRFC3339(new Date(end.getTime() - 1)),
  }
}

function getOriginalTime(value) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return { hour: 18, minute: 0 }
  return { hour: date.getHours(), minute: date.getMinutes() }
}

function withTime(value, hour, minute) {
  const date = startOfLocalDay(value)
  date.setHours(hour, minute, 0, 0)
  return date
}

function formatOptionDate(date) {
  const key = formatDateKey(date)
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  return `${date.getMonth() + 1}月${date.getDate()}日 ${getWeekdayLabel(key)} · ${hour}:${minute}`
}

function buildQuickPostponeOptions(now = new Date(), currentDueAt = '') {
  const { hour, minute } = getOriginalTime(currentDueAt)
  const today = withTime(now, 23, 59)
  const tomorrow = withTime(addLocalDays(now, 1), hour, minute)
  const nextWeek = withTime(addLocalDays(now, 7), hour, minute)
  return [
    { key: 'today', label: `今天 · ${formatOptionDate(today)}`, dueAt: today.toISOString() },
    { key: 'tomorrow', label: `明天 · ${formatOptionDate(tomorrow)}`, dueAt: tomorrow.toISOString() },
    { key: 'next_week', label: `一周后 · ${formatOptionDate(nextWeek)}`, dueAt: nextWeek.toISOString() },
  ]
}

module.exports = {
  buildQuickPostponeOptions,
  getTodayRange,
}
