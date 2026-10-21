const assert = require('node:assert/strict')
const test = require('node:test')
const Module = require('node:module')
const path = require('node:path')

function loadTaskIndexPage() {
  const pagePath = path.resolve(__dirname, '../pages/tasks/index.js')
  const originalLoad = Module._load
  const originalPage = global.Page
  const originalWx = global.wx
  let pageOptions
  let navigation
  let currentUserCalls = 0

  Module._load = function load(request, parent, isMain) {
    if (request === '../../services/task-service') return { TaskService: {} }
    if (request === '../../services/ai-summary-service') return { AISummaryService: {} }
    if (request === '../../utils/access') {
      return {
        getCurrentUser: async () => {
          currentUserCalls += 1
          return { id: '1', accessStatus: 'approved' }
        },
        requireWriteAccess: async () => ({ id: '1', accessStatus: 'approved' }),
      }
    }
    if (request === '../../utils/page-scroll') return { withScrollRestoration: (options) => options }
    return originalLoad.call(this, request, parent, isMain)
  }

  global.Page = (options) => { pageOptions = options }
  global.wx = {
    getWindowInfo: () => ({ windowWidth: 375 }),
    navigateTo(options) { navigation = options },
  }

  delete require.cache[pagePath]
  require(pagePath)
  Module._load = originalLoad
  global.Page = originalPage

  const page = {
    ...pageOptions,
    data: JSON.parse(JSON.stringify(pageOptions.data)),
    setData(patch) { Object.assign(this.data, patch) },
    async loadTasks(options) {
      this.loadCalls = [...(this.loadCalls || []), options]
    },
  }

  return {
    page,
    getNavigation: () => navigation,
    getCurrentUserCalls: () => currentUserCalls,
    restore() {
      global.wx = originalWx
      delete require.cache[pagePath]
    },
  }
}

test('查看任务详情后返回复用列表并保留已加载分页', async (t) => {
  const harness = loadTaskIndexPage()
  t.after(harness.restore)

  await harness.page.onShow()
  harness.page.data.page = 4
  harness.page.openDetail({ currentTarget: { dataset: { id: '8' } } })
  await harness.page.onShow()

  assert.equal(harness.getNavigation().url, '/pages/tasks/detail?id=8')
  assert.equal(harness.page.loadCalls.length, 1)
  assert.equal(harness.getCurrentUserCalls(), 1)
  assert.equal(harness.page.data.page, 4)
})

test('任务详情发生变更后重新加载原有页数', async (t) => {
  const harness = loadTaskIndexPage()
  t.after(harness.restore)

  await harness.page.onShow()
  harness.page.data.page = 4
  harness.page.openDetail({ currentTarget: { dataset: { id: '8' } } })
  harness.getNavigation().events.taskChanged({ task: { id: '8' } })
  await harness.page.onShow()

  assert.equal(harness.page.loadCalls.length, 2)
  assert.deepEqual(harness.page.loadCalls[1], { preserveLoadedPages: true })
})

test('任务列表标签使用轻量 Emoji 展示', (t) => {
  const harness = loadTaskIndexPage()
  t.after(harness.restore)

  const task = harness.page.toViewTask({
    id: '8',
    title: '更新接口',
    priority: 'medium',
    status: 'todo',
    dueAt: '2099-09-20T18:00:00+08:00',
    tags: ['工作'],
    creator: { id: '1' },
  })

  assert.deepEqual(task.tagViews, [{ label: '工作', icon: '💼' }])
  assert.equal(task.dueWeekend, undefined)
})

test('已完成任务只标记第一项作为折叠入口', (t) => {
  const harness = loadTaskIndexPage()
  t.after(harness.restore)

  const tasks = harness.page.decorateTaskList([
    { id: '1', status: 'todo' },
    { id: '2', status: 'done' },
    { id: '3', status: 'done' },
  ])

  assert.equal(tasks[0].firstCompleted, false)
  assert.equal(tasks[1].firstCompleted, true)
  assert.equal(tasks[2].firstCompleted, false)
})

test('任务筛选包含今天和我创建的', (t) => {
  const harness = loadTaskIndexPage()
  t.after(harness.restore)

  assert.ok(harness.page.data.statusFilters.some((item) => item.value === 'today'))
  assert.ok(harness.page.data.ownerFilters.some((item) => item.value === 'mine'))
})
