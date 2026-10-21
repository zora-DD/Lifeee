function restorePageScroll(page) {
  if (!page.__shouldRestorePageScroll) return

  const scrollTop = Math.max(0, Number(page.__savedPageScrollTop) || 0)
  page.__shouldRestorePageScroll = false
  const restore = () => {
    if (typeof wx !== 'undefined' && typeof wx.pageScrollTo === 'function') {
      wx.pageScrollTo({ scrollTop, duration: 0 })
    }
  }

  if (typeof wx !== 'undefined' && typeof wx.nextTick === 'function') {
    wx.nextTick(restore)
    return
  }
  setTimeout(restore, 0)
}

// 页面进入下一级后返回时，在数据刷新和视图渲染完成后恢复原滚动位置。
function withScrollRestoration(options) {
  const originalOnShow = options.onShow
  const originalOnHide = options.onHide
  const originalOnPageScroll = options.onPageScroll

  return {
    ...options,

    async onShow(...args) {
      try {
        if (typeof originalOnShow === 'function') {
          return await originalOnShow.apply(this, args)
        }
        return undefined
      } finally {
        restorePageScroll(this)
      }
    },

    onHide(...args) {
      this.__shouldRestorePageScroll = true
      if (typeof originalOnHide === 'function') {
        return originalOnHide.apply(this, args)
      }
      return undefined
    },

    onPageScroll(event) {
      const scrollTop = Number(event && event.scrollTop)
      if (Number.isFinite(scrollTop)) this.__savedPageScrollTop = Math.max(0, scrollTop)
      if (typeof originalOnPageScroll === 'function') {
        return originalOnPageScroll.call(this, event)
      }
      return undefined
    },
  }
}

module.exports = { withScrollRestoration }
