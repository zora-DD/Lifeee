const PENDING_ROUTE_KEY = 'life_pending_route_v1'
const PENDING_ROUTE_TTL = 7 * 24 * 60 * 60 * 1000

function isSafePagePath(path) {
  return typeof path === 'string'
    && path.startsWith('/pages/')
    && !path.includes('\\')
    && !path.includes('://')
}

function savePendingRoute(path) {
  if (!isSafePagePath(path) || path === '/pages/index/index') return
  wx.setStorageSync(PENDING_ROUTE_KEY, {
    path,
    savedAt: Date.now(),
  })
}

function clearPendingRoute() {
  wx.removeStorageSync(PENDING_ROUTE_KEY)
}

function getPendingRoute() {
  const pending = wx.getStorageSync(PENDING_ROUTE_KEY)
  if (!pending || !isSafePagePath(pending.path)) {
    clearPendingRoute()
    return ''
  }
  if (!pending.savedAt || Date.now() - Number(pending.savedAt) > PENDING_ROUTE_TTL) {
    clearPendingRoute()
    return ''
  }
  return pending.path
}

function resumePendingRoute() {
  const path = getPendingRoute()
  if (!path) return false

  wx.navigateTo({
    url: path,
    success: clearPendingRoute,
  })
  return true
}

module.exports = {
  clearPendingRoute,
  getPendingRoute,
  resumePendingRoute,
  savePendingRoute,
}
