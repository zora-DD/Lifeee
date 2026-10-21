const assert = require('node:assert/strict')
const test = require('node:test')
const Module = require('node:module')
const path = require('node:path')

function loadPage(relativePath, replacements = {}) {
  const pagePath = path.resolve(__dirname, relativePath)
  const originalLoad = Module._load
  const originalPage = global.Page
  const originalWx = global.wx
  let options
  let navigation

  Module._load = function load(request, parent, isMain) {
    if (replacements[request]) return replacements[request]
    return originalLoad.call(this, request, parent, isMain)
  }
  global.Page = (value) => { options = value }
  global.wx = {
    navigateTo(value) { navigation = value },
    setNavigationBarTitle() {},
    getRecorderManager() {
      return { onStart() {}, onStop() {}, onError() {} }
    },
  }
  delete require.cache[pagePath]
  require(pagePath)
  Module._load = originalLoad
  global.Page = originalPage

  return {
    page: {
      ...options,
      data: JSON.parse(JSON.stringify(options.data)),
      setData(patch) { Object.assign(this.data, patch) },
    },
    getNavigation: () => navigation,
    restore() {
      global.wx = originalWx
      delete require.cache[pagePath]
    },
  }
}

test('日历空日期直接打开新建任务并预带日期', (t) => {
  const harness = loadPage('../pages/tasks/calendar.js', {
    '../../services/task-service': { TaskService: {} },
    '../../utils/access': { getCurrentUser: async () => null },
    '../../utils/permissions': { MODULES: { TASK: 'task' }, hasPermission: () => true, loadPermissions: async () => ({}) },
  })
  t.after(harness.restore)
  harness.page.data.canWrite = true
  harness.page.allTasks = []

  harness.page.selectDay({ currentTarget: { dataset: { date: '2026-09-20', taskCount: 0 } } })

  assert.equal(harness.getNavigation().url, '/pages/tasks/form?date=2026-09-20')
  assert.equal(harness.page.data.selectedDate, '2026-09-20')
})

test('从日历新建时表单使用传入日期', async (t) => {
  const harness = loadPage('../pages/tasks/form.js', {
    '../../services/task-service': { TaskService: {} },
    '../../utils/access': { requireWriteAccess: async () => ({ id: '1' }) },
    '../../utils/permissions': { MODULES: { TASK: 'task' } },
    '../../utils/page-scroll': { withScrollRestoration: (options) => options },
  })
  t.after(harness.restore)

  await harness.page.onLoad({ date: '2026-09-20' })

  assert.equal(harness.page.data.dueDate, '2026-09-20')
  assert.equal(harness.page.data.dueWeekday, '周日')
  assert.equal(harness.page.data.dueTime, '18:00')
})

test('普通新建任务默认当天截止', async (t) => {
  const harness = loadPage('../pages/tasks/form.js', {
    '../../services/task-service': { TaskService: {} },
    '../../utils/access': { requireWriteAccess: async () => ({ id: '1' }) },
    '../../utils/permissions': { MODULES: { TASK: 'task' } },
    '../../utils/page-scroll': { withScrollRestoration: (options) => options },
  })
  t.after(harness.restore)
  const now = new Date()
  const expectedDate = [
    now.getFullYear(),
    String(now.getMonth() + 1).padStart(2, '0'),
    String(now.getDate()).padStart(2, '0'),
  ].join('-')

  await harness.page.onLoad({})

  assert.equal(harness.page.data.dueDate, expectedDate)
  assert.equal(harness.page.data.dueTime, '18:00')
})
