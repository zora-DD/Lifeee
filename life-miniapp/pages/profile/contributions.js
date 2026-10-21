const { ProfileService } = require('../../services/profile-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { requireLogin } = require('../../utils/access')
const { toContentCard } = require('../../utils/content-summary')
const { MODULES, hasPermission, loadPermissions } = require('../../utils/permissions')

const contentFilters = [
  { value: 'task', label: '任务', module: MODULES.TASK },
  { value: 'restaurant', label: '吃喝', module: MODULES.RESTAURANT },
  { value: 'place', label: '地点', module: MODULES.PLACE },
]

Page(withScrollRestoration({
  data: {
    loading: true,
    loadingMore: false,
    filters: [],
    activeType: 'all',
    items: [],
    page: 1,
    hasMore: false,
  },

  async onShow() {
    const user = await requireLogin()
    if (!user || user.accessStatus !== 'approved') return
    const levels = await loadPermissions(user)
    const available = contentFilters.filter((item) => hasPermission(levels, item.module))
    const filters = available.length === contentFilters.length ? [{ value: 'all', label: '全部' }, ...available] : available
    const activeType = filters.some((item) => item.value === this.data.activeType)
      ? this.data.activeType
      : (filters[0]?.value || '')
    this.setData({ filters, activeType })
    if (!activeType) {
      this.setData({ loading: false, items: [], hasMore: false })
      return
    }
    await this.loadContributions(true)
  },

  onPullDownRefresh() {
    this.loadContributions(true).finally(() => wx.stopPullDownRefresh())
  },

  onReachBottom() {
    if (!this.data.loading && !this.data.loadingMore && this.data.hasMore) {
      this.loadContributions(false)
    }
  },

  async loadContributions(reset) {
    const page = reset ? 1 : this.data.page + 1
    this.setData(reset ? { loading: true } : { loadingMore: true })
    try {
      const result = await ProfileService.listContributions({
        targetType: this.data.activeType,
        page,
        pageSize: 20,
      })
      const nextItems = result.items.map(toContentCard)
      this.setData({
        items: reset ? nextItems : [...this.data.items, ...nextItems],
        page: result.page,
        hasMore: result.hasMore,
      })
    } catch (error) {
      wx.showToast({ title: error.message || '加载失败', icon: 'none' })
    } finally {
      this.setData({ loading: false, loadingMore: false })
    }
  },

  selectType({ currentTarget }) {
    const activeType = currentTarget.dataset.value
    if (activeType === this.data.activeType) return
    this.setData({ activeType, items: [], page: 1, hasMore: false }, () => this.loadContributions(true))
  },

  openItem({ currentTarget }) {
    wx.navigateTo({ url: currentTarget.dataset.url })
  },
}))
