const { request } = require('./request')

const API_PATHS = {
  list: '/v1/life-core/favorite/list',
  status: '/v1/life-core/favorite/status',
  modify: '/v1/life-core/favorite/modify',
}

const FavoriteService = {
  list(filters = {}) {
    const page = filters.page || 1
    const pageSize = filters.pageSize || 20
    return request({
      path: API_PATHS.list,
      method: 'POST',
      data: {
        targetType: filters.targetType && filters.targetType !== 'all' ? filters.targetType : undefined,
        page,
        pageSize,
      },
    }).then((result) => ({
      items: Array.isArray(result?.items) ? result.items : [],
      page: Number(result?.page) || page,
      pageSize: Number(result?.pageSize) || pageSize,
      total: Number(result?.total) || 0,
      hasMore: Boolean(result?.has_more ?? result?.hasMore),
    }))
  },

  status(targetType, targetId) {
    return request({ path: API_PATHS.status, method: 'POST', data: { targetType, targetId } })
      .then((result) => Boolean(result?.favorited))
  },

  modify(targetType, targetId, favorited) {
    return request({ path: API_PATHS.modify, method: 'POST', data: { targetType, targetId, favorited } })
      .then((result) => Boolean(result?.favorited))
  },
}

module.exports = { API_PATHS, FavoriteService }

