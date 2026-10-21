const { request } = require('./request')

const API_PATHS = {
  list: '/v1/life-core/bill/list',
  detail: '/v1/life-core/bill/detail',
  create: '/v1/life-core/bill/create',
  update: '/v1/life-core/bill/modify',
  remove: '/v1/life-core/bill/delete',
  stats: '/v1/life-core/bill/stats',
  parse: '/v1/life-core/record/parse',
  recognizeVoice: '/v1/life-core/record/voice/recognize',
}

function normalizeBill(item) {
  const creator = item?.creator || {}
  return {
    ...item,
    amount: Number(item?.amount) || 0,
    creator: {
      id: creator.id || item?.createdBy || '',
      nickname: creator.nickname || '未命名用户',
      avatarEmoji: creator.avatarEmoji || '记',
      avatarColor: creator.avatarColor || '#789b8b',
    },
  }
}

const BillService = {
  list(filters = {}) {
    return request({
      path: API_PATHS.list,
      method: 'POST',
      data: {
        type: filters.type || undefined,
        category: filters.category || undefined,
        keyword: (filters.keyword || '').trim() || undefined,
        createdBy: filters.createdBy || undefined,
        from: filters.from || undefined,
        to: filters.to || undefined,
        page: filters.page || 1,
        pageSize: filters.pageSize || 20,
      },
    }).then((result) => ({
      items: Array.isArray(result?.items) ? result.items.map(normalizeBill) : [],
      creators: Array.isArray(result?.creators) ? result.creators : [],
      page: Number(result?.page) || 1,
      pageSize: Number(result?.pageSize) || filters.pageSize || 20,
      total: Number(result?.total) || 0,
      hasMore: Boolean(result?.has_more ?? result?.hasMore),
    }))
  },

  get(id) {
    return request({ path: API_PATHS.detail, method: 'POST', data: { id } }).then(normalizeBill)
  },

  create(input) {
    return request({ path: API_PATHS.create, method: 'POST', data: input }).then(normalizeBill)
  },

  update(id, input) {
    return request({ path: API_PATHS.update, method: 'POST', data: { id, ...input } }).then(normalizeBill)
  },

  remove(id) {
    return request({ path: API_PATHS.remove, method: 'POST', data: { id } })
  },

  stats(params = {}) {
    return request({
      path: API_PATHS.stats,
      method: 'POST',
      data: {
        year: params.year || undefined,
        month: params.month || undefined,
        createdBy: params.createdBy || undefined,
        scope: params.scope || undefined,
      },
    })
  },

  parse(text) {
    return request({ path: API_PATHS.parse, method: 'POST', data: { text } })
  },

  recognizeVoice(input) {
    return request({ path: API_PATHS.recognizeVoice, method: 'POST', data: input })
  },
}

module.exports = { API_PATHS, BillService }
