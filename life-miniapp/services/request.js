const PROFILE_KEY = 'life_profile_v1'
const LOGIN_PATH = '/v1/life-core/auth/login'
const { API_BASE_URL, resolveApiPath } = require('../config/env')
const {
  clearAccessToken,
  getAccessToken,
  hasStoredAccessToken,
  saveAccessToken,
} = require('./auth-session')
const { invalidateSessionCache } = require('../utils/session-cache')

let reloginPromise = null

function clearLoginState() {
  clearAccessToken()
  wx.removeStorageSync(PROFILE_KEY)
  invalidateSessionCache()
}

function isUnauthorized(error) {
  return error?.statusCode === 401 || error?.code === 1204
}

function sendRequest({ path, method = 'GET', data = {} }, accessToken = '') {
  const header = accessToken ? { Authorization: `Bearer ${accessToken}` } : {}
  // 微信请求会把 undefined 查询参数转成字符串，发送前统一剔除空值。
  const requestData = Object.fromEntries(
    Object.entries(data).filter(([, value]) => value !== undefined && value !== null),
  )

  return new Promise((resolve, reject) => {
    wx.request({
      url: `${API_BASE_URL}${resolveApiPath(path)}`,
      method,
      data: requestData,
      header,
      success: ({ statusCode, data: response }) => {
        if (statusCode >= 200 && statusCode < 300 && response && response.code === 0) {
          resolve(response.data)
          return
        }
        const error = new Error(response?.msg || `接口返回 ${statusCode}`)
        error.statusCode = statusCode
        error.code = response?.code
        reject(error)
      },
      fail: ({ errMsg }) => reject(new Error(errMsg || '网络请求失败')),
    })
  })
}

function getWechatLoginCode() {
  return new Promise((resolve, reject) => {
    wx.login({
      success: ({ code }) => {
        if (code) resolve(code)
        else reject(new Error('微信登录未返回 code'))
      },
      fail: ({ errMsg }) => reject(new Error(errMsg || '微信登录失败')),
    })
  })
}

// 多个请求同时遇到 401 时只执行一次微信静默登录。
function refreshAccessToken() {
  if (reloginPromise) return reloginPromise

  reloginPromise = getWechatLoginCode()
    .then((code) => sendRequest({ path: LOGIN_PATH, method: 'POST', data: { code } }))
    .then((result) => {
      const accessToken = result?.access_token ?? result?.accessToken
      const expiresAt = result?.expires_at ?? result?.expiresAt
      saveAccessToken(accessToken, expiresAt)
      reloginPromise = null
      return accessToken
    }, (error) => {
      reloginPromise = null
      // 网络抖动或微信登录临时失败不应让用户看起来被强制退出。
      // 只有服务端明确判定未授权时，才清理完整登录状态。
      if (isUnauthorized(error)) clearLoginState()
      throw error
    })

  return reloginPromise
}

function request(options) {
  const hadLoginSession = hasStoredAccessToken() || Boolean(wx.getStorageSync(PROFILE_KEY))
  const accessToken = getAccessToken()
  const isLoginRequest = options.path === LOGIN_PATH
  const shouldRelogin = hadLoginSession || Boolean(reloginPromise)

  // 本地已知 Token 过期时，先静默登录，避免发送一次必然失败的请求。
  if (!accessToken && shouldRelogin && !isLoginRequest) {
    return refreshAccessToken()
      .then((newAccessToken) => sendRequest(options, newAccessToken))
      .catch((error) => {
        if (isUnauthorized(error)) clearLoginState()
        throw error
      })
  }

  return sendRequest(options, accessToken).catch((error) => {
    if (!isUnauthorized(error) || isLoginRequest || !shouldRelogin) {
      if (isUnauthorized(error)) clearLoginState()
      throw error
    }

    // 其他并发请求可能已经完成续登，优先复用刚写入的新 Token。
    const latestAccessToken = getAccessToken()
    if (latestAccessToken && latestAccessToken !== accessToken) {
      return sendRequest(options, latestAccessToken)
    }

    return refreshAccessToken()
      .then((newAccessToken) => sendRequest(options, newAccessToken))
      .catch((retryError) => {
        if (isUnauthorized(retryError)) clearLoginState()
        throw retryError
      })
  })
}

module.exports = { isUnauthorized, request, refreshAccessToken }
