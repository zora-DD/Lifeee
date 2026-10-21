const WEEKDAYS = ['一', '二', '三', '四', '五', '六', '日']

function startOfDay(value) {
  const date = new Date(value)
  return new Date(date.getFullYear(), date.getMonth(), date.getDate())
}

function addDays(value, count) {
  const date = startOfDay(value)
  date.setDate(date.getDate() + count)
  return date
}

function startOfWeek(value) {
  const date = startOfDay(value)
  const offset = (date.getDay() + 6) % 7
  return addDays(date, -offset)
}

function formatDateKey(value) {
  const date = startOfDay(value)
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${date.getFullYear()}-${month}-${day}`
}

function formatLocalRFC3339(value) {
  const date = new Date(value)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  const hour = String(date.getHours()).padStart(2, '0')
  const minute = String(date.getMinutes()).padStart(2, '0')
  const second = String(date.getSeconds()).padStart(2, '0')
  const millisecond = String(date.getMilliseconds()).padStart(3, '0')
  const offsetMinutes = -date.getTimezoneOffset()
  const offsetSign = offsetMinutes >= 0 ? '+' : '-'
  const offsetHour = String(Math.floor(Math.abs(offsetMinutes) / 60)).padStart(2, '0')
  const offsetMinute = String(Math.abs(offsetMinutes) % 60).padStart(2, '0')
  return `${year}-${month}-${day}T${hour}:${minute}:${second}.${millisecond}${offsetSign}${offsetHour}:${offsetMinute}`
}

function parseDateKey(value) {
  const [year, month, day] = String(value || '').split('-').map(Number)
  return new Date(year, month - 1, day)
}

function getViewRange(mode, anchor) {
  if (mode === 'month') {
    const monthStart = new Date(anchor.getFullYear(), anchor.getMonth(), 1)
    const start = startOfWeek(monthStart)
    return { start, end: addDays(start, 42) }
  }
  const start = startOfWeek(anchor)
  return { start, end: addDays(start, 7) }
}

function buildDays(mode, anchor, tasks = []) {
  const { start, end } = getViewRange(mode, anchor)
  const count = Math.round((end.getTime() - start.getTime()) / 86400000)
  const todayKey = formatDateKey(new Date())
  const taskMap = new Map()
  tasks.forEach((task) => {
    if (!task.dueAt) return
    const key = formatDateKey(new Date(task.dueAt))
    if (!taskMap.has(key)) taskMap.set(key, [])
    taskMap.get(key).push(task)
  })
  return Array.from({ length: count }, (_, index) => {
    const date = addDays(start, index)
    const key = formatDateKey(date)
    const dayTasks = taskMap.get(key) || []
    return {
      key,
      day: date.getDate(),
      weekday: WEEKDAYS[index % 7],
      isToday: key === todayKey,
      isWeekend: index % 7 >= 5,
      inCurrentMonth: date.getMonth() === anchor.getMonth(),
      taskCount: dayTasks.length,
      todoCount: dayTasks.filter((task) => task.status !== 'done').length,
      markers: dayTasks.slice(0, 3).map((task) => ({
        id: task.id,
        priority: task.priority || 'medium',
        done: task.status === 'done',
      })),
      overflowCount: Math.max(0, dayTasks.length - 3),
    }
  })
}

function getDaySectionMinHeight(tasks = []) {
  const dailyCounts = new Map()
  tasks.forEach((task) => {
    if (!task.dueAt) return
    const key = formatDateKey(new Date(task.dueAt))
    dailyCounts.set(key, (dailyCounts.get(key) || 0) + 1)
  })
  const maxCount = Math.max(0, ...dailyCounts.values())
  // 标题区、空状态和任务卡片预留稳定空间，避免切换日期时页面高度突变。
  return Math.max(250, 110 + maxCount * 125)
}

function formatPeriodTitle(mode, anchor) {
  if (mode === 'month') return `${anchor.getFullYear()}年${anchor.getMonth() + 1}月`
  const { start, end } = getViewRange('week', anchor)
  const last = addDays(end, -1)
  if (start.getFullYear() !== last.getFullYear()) {
    return `${start.getFullYear()}年${start.getMonth() + 1}月${start.getDate()}日 – ${last.getFullYear()}年${last.getMonth() + 1}月${last.getDate()}日`
  }
  return `${start.getMonth() + 1}月${start.getDate()}日 – ${last.getMonth() + 1}月${last.getDate()}日`
}

function formatSelectedDate(value) {
  const date = parseDateKey(value)
  const weekday = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()]
  return `${date.getMonth() + 1}月${date.getDate()}日 ${weekday}`
}

module.exports = {
  addDays,
  buildDays,
  formatDateKey,
  formatLocalRFC3339,
  formatPeriodTitle,
  formatSelectedDate,
  getDaySectionMinHeight,
  getViewRange,
  parseDateKey,
  startOfWeek,
}
