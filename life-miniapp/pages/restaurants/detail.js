const { RestaurantService } = require('../../services/restaurant-service')
const { BillService } = require('../../services/bill-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { FavoriteService } = require('../../services/favorite-service')
const { requireApprovedAccess, requireWriteAccess } = require('../../utils/access')
const { buildShareCard } = require('../../utils/share')
const { MODULES, hasPermission, loadPermissions } = require('../../utils/permissions')
const { pickerValuesToISO } = require('../../utils/date')

const preferenceMap = {
  okay: { icon: '○', label: '还行' },
  good: { icon: '✦', label: '不错' },
  love: { icon: '♥', label: '喜欢' },
  must: { icon: '🔥', label: '必吃' },
}

function formatVisitDate(value) {
  if (!value) return '还没有记录'
  const date = new Date(value)
  return `${date.getFullYear()}年${date.getMonth() + 1}月${date.getDate()}日`
}

function formatMoney(value, emptyText = '¥0') {
  const amount = Math.max(0, Number(value) || 0)
  if (!amount) return emptyText
  return `¥${Number.isInteger(amount) ? amount : amount.toFixed(2)}`
}

function pad(value) {
  return String(value).padStart(2, '0')
}

function today() {
  const date = new Date()
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function confirmBillLink(name, amount) {
  return new Promise((resolve) => {
    wx.showModal({
      title: '同步到账单？',
      content: `${name}本次消费 ${formatMoney(amount)}，是否同时记入餐饮账单？`,
      confirmText: '记入账单',
      confirmColor: '#6f9785',
      success: ({ confirm }) => resolve(Boolean(confirm)),
      fail: () => resolve(false),
    })
  })
}

Page(withScrollRestoration({
  data: {
    id: '',
    restaurant: null,
    favorited: false,
    favoriteSaving: false,
    canWrite: false,
    canWriteBill: false,
    showConsumptionForm: false,
    savingConsumption: false,
    canEdit: false,
    consumptionForm: {
      consumedAt: today(),
      amount: '',
      meal: '',
      note: '',
    },
  },

  onLoad({ id }) {
    this.setData({ id })
  },

  async onShow() {
    if (this.skipNextShowRefresh && this.data.restaurant) {
      this.skipNextShowRefresh = false
      return
    }
    this.skipNextShowRefresh = false
    this.setData({ restaurant: null })
    const user = await requireApprovedAccess(`/pages/restaurants/detail?id=${encodeURIComponent(this.data.id)}`, MODULES.RESTAURANT)
    if (!user) return
    this.currentUser = user
    const levels = await loadPermissions(user)
    this.canWrite = hasPermission(levels, MODULES.RESTAURANT, 'write')
    this.canWriteBill = hasPermission(levels, MODULES.BILL, 'write')
    await this.loadRestaurant()
  },

  onShareAppMessage() {
    const restaurant = this.data.restaurant
    return buildShareCard({
      title: restaurant ? `吃喝推荐｜${restaurant.name}` : '',
      path: `/pages/restaurants/detail?id=${encodeURIComponent(this.data.id)}`,
      imageUrl: restaurant && restaurant.coverUrl ? restaurant.coverUrl : undefined,
    })
  },

  async loadRestaurant() {
    const restaurant = await RestaurantService.get(this.data.id)
    if (!restaurant) {
      wx.showToast({ title: '记录不存在', icon: 'none' })
      return
    }

    this.renderRestaurant(restaurant)
    this.loadFavoriteStatus()
  },

  renderRestaurant(restaurant) {
    this.setData({
      restaurant: {
        ...restaurant,
        lastVisitedText: formatVisitDate(restaurant.lastVisitedAt),
        totalSpentText: formatMoney(restaurant.totalSpent),
        lastAmountText: formatMoney(restaurant.lastAmount, '金额未记录'),
        preferenceInfo: preferenceMap[restaurant.preference] || preferenceMap.good,
        hasLocation: Number.isFinite(Number(restaurant.latitude)) && Number.isFinite(Number(restaurant.longitude)),
        consumptions: (restaurant.consumptions || []).map((item) => ({
          ...item,
          dateText: formatVisitDate(item.consumedAt),
          amountText: formatMoney(item.amount, '金额未记录'),
        })),
      },
      canWrite: this.canWrite,
      canWriteBill: this.canWriteBill,
      canEdit: this.canWrite && (this.currentUser?.role === 'admin' || String(restaurant.creator?.id || restaurant.createdBy || '') === String(this.currentUser?.id || '')),
    })
  },

  async loadFavoriteStatus() {
    try {
      const favorited = await FavoriteService.status('restaurant', this.data.id)
      this.setData({ favorited })
    } catch (error) {
      this.setData({ favorited: false })
    }
  },

  async toggleFavorite() {
    if (this.data.favoriteSaving || !await requireWriteAccess(MODULES.RESTAURANT)) return
    const next = !this.data.favorited
    this.setData({ favoriteSaving: true })
    try {
      const favorited = await FavoriteService.modify('restaurant', this.data.id, next)
      this.setData({ favorited })
      wx.showToast({ title: favorited ? '已收藏' : '已取消收藏', icon: 'none' })
    } catch (error) {
      wx.showToast({ title: error.message || '操作失败', icon: 'none' })
    } finally {
      this.setData({ favoriteSaving: false })
    }
  },

  async editRestaurant() {
    const ownerID = this.data.restaurant?.creator?.id || this.data.restaurant?.createdBy || ''
    if ((ownerID && !this.data.canEdit) || !await requireWriteAccess(MODULES.RESTAURANT)) return
    this.skipNextShowRefresh = true
    wx.navigateTo({
      url: `/pages/restaurants/form?id=${this.data.id}`,
      events: {
        restaurantChanged: ({ restaurant } = {}) => {
          if (restaurant) this.renderRestaurant(restaurant)
          this.notifyRestaurantChanged()
        },
      },
      fail: () => {
        this.skipNextShowRefresh = false
      },
    })
  },

  notifyRestaurantChanged() {
    if (typeof this.getOpenerEventChannel !== 'function') return
    const eventChannel = this.getOpenerEventChannel()
    if (eventChannel && typeof eventChannel.emit === 'function') {
      eventChannel.emit('restaurantChanged', { id: this.data.id })
    }
  },

  async openConsumptionForm() {
    if (!await requireWriteAccess(MODULES.RESTAURANT)) return
    this.setData({
      showConsumptionForm: true,
      consumptionForm: {
        consumedAt: today(),
        amount: '',
        meal: '',
        note: '',
      },
    })
  },

  closeConsumptionForm() {
    if (this.data.savingConsumption) return
    this.setData({ showConsumptionForm: false })
  },

  onConsumptionInput({ currentTarget, detail }) {
    this.setData({ [`consumptionForm.${currentTarget.dataset.field}`]: detail.value })
  },

  onConsumptionDateChange({ detail }) {
    this.setData({ 'consumptionForm.consumedAt': detail.value })
  },

  async saveConsumption() {
    if (this.data.savingConsumption) return
    const form = this.data.consumptionForm
    const amountText = String(form.amount || '').trim()
    if (amountText && (!/^\d+(\.\d{1,2})?$/.test(amountText) || Number(amountText) < 0)) {
      wx.showToast({ title: '请输入正确金额', icon: 'none' })
      return
    }
    this.setData({ savingConsumption: true })

    try {
      const consumption = await RestaurantService.addConsumption(this.data.id, {
        consumedAt: form.consumedAt,
        amount: amountText,
        meal: form.meal.trim(),
        note: form.note.trim(),
      })
      this.setData({ showConsumptionForm: false, savingConsumption: false })
      await this.loadRestaurant()
      this.notifyRestaurantChanged()
      await this.offerBillLink(form, consumption)
    } catch (error) {
      this.setData({ savingConsumption: false })
      wx.showToast({ title: error.message || '记录失败', icon: 'none' })
    }
  },

  async offerBillLink(form, consumption) {
    const amount = Number(form.amount) || 0
    if (!amount || !this.canWriteBill) {
      wx.showToast({ title: '已记录本次消费' })
      return
    }
    const restaurantName = this.data.restaurant?.name || '吃喝消费'
    const confirmed = await confirmBillLink(restaurantName, amount)
    if (!confirmed) {
      wx.showToast({ title: '消费记录已保存' })
      return
    }
    if (!await requireWriteAccess(MODULES.BILL)) return
    const noteParts = [form.meal.trim(), form.note.trim(), consumption?.id ? `吃喝记录 #${consumption.id}` : '来自吃喝记录']
      .filter(Boolean)
    try {
      await BillService.create({
        type: 'expense',
        title: restaurantName,
        amount: String(form.amount).trim(),
        category: '餐饮',
        occurredAt: pickerValuesToISO(form.consumedAt, '12:00'),
        note: noteParts.join(' · '),
        inputType: 'text',
        rawText: '',
      })
      wx.showToast({ title: '消费与账单已保存' })
    } catch (error) {
      wx.showModal({
        title: '消费已保存',
        content: `同步账单失败：${error.message || '请稍后重试'}`,
        showCancel: false,
      })
    }
  },

  openNavigation() {
    const restaurant = this.data.restaurant
    const latitude = Number(restaurant?.latitude)
    const longitude = Number(restaurant?.longitude)
    if (!Number.isFinite(latitude) || !Number.isFinite(longitude)) {
      wx.showToast({ title: '还没有保存地图位置', icon: 'none' })
      return
    }
    wx.openLocation({
      latitude,
      longitude,
      name: restaurant.name,
      address: restaurant.address || '',
      scale: 16,
    })
  },

  noop() {
  },
}))
