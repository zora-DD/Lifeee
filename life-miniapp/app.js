const { API_BASE_URL, ENV } = require('./config/env')
const { CACHE_CONFIG } = require('./config/cache')
const { invalidateSessionCache } = require('./utils/session-cache')

let backgroundAt = 0

App({
  globalData: {
    env: ENV,
    apiBaseUrl: API_BASE_URL,
  },

  onHide() {
    backgroundAt = Date.now()
  },

  onShow() {
    if (!backgroundAt) return
    const backgroundDuration = Math.max(0, Date.now() - backgroundAt)
    backgroundAt = 0
    if (backgroundDuration >= CACHE_CONFIG.session.refreshAfterBackgroundMs) {
      invalidateSessionCache()
    }
  },
})
