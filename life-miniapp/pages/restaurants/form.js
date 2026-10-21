const { RestaurantService } = require('../../services/restaurant-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { requireWriteAccess } = require('../../utils/access')
const { RESTAURANT_CATEGORIES: categories } = require('../../utils/restaurant-category')
const { MODULES } = require('../../utils/permissions')

const defaultCategoryIndex = categories.length - 1

const preferences = [
  { value: 'okay', icon: '○', label: '还行' },
  { value: 'good', icon: '✦', label: '不错' },
  { value: 'love', icon: '♥', label: '喜欢' },
  { value: 'must', icon: '🔥', label: '必吃' },
]

Page(withScrollRestoration({
  data: {
    id: '',
    saving: false,
    categories,
    categoryIndex: defaultCategoryIndex,
    preferences,
    form: {
      name: '',
      category: '其他',
      icon: '🍽️',
      cuisine: '',
      preference: 'good',
      address: '',
      hours: '',
      tags: '',
      coverUrl: '',
      coverObjectKey: '',
      latitude: '',
      longitude: '',
    },
  },

  async onLoad({ id }) {
    const user = await requireWriteAccess(MODULES.RESTAURANT)
    if (!user) return
    this.currentUser = user
    if (!id) return
    this.setData({ id })
    wx.setNavigationBarTitle({ title: '编辑店铺' })
    this.loadRestaurant(id)
  },

  async loadRestaurant(id) {
    const restaurant = await RestaurantService.get(id)
    if (!restaurant) {
      wx.showToast({ title: '记录不存在', icon: 'none' })
      return
    }
    if (this.currentUser?.role !== 'admin' && String(restaurant.createdBy || '') !== String(this.currentUser?.id || '')) {
      wx.showToast({ title: '只能编辑自己创建的店铺', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 400)
      return
    }

    const matchedCategoryIndex = categories.findIndex((item) => item.label === restaurant.category)
    const categoryIndex = matchedCategoryIndex >= 0 ? matchedCategoryIndex : defaultCategoryIndex
    this.originalCoverUrl = restaurant.coverUrl || ''
    this.setData({
      categoryIndex,
      form: {
        name: restaurant.name || '',
        category: restaurant.category || categories[categoryIndex].label,
        icon: restaurant.icon || categories[categoryIndex].icon,
        cuisine: restaurant.cuisine || '',
        preference: restaurant.preference || 'good',
        address: restaurant.address || '',
        hours: restaurant.hours || '',
        tags: restaurant.tags || '',
        coverUrl: restaurant.coverUrl || '',
        coverObjectKey: restaurant.coverObjectKey || '',
        latitude: restaurant.latitude || '',
        longitude: restaurant.longitude || '',
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

  selectPreference({ currentTarget }) {
    this.setData({ 'form.preference': currentTarget.dataset.value })
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

  // wx.chooseLocation 暂时停用，避免小程序审核依赖位置接口权限。
  // 后续申请接口权限后，可恢复地图选点入口和此方法。
  // chooseLocation() {
  //   if (this.data.saving) return
  //   wx.chooseLocation({
  //     latitude: Number(this.data.form.latitude) || undefined,
  //     longitude: Number(this.data.form.longitude) || undefined,
  //     success: ({ name, address, latitude, longitude }) => {
  //       this.setData({
  //         'form.address': address || name || this.data.form.address,
  //         'form.latitude': String(latitude),
  //         'form.longitude': String(longitude),
  //       })
  //     },
  //   })
  // },

  clearLocation() {
    this.setData({ 'form.latitude': '', 'form.longitude': '' })
  },

  async saveRestaurant() {
    if (this.data.saving) return

    const form = this.data.form
    const name = form.name.trim()
    if (!name) {
      wx.showToast({ title: '请填写店名', icon: 'none' })
      return
    }

    this.setData({ saving: true })
    try {
      let coverUrl = form.coverUrl
      let coverObjectKey = form.coverObjectKey
      if (coverUrl && coverUrl !== this.originalCoverUrl && !coverObjectKey) {
        const uploaded = await RestaurantService.uploadCover(coverUrl)
        coverUrl = uploaded.url
        coverObjectKey = uploaded.objectKey
      }

      const input = {
        ...form,
        name,
        cuisine: form.cuisine.trim(),
        address: form.address.trim(),
        hours: form.hours.trim(),
        coverUrl,
        coverObjectKey,
      }

      const savedRestaurant = this.data.id
        ? await RestaurantService.update(this.data.id, input)
        : await RestaurantService.create(input)

      if (typeof this.getOpenerEventChannel === 'function') {
        const eventChannel = this.getOpenerEventChannel()
        if (eventChannel && typeof eventChannel.emit === 'function') {
          eventChannel.emit('restaurantChanged', {
            id: savedRestaurant.id,
            restaurant: savedRestaurant,
          })
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
