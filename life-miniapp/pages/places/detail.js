const { PlaceService } = require('../../services/place-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { FavoriteService } = require('../../services/favorite-service')
const { requireApprovedAccess, requireWriteAccess } = require('../../utils/access')
const { formatDriveDuration } = require('../../utils/place-travel')
const { MODULES, hasPermission, loadPermissions } = require('../../utils/permissions')

function formatVisitDate(value) {
  if (!value) return '还没有到访记录'
  const date = new Date(value)
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日`
}

function decoratePlace(place) {
  return {
    ...place,
    lastVisitedText: formatVisitDate(place.lastVisitedAt),
    distanceText: place.distanceKm ? `约 ${place.distanceKm} km` : '待补充',
    driveDurationText: place.driveDurationMinutes
      ? `约 ${formatDriveDuration(place.driveDurationMinutes)}`
      : '待补充',
  }
}

Page(withScrollRestoration({
  data: {
    id: '',
    place: null,
    favorited: false,
    favoriteSaving: false,
    canWrite: false,
    canEdit: false,
  },

  onLoad({ id }) {
    this.setData({ id })
  },

  async onShow() {
    if (this.skipNextShowRefresh && this.data.place) {
      this.skipNextShowRefresh = false
      return
    }
    this.skipNextShowRefresh = false
    this.setData({ place: null })
    const user = await requireApprovedAccess(`/pages/places/detail?id=${encodeURIComponent(this.data.id)}`, MODULES.PLACE)
    if (!user) return
    this.currentUser = user
    this.canWrite = hasPermission(await loadPermissions(user), MODULES.PLACE, 'write')
    await this.loadPlace()
  },

  async loadPlace() {
    const place = await PlaceService.get(this.data.id)
    if (!place) {
      wx.showToast({ title: '地点不存在', icon: 'none' })
      return
    }
    this.setData({
      place: decoratePlace(place),
      canWrite: this.canWrite,
      canEdit: this.canWrite && (this.currentUser?.role === 'admin' || String(place.creator?.id || place.createdBy || '') === String(this.currentUser?.id || '')),
    })
    this.loadFavoriteStatus()
  },

  async loadFavoriteStatus() {
    try {
      const favorited = await FavoriteService.status('place', this.data.id)
      this.setData({ favorited })
    } catch (error) {
      this.setData({ favorited: false })
    }
  },

  async toggleFavorite() {
    if (this.data.favoriteSaving || !await requireWriteAccess(MODULES.PLACE)) return
    const next = !this.data.favorited
    this.setData({ favoriteSaving: true })
    try {
      const favorited = await FavoriteService.modify('place', this.data.id, next)
      this.setData({ favorited })
      wx.showToast({ title: favorited ? '已收藏' : '已取消收藏', icon: 'none' })
    } catch (error) {
      wx.showToast({ title: error.message || '操作失败', icon: 'none' })
    } finally {
      this.setData({ favoriteSaving: false })
    }
  },

  openGuideArticle() {
    const place = this.data.place || {}
    if (!place.guideUrl) return
    if (typeof wx.openOfficialAccountArticle !== 'function') {
      wx.showModal({
        title: '暂时无法打开',
        content: '当前微信版本不支持打开公众号文章，请升级微信后重试。',
        showCancel: false,
      })
      return
    }
    wx.openOfficialAccountArticle({
      url: place.guideUrl,
      fail: () => wx.showToast({ title: '文章打开失败', icon: 'none' }),
    })
  },

  async editPlace() {
    if (!this.data.canEdit || !await requireWriteAccess(MODULES.PLACE)) return
    wx.navigateTo({
      url: `/pages/places/form?id=${this.data.id}`,
      events: {
        placeChanged: ({ place } = {}) => {
          if (place) {
            this.setData({
              place: decoratePlace(place),
            })
            this.skipNextShowRefresh = true
          }
          this.emitToPlaceList('placeChanged', { place })
        },
      },
    })
  },

  emitToPlaceList(eventName, payload) {
    if (typeof this.getOpenerEventChannel !== 'function') return
    const eventChannel = this.getOpenerEventChannel()
    if (eventChannel && typeof eventChannel.emit === 'function') eventChannel.emit(eventName, payload)
  },
}))
