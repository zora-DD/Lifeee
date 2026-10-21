const assert = require('node:assert/strict')
const test = require('node:test')

const {
  buildDays,
  formatDateKey,
  formatLocalRFC3339,
  formatPeriodTitle,
  getDaySectionMinHeight,
  getViewRange,
} = require('../utils/task-calendar')

test('周视图从周一开始并生成七天', () => {
  const anchor = new Date(2026, 8, 18)
  const { start, end } = getViewRange('week', anchor)
  const days = buildDays('week', anchor)

  assert.equal(formatDateKey(start), '2026-09-14')
  assert.equal(formatDateKey(end), '2026-09-21')
  assert.equal(days.length, 7)
  assert.equal(days[0].weekday, '一')
  assert.equal(days[6].weekday, '日')
  assert.equal(formatPeriodTitle('week', anchor), '9月14日 – 9月20日')
})

test('月视图生成六周并汇总每日任务标记', () => {
  const anchor = new Date(2026, 8, 18)
  const tasks = [
    { id: '1', dueAt: new Date(2026, 8, 18, 9).toISOString(), priority: 'high', status: 'todo' },
    { id: '2', dueAt: new Date(2026, 8, 18, 18).toISOString(), priority: 'low', status: 'done' },
  ]
  const days = buildDays('month', anchor, tasks)
  const target = days.find((day) => day.key === '2026-09-18')

  assert.equal(days.length, 42)
  assert.equal(target.taskCount, 2)
  assert.equal(target.todoCount, 1)
  assert.deepEqual(target.markers.map((item) => item.priority), ['high', 'low'])
  assert.equal(formatPeriodTitle('month', anchor), '2026年9月')
})

test('接口日期范围保留本地时区和自然日边界', () => {
  const value = formatLocalRFC3339(new Date(2026, 8, 14, 0, 0, 0, 0))
  assert.match(value, /^2026-09-14T00:00:00\.000[+-]\d{2}:\d{2}$/)
})

test('日期任务区域按本周期最大任务数保持稳定高度', () => {
  const tasks = [
    { dueAt: new Date(2026, 8, 18, 9).toISOString() },
    { dueAt: new Date(2026, 8, 18, 10).toISOString() },
    { dueAt: new Date(2026, 8, 19, 9).toISOString() },
  ]

  assert.equal(getDaySectionMinHeight([]), 250)
  assert.equal(getDaySectionMinHeight(tasks), 360)
})

test('任务列表请求会传递日历日期范围', async (t) => {
  const originalWx = global.wx
  let requestData
  global.wx = {
    getStorageSync: () => 'test-token',
    request(options) {
      requestData = options.data
      options.success({
        statusCode: 200,
        data: { code: 0, data: { items: [], page: 1, pageSize: 100, total: 0, has_more: false } },
      })
    },
  }
  t.after(() => { global.wx = originalWx })
  delete require.cache[require.resolve('../services/task-service')]
  delete require.cache[require.resolve('../services/request')]
  const { TaskService } = require('../services/task-service')

  await TaskService.listPage({
    dueFrom: '2026-09-14T00:00:00.000+08:00',
    dueTo: '2026-09-20T23:59:59.999+08:00',
    pageSize: 100,
  })

  assert.equal(requestData.dueFrom, '2026-09-14T00:00:00.000+08:00')
  assert.equal(requestData.dueTo, '2026-09-20T23:59:59.999+08:00')
})
