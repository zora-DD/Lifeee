// 缓存策略集中配置，后续按实际使用频率调整即可。
const CACHE_CONFIG = Object.freeze({
  image: Object.freeze({
    ttlMs: 24 * 60 * 60 * 1000,
    maxFiles: 80,
  }),
  session: Object.freeze({
    // 0 表示小程序进入后台后，下次回到前台立即重新校验用户和权限。
    refreshAfterBackgroundMs: 0,
  }),
})

module.exports = { CACHE_CONFIG }
