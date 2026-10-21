const { restaurants: seedRestaurants } = require('../data/catalog')
const { request } = require('./request')
const { MediaService, resolveMediaURL } = require('./media-service')
const {
  RESTAURANT_CATEGORY_ICONS: categoryIcons,
  RESTAURANT_CATEGORY_ICON_GROUPS: categoryIconGroups,
} = require('../utils/restaurant-category')

const STORAGE_KEY = 'life_restaurants_v2'

// 吃喝模块已接入后端；本地实现仅作开发回退。
const REMOTE_RESTAURANT_ENABLED = true

const API_PATHS = {
  list: '/v1/life-core/restaurant/list',
  create: '/v1/life-core/restaurant/create',
  detail: '/v1/life-core/restaurant/detail',
  update: '/v1/life-core/restaurant/modify',
  remove: '/v1/life-core/restaurant/delete',
  createConsumption: '/v1/life-core/restaurant/consumption/create',
  consumptions: '/v1/life-core/restaurant/consumption/list',
  updateConsumption: '/v1/life-core/restaurant/consumption/modify',
  removeConsumption: '/v1/life-core/restaurant/consumption/delete',
  recommendation: '/v1/life-core/restaurant/recommend',
}

function fallbackVisual(name, category) {
  const icons = categoryIconGroups[category] || [categoryIcons.其他]
  const hash = Array.from(name || category || '吃').reduce((total, char) => total + char.charCodeAt(0), 0)
  return { icon: icons[hash % icons.length], coverTone: (hash % 4) + 1 }
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function createSeedRestaurants() {
  const now = new Date().toISOString()
  return seedRestaurants.map((item) => normalizeRestaurant({
    coverUrl: '',
    createdAt: now,
    updatedAt: now,
    ...item,
  }))
}

function normalizeConsumption(item, index) {
  return {
    id: item.id || `consumption-${index}`,
    consumedAt: item.consumedAt || item.visitedAt || '',
    amount: Math.max(0, Number(item.amount) || 0),
    meal: item.meal || '',
    note: item.note || '',
    createdAt: item.createdAt || '',
  }
}

function normalizeRestaurant(item) {
  const { averagePrice, price, ...rest } = item
  const seedDefaults = seedRestaurants.find((seed) => seed.id === item.id)
  const fallbackLastAmount = item.lastAmount === undefined ? seedDefaults?.lastAmount : item.lastAmount
  let consumptions = (Array.isArray(item.consumptions) ? item.consumptions : [])
    .map(normalizeConsumption)
    .sort((left, right) => new Date(right.consumedAt || 0).getTime() - new Date(left.consumedAt || 0).getTime())

  // 兼容旧版本：保留已有的最近一次记录，不把历史到访次数误删。
  if (!consumptions.length && Number(item.visitCount) > 0 && item.lastVisitedAt) {
    consumptions = [normalizeConsumption({
      id: `legacy-${item.id || 'restaurant'}`,
      consumedAt: item.lastVisitedAt,
      amount: fallbackLastAmount,
      meal: item.lastMeal,
      note: item.note,
      createdAt: item.updatedAt || item.createdAt || '',
    }, 0)]
  }

  const latest = consumptions[0]
  const calculatedTotal = consumptions.reduce((total, record) => total + record.amount, 0)
  const visual = fallbackVisual(item.name, item.category)

  return {
    coverUrl: '',
    cuisine: '',
    note: '',
    lastMeal: '',
    visitCount: 0,
    totalSpent: 0,
    lastAmount: 0,
    ...rest,
    coverUrl: resolveMediaURL(item.coverUrl || ''),
    coverObjectKey: item.coverObjectKey || '',
    tags: item.features || item.tags || '',
    icon: item.icon || visual.icon,
    coverTone: item.coverTone || visual.coverTone,
    consumptions,
    visitCount: Math.max(consumptions.length, Number(item.visitCount) || 0),
    totalSpent: item.totalSpent === undefined
      ? Math.max(calculatedTotal, Number(seedDefaults?.totalSpent) || 0)
      : Math.max(0, Number(item.totalSpent) || 0),
    lastVisitedAt: latest ? latest.consumedAt : item.lastVisitedAt || '',
    lastAmount: latest ? latest.amount : Math.max(0, Number(fallbackLastAmount) || 0),
    lastMeal: latest ? latest.meal : item.lastMeal || '',
    note: latest ? latest.note : item.note || '',
  }
}

function loadLocalRestaurants() {
  const saved = wx.getStorageSync(STORAGE_KEY)
  if (Array.isArray(saved)) return saved.map(normalizeRestaurant)

  const restaurants = createSeedRestaurants()
  wx.setStorageSync(STORAGE_KEY, restaurants)
  return restaurants
}

function saveLocalRestaurants(restaurants) {
  wx.setStorageSync(STORAGE_KEY, restaurants)
}

function listLocal(filters = {}) {
  const category = filters.category || 'all'
  return loadLocalRestaurants()
    .filter((item) => category === 'all' || item.category === category)
    .filter((item) => matchesVisitStatus(item, filters.status))
    .sort((left, right) => new Date(right.lastVisitedAt).getTime() - new Date(left.lastVisitedAt).getTime())
}

function matchesVisitStatus(item, status) {
  const visitCount = Number(item.visitCount) || 0
  if (!status || status === 'all') return true
  if (status === 'wishlist') return visitCount === 0
  if (status === 'visited') return visitCount > 0 && visitCount < 3
  if (status === 'frequent') return visitCount >= 3
  return true
}

function hasLocalDuplicateName(name, excludeID = '') {
  const normalizedName = String(name || '').trim().toLocaleLowerCase()
  return loadLocalRestaurants().some((item) => (
    item.id !== excludeID && String(item.name || '').trim().toLocaleLowerCase() === normalizedName
  ))
}

function normalizeRemoteRestaurant(item) {
  return normalizeRestaurant({
    ...item,
    hours: item.businessHours || '',
    note: item.lastNote || '',
    consumptions: Array.isArray(item.consumptions) ? item.consumptions : [],
  })
}

function listRemote(filters = {}) {
  return request({
    path: API_PATHS.list,
    method: 'POST',
    data: {
      category: filters.category && filters.category !== 'all' ? filters.category : undefined,
      preference: filters.preference && filters.preference !== 'all' ? filters.preference : undefined,
      status: filters.status && filters.status !== 'all' ? filters.status : undefined,
      keyword: (filters.keyword || '').trim() || undefined,
      page: filters.page || 1,
      pageSize: filters.pageSize || 20,
    },
  }).then((result) => ({
    items: Array.isArray(result?.items) ? result.items.map(normalizeRemoteRestaurant) : [],
    page: Number(result?.page) || 1,
    pageSize: Number(result?.pageSize) || filters.pageSize || 20,
    total: Number(result?.total) || 0,
    hasMore: Boolean(result?.has_more ?? result?.hasMore),
    overallTotal: Number(result?.overallTotal) || 0,
    totalVisitCount: Number(result?.totalVisitCount) || 0,
    totalSpent: Number(result?.totalSpent) || 0,
  }))
}

function toRemoteRestaurantInput(input) {
  return {
    name: input.name,
    category: input.category,
    cuisine: input.cuisine || '',
    preference: input.preference,
    address: input.address || '',
    businessHours: input.hours || '',
    features: input.tags || '',
    coverUrl: input.coverUrl || '',
    coverObjectKey: input.coverObjectKey || '',
    latitude: input.latitude === undefined || input.latitude === null ? '' : String(input.latitude),
    longitude: input.longitude === undefined || input.longitude === null ? '' : String(input.longitude),
  }
}

const RestaurantService = {
  listPage(filters) {
    if (REMOTE_RESTAURANT_ENABLED) return listRemote(filters)
    const page = Math.max(1, Number(filters?.page) || 1)
    const pageSize = Math.max(1, Number(filters?.pageSize) || 20)
    const start = (page - 1) * pageSize
    const allItems = clone(listLocal(filters))
    return Promise.resolve({
      items: allItems.slice(start, start + pageSize),
      page,
      pageSize,
      total: allItems.length,
      hasMore: start + pageSize < allItems.length,
      overallTotal: loadLocalRestaurants().length,
      totalVisitCount: loadLocalRestaurants().reduce((total, item) => total + Number(item.visitCount || 0), 0),
      totalSpent: loadLocalRestaurants().reduce((total, item) => total + Number(item.totalSpent || 0), 0),
    })
  },

  list(filters) {
    return this.listPage(filters).then((result) => result.items)
  },

  all() {
    if (REMOTE_RESTAURANT_ENABLED) return listRemote({ page: 1, pageSize: 20 }).then((result) => result.items)
    return Promise.resolve(clone(loadLocalRestaurants()))
  },

  recommend(filters = {}) {
    if (REMOTE_RESTAURANT_ENABLED) {
      return request({
        path: API_PATHS.recommendation,
        method: 'POST',
        data: {
          category: filters.category && filters.category !== 'all' ? filters.category : undefined,
          preference: filters.preference && filters.preference !== 'all' ? filters.preference : undefined,
          status: filters.status && filters.status !== 'all' ? filters.status : undefined,
        },
      }).then(normalizeRemoteRestaurant)
    }
    const items = listLocal(filters)
    return Promise.resolve(items.length ? clone(items[Math.floor(Math.random() * items.length)]) : null)
  },

  get(id) {
    if (REMOTE_RESTAURANT_ENABLED) {
      return request({ path: API_PATHS.detail, method: 'POST', data: { id } }).then(normalizeRemoteRestaurant)
    }
    const restaurant = loadLocalRestaurants().find((item) => item.id === id)
    return Promise.resolve(restaurant ? clone(restaurant) : null)
  },

  create(input) {
    if (REMOTE_RESTAURANT_ENABLED) {
      return request({
        path: API_PATHS.create,
        method: 'POST',
        data: toRemoteRestaurantInput(input),
      }).then(normalizeRemoteRestaurant)
    }
    if (hasLocalDuplicateName(input.name)) {
      return Promise.reject(new Error('已存在同名吃喝记录'))
    }
    const now = new Date().toISOString()
    const restaurant = normalizeRestaurant({
      ...input,
      id: `restaurant-${Date.now()}`,
      visitCount: 0,
      totalSpent: 0,
      lastVisitedAt: '',
      lastAmount: 0,
      lastMeal: '',
      note: '',
      consumptions: [],
      createdAt: now,
      updatedAt: now,
    })
    saveLocalRestaurants([restaurant, ...loadLocalRestaurants()])
    return Promise.resolve(clone(restaurant))
  },

  update(id, input) {
    if (REMOTE_RESTAURANT_ENABLED) {
      return request({
        path: API_PATHS.update,
        method: 'POST',
        data: { id, ...toRemoteRestaurantInput(input) },
      }).then(normalizeRemoteRestaurant)
    }
    if (hasLocalDuplicateName(input.name, id)) {
      return Promise.reject(new Error('已存在同名吃喝记录'))
    }
    let updatedRestaurant = null
    const restaurants = loadLocalRestaurants().map((item) => {
      if (item.id !== id) return item
      updatedRestaurant = normalizeRestaurant({ ...item, ...input, updatedAt: new Date().toISOString() })
      return updatedRestaurant
    })
    saveLocalRestaurants(restaurants)
    return Promise.resolve(updatedRestaurant ? clone(updatedRestaurant) : null)
  },

  remove(id) {
    if (REMOTE_RESTAURANT_ENABLED) return request({ path: API_PATHS.remove, method: 'POST', data: { id } })
    saveLocalRestaurants(loadLocalRestaurants().filter((item) => item.id !== id))
    return Promise.resolve()
  },

  addConsumption(id, input) {
    if (REMOTE_RESTAURANT_ENABLED) {
      return request({
        path: API_PATHS.createConsumption,
        method: 'POST',
        data: {
          restaurantId: id,
          consumedAt: input.consumedAt,
          amount: input.amount === undefined || input.amount === null ? '' : String(input.amount),
          meal: input.meal || '',
          note: input.note || '',
        },
      }).then((item) => normalizeConsumption(item, 0))
    }

    let updatedRestaurant = null
    const now = new Date().toISOString()
    const consumption = normalizeConsumption({
      ...input,
      id: `consumption-${Date.now()}`,
      createdAt: now,
    }, 0)

    const restaurants = loadLocalRestaurants().map((item) => {
      if (item.id !== id) return item
      updatedRestaurant = normalizeRestaurant({
        ...item,
        consumptions: [consumption, ...(item.consumptions || [])],
        visitCount: Number(item.visitCount || 0) + 1,
        totalSpent: Number(item.totalSpent || 0) + consumption.amount,
        lastVisitedAt: consumption.consumedAt,
        lastAmount: consumption.amount,
        lastMeal: consumption.meal,
        note: consumption.note,
        updatedAt: now,
      })
      return updatedRestaurant
    })
    saveLocalRestaurants(restaurants)
    return Promise.resolve(updatedRestaurant ? clone(updatedRestaurant) : null)
  },

  listConsumptions(id, filters = {}) {
    if (REMOTE_RESTAURANT_ENABLED) {
      return request({
        path: API_PATHS.consumptions,
        method: 'POST',
        data: { restaurantId: id, page: filters.page || 1, pageSize: filters.pageSize || 20 },
      }).then((result) => ({
        items: Array.isArray(result?.items) ? result.items.map(normalizeConsumption) : [],
        page: Number(result?.page) || 1,
        pageSize: Number(result?.pageSize) || filters.pageSize || 20,
        total: Number(result?.total) || 0,
        hasMore: Boolean(result?.has_more ?? result?.hasMore),
      }))
    }
    const restaurant = loadLocalRestaurants().find((item) => item.id === id)
    const items = restaurant ? clone(restaurant.consumptions || []) : []
    return Promise.resolve({ items, page: 1, pageSize: items.length, total: items.length, hasMore: false })
  },

  updateConsumption(id, consumptionId, input) {
    if (REMOTE_RESTAURANT_ENABLED) {
      return request({
        path: API_PATHS.updateConsumption,
        method: 'POST',
        data: {
          restaurantId: id,
          id: consumptionId,
          consumedAt: input.consumedAt,
          amount: input.amount === undefined || input.amount === null ? '' : String(input.amount),
          meal: input.meal || '',
          note: input.note || '',
        },
      }).then(normalizeConsumption)
    }
    return Promise.resolve(null)
  },

  removeConsumption(id, consumptionId) {
    if (REMOTE_RESTAURANT_ENABLED) {
      return request({
        path: API_PATHS.removeConsumption,
        method: 'POST',
        data: { restaurantId: id, id: consumptionId },
      })
    }
    return Promise.resolve()
  },

  uploadCover(filePath) {
    if (REMOTE_RESTAURANT_ENABLED) return MediaService.uploadImage(filePath, 'restaurant')

    if (!filePath || filePath.includes('/usr/')) {
      return Promise.resolve({ url: filePath, objectKey: '' })
    }
    return new Promise((resolve, reject) => {
      wx.saveFile({
        tempFilePath: filePath,
        success: ({ savedFilePath }) => resolve({ url: savedFilePath, objectKey: '' }),
        fail: reject,
      })
    })
  },
}

module.exports = { API_PATHS, RestaurantService }
