const assert = require('node:assert/strict')
const test = require('node:test')

const { buildCurrentYearMonthOptions, getMonthRange } = require('../utils/month-filter')

test('月份筛选只展示本年已到来的月份', () => {
  const options = buildCurrentYearMonthOptions(new Date(2026, 8, 15))

  assert.equal(options.length, 9)
  assert.deepEqual(options[0], { year: 2026, month: 1, label: '1月', shortLabel: '1月' })
  assert.deepEqual(options[8], { year: 2026, month: 9, label: '本月', shortLabel: '9月' })
})

test('指定月份生成左闭右开的时间范围', () => {
  const range = getMonthRange({ year: 2026, month: 9, label: '2026年9月' })

  assert.equal(range.scope, 'month')
  assert.equal(new Date(range.from).getFullYear(), 2026)
  assert.equal(new Date(range.from).getMonth(), 8)
  assert.equal(new Date(range.to).getMonth(), 9)
  assert.equal(getMonthRange({ year: 0, month: 0 }), null)
})
