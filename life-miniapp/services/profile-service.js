const PROFILE_KEY = 'life_profile_v1'
const { isUnauthorized, request } = require('./request')
const {
  clearAccessToken,
  hasStoredAccessToken,
  saveAccessToken,
} = require('./auth-session')
const {
  invalidateSessionCache,
  readSessionUser,
  saveSessionUser,
} = require('../utils/session-cache')

let currentUserRequest = null

const API_PATHS = {
  login: '/v1/life-core/auth/login',
  me: '/v1/life-core/user/current',
  contributions: '/v1/life-core/user/contribution/list',
  applyAccess: '/v1/life-core/access/apply',
  accessStatus: '/v1/life-core/access/status',
  adminRequests: '/v1/life-core/admin/access_request/list',
  reviewAccess: '/v1/life-core/admin/access_request/modify',
  myPermissions: '/v1/life-core/user/permission/list',
  adminPermissionDetail: '/v1/life-core/admin/user_permission/detail',
  adminPermissionModify: '/v1/life-core/admin/user_permission/modify',
}

function normalizeUser(user) {
  if (!user) return null
  return {
    ...user,
    // 管理端列表可能来自不同序列化方式，统一保留字符串用户 ID。
    id: String(user.id ?? user.userId ?? user.user_id ?? ''),
    avatarUrl: user.avatarUrl ?? user.avatar_url ?? '',
    avatarEmoji: user.avatarEmoji ?? user.avatar_emoji ?? '',
    avatarColor: user.avatarColor ?? user.avatar_color ?? '#759b89',
    accessStatus: user.accessStatus ?? user.access_status ?? 'pending',
    applicationNote: user.applicationNote ?? user.application_note ?? '',
    createdAt: user.createdAt ?? user.created_at ?? '',
  }
}

const ProfileService = {
  getCurrent({ force = false } = {}) {
    const sessionUser = readSessionUser()
    if (!force && sessionUser.loaded) return Promise.resolve(sessionUser.user)
    if (!force && currentUserRequest) return currentUserRequest

    const cachedUser = wx.getStorageSync(PROFILE_KEY)
    if (!hasStoredAccessToken() && !cachedUser) {
      wx.removeStorageSync(PROFILE_KEY)
      saveSessionUser(null)
      return Promise.resolve(null)
    }

    currentUserRequest = request({ path: API_PATHS.me, method: 'POST' })
      .then((rawUser) => {
        const user = normalizeUser(rawUser)
        wx.setStorageSync(PROFILE_KEY, user)
        saveSessionUser(user)
        return user
      })
      .catch((error) => {
        // 短暂断网或服务异常时沿用最近一次用户信息，避免页面误显示退出。
        if (cachedUser && !isUnauthorized(error)) {
          const user = normalizeUser(cachedUser)
          saveSessionUser(user)
          return user
        }
        if (isUnauthorized(error)) saveSessionUser(null)
        throw error
      })
      .finally(() => { currentUserRequest = null })

    return currentUserRequest
  },

  login(code) {
    return request({ path: API_PATHS.login, method: 'POST', data: { code } })
      .then((result) => {
        const accessToken = result.access_token ?? result.accessToken
        const expiresAt = result.expires_at ?? result.expiresAt
        saveAccessToken(accessToken, expiresAt)
        const user = normalizeUser(result.user)
        wx.setStorageSync(PROFILE_KEY, user)
        saveSessionUser(user)
        return user
      })
  },

  getAccessStatus() {
    return request({ path: API_PATHS.accessStatus, method: 'POST' }).then((rawUser) => {
      const user = normalizeUser(rawUser)
      wx.setStorageSync(PROFILE_KEY, user)
      saveSessionUser(user)
      return user
    })
  },

  applyAccess(note) {
    return request({
      path: API_PATHS.applyAccess,
      method: 'POST',
      data: { note: String(note || '').trim() },
    }).then((rawUser) => {
      const user = normalizeUser(rawUser)
      wx.setStorageSync(PROFILE_KEY, user)
      saveSessionUser(user)
      return user
    })
  },

  listAccessRequests(status = 'pending', page = 1, pageSize = 20) {
    return request({
      path: API_PATHS.adminRequests,
      method: 'POST',
      data: { status, page, page_size: pageSize },
    }).then((result) => ({
      items: Array.isArray(result?.items) ? result.items.map(normalizeUser) : [],
      page: Number(result?.page) || page,
      pageSize: Number(result?.pageSize) || pageSize,
      total: Number(result?.total) || 0,
      hasMore: Boolean(result?.has_more ?? result?.hasMore),
    }))
  },

  reviewAccessRequest(userId, status) {
    return request({
      path: API_PATHS.reviewAccess,
      method: 'POST',
      data: { user_id: String(userId), status },
    }).then(normalizeUser)
  },

  getMyPermissions() {
    return request({ path: API_PATHS.myPermissions, method: 'POST' })
  },

  getUserPermissions(userId) {
    return request({
      path: API_PATHS.adminPermissionDetail,
      method: 'POST',
      data: { user_id: String(userId) },
    }).then((result) => ({
      user: normalizeUser(result?.user),
      permissions: Array.isArray(result?.permissions) ? result.permissions : [],
    }))
  },

  modifyUserPermissions(userId, permissions) {
    return request({
      path: API_PATHS.adminPermissionModify,
      method: 'POST',
      data: { user_id: String(userId), permissions },
    }).then((result) => ({
      user: normalizeUser(result?.user),
      permissions: Array.isArray(result?.permissions) ? result.permissions : [],
    }))
  },

  listContributions(filters = {}) {
    const page = filters.page || 1
    const pageSize = filters.pageSize || 20
    return request({
      path: API_PATHS.contributions,
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

  logout() {
    wx.removeStorageSync(PROFILE_KEY)
    wx.removeStorageSync('life_module_permissions_v1')
    clearAccessToken()
    invalidateSessionCache()
    return Promise.resolve()
  },
}

module.exports = { API_PATHS, ProfileService }
