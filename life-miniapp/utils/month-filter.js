function buildCurrentYearMonthOptions(now = new Date()) {
  const currentMonth = now.getMonth() + 1
  const options = []
  for (let month = 1; month <= currentMonth; month += 1) {
    options.push({
      year: now.getFullYear(),
      month,
      label: month === currentMonth ? '本月' : `${month}月`,
      shortLabel: `${month}月`,
    })
  }
  return options
}

function getMonthRange(option) {
  if (!option || !Number(option.year) || !Number(option.month)) return null
  const from = new Date(Number(option.year), Number(option.month) - 1, 1)
  const to = new Date(Number(option.year), Number(option.month), 1)
  return {
    scope: 'month',
    year: Number(option.year),
    month: Number(option.month),
    label: option.label || `${option.year}年${option.month}月`,
    from: from.toISOString(),
    to: to.toISOString(),
  }
}

module.exports = { buildCurrentYearMonthOptions, getMonthRange }
