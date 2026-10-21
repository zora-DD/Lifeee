const { PlaceService } = require('../../services/place-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { cacheImage, getCachedImageURL } = require('../../utils/image-cache')
const { getCurrentUser, requireWriteAccess } = require('../../utils/access')
const { buildTravelMeta } = require('../../utils/place-travel')
const {
  MODULES,
  hasPermission,
  loadPermissions,
  redirectToFirstAllowedTab,
  syncTabBar,
} = require('../../utils/permissions')

const categories = [
  { label: '全部分类', value: 'all' },
  { label: '自然风景', value: '自然风景' },
  { label: '城市探索', value: '城市探索' },
  { label: '徒步登山', value: '徒步登山' },
  { label: '玩水亲水', value: '玩水亲水' },
  { label: '商场街区', value: '商场街区' },
  { label: '露营野餐', value: '露营野餐' },
  { label: '自驾路线', value: '自驾路线' },
  { label: '其他', value: '其他' },
]

const statusFilters = [
  { label: '全部', value: 'all' },
  { label: '去过', value: 'visited' },
  { label: '想去', value: 'wishlist' },
]

const PLACE_PAGE_SIZE = 20

function formatVisitDate(value) {
  if (!value) return ''
  const date = new Date(value)
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

Page(withScrollRestoration({
  data: {
    authChecked: false,
    loggedIn: false,
    canWrite: false,
    categories,
    categoryIndex: 0,
    selectedCategoryLabel: categories[0].label,
    categoryDropdownOpen: false,
    statusFilters,
    statusFilter: 'all',
    places: [],
    resultCount: 0,
    page: 1,
    hasMore: false,
    loadingMore: false,
    totalCount: 0,
    visitedCount: 0,
    wishlistCount: 0,
  },

  async onShow() {
    const reuseCurrentList = this.skipNextShowRefresh &&
      !this.placeListDirty &&
      this.data.authChecked &&
      this.data.loggedIn
    this.skipNextShowRefresh = false

    // 详情未发生变更时保留列表和滚动位置，不重复校验和请求。
    if (reuseCurrentList) return

    const preserveLoadedPages = this.placeListDirty && this.data.page > 1
    this.placeListDirty = false
    this.setData({ authChecked: false })
    const user = await getCurrentUser()
    if (!user || user.accessStatus !== 'approved') {
      this.setData({
        authChecked: true,
        loggedIn: false,
        places: [],
        totalCount: 0,
        visitedCount: 0,
        wishlistCount: 0,
      })
      return
    }
    const levels = await loadPermissions(user)
    syncTabBar(this, 3, levels)
    if (!hasPermission(levels, MODULES.PLACE)) {
      this.setData({ authChecked: true, loggedIn: false, places: [] })
      redirectToFirstAllowedTab(levels)
      return
    }
    this.setData({
      authChecked: true,
      loggedIn: true,
      canWrite: hasPermission(levels, MODULES.PLACE, 'write'),
    })
    await this.loadPlaces({ preserveLoadedPages })
  },

  onHide() {
    if (this.data.categoryDropdownOpen) {
      this.setData({ categoryDropdownOpen: false })
    }
  },

  async loadPlaces({ append = false, preserveLoadedPages = false } = {}) {
    if (append && (this.data.loadingMore || !this.data.hasMore)) return
    const category = categories[this.data.categoryIndex]
    const page = append ? this.data.page + 1 : 1
    const loadedPageCount = preserveLoadedPages ? Math.max(1, this.data.page) : 1
    const pages = preserveLoadedPages
      ? Array.from({ length: loadedPageCount }, (_, index) => index + 1)
      : [page]
    if (append) this.setData({ loadingMore: true })
    let results
    try {
      results = await Promise.all(pages.map((pageNumber) => PlaceService.listPage({
        category: category.value,
        status: this.data.statusFilter,
        page: pageNumber,
        pageSize: PLACE_PAGE_SIZE,
      })))
    } finally {
      if (append) this.setData({ loadingMore: false })
    }
    const result = results[results.length - 1]
    const summary = results[0]
    const pagePlaces = results.flatMap((pageResult) => pageResult.items.map((item) => {
      const coverSourceUrl = item.coverUrl || ''
      return {
        ...item,
        coverSourceUrl,
        coverUrl: getCachedImageURL(coverSourceUrl),
        lastVisitedText: formatVisitDate(item.lastVisitedAt),
        travelMeta: buildTravelMeta(item),
        visitText: item.visited
          ? `去过 ${item.visitCount} 次 · 上次 ${formatVisitDate(item.lastVisitedAt)}`
          : '还没去过 · 已加入想去',
      }
    }))
    this.setData({
      places: append ? [...this.data.places, ...pagePlaces] : pagePlaces,
      resultCount: result.total,
      page: preserveLoadedPages ? loadedPageCount : result.page,
      hasMore: result.hasMore,
      totalCount: summary.overallTotal,
      visitedCount: summary.visitedCount,
      wishlistCount: summary.wishlistCount,
    })
    this.refreshCoverImages(pagePlaces)
  },

  async refreshCoverImages(items) {
    const resolved = await Promise.all(items.map(async (item) => ({
      id: String(item.id),
      sourceURL: item.coverSourceUrl,
      localURL: await cacheImage(item.coverSourceUrl),
    })))
    const covers = new Map(resolved.filter((item) => item.localURL).map((item) => [item.id, item]))
    if (!covers.size) return
    this.setData({
      places: this.data.places.map((item) => {
        const cover = covers.get(String(item.id))
        if (!cover || item.coverSourceUrl !== cover.sourceURL || item.coverUrl === cover.localURL) return item
        return { ...item, coverUrl: cover.localURL }
      }),
    })
  },

  onReachBottom() {
    this.loadPlaces({ append: true })
  },

  toggleCategoryDropdown() {
    this.setData({ categoryDropdownOpen: !this.data.categoryDropdownOpen })
  },

  closeCategoryDropdown() {
    this.setData({ categoryDropdownOpen: false })
  },

  selectCategory({ currentTarget }) {
    const categoryIndex = Number(currentTarget.dataset.index)
    const selected = categories[categoryIndex]
    if (!selected) return
    if (categoryIndex === this.data.categoryIndex) {
      this.closeCategoryDropdown()
      return
    }
    this.setData({
      categoryIndex,
      selectedCategoryLabel: selected.label,
      categoryDropdownOpen: false,
    }, () => this.loadPlaces())
  },

  noop() {},

  selectStatus({ currentTarget }) {
    this.setData({
      statusFilter: currentTarget.dataset.value,
      categoryDropdownOpen: false,
    }, () => this.loadPlaces())
  },

  openDetail({ currentTarget }) {
    this.openPlacePage(`/pages/places/detail?id=${currentTarget.dataset.id}`)
  },

  openPlacePage(url) {
    this.skipNextShowRefresh = true
    wx.navigateTo({
      url,
      events: {
        placeChanged: () => {
          this.placeListDirty = true
        },
      },
      fail: () => {
        this.skipNextShowRefresh = false
      },
    })
  },

  async createPlace() {
    if (!await requireWriteAccess(MODULES.PLACE)) return
    this.openPlacePage('/pages/places/form')
  },
}))
