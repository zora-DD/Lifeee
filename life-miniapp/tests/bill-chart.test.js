const assert = require('node:assert/strict')
const test = require('node:test')

const { buildExpenseDonutItems, buildDonutGradient, formatPercent } = require('../utils/bill-chart')

test('环形图按金额排序并计算占比', () => {
  const items = buildExpenseDonutItems([
    { category: '餐饮', amount: '30', count: 2 },
    { category: '交通', amount: '70', count: 1 },
  ], 100)

  assert.equal(items[0].category, '交通')
  assert.equal(items[0].percentText, '70.0%')
  assert.equal(items[1].ratio, 0.3)
})

test('分类过多时合并为其他', () => {
  const categories = Array.from({ length: 8 }, (_, index) => ({
    category: `分类${index + 1}`,
    amount: 80 - index * 5,
    count: 1,
  }))
  const total = categories.reduce((sum, item) => sum + item.amount, 0)
  const items = buildExpenseDonutItems(categories, total, 6)

  assert.equal(items.length, 6)
  assert.equal(items[5].category, '其他')
  assert.equal(items[5].grouped, true)
  assert.equal(formatPercent(1, 1000), '0.1%')
})

test('环形图渐变完整覆盖一圈', () => {
  const gradient = buildDonutGradient([
    { amount: 3, color: '#111111' },
    { amount: 1, color: '#222222' },
  ])

  assert.equal(gradient, 'conic-gradient(#111111 0.00% 75.00%, #222222 75.00% 100.00%)')
})
