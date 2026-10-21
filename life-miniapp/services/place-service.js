const { places: seedPlaces } = require('../data/catalog')
const { request } = require('./request')
const { MediaService, resolveMediaURL } = require('./media-service')

const STORAGE_KEY = 'life_places_v1'

// 地点模块已接入后端；本地实现仅作开发回退。
const REMOTE_PLACE_ENABLED = true

const API_PATHS = {
  list: '/v1/life-core/place/list',
  create: '/v1/life-core/place/create',
  detail: '/v1/life-core/place/detail',
  update: '/v1/life-core/place/modify',
  remove: '/v1/life-core/place/delete',
}

const categoryIconGroups = {
  自然风景: ['🌿', '🌲', '🏞️'],
  城市探索: ['🌇', '🏛️', '🎨'],
  徒步登山: ['🥾', '⛰️', '🧭'],
  玩水亲水: ['🌊', '🏖️', '🚣'],
  商场街区: ['🏙️', '🛍️', '✨'],
  露营野餐: ['⛺', '🧺', '🌤️'],
  自驾路线: ['🚗', '🛣️', '🌄'],
  其他: ['📍', '🗺️', '✨'],
}

function fallbackVisual(name, category) {
  const icons = categoryIconGroups[category] || categoryIconGroups.其他
  const hash = Array.from(name || category || '地点').reduce((total, char) => total + char.charCodeAt(0), 0)
  return { icon: icons[hash % icons.length], coverTone: (hash % 4) + 1 }
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function normalizePlace(item) {
  const visual = fallbackVisual(item.name, item.category)
  return {
    coverUrl: '',
    visited: false,
    visitCount: 0,
    lastVisitedAt: '',
    feature: item.description || '',
    note: item.tip || '',
    guideUrl: '',
    distanceKm: 0,
    driveDurationMinutes: 0,
    latitude: null,
    longitude: null,
    ...item,
    coverUrl: resolveMediaURL(item.coverUrl || ''),
    coverObjectKey: item.coverObjectKey || '',
    distanceKm: Number(item.distanceKm) || 0,
    driveDurationMinutes: Number(item.driveDurationMinutes) || 0,
    icon: item.icon || visual.icon,
    coverTone: item.coverTone || visual.coverTone,
  }
}

function createSeedPlaces() {
  const now = new Date().toISOString()
  return seedPlaces.map((item) => ({
    creator: { id: 'me', name: '我' },
    createdAt: now,
    updatedAt: now,
    ...normalizePlace(item),
  }))
}

function loadLocalPlaces() {
  const saved = wx.getStorageSync(STORAGE_KEY)
  if (Array.isArray(saved)) return saved.map(normalizePlace)

  const places = createSeedPlaces()
  wx.setStorageSync(STORAGE_KEY, places)
  return places
}

function saveLocalPlaces(places) {
  wx.setStorageSync(STORAGE_KEY, places)
}

function matchesStatus(place, status) {
  if (!status || status === 'all') return true
  if (status === 'visited') return place.visited
  if (status === 'wishlist') return !place.visited
  return true
}

function listLocal(filters = {}) {
  const category = filters.category || 'all'
  return loadLocalPlaces()
    .filter((item) => category === 'all' || item.category === category)
    .filter((item) => matchesStatus(item, filters.status))
    .sort((left, right) => {
      if (left.visited !== right.visited) return left.visited ? -1 : 1
      return new Date(right.lastVisitedAt || 0).getTime() - new Date(left.lastVisitedAt || 0).getTime()
    })
}

function normalizeRemotePlace(item) {
  const creator = item && item.creator ? item.creator : {}
  return normalizePlace({
    ...item,
    creator: {
      id: creator.id || item.createdBy || '',
      name: creator.nickname || creator.name || '未知用户',
      avatarEmoji: creator.avatarEmoji || '',
      avatarColor: creator.avatarColor || '#789b8b',
    },
  })
}

function listRemote(filters = {}) {
  return request({
    path: API_PATHS.list,
    method: 'POST',
    data: {
      category: filters.category && filters.category !== 'all' ? filters.category : undefined,
      status: filters.status && filters.status !== 'all' ? filters.status : undefined,
      keyword: (filters.keyword || '').trim() || undefined,
      page: filters.page || 1,
      pageSize: filters.pageSize || 20,
    },
  }).then((result) => ({
    items: Array.isArray(result?.items) ? result.items.map(normalizeRemotePlace) : [],
    page: Number(result?.page) || 1,
    pageSize: Number(result?.pageSize) || filters.pageSize || 20,
    total: Number(result?.total) || 0,
    hasMore: Boolean(result?.has_more ?? result?.hasMore),
    overallTotal: Number(result?.overallTotal) || 0,
    visitedCount: Number(result?.visitedCount) || 0,
    wishlistCount: Number(result?.wishlistCount) || 0,
  }))
}

function toRemoteInput(input) {
  return {
    name: input.name,
    category: input.category,
    icon: input.icon || '',
    visited: Boolean(input.visited),
    visitCount: Number(input.visitCount) || 0,
    lastVisitedAt: input.visited ? input.lastVisitedAt : '',
    feature: input.feature || '',
    note: input.note || '',
    address: input.address || '',
    bestTime: input.bestTime || '',
    guideUrl: input.guideUrl || '',
    distanceKm: Number(input.distanceKm) || 0,
    driveDurationMinutes: Number(input.driveDurationMinutes) || 0,
    coverUrl: input.coverUrl || '',
    coverObjectKey: input.coverObjectKey || '',
    latitude: input.latitude === null || input.latitude === undefined ? '' : String(input.latitude),
    longitude: input.longitude === null || input.longitude === undefined ? '' : String(input.longitude),
  }
}

const PlaceService = {
  listPage(filters) {
    if (REMOTE_PLACE_ENABLED) return listRemote(filters)
    const allItems = clone(listLocal(filters))
    const page = Math.max(1, Number(filters?.page) || 1)
    const pageSize = Math.max(1, Number(filters?.pageSize) || 20)
    const start = (page - 1) * pageSize
    const places = loadLocalPlaces()
    const visitedCount = places.filter((item) => item.visited).length
    return Promise.resolve({
      items: allItems.slice(start, start + pageSize),
      page,
      pageSize,
      total: allItems.length,
      hasMore: start + pageSize < allItems.length,
      overallTotal: places.length,
      visitedCount,
      wishlistCount: places.length - visitedCount,
    })
  },

  list(filters) {
    return this.listPage(filters).then((result) => result.items)
  },

  all() {
    if (REMOTE_PLACE_ENABLED) return listRemote({ page: 1, pageSize: 20 }).then((result) => result.items)
    return Promise.resolve(clone(loadLocalPlaces()))
  },

  get(id) {
    if (REMOTE_PLACE_ENABLED) {
      return request({ path: API_PATHS.detail, method: 'POST', data: { id } }).then(normalizeRemotePlace)
    }
    const place = loadLocalPlaces().find((item) => item.id === id)
    return Promise.resolve(place ? clone(place) : null)
  },

  create(input) {
    if (REMOTE_PLACE_ENABLED) {
      return request({ path: API_PATHS.create, method: 'POST', data: toRemoteInput(input) }).then(normalizeRemotePlace)
    }
    const now = new Date().toISOString()
    const place = {
      ...input,
      id: `place-${Date.now()}`,
      creator: { id: 'me', name: '我' },
      createdAt: now,
      updatedAt: now,
    }
    saveLocalPlaces([place, ...loadLocalPlaces()])
    return Promise.resolve(clone(place))
  },

  update(id, input) {
    if (REMOTE_PLACE_ENABLED) {
      return request({
        path: API_PATHS.update,
        method: 'POST',
        data: { id, ...toRemoteInput(input) },
      }).then(normalizeRemotePlace)
    }
    let updatedPlace = null
    const places = loadLocalPlaces().map((item) => {
      if (item.id !== id) return item
      updatedPlace = { ...item, ...input, updatedAt: new Date().toISOString() }
      return updatedPlace
    })
    saveLocalPlaces(places)
    return Promise.resolve(updatedPlace ? clone(updatedPlace) : null)
  },

  remove(id) {
    if (REMOTE_PLACE_ENABLED) return request({ path: API_PATHS.remove, method: 'POST', data: { id } })
    saveLocalPlaces(loadLocalPlaces().filter((item) => item.id !== id))
    return Promise.resolve()
  },

  uploadCover(filePath) {
    if (REMOTE_PLACE_ENABLED) return MediaService.uploadImage(filePath, 'place')

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

module.exports = { API_PATHS, PlaceService }
