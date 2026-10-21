const assert = require('node:assert/strict')
const test = require('node:test')

const { formatDateTimeWithWeekday, getWeekdayLabel } = require('../utils/date')

test('截止日期显示对应星期并区分周末', () => {
  assert.equal(getWeekdayLabel('2026-09-18'), '周五')
  assert.equal(getWeekdayLabel('2026-09-19'), '周六')
  assert.equal(getWeekdayLabel('2026-09-20'), '周日')
})

test('空日期不显示星期', () => {
  assert.equal(getWeekdayLabel(''), '')
})

test('任务列表截止时间包含星期', () => {
  assert.match(formatDateTimeWithWeekday('2026-09-20T10:00:00.000Z'), /9月20日 周日 · /)
})
