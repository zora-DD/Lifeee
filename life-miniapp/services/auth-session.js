const ACCESS_TOKEN_KEY = 'access_token'
const ACCESS_TOKEN_EXPIRES_AT_KEY = 'access_token_expires_at'
const LEGACY_TOKEN_KEY = 'token'

function clearAccessToken() {
  wx.removeStorageSync(ACCESS_TOKEN_KEY)
  wx.removeStorageSync(ACCESS_TOKEN_EXPIRES_AT_KEY)
  wx.removeStorageSync(LEGACY_TOKEN_KEY)
}

function hasStoredAccessToken() {
  return Boolean(wx.getStorageSync(ACCESS_TOKEN_KEY) || wx.getStorageSync(LEGACY_TOKEN_KEY))
}

function getAccessToken() {
  let accessToken = wx.getStorageSync(ACCESS_TOKEN_KEY)

  // 平滑迁移开发阶段缓存过的旧 token，避免升级后被意外退出。
  if (!accessToken) {
    accessToken = wx.getStorageSync(LEGACY_TOKEN_KEY)
    if (accessToken) {
      wx.setStorageSync(ACCESS_TOKEN_KEY, accessToken)
      wx.removeStorageSync(LEGACY_TOKEN_KEY)
    }
  }

  const expiresAt = wx.getStorageSync(ACCESS_TOKEN_EXPIRES_AT_KEY)
  if (expiresAt) {
    const expiresAtTime = new Date(expiresAt).getTime()
    if (Number.isFinite(expiresAtTime) && expiresAtTime <= Date.now()) {
      clearAccessToken()
      return ''
    }
  }

  return accessToken || ''
}

function saveAccessToken(accessToken, expiresAt) {
  if (!accessToken) throw new Error('登录接口未返回 access_token')
  wx.setStorageSync(ACCESS_TOKEN_KEY, accessToken)
  if (expiresAt) wx.setStorageSync(ACCESS_TOKEN_EXPIRES_AT_KEY, expiresAt)
  else wx.removeStorageSync(ACCESS_TOKEN_EXPIRES_AT_KEY)
  wx.removeStorageSync(LEGACY_TOKEN_KEY)
}

module.exports = {
  ACCESS_TOKEN_KEY,
  clearAccessToken,
  getAccessToken,
  hasStoredAccessToken,
  saveAccessToken,
}
