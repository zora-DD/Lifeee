const { formatDateTime } = require('./date')

const priorityText = { high: '高', medium: '中', low: '低' }
const preferenceText = { okay: '还行', good: '不错', love: '♥ 喜欢', must: '🔥 必吃' }

function toContentCard(item) {
  const type = item.targetType
  if (type === 'task') {
    return {
      ...item,
      id: item.targetId,
      type,
      icon: '✓',
      typeLabel: '任务',
      meta: `${priorityText[item.priority] || '中'} · ${item.status === 'done' ? '已完成' : '待完成'}`,
      detail: `截止 ${formatDateTime(item.dueAt)}`,
      url: `/pages/tasks/detail?id=${item.targetId}`,
    }
  }
  if (type === 'restaurant') {
    return {
      ...item,
      id: item.targetId,
      type,
      icon: '吃',
      typeLabel: '吃喝',
      meta: `${item.category || '其他'} · ${preferenceText[item.status] || '不错'}`,
      detail: `消费 ${item.visitCount || 0} 次 · 累计 ¥${Number(item.totalSpent || 0)}`,
      url: `/pages/restaurants/detail?id=${item.targetId}`,
    }
  }
  return {
    ...item,
    id: item.targetId,
    type: 'place',
    icon: item.icon || '⌖',
    typeLabel: '地点',
    meta: item.category || '其他',
    detail: item.visited ? `去过 ${item.visitCount || 0} 次` : '还没去过 · 想去',
    url: `/pages/places/detail?id=${item.targetId}`,
  }
}

module.exports = { toContentCard }
