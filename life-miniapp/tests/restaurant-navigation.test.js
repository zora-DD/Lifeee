const assert = require('node:assert/strict')
const test = require('node:test')
const Module = require('node:module')
const path = require('node:path')

function loadRestaurantIndexPage() {
  const pagePath = path.resolve(__dirname, '../pages/restaurants/index.js')
  const originalLoad = Module._load
  const originalPage = global.Page
  const originalWx = global.wx
  let pageOptions
  let navigation
  let currentUserCalls = 0

  Module._load = function load(request, parent, isMain) {
    if (request === '../../services/restaurant-service') {
      return { RestaurantService: {} }
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
    if (request === '../../utils/page-scroll') {
      return { withScrollRestoration: (options) => options }
    }
    return originalLoad.call(this, request, parent, isMain)
  }

  global.Page = (options) => { pageOptions = options }
  global.wx = {
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
    async loadRestaurants() {
      this.loadCount = (this.loadCount || 0) + 1
      this.lastRestaurantLoadAt = Date.now()
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

function loadRestaurantDetailPage() {
  const pagePath = path.resolve(__dirname, '../pages/restaurants/detail.js')
  const originalLoad = Module._load
  const originalPage = global.Page
  const originalWx = global.wx
  let pageOptions
  let navigation
  const emitted = []

  Module._load = function load(request, parent, isMain) {
    if (request === '../../services/restaurant-service') {
      return { RestaurantService: {} }
    }
    if (request === '../../services/favorite-service') {
      return { FavoriteService: {} }
    }
    if (request === '../../utils/access') {
      return {
        requireApprovedAccess: async () => ({ id: '1', accessStatus: 'approved' }),
        requireWriteAccess: async () => ({ id: '1', accessStatus: 'approved' }),
      }
    }
    if (request === '../../utils/page-scroll') {
      return { withScrollRestoration: (options) => options }
    }
    return originalLoad.call(this, request, parent, isMain)
  }

  global.Page = (options) => { pageOptions = options }
  global.wx = {
    navigateTo(options) { navigation = options },
  }

  delete require.cache[pagePath]
  require(pagePath)
  Module._load = originalLoad
  global.Page = originalPage

  const page = {
    ...pageOptions,
    data: {
      ...JSON.parse(JSON.stringify(pageOptions.data)),
      id: '8',
      restaurant: { id: '8', name: '旧店名' },
    },
    setData(patch) { Object.assign(this.data, patch) },
    async loadRestaurant() { this.loadCount = (this.loadCount || 0) + 1 },
    getOpenerEventChannel() {
      return { emit: (name, payload) => emitted.push({ name, payload }) }
    },
  }

  return {
    page,
    emitted,
    getNavigation: () => navigation,
    restore() {
      global.wx = originalWx
      delete require.cache[pagePath]
    },
  }
}

test('查看吃喝详情后返回会复用列表，数据变更后才刷新', async (t) => {
  const harness = loadRestaurantIndexPage()
  t.after(harness.restore)

  await harness.page.onShow()
  assert.equal(harness.page.loadCount, 1)
  assert.equal(harness.getCurrentUserCalls(), 1)

  harness.page.openDetail({ currentTarget: { dataset: { id: '8' } } })
  const navigation = harness.getNavigation()
  assert.equal(navigation.url, '/pages/restaurants/detail?id=8')

  await harness.page.onShow()
  assert.equal(harness.page.loadCount, 1)
  assert.equal(harness.getCurrentUserCalls(), 1)

  harness.page.openDetail({ currentTarget: { dataset: { id: '8' } } })
  harness.getNavigation().events.restaurantChanged()
  await harness.page.onShow()
  assert.equal(harness.page.loadCount, 2)
})

test('切换 Tab 时五分钟内复用吃喝列表，过期后重新刷新', async (t) => {
  const originalNow = Date.now
  let now = 1_000_000
  Date.now = () => now
  const harness = loadRestaurantIndexPage()
  t.after(() => {
    Date.now = originalNow
    harness.restore()
  })

  await harness.page.onShow()
  assert.equal(harness.page.loadCount, 1)

  now += 4 * 60 * 1000
  await harness.page.onShow()
  assert.equal(harness.page.loadCount, 1)

  now += 61 * 1000
  await harness.page.onShow()
  assert.equal(harness.page.loadCount, 2)
})

test('吃喝数据变更后忽略列表缓存并立即刷新', async (t) => {
  const harness = loadRestaurantIndexPage()
  t.after(harness.restore)

  await harness.page.onShow()
  harness.page.openDetail({ currentTarget: { dataset: { id: '8' } } })
  harness.getNavigation().events.restaurantChanged()
  await harness.page.onShow()

  assert.equal(harness.page.loadCount, 2)
})

test('编辑店铺返回详情时直接应用保存结果，不清空页面重新加载', async (t) => {
  const harness = loadRestaurantDetailPage()
  t.after(harness.restore)

  await harness.page.editRestaurant()
  const navigation = harness.getNavigation()
  navigation.events.restaurantChanged({
    restaurant: {
      id: '8', name: '新店名', preference: 'love', visitCount: 1,
      totalSpent: 88, lastAmount: 88, tags: '鱼鲜 · 菜味很好',
    },
  })

  await harness.page.onShow()

  assert.equal(harness.page.data.restaurant.name, '新店名')
  assert.equal(harness.page.data.restaurant.tags, '鱼鲜 · 菜味很好')
  assert.equal(harness.page.loadCount || 0, 0)
  assert.equal(harness.emitted[0].name, 'restaurantChanged')
})
