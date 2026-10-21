function formatMoney(value) {
  const amount = Number(value) || 0
  const [integer, fraction] = amount.toFixed(2).split('.')
  return `${integer.replace(/\B(?=(\d{3})+(?!\d))/g, ',')}.${fraction}`
}

function getBillPeriod(year, month = 0) {
  const normalizedYear = Number(year) || new Date().getFullYear()
  const normalizedMonth = Number(month)
  const hasMonth = normalizedMonth >= 1 && normalizedMonth <= 12
  const from = hasMonth
    ? new Date(normalizedYear, normalizedMonth - 1, 1)
    : new Date(normalizedYear, 0, 1)
  const to = hasMonth
    ? new Date(normalizedYear, normalizedMonth, 1)
    : new Date(normalizedYear + 1, 0, 1)

  return {
    scope: hasMonth ? 'month' : 'year',
    year: normalizedYear,
    month: hasMonth ? normalizedMonth : 0,
    label: hasMonth ? `${normalizedYear}年${normalizedMonth}月` : `${normalizedYear}年`,
    from: from.toISOString(),
    to: to.toISOString(),
  }
}

function buildBillInsights({ expenseTotal, categories = [], monthly = [], year, month = 0, scope = 'year' }) {
  const total = Number(expenseTotal) || 0
  if (total <= 0) return []

  const insights = []
  const periodText = scope === 'month' && Number(month) ? `${Number(month)}月` : '全年'
  const sortedCategories = [...categories].sort((a, b) => Number(b.amount) - Number(a.amount))
  const topCategory = sortedCategories[0]
  if (topCategory) {
    const percent = Math.round((Number(topCategory.amount) || 0) / total * 100)
    insights.push({
      type: percent >= 50 ? 'attention' : 'steady',
      title: `主要支出在${topCategory.category}`,
      text: percent >= 50
        ? `${topCategory.category}占${periodText}支出的 ${percent}%，可以先看看其中是否有可减少的小额高频消费。`
        : `${topCategory.category}占${periodText}支出的 ${percent}%，目前支出结构相对分散。`,
    })
  }

  if (scope === 'year') {
    const currentYear = new Date().getFullYear()
    const monthLimit = Number(year) === currentYear ? new Date().getMonth() + 1 : 12
    const available = monthly
      .filter((item) => Number(item.month) <= monthLimit && Number(item.expenseAmount) > 0)
      .sort((a, b) => Number(a.month) - Number(b.month))
    const latest = available[available.length - 1]
    const previous = available.slice(Math.max(0, available.length - 4), -1)
    if (latest && previous.length) {
      const latestAmount = Number(latest.expenseAmount) || 0
      const average = previous.reduce((sum, item) => sum + Number(item.expenseAmount || 0), 0) / previous.length
      const change = average ? Math.round((latestAmount - average) / average * 100) : 0
      if (change > 20) {
        insights.push({
          type: 'attention',
          title: '近期支出有所上升',
          text: `${latest.month}月支出比此前有消费记录月份的均值高约 ${change}%，可以回看主要增加在哪一类。`,
        })
      } else if (change < -20) {
        insights.push({
          type: 'good',
          title: '近期支出有所回落',
          text: `${latest.month}月支出比此前有消费记录月份的均值低约 ${Math.abs(change)}%，当前节奏保持得不错。`,
        })
      } else {
        insights.push({
          type: 'steady',
          title: '近期消费较平稳',
          text: `${latest.month}月支出与此前有消费记录月份的均值接近，可以继续保持记录。`,
        })
      }
    } else {
      insights.push({
        type: 'steady',
        title: '继续积累月度记录',
        text: '目前月度数据还比较少，再记录一段时间后会给出更准确的变化建议。',
      })
    }
  }

  const expenseCount = sortedCategories.reduce((sum, item) => sum + (Number(item.count) || 0), 0)
  if (expenseCount > 0) {
    insights.push({
      type: 'tip',
      title: '留意单笔消费',
      text: `${periodText}共记录 ${expenseCount} 笔支出，平均每笔 ¥${formatMoney(total / expenseCount)}。给大额消费补充备注，月底会更容易复盘。`,
    })
  }
  return insights.slice(0, 3)
}

module.exports = { buildBillInsights, formatMoney, getBillPeriod }
