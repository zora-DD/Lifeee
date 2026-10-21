const assert = require('node:assert/strict')
const test = require('node:test')
const Module = require('node:module')
const path = require('node:path')

function loadLogsPage({ created = [], createdLogs = [] } = {}) {
  const pagePath = path.resolve(__dirname, '../pages/logs/index.js')
  const originalLoad = Module._load
  const originalPage = global.Page
  const originalWx = global.wx
  let pageOptions
  let navigation
  let currentUserCalls = 0

  Module._load = function load(request, parent, isMain) {
    if (request === '../../services/bill-service') {
      return {
        BillService: {
          create: async (input) => {
            created.push(input)
            return { id: String(created.length), ...input }
          },
        },
      }
    }
    if (request === '../../services/activity-log-service') {
      return {
        ActivityLogService: {
          create: async (input) => {
            createdLogs.push(input)
            return { id: String(createdLogs.length), ...input }
          },
        },
      }
    }
    if (request === '../../utils/access') {
      return {
        getCurrentUser: async () => {
          currentUserCalls += 1
          return { id: '1', accessStatus: 'approved' }
        },
        requireWriteAccess: async () => ({ id: '1', accessStatus: 'approved' }),
      }
    }
    return originalLoad.call(this, request, parent, isMain)
  }

  global.Page = (options) => { pageOptions = options }
  const toasts = []
  const scrollCalls = []
  global.wx = {
    showToast: (options) => { toasts.push(options) },
    showModal: () => {},
    getStorageSync: () => '',
    removeStorageSync: () => {},
    navigateTo: (options) => { navigation = options },
    nextTick: (callback) => callback(),
    pageScrollTo: (options) => { scrollCalls.push(options) },
  }

  delete require.cache[pagePath]
  require(pagePath)
  Module._load = originalLoad
  global.Page = originalPage

  const page = {
    ...pageOptions,
    data: JSON.parse(JSON.stringify(pageOptions.data)),
    setData(patch, callback) {
      Object.assign(this.data, patch)
      if (callback) callback()
    },
    async loadBills() { this.reloaded = true },
    async loadLogs() { this.logsReloaded = true },
    async loadCurrentMode() { this.loadCount = (this.loadCount || 0) + 1 },
  }

  return {
    page,
    toasts,
    scrollCalls,
    getNavigation: () => navigation,
    getCurrentUserCalls: () => currentUserCalls,
    restore() {
      global.wx = originalWx
      delete require.cache[pagePath]
    },
  }
}

test('确认两笔账单时会全部保存并关闭确认弹窗', async (t) => {
  const created = []
  const harness = loadLogsPage({ created })
  t.after(harness.restore)

  harness.page.data.showQuickDraft = true
  harness.page.data.billDrafts = [
    {
      type: 'expense', title: '午饭', amount: '85.00', category: '餐饮',
      occurredAt: '2026-09-05T04:00:00.000Z', note: '', inputType: 'text', rawText: '午饭85',
    },
    {
      type: 'expense', title: '停车费', amount: '25.00', category: '交通',
      occurredAt: '2026-09-05T04:00:00.000Z', note: '', inputType: 'text', rawText: '停车费25',
    },
  ]

  await harness.page.saveBillDrafts()

  assert.equal(created.length, 2)
  assert.deepEqual(created.map((item) => item.title), ['午饭', '停车费'])
  assert.equal(harness.page.data.showQuickDraft, false)
  assert.deepEqual(harness.page.data.billDrafts, [])
  assert.equal(harness.page.reloaded, true)
  assert.equal(harness.page.data.saving, false)
})

test('生活记录未填写花费时间时不会提交', async (t) => {
  const createdLogs = []
  const harness = loadLogsPage({ createdLogs })
  t.after(harness.restore)

  harness.page.data.content = '收拾客厅'
  harness.page.data.dateValue = '2026-09-05'
  harness.page.data.timeValue = '11:30'
  harness.page.data.durationValue = ''

  await harness.page.saveLog()

  assert.equal(createdLogs.length, 0)
  assert.equal(harness.toasts.at(-1).title, '请填写花费时间')
})

test('查看生活记录详情后返回复用列表并恢复滚动位置', async (t) => {
  const harness = loadLogsPage()
  t.after(harness.restore)

  harness.page.data.activeMode = 'activity'
  await harness.page.onShow()
  harness.page.onPageScroll({ scrollTop: 860 })
  harness.page.openLogDetail({ currentTarget: { dataset: { id: '12' } } })
  harness.page.onHide()
  await harness.page.onShow()

  assert.equal(harness.getNavigation().url, '/pages/logs/detail?id=12')
  assert.equal(harness.page.loadCount, 1)
  assert.equal(harness.getCurrentUserCalls(), 1)
  assert.deepEqual(harness.scrollCalls, [{ scrollTop: 860, duration: 0 }])
})
