const SOURCE_TYPES = ['task', 'restaurant', 'place', 'log']

const priorityText = {
  high: '高',
  medium: '中',
  low: '低',
}

const preferenceText = {
  okay: '还行',
  good: '不错',
  love: '喜欢',
  must: '必吃',
}

function pad(value) {
  return String(value).padStart(2, '0')
}

function truncate(value, maxLength) {
  const text = String(value || '').replace(/\s+/g, ' ').trim()
  return text.length > maxLength ? `${text.slice(0, maxLength)}…` : text
}

function daySerial(now = new Date()) {
  return Math.floor(Date.UTC(now.getFullYear(), now.getMonth(), now.getDate()) / 86400000)
}

function getDailySourceIndex(now = new Date()) {
  return Math.abs(daySerial(now)) % SOURCE_TYPES.length
}

function pickDaily(items, now = new Date(), offset = 0) {
  if (!Array.isArray(items) || !items.length) return null
  return items[(Math.abs(daySerial(now)) + offset) % items.length]
}

function visualVariant(id, now = new Date()) {
  const hash = Array.from(String(id || 'inspiration'))
    .reduce((total, char) => total + char.charCodeAt(0), daySerial(now))
  const motions = ['rise', 'slide', 'zoom']
  return {
    layout: hash % 2 === 0 ? 'left' : 'right',
    visualSize: hash % 3 === 0 ? 'large' : 'medium',
    motion: motions[Math.abs(hash) % motions.length],
  }
}

function decorateVisual(inspiration, id, now, imageUrl = '') {
  return {
    ...inspiration,
    ...visualVariant(id, now),
    imageUrl: imageUrl || '',
  }
}

function formatDue(value, now = new Date()) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return '截止时间待确定'

  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
  const target = new Date(date.getFullYear(), date.getMonth(), date.getDate())
  const dayDiff = Math.round((target.getTime() - today.getTime()) / 86400000)
  const time = `${pad(date.getHours())}:${pad(date.getMinutes())}`
  if (dayDiff < 0) return `已逾期 ${Math.abs(dayDiff)} 天`
  if (dayDiff === 0) return `今天 ${time} 截止`
  if (dayDiff === 1) return `明天 ${time} 截止`
  return `${date.getMonth() + 1}月${date.getDate()}日 ${time} 截止`
}

function buildTaskInspiration(tasks, now = new Date()) {
  const candidates = (Array.isArray(tasks) ? tasks : [])
    .filter((task) => task && task.id && task.status !== 'done')
    .sort((left, right) => {
      const leftTime = new Date(left.dueAt).getTime()
      const rightTime = new Date(right.dueAt).getTime()
      return (Number.isFinite(leftTime) ? leftTime : Number.MAX_SAFE_INTEGER)
        - (Number.isFinite(rightTime) ? rightTime : Number.MAX_SAFE_INTEGER)
    })
  const task = candidates[0]
  if (!task) return null

  return decorateVisual({
    type: 'task',
    icon: '✓',
    eyebrow: '先完成一件事',
    title: task.title,
    description: `${formatDue(task.dueAt, now)} · ${priorityText[task.priority] || '中'}优先级`,
    sourceLabel: '来自任务',
    path: `/pages/tasks/detail?id=${encodeURIComponent(task.id)}`,
  }, task.id, now, task.coverUrl)
}

function buildRestaurantInspiration(restaurant, now = new Date()) {
  if (!restaurant || !restaurant.id) return null
  const details = [
    restaurant.category || '吃喝',
    preferenceText[restaurant.preference] || '不错',
  ]
  if (Number(restaurant.visitCount) > 0) details.push(`已去 ${Number(restaurant.visitCount)} 次`)

  return decorateVisual({
    type: 'restaurant',
    icon: '吃',
    eyebrow: '今天吃点什么',
    title: restaurant.name,
    description: details.join(' · '),
    sourceLabel: '来自吃喝',
    path: `/pages/restaurants/detail?id=${encodeURIComponent(restaurant.id)}`,
  }, restaurant.id, now, restaurant.coverUrl)
}

function buildPlaceInspiration(places, now = new Date()) {
  const candidates = (Array.isArray(places) ? places : []).filter((place) => place && place.id && !place.visited)
  const place = pickDaily(candidates, now, 11)
  if (!place) return null

  return decorateVisual({
    type: 'place',
    icon: '游',
    eyebrow: '找个地方走走',
    title: place.name,
    description: truncate([place.category, place.feature].filter(Boolean).join(' · '), 42) || '还没去过，可以收进近期计划',
    sourceLabel: '来自地点',
    path: `/pages/places/detail?id=${encodeURIComponent(place.id)}`,
  }, place.id, now, place.coverUrl)
}

function buildLogInspiration(logs, now = new Date()) {
  const candidates = (Array.isArray(logs) ? logs : []).filter((log) => log && log.id && log.content)
  const log = pickDaily(candidates, now, 23)
  if (!log) return null
  const occurredAt = new Date(log.occurredAt)
  const dateText = Number.isNaN(occurredAt.getTime())
    ? '以前的一天'
    : `${occurredAt.getFullYear()}年${occurredAt.getMonth() + 1}月${occurredAt.getDate()}日`
  const durationText = Number(log.durationMinutes) > 0 ? ` · ${Number(log.durationMinutes)} 分钟` : ''

  return decorateVisual({
    type: 'log',
    icon: '记',
    eyebrow: '偶遇旧时光',
    title: truncate(log.content, 34),
    description: `${dateText}${durationText}`,
    sourceLabel: '来自生活记录',
    path: `/pages/logs/detail?id=${encodeURIComponent(log.id)}`,
  }, log.id, now, log.coverUrl)
}

module.exports = {
  SOURCE_TYPES,
  buildLogInspiration,
  buildPlaceInspiration,
  buildRestaurantInspiration,
  buildTaskInspiration,
  getDailySourceIndex,
}
