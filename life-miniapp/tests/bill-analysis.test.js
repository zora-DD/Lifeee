const assert = require('node:assert/strict')
const test = require('node:test')

const { buildBillInsights, formatMoney, getBillPeriod } = require('../utils/bill-analysis')

test('金额格式化保留两位小数并添加千分位', () => {
  assert.equal(formatMoney(1080), '1,080.00')
  assert.equal(formatMoney('25.5'), '25.50')
})

test('消费建议识别主要分类和近期增长', () => {
  const insights = buildBillInsights({
    expenseTotal: 1080,
    year: new Date().getFullYear(),
    categories: [
      { category: '购物', amount: '720', count: 3 },
      { category: '餐饮', amount: '360', count: 4 },
    ],
    monthly: [
      { month: 7, expenseAmount: '300' },
      { month: 8, expenseAmount: '780' },
    ],
  })

  assert.equal(insights.length, 3)
  assert.match(insights[0].text, /购物占全年支出的 67%/)
  assert.match(insights[1].title, /支出有所上升/)
  assert.match(insights[2].text, /平均每笔/)
})

test('没有消费时不生成建议', () => {
  assert.deepEqual(buildBillInsights({ expenseTotal: 0 }), [])
})

test('月份筛选生成对应月份范围和建议文案', () => {
  const period = getBillPeriod(2026, 8)
  const insights = buildBillInsights({
    expenseTotal: 200,
    categories: [{ category: '餐饮', amount: '200', count: 2 }],
    scope: period.scope,
    year: period.year,
    month: period.month,
  })

  assert.equal(period.scope, 'month')
  assert.equal(period.label, '2026年8月')
  assert.equal(new Date(period.from).getMonth(), 7)
  assert.equal(new Date(period.to).getMonth(), 8)
  assert.match(insights[0].text, /8月支出/)
  assert.match(insights.at(-1).text, /8月共记录 2 笔/)
})

test('月份留空时统计全年', () => {
  const period = getBillPeriod(2026, 0)

  assert.equal(period.scope, 'year')
  assert.equal(period.label, '2026年')
})
