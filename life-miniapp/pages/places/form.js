const { PlaceService } = require('../../services/place-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { requireWriteAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')

const categories = [
  { label: '自然风景', icon: '🌿' },
  { label: '城市探索', icon: '🌇' },
  { label: '徒步登山', icon: '🥾' },
  { label: '玩水亲水', icon: '🌊' },
  { label: '商场街区', icon: '🏙️' },
  { label: '露营野餐', icon: '⛺' },
  { label: '自驾路线', icon: '🚗' },
  { label: '其他', icon: '📍' },
]

function pad(value) {
  return String(value).padStart(2, '0')
}

function today() {
  const date = new Date()
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

Page(withScrollRestoration({
  data: {
    id: '',
    saving: false,
    categories,
    categoryIndex: 0,
    form: {
      name: '',
      category: '自然风景',
      icon: '🌿',
      visited: false,
      visitCount: '1',
      lastVisitedAt: today(),
      feature: '',
      note: '',
      address: '',
      bestTime: '',
      guideUrl: '',
      distanceKm: '',
      driveDurationHours: '',
      coverUrl: '',
      coverObjectKey: '',
      latitude: null,
      longitude: null,
    },
  },

  async onLoad({ id }) {
    const user = await requireWriteAccess(MODULES.PLACE)
    if (!user) return
    this.currentUser = user
    if (!id) return
    this.setData({ id })
    wx.setNavigationBarTitle({ title: '编辑地点' })
    this.loadPlace(id)
  },

  async loadPlace(id) {
    const place = await PlaceService.get(id)
    if (!place) {
      wx.showToast({ title: '地点不存在', icon: 'none' })
      return
    }
    if (this.currentUser?.role !== 'admin' && String(place.creator?.id || place.createdBy || '') !== String(this.currentUser?.id || '')) {
      wx.showToast({ title: '只能编辑自己创建的地点', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 400)
      return
    }

    const categoryIndex = Math.max(0, categories.findIndex((item) => item.label === place.category))
    this.originalCoverUrl = place.coverUrl || ''
    this.setData({
      categoryIndex,
      form: {
        name: place.name || '',
        category: place.category || categories[categoryIndex].label,
        icon: place.icon || categories[categoryIndex].icon,
        visited: Boolean(place.visited),
        visitCount: String(place.visitCount || 1),
        lastVisitedAt: place.lastVisitedAt || today(),
        feature: place.feature || '',
        note: place.note || '',
        address: place.address || '',
        bestTime: place.bestTime || '',
        guideUrl: place.guideUrl || '',
        distanceKm: place.distanceKm ? String(place.distanceKm) : '',
        driveDurationHours: place.driveDurationMinutes
          ? String(Math.round(place.driveDurationMinutes / 6) / 10)
          : '',
        coverUrl: place.coverUrl || '',
        coverObjectKey: place.coverObjectKey || '',
        latitude: place.latitude || null,
        longitude: place.longitude || null,
      },
    })
  },

  onInput({ currentTarget, detail }) {
    this.setData({ [`form.${currentTarget.dataset.field}`]: detail.value })
  },

  onCategoryChange({ detail }) {
    const categoryIndex = Number(detail.value)
    const category = categories[categoryIndex]
    this.setData({
      categoryIndex,
      'form.category': category.label,
      'form.icon': category.icon,
    })
  },

  selectVisitStatus({ currentTarget }) {
    this.setData({ 'form.visited': currentTarget.dataset.value === 'visited' })
  },

  onVisitDateChange({ detail }) {
    this.setData({ 'form.lastVisitedAt': detail.value })
  },

  chooseCover() {
    if (this.data.saving) return
    wx.chooseImage({
      count: 1,
      sizeType: ['compressed'],
      sourceType: ['album', 'camera'],
      success: ({ tempFilePaths }) => {
        if (!tempFilePaths || !tempFilePaths[0]) return
        this.setData({
          'form.coverUrl': tempFilePaths[0],
          'form.coverObjectKey': '',
        })
      },
    })
  },

  removeCover() {
    this.setData({ 'form.coverUrl': '', 'form.coverObjectKey': '' })
  },

  async savePlace() {
    if (this.data.saving) return
    const form = this.data.form
    const name = form.name.trim()
    if (!name) {
      wx.showToast({ title: '请填写地点名称', icon: 'none' })
      return
    }
    if (!form.feature.trim()) {
      wx.showToast({ title: '请填写地点特色', icon: 'none' })
      return
    }

    this.setData({ saving: true })
    try {
      let coverUrl = form.coverUrl
      let coverObjectKey = form.coverObjectKey
      if (coverUrl && coverUrl !== this.originalCoverUrl && !coverObjectKey) {
        const uploaded = await PlaceService.uploadCover(coverUrl)
        coverUrl = uploaded.url
        coverObjectKey = uploaded.objectKey
      }

      const input = {
        ...form,
        name,
        feature: form.feature.trim(),
        note: form.note.trim(),
        address: form.address.trim(),
        bestTime: form.bestTime.trim(),
        guideUrl: form.guideUrl.trim(),
        distanceKm: Math.max(0, Math.round(Number(form.distanceKm) || 0)),
        driveDurationMinutes: Math.max(0, Math.round((Number(form.driveDurationHours) || 0) * 60)),
        visitCount: form.visited ? Math.max(1, Number(form.visitCount) || 1) : 0,
        lastVisitedAt: form.visited ? form.lastVisitedAt : '',
        coverUrl,
        coverObjectKey,
      }

      let savedPlace
      if (this.data.id) {
        savedPlace = await PlaceService.update(this.data.id, input)
      } else {
        savedPlace = await PlaceService.create(input)
      }
      if (typeof this.getOpenerEventChannel === 'function') {
        const eventChannel = this.getOpenerEventChannel()
        if (eventChannel && typeof eventChannel.emit === 'function') {
          eventChannel.emit('placeChanged', { place: savedPlace })
        }
      }

      wx.showToast({ title: this.data.id ? '已更新' : '已保存' })
      setTimeout(() => wx.navigateBack(), 450)
    } catch (error) {
      wx.showToast({ title: error.message || '保存失败', icon: 'none' })
      this.setData({ saving: false })
    }
  },
}))
