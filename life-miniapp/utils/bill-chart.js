const DONUT_COLORS = ['#4fae7b', '#78c29a', '#a9d6be', '#d2b48c', '#7fa7a1', '#b8a4c8']

function formatPercent(amount, total) {
  if (!total) return '0%'
  const percent = amount / total * 100
  if (percent < 1) return `${percent.toFixed(1)}%`
  return `${percent.toFixed(1)}%`
}

function buildExpenseDonutItems(categories = [], expenseTotal = 0, maxItems = 6) {
  const total = Number(expenseTotal) || 0
  if (total <= 0) return []

  const sorted = categories
    .map((item) => ({ ...item, amount: Number(item.amount) || 0, count: Number(item.count) || 0 }))
    .filter((item) => item.amount > 0)
    .sort((left, right) => right.amount - left.amount)

  const limit = Math.max(2, Number(maxItems) || 6)
  let displayItems = sorted
  if (sorted.length > limit) {
    const visible = sorted.slice(0, limit - 1)
    const remainder = sorted.slice(limit - 1)
    displayItems = visible.concat({
      category: visible.some((item) => item.category === '其他') ? '其余' : '其他',
      amount: remainder.reduce((sum, item) => sum + item.amount, 0),
      count: remainder.reduce((sum, item) => sum + item.count, 0),
      grouped: true,
    })
  }

  return displayItems.map((item, index) => ({
    ...item,
    color: DONUT_COLORS[index % DONUT_COLORS.length],
    ratio: item.amount / total,
    percentText: formatPercent(item.amount, total),
  }))
}

function buildDonutGradient(items = []) {
  const total = items.reduce((sum, item) => sum + (Number(item.amount) || 0), 0)
  if (total <= 0) return '#e8efeb'

  let current = 0
  const stops = items.map((item, index) => {
    const start = current
    current += (Number(item.amount) || 0) / total * 100
    const end = index === items.length - 1 ? 100 : current
    return `${item.color} ${start.toFixed(2)}% ${end.toFixed(2)}%`
  })
  return `conic-gradient(${stops.join(', ')})`
}

module.exports = { DONUT_COLORS, buildExpenseDonutItems, buildDonutGradient, formatPercent }
