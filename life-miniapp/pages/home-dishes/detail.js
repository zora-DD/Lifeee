const { HomeDishService } = require('../../services/home-dish-service')
const { getModuleAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')

const difficultyLabels = { easy: '简单', normal: '适中', hard: '挑战' }
const statusLabels = {
  wishlist: { value: 'wishlist', label: '想吃', icon: '♡' },
  eaten: { value: 'eaten', label: '吃过', icon: '✓' },
  frequent: { value: 'frequent', label: '常吃', icon: '♥' },
}

function today() {
  const date = new Date()
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}

function splitSteps(value) {
  return String(value || '')
    .split(/\r?\n/)
    .map((step) => step.trim().replace(/^(?:第?\s*\d+\s*[.、：:]|[-•])\s*/, ''))
    .filter(Boolean)
}

function toDishView(dish) {
  const status = dish.status || (dish.cookCount === 0 ? 'wishlist' : dish.cookCount >= 3 ? 'frequent' : 'eaten')
  return {
    ...dish,
    difficultyLabel: difficultyLabels[dish.difficulty] || '适中',
    statusInfo: statusLabels[status] || statusLabels.wishlist,
    stepItems: splitSteps(dish.steps),
  }
}

Page({
  data: {
    id: '',
    dish: null,
    canWrite: false,
    showCook: false,
    saving: false,
    ratingOptions: [1, 2, 3, 4, 5],
    record: { cookedAt: today(), durationMinutes: '', rating: 5, note: '' },
  },

  async onLoad(options) {
    this.setData({ id: options.id })
    await this.load()
  },

  async onShow() {
    if (getApp().globalData.homeDishChanged) {
      getApp().globalData.homeDishChanged = false
      await this.load()
    }
  },

  async load() {
    try {
      const [dish, access] = await Promise.all([
        HomeDishService.detail(this.data.id),
        getModuleAccess(MODULES.RESTAURANT, 'write'),
      ])
      this.setData({ dish: toDishView(dish), canWrite: Boolean(access.allowed) })
    } catch (error) {
      wx.showToast({ title: error.message || '加载失败', icon: 'none' })
    }
  },

  edit() {
    wx.navigateTo({ url: `/pages/home-dishes/form?id=${this.data.id}` })
  },

  noop() {},

  toggleCook() {
    this.setData({ showCook: !this.data.showCook })
  },

  input({ currentTarget, detail }) {
    this.setData({ [`record.${currentTarget.dataset.field}`]: detail.value })
  },

  dateChange({ detail }) {
    this.setData({ 'record.cookedAt': detail.value })
  },

  ratingChange({ detail }) {
    this.setData({ 'record.rating': Number(detail.value) + 1 })
  },

  async saveCook() {
    if (this.data.saving) return
    this.setData({ saving: true })
    try {
      const dish = await HomeDishService.cook(this.data.id, this.data.record)
      this.setData({
        dish: toDishView(dish),
        showCook: false,
        record: { cookedAt: today(), durationMinutes: '', rating: 5, note: '' },
      })
      getApp().globalData.homeDishChanged = true
      wx.showToast({ title: '已记录这次制作' })
    } catch (error) {
      wx.showToast({ title: error.message || '记录失败', icon: 'none' })
    } finally {
      this.setData({ saving: false })
    }
  },

  remove() {
    wx.showModal({
      title: '删除这道菜？',
      content: '制作记录也将不再展示。',
      success: async ({ confirm }) => {
        if (!confirm) return
        try {
          await HomeDishService.remove(this.data.id)
          getApp().globalData.homeDishChanged = true
          wx.navigateBack()
        } catch (error) {
          wx.showToast({ title: error.message || '删除失败', icon: 'none' })
        }
      },
    })
  },
})
