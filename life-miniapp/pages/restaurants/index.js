const { RestaurantService } = require('../../services/restaurant-service')
const { HomeDishService } = require('../../services/home-dish-service')
const { SnackService } = require('../../services/snack-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { cacheImage, getCachedImageURL } = require('../../utils/image-cache')
const { getCurrentUser, requireWriteAccess } = require('../../utils/access')
const { RESTAURANT_FILTER_CATEGORIES: categories } = require('../../utils/restaurant-category')
const {
  MODULES,
  hasPermission,
  loadPermissions,
  redirectToFirstAllowedTab,
  syncTabBar,
} = require('../../utils/permissions')

const RESTAURANT_LIST_CACHE_TTL = 5 * 60 * 1000
const RESTAURANT_PAGE_SIZE = 20
const HOME_DISH_PAGE_SIZE = 20
const homeCategories = ['全部分类', '家常菜', '快手菜', '汤羹', '面食', '烘焙', '其他']
const homeStatuses = [
  { value: '', label: '全部' },
  { value: 'wishlist', label: '想吃' },
  { value: 'eaten', label: '吃过' },
  { value: 'frequent', label: '常吃' },
]
const snackCategories = ['全部分类', '肉干卤味', '坚果炒货', '方便速食', '糖果巧克力', '饼干糕点', '饮料乳品', '其他']
const snackStatuses = [
  { value: '', label: '全部状态' },
  { value: 'wishlist', label: '想吃' },
  { value: 'eaten', label: '吃过' },
  { value: 'frequent', label: '常买' },
  { value: 'avoid', label: '避雷' },
]

const visitStatuses = [
  { value: 'all', label: '全部状态' },
  { value: 'wishlist', label: '想去' },
  { value: 'visited', label: '吃过' },
  { value: 'frequent', label: '常去' },
]

const preferenceMap = {
  okay: { icon: '○', label: '还行' },
  good: { icon: '✦', label: '不错' },
  love: { icon: '♥', label: '喜欢' },
  must: { icon: '🔥', label: '必吃' },
}

function formatVisitDate(value) {
  if (!value) return '还没有记录'
  const date = new Date(value)
  return `${date.getMonth() + 1}月${date.getDate()}日`
}

function formatMoney(value) {
  const amount = Math.max(0, Number(value) || 0)
  return `¥${Number.isInteger(amount) ? amount : amount.toFixed(2)}`
}

function getVisitStatus(visitCount) {
  const count = Number(visitCount) || 0
  if (count === 0) return { value: 'wishlist', label: '想去' }
  if (count >= 3) return { value: 'frequent', label: '常去' }
  return { value: 'visited', label: '吃过' }
}

function toRestaurantView(item) {
  const coverSourceUrl = item.coverUrl || ''
  return {
    ...item,
    coverSourceUrl,
    coverUrl: getCachedImageURL(coverSourceUrl),
    lastVisitedText: formatVisitDate(item.lastVisitedAt),
    totalSpentText: formatMoney(item.totalSpent),
    preferenceInfo: preferenceMap[item.preference] || preferenceMap.good,
    visitStatus: getVisitStatus(item.visitCount),
  }
}

// 接口异常时退回当前筛选结果随机选择，保证“帮我选”仍可使用。
function pickRestaurant(items) {
  if (!items.length) return null
  return items[Math.floor(Math.random() * items.length)]
}

function getHomeDishStatus(cookCount, status) {
  const value = status || (Number(cookCount) === 0 ? 'wishlist' : Number(cookCount) >= 3 ? 'frequent' : 'eaten')
  return {
    wishlist: { value, label: '想吃', icon: '♡' },
    eaten: { value, label: '吃过', icon: '✓' },
    frequent: { value, label: '常吃', icon: '♥' },
  }[value] || { value: 'wishlist', label: '想吃', icon: '♡' }
}

function toHomeDishView(item) {
  const source = item.coverUrl || ''
  const cookCount = Number(item.cookCount) || 0
  const lastCookedText = item.lastCookedAt ? item.lastCookedAt.slice(5).replace('-', '月') + '日' : '还没做过'
  return {
    ...item,
    coverUrl: getCachedImageURL(source) || source,
    difficultyLabel: { easy: '简单', normal: '适中', hard: '挑战' }[item.difficulty] || '适中',
    lastCookedText,
    statusInfo: getHomeDishStatus(cookCount, item.status),
    progressText: cookCount ? `做过 ${cookCount} 次 · ${lastCookedText}` : '还没做过，找一天试试看',
  }
}

function toSnackView(item) {
  const source = item.coverUrl || ''
  const statusInfo = {
    wishlist: { label: '想吃', icon: '♡' }, eaten: { label: '吃过', icon: '✓' },
    frequent: { label: '常买', icon: '♥' }, avoid: { label: '避雷', icon: '!' },
  }[item.status] || { label: '想吃', icon: '♡' }
  return {
    ...item,
    coverSourceUrl: source,
    coverUrl: getCachedImageURL(source) || source,
    statusInfo,
    lastTastedText: item.lastTastedAt ? item.lastTastedAt.slice(5).replace('-', '月') + '日' : '还没吃过',
  }
}

Page(withScrollRestoration({
  data: {
    authChecked: false,
    loggedIn: false,
    canWrite: false,
    viewMode: 'outside',
    categories,
    categoryIndex: 0,
    selectedCategoryLabel: categories[0].label,
    visitStatuses,
    statusIndex: 0,
    selectedStatusLabel: visitStatuses[0].label,
    restaurants: [],
    page: 1,
    hasMore: false,
    loadingMore: false,
    totalCount: 0,
    totalVisitCount: 0,
    totalSpentText: '¥0',
    showRecommendation: false,
    isPicking: false,
    rollingRestaurant: null,
    recommendedRestaurant: null,
    homeCategories,
    homeCategoryIndex: 0,
    homeStatuses,
    homeStatusIndex: 0,
    showHomeCategoryMenu: false,
    showHomeStatusMenu: false,
    homeDishes: [],
    homePage: 1,
    homeHasMore: false,
    homeLoadingMore: false,
    homeTotal: 0,
    homePicking: false,
    showHomeRecommendation: false,
    recommendedHomeDish: null,
    snackCategories,
    snackCategoryIndex: 0,
    snackStatuses,
    snackStatusIndex: 0,
    showSnackCategoryMenu: false,
    showSnackStatusMenu: false,
    snacks: [],
    snackPage: 1,
    snackHasMore: false,
    snackLoadingMore: false,
    snackTotal: 0,
    snackPicking: false,
    showSnackRecommendation: false,
    recommendedSnack: null,
  },

  async onShow() {
    const now = Date.now()
    const returningFromDetail = this.skipNextShowRefresh &&
      !this.restaurantListDirty &&
      this.data.authChecked &&
      this.data.loggedIn
    const cacheIsFresh = this.lastRestaurantLoadAt &&
      now >= this.lastRestaurantLoadAt &&
      now - this.lastRestaurantLoadAt < RESTAURANT_LIST_CACHE_TTL
    const reuseCurrentList = (this.skipNextShowRefresh || cacheIsFresh) &&
      !this.restaurantListDirty &&
      this.data.authChecked &&
      this.data.loggedIn
    this.skipNextShowRefresh = false

    // 详情未发生变更时直接复用当前页面，避免返回时权限请求造成页面闪烁。
    if (returningFromDetail) return

    const preserveLoadedPages = this.restaurantListDirty && this.data.page > 1
    this.restaurantListDirty = false
    if (!reuseCurrentList) this.setData({ authChecked: false })
    const user = await getCurrentUser()
    if (!user || user.accessStatus !== 'approved') {
      this.clearRecommendationTimer()
      this.setData({
        authChecked: true,
        loggedIn: false,
        restaurants: [],
        totalCount: 0,
        totalVisitCount: 0,
        totalSpentText: '¥0',
        showRecommendation: false,
        isPicking: false,
      })
      this.lastRestaurantLoadAt = 0
      return
    }
    const levels = await loadPermissions(user)
    syncTabBar(this, 2, levels)
    if (!hasPermission(levels, MODULES.RESTAURANT)) {
      this.setData({ authChecked: true, loggedIn: false, restaurants: [] })
      redirectToFirstAllowedTab(levels)
      return
    }
    this.setData({
      authChecked: true,
      loggedIn: true,
      canWrite: hasPermission(levels, MODULES.RESTAURANT, 'write'),
    })
    if (this.data.viewMode === 'home') {
      if (!this.data.homeDishes.length || getApp().globalData.homeDishChanged) {
        getApp().globalData.homeDishChanged = false
        await this.loadHomeDishes()
      }
      return
    }
    if (this.data.viewMode === 'snack') {
      if (!this.data.snacks.length || getApp().globalData.snackChanged) {
        getApp().globalData.snackChanged = false
        await this.loadSnacks()
      }
      return
    }
    if (reuseCurrentList) return
    await this.loadRestaurants({ preserveLoadedPages })
  },

  onUnload() {
    this.clearRecommendationTimer()
  },

  onHide() {
    if (this.data.isPicking) this.closeRecommendation()
    if (this.data.showHomeRecommendation) this.closeHomeRecommendation()
    if (this.data.showHomeCategoryMenu || this.data.showHomeStatusMenu) this.closeHomeFilterMenus()
    if (this.data.showSnackRecommendation) this.closeSnackRecommendation()
    if (this.data.showSnackCategoryMenu || this.data.showSnackStatusMenu) this.closeSnackFilterMenus()
  },

  openDetail({ currentTarget }) {
    this.openRestaurantPage(`/pages/restaurants/detail?id=${currentTarget.dataset.id}`)
  },

  openRestaurantPage(url) {
    this.skipNextShowRefresh = true
    wx.navigateTo({
      url,
      events: {
        restaurantChanged: () => {
          this.restaurantListDirty = true
        },
      },
      fail: () => {
        this.skipNextShowRefresh = false
      },
    })
  },

  onCategoryChange({ detail }) {
    this.applyFilter(Number(detail.value))
  },

  onStatusChange({ detail }) {
    this.applyFilter(this.data.categoryIndex, { statusIndex: Number(detail.value) })
  },

  onReachBottom() {
    if (this.data.viewMode === 'home') {
      this.loadHomeDishes({ append: true })
      return
    }
    if (this.data.viewMode === 'snack') {
      this.loadSnacks({ append: true })
      return
    }
    this.loadRestaurants({ append: true })
  },

  switchFoodMode({currentTarget}){const mode=currentTarget.dataset.mode;if(mode===this.data.viewMode)return;this.clearRecommendationTimer();this.setData({viewMode:mode,showRecommendation:false,showHomeRecommendation:false,showSnackRecommendation:false,showHomeCategoryMenu:false,showHomeStatusMenu:false,showSnackCategoryMenu:false,showSnackStatusMenu:false,recommendedHomeDish:null,recommendedSnack:null});if(mode==='home'&&this.data.loggedIn&&!this.data.homeDishes.length)this.loadHomeDishes();if(mode==='snack'&&this.data.loggedIn&&!this.data.snacks.length)this.loadSnacks()},
  async loadHomeDishes({append=false}={}){if(append&&(this.data.homeLoadingMore||!this.data.homeHasMore))return;const page=append?this.data.homePage+1:1;if(append)this.setData({homeLoadingMore:true});try{const result=await HomeDishService.list({category:this.data.homeCategoryIndex?homeCategories[this.data.homeCategoryIndex]:'',status:homeStatuses[this.data.homeStatusIndex].value,page,pageSize:HOME_DISH_PAGE_SIZE});const rows=result.items.map(toHomeDishView);this.setData({homeDishes:append?[...this.data.homeDishes,...rows]:rows,homePage:result.page||page,homeHasMore:result.hasMore,homeTotal:result.total||0})}catch(e){wx.showToast({title:e.message||'加载失败',icon:'none'})}finally{this.setData({homeLoadingMore:false})}},
  toggleHomeCategoryMenu(){this.setData({showHomeCategoryMenu:!this.data.showHomeCategoryMenu,showHomeStatusMenu:false})},
  toggleHomeStatusMenu(){this.setData({showHomeStatusMenu:!this.data.showHomeStatusMenu,showHomeCategoryMenu:false})},
  closeHomeFilterMenus(){this.setData({showHomeCategoryMenu:false,showHomeStatusMenu:false})},
  selectHomeCategory({currentTarget}){const index=Number(currentTarget.dataset.index);const changed=index!==this.data.homeCategoryIndex;this.setData({homeCategoryIndex:index,showHomeCategoryMenu:false,showHomeStatusMenu:false,showHomeRecommendation:false,recommendedHomeDish:null},()=>{if(changed)this.loadHomeDishes()})},
  selectHomeStatus({currentTarget}){const index=Number(currentTarget.dataset.index);const changed=index!==this.data.homeStatusIndex;this.setData({homeStatusIndex:index,showHomeCategoryMenu:false,showHomeStatusMenu:false,showHomeRecommendation:false,recommendedHomeDish:null},()=>{if(changed)this.loadHomeDishes()})},
  createHomeDish(){wx.navigateTo({url:'/pages/home-dishes/form'})},
  openHomeDish({currentTarget}){wx.navigateTo({url:`/pages/home-dishes/detail?id=${currentTarget.dataset.id}`})},
  async recommendHomeDish(){if(this.data.homePicking)return;this.setData({showHomeRecommendation:true,homePicking:true,recommendedHomeDish:null});try{const dish=await HomeDishService.recommend({category:this.data.homeCategoryIndex?homeCategories[this.data.homeCategoryIndex]:'',status:homeStatuses[this.data.homeStatusIndex].value});this.setData({recommendedHomeDish:toHomeDishView(dish)})}catch(e){this.setData({showHomeRecommendation:false});if(!this.data.homeTotal&&!this.data.homeDishes.length){wx.showModal({title:'还没有家常菜',content:'先记录几道自己会做、喜欢吃的菜，之后就可以帮你随机推荐。',confirmText:'新增一道',cancelText:'稍后再说',success:({confirm})=>{if(confirm)this.createHomeDish()}})}else{wx.showToast({title:'当前筛选下没有合适的菜，换个状态或分类试试',icon:'none'})}}finally{this.setData({homePicking:false})}},
  closeHomeRecommendation(){if(this.data.homePicking)return;this.setData({showHomeRecommendation:false})},
  viewHomeRecommendation(){const dish=this.data.recommendedHomeDish;if(!dish)return;this.setData({showHomeRecommendation:false});wx.navigateTo({url:`/pages/home-dishes/detail?id=${dish.id}`})},
  async loadSnacks({append=false}={}){if(append&&(this.data.snackLoadingMore||!this.data.snackHasMore))return;const page=append?this.data.snackPage+1:1;if(append)this.setData({snackLoadingMore:true});try{const result=await SnackService.list({category:this.data.snackCategoryIndex?snackCategories[this.data.snackCategoryIndex]:'',status:snackStatuses[this.data.snackStatusIndex].value,page,pageSize:HOME_DISH_PAGE_SIZE});const rows=result.items.map(toSnackView);this.setData({snacks:append?[...this.data.snacks,...rows]:rows,snackPage:result.page||page,snackHasMore:result.hasMore,snackTotal:result.total||0});this.refreshSnackCoverImages(rows)}catch(e){wx.showToast({title:e.message||'加载失败',icon:'none'})}finally{this.setData({snackLoadingMore:false})}},
  async refreshSnackCoverImages(items) {
    const resolved = await Promise.all(items.map(async (item) => ({
      id: String(item.id),
      sourceURL: item.coverSourceUrl,
      localURL: await cacheImage(item.coverSourceUrl),
    })))
    const covers = new Map(resolved.filter((item) => item.localURL).map((item) => [item.id, item]))
    if (!covers.size) return
    this.setData({
      snacks: this.data.snacks.map((item) => {
        const cover = covers.get(String(item.id))
        if (!cover || item.coverSourceUrl !== cover.sourceURL || item.coverUrl === cover.localURL) return item
        return { ...item, coverUrl: cover.localURL }
      }),
    })
  },
  toggleSnackCategoryMenu(){this.setData({showSnackCategoryMenu:!this.data.showSnackCategoryMenu,showSnackStatusMenu:false})},
  toggleSnackStatusMenu(){this.setData({showSnackStatusMenu:!this.data.showSnackStatusMenu,showSnackCategoryMenu:false})},
  closeSnackFilterMenus(){this.setData({showSnackCategoryMenu:false,showSnackStatusMenu:false})},
  selectSnackCategory({currentTarget}){const index=Number(currentTarget.dataset.index);const changed=index!==this.data.snackCategoryIndex;this.setData({snackCategoryIndex:index,showSnackCategoryMenu:false,showSnackStatusMenu:false,recommendedSnack:null},()=>{if(changed)this.loadSnacks()})},
  selectSnackStatus({currentTarget}){const index=Number(currentTarget.dataset.index);const changed=index!==this.data.snackStatusIndex;this.setData({snackStatusIndex:index,showSnackCategoryMenu:false,showSnackStatusMenu:false,recommendedSnack:null},()=>{if(changed)this.loadSnacks()})},
  createSnack(){wx.navigateTo({url:'/pages/snacks/form'})},
  openSnack({currentTarget}){wx.navigateTo({url:`/pages/snacks/detail?id=${currentTarget.dataset.id}`})},
  async recommendSnack(){if(this.data.snackPicking)return;this.setData({showSnackRecommendation:true,snackPicking:true,recommendedSnack:null});try{const snack=await SnackService.recommend({category:this.data.snackCategoryIndex?snackCategories[this.data.snackCategoryIndex]:'',status:snackStatuses[this.data.snackStatusIndex].value});this.setData({recommendedSnack:toSnackView(snack)})}catch(e){this.setData({showSnackRecommendation:false});if(!this.data.snackTotal&&!this.data.snacks.length){wx.showModal({title:'零食柜还是空的',content:'先收藏几种想吃或喜欢的零食，再来帮你挑选。',confirmText:'新增零食',cancelText:'稍后再说',success:({confirm})=>{if(confirm)this.createSnack()}})}else{wx.showToast({title:'当前筛选下没有适合推荐的零食',icon:'none'})}}finally{this.setData({snackPicking:false})}},
  closeSnackRecommendation(){if(this.data.snackPicking)return;this.setData({showSnackRecommendation:false})},
  viewSnackRecommendation(){const snack=this.data.recommendedSnack;if(!snack)return;this.setData({showSnackRecommendation:false});wx.navigateTo({url:`/pages/snacks/detail?id=${snack.id}`})},

  async loadRestaurants({ append = false, preserveLoadedPages = false } = {}) {
    await this.applyFilter(this.data.categoryIndex, { append, preserveLoadedPages })
  },

  async applyFilter(categoryIndex, { append = false, preserveLoadedPages = false, statusIndex = this.data.statusIndex } = {}) {
    if (append && (this.data.loadingMore || !this.data.hasMore)) return
    const selected = categories[categoryIndex]
    const selectedStatus = visitStatuses[statusIndex]
    const page = append ? this.data.page + 1 : 1
    const loadedPageCount = preserveLoadedPages ? Math.max(1, this.data.page) : 1
    const pages = preserveLoadedPages
      ? Array.from({ length: loadedPageCount }, (_, index) => index + 1)
      : [page]
    if (append) this.setData({ loadingMore: true })
    let results
    try {
      results = await Promise.all(pages.map((pageNumber) => RestaurantService.listPage({
        category: selected.value,
        status: selectedStatus.value,
        page: pageNumber,
        pageSize: RESTAURANT_PAGE_SIZE,
      })))
    } finally {
      if (append) this.setData({ loadingMore: false })
    }

    const result = results[results.length - 1]
    const summary = results[0]
    const pageRestaurants = results.flatMap((pageResult) => pageResult.items.map(toRestaurantView))

    this.setData({
      categoryIndex,
      selectedCategoryLabel: selected.label,
      statusIndex,
      selectedStatusLabel: selectedStatus.label,
      restaurants: append ? [...this.data.restaurants, ...pageRestaurants] : pageRestaurants,
      page: preserveLoadedPages ? loadedPageCount : result.page,
      hasMore: result.hasMore,
      totalCount: summary.overallTotal,
      totalVisitCount: summary.totalVisitCount,
      totalSpentText: formatMoney(summary.totalSpent),
    })
    this.lastRestaurantLoadAt = Date.now()
    this.refreshCoverImages(pageRestaurants)
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
      restaurants: this.data.restaurants.map((item) => {
        const cover = covers.get(String(item.id))
        if (!cover || item.coverSourceUrl !== cover.sourceURL || item.coverUrl === cover.localURL) return item
        return { ...item, coverUrl: cover.localURL }
      }),
    })
  },

  async createRestaurant() {
    if (!await requireWriteAccess(MODULES.RESTAURANT)) return
    this.openRestaurantPage('/pages/restaurants/form')
  },

  recommendRestaurant() {
    if (!this.data.restaurants.length) {
      wx.showToast({ title: '当前分类暂无餐厅', icon: 'none' })
      return
    }

    this.setData({
      showRecommendation: true,
      recommendedRestaurant: null,
    })
    this.startPicking()
  },

  startPicking() {
    if (this.data.isPicking || !this.data.restaurants.length) return

    this.clearRecommendationTimer()
    const category = categories[this.data.categoryIndex].value
    const status = visitStatuses[this.data.statusIndex].value
    const recommendation = RestaurantService.recommend({ category, status })
      .catch(() => pickRestaurant(this.data.restaurants))
    let round = 0
    this.setData({
      isPicking: true,
      recommendedRestaurant: null,
      rollingRestaurant: this.data.restaurants[0],
    })

    this.recommendationTimer = setInterval(() => {
      round += 1
      const rollingRestaurant = pickRestaurant(this.data.restaurants)
      this.setData({ rollingRestaurant })

      if (round < 13) return
      this.clearRecommendationTimer()
      recommendation.then((recommendedRestaurant) => {
        const viewRestaurant = recommendedRestaurant ? toRestaurantView(recommendedRestaurant) : null
        this.setData({
          isPicking: false,
          rollingRestaurant: viewRestaurant,
          recommendedRestaurant: viewRestaurant,
        })
      })
    }, 90)
  },

  clearRecommendationTimer() {
    if (!this.recommendationTimer) return
    clearInterval(this.recommendationTimer)
    this.recommendationTimer = null
  },

  closeRecommendation() {
    this.clearRecommendationTimer()
    this.setData({
      showRecommendation: false,
      isPicking: false,
    })
  },

  viewRecommendation() {
    const restaurant = this.data.recommendedRestaurant
    if (!restaurant) return
    this.closeRecommendation()
    this.openRestaurantPage(`/pages/restaurants/detail?id=${restaurant.id}`)
  },

  noop() {
  },
}))
