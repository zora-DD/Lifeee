const { request } = require('./request')
const { MediaService, resolveMediaURL } = require('./media-service')

const API_PATHS = {
  list: '/v1/life-core/restaurant/snack/list',
  detail: '/v1/life-core/restaurant/snack/detail',
  create: '/v1/life-core/restaurant/snack/create',
  update: '/v1/life-core/restaurant/snack/modify',
  remove: '/v1/life-core/restaurant/snack/delete',
  taste: '/v1/life-core/restaurant/snack/taste',
  recommend: '/v1/life-core/restaurant/snack/recommend',
}

function normalizeRecord(item = {}) {
  return {
    ...item,
    id: String(item.id || ''),
    snackId: String(item.snackId || item.snack_id || ''),
    tastedBy: String(item.tastedBy || item.tasted_by || ''),
    tastedAt: item.tastedAt || item.tasted_at || '',
  }
}

function normalizeSnack(item = {}) {
  return {
    ...item,
    id: String(item.id || ''),
    purchasePlace: item.purchasePlace || item.purchase_place || '',
    coverUrl: resolveMediaURL(item.coverUrl || item.cover_url || ''),
    coverObjectKey: item.coverObjectKey || item.cover_object_key || '',
    memoryTag: item.memoryTag || item.memory_tag || '',
    tastingCount: Number(item.tastingCount ?? item.tasting_count) || 0,
    lastTastedAt: item.lastTastedAt || item.last_tasted_at || '',
    createdBy: String(item.createdBy || item.created_by || ''),
    records: (item.records || []).map(normalizeRecord),
  }
}

function payload(input = {}) {
  return {
    name: input.name || '',
    brand: input.brand || '',
    purchase_place: input.purchasePlace || '',
    category: input.category || '',
    status: input.status || 'wishlist',
    cover_url: input.coverUrl || '',
    cover_object_key: input.coverObjectKey || '',
    note: input.note || '',
    memory_tag: input.memoryTag || '',
  }
}

const SnackService = {
  list(filters = {}) {
    return request({
      path: API_PATHS.list,
      method: 'POST',
      data: {
        category: filters.category || '',
        status: filters.status || '',
        keyword: filters.keyword || '',
        page: filters.page || 1,
        page_size: filters.pageSize || 20,
      },
    }).then((result) => ({
      ...result,
      items: (result.items || []).map(normalizeSnack),
      pageSize: Number(result.pageSize || result.page_size) || 20,
      hasMore: Boolean(result.hasMore ?? result.has_more),
    }))
  },
  detail(id) {
    return request({ path: API_PATHS.detail, method: 'POST', data: { id: String(id) } }).then(normalizeSnack)
  },
  create(input) {
    return request({ path: API_PATHS.create, method: 'POST', data: payload(input) }).then(normalizeSnack)
  },
  update(id, input) {
    return request({ path: API_PATHS.update, method: 'POST', data: { id: String(id), ...payload(input) } }).then(normalizeSnack)
  },
  remove(id) {
    return request({ path: API_PATHS.remove, method: 'POST', data: { id: String(id) } })
  },
  taste(id, input = {}) {
    return request({ path: API_PATHS.taste, method: 'POST', data: { snack_id: String(id), tasted_at: input.tastedAt || '', note: input.note || '' } }).then(normalizeSnack)
  },
  recommend(filters = {}) {
    return request({ path: API_PATHS.recommend, method: 'POST', data: { category: filters.category || '', status: filters.status || '' } }).then(normalizeSnack)
  },
  uploadCover(filePath) {
    return MediaService.uploadImage(filePath, 'snack')
  },
}

module.exports = { API_PATHS, SnackService, normalizeSnack }
