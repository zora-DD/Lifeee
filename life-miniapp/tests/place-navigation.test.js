const assert = require('node:assert/strict')
const test = require('node:test')
const Module = require('node:module')
const path = require('node:path')

function loadPlaceIndexPage() {
  const pagePath = path.resolve(__dirname, '../pages/places/index.js')
  const originalLoad = Module._load
  const originalPage = global.Page
  const originalWx = global.wx
  let pageOptions
  let navigation
  let currentUserCalls = 0

  Module._load = function load(request, parent, isMain) {
    if (request === '../../services/place-service') return { PlaceService: {} }
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
  global.wx = { navigateTo: (options) => { navigation = options } }

  delete require.cache[pagePath]
  require(pagePath)
  Module._load = originalLoad
  global.Page = originalPage

  const page = {
    ...pageOptions,
    data: JSON.parse(JSON.stringify(pageOptions.data)),
    setData(patch) { Object.assign(this.data, patch) },
    async loadPlaces() { this.loadCount = (this.loadCount || 0) + 1 },
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

test('查看地点详情后返回不重复请求用户和列表', async (t) => {
  const harness = loadPlaceIndexPage()
  t.after(harness.restore)

  await harness.page.onShow()
  harness.page.openDetail({ currentTarget: { dataset: { id: '8' } } })
  await harness.page.onShow()

  assert.equal(harness.getNavigation().url, '/pages/places/detail?id=8')
  assert.equal(harness.page.loadCount, 1)
  assert.equal(harness.getCurrentUserCalls(), 1)
})
