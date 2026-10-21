const assert = require('node:assert/strict')
const test = require('node:test')

function setupRequestMock(responseData) {
  let captured
  global.wx = {
    getStorageSync: () => 'test-token',
    setStorageSync: () => {},
    removeStorageSync: () => {},
    request: (options) => {
      captured = options
      options.success({ statusCode: 200, data: { code: 0, data: responseData } })
    },
  }
  return () => captured
}

test('任务 AI 总结会传递刷新参数并补齐数组字段', async () => {
  const getCaptured = setupRequestMock({
    summary: '今天先做重要事项',
    todo_count: 3,
    overdue_count: 1,
    ai_generated: true,
    focus_tasks: [{ id: '8', title: '买猫粮', due_at: '2026-09-14T18:00:00+08:00' }],
  })
  const { AISummaryService } = require('../services/ai-summary-service')

  const result = await AISummaryService.taskSummary({ refresh: true })

  assert.equal(getCaptured().url.endsWith('/v1/task/ai_summary'), true)
  assert.deepEqual(getCaptured().data, { refresh: true })
  assert.equal(result.todoCount, 3)
  assert.equal(result.overdueCount, 1)
  assert.equal(result.aiGenerated, true)
  assert.equal(result.focusTasks[0].dueAt, '2026-09-14T18:00:00+08:00')
  assert.deepEqual(result.suggestions, [])
})

test('生活 AI 总结发送月度范围', async () => {
  const getCaptured = setupRequestMock({
    summary: '本月生活很充实',
    period_label: '2026年9月',
    task_done_count: 6,
    expense_total: '225.00',
    highlights: ['坚持记录'],
  })
  const { AISummaryService } = require('../services/ai-summary-service')

  const result = await AISummaryService.lifeSummary({ scope: 'month', year: 2026, month: 9 })

  assert.deepEqual(getCaptured().data, { scope: 'month', year: 2026, month: 9, refresh: false })
  assert.equal(result.periodLabel, '2026年9月')
  assert.equal(result.taskDoneCount, 6)
  assert.equal(result.expenseTotal, '225.00')
  assert.deepEqual(result.highlights, ['坚持记录'])
  assert.deepEqual(result.suggestions, [])
})
