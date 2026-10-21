const assert = require('node:assert/strict')
const test = require('node:test')

const { buildQuickPostponeOptions, getTodayRange } = require('../utils/task-schedule')

test('今日筛选覆盖本地自然日', () => {
  const range = getTodayRange(new Date(2026, 8, 18, 12, 30))
  assert.match(range.dueFrom, /^2026-09-18T00:00:00\.000[+-]\d{2}:\d{2}$/)
  assert.match(range.dueTo, /^2026-09-18T23:59:59\.999[+-]\d{2}:\d{2}$/)
})

test('快速延期提供今天、明天和一周后', () => {
  const options = buildQuickPostponeOptions(
    new Date(2026, 8, 18, 12, 30),
    new Date(2026, 8, 10, 18, 0).toISOString(),
  )

  assert.deepEqual(options.map((item) => item.key), ['today', 'tomorrow', 'next_week'])
  assert.match(options[0].label, /^今天 · 9月18日 周五 · 23:59$/)
  assert.match(options[1].label, /^明天 · 9月19日 周六 · 18:00$/)
  assert.match(options[2].label, /^一周后 · 9月25日 周五 · 18:00$/)
})
