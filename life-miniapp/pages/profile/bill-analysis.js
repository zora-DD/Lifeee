const { BillService } = require('../../services/bill-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { requireApprovedAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')
const { buildBillInsights, formatMoney, getBillPeriod } = require('../../utils/bill-analysis')
const { buildExpenseDonutItems, buildDonutGradient } = require('../../utils/bill-chart')
const { getBillCategoryIcon } = require('../../utils/bill-category')

const MONTH_OPTIONS = [
  { label: '全年', value: 0 },
  ...Array.from({ length: 12 }, (_, index) => ({ label: `${index + 1}月`, value: index + 1 })),
]
const CURRENT_MONTH = new Date().getMonth() + 1

function initialYear() {
  return String(new Date().getFullYear())
}

const YEAR_OPTIONS = Array.from({ length: 7 }, (_, index) => {
  const year = new Date().getFullYear() - index
  return { label: `${year}年`, value: String(year) }
})

Page(withScrollRestoration({
  data: {
    loading: true,
    errorText: '',
    yearOptions: YEAR_OPTIONS,
    selectedYear: initialYear(),
    selectedYearIndex: 0,
    monthOptions: MONTH_OPTIONS,
    selectedMonth: CURRENT_MONTH,
    selectedMonthIndex: CURRENT_MONTH,
    selectedMonthLabel: `${CURRENT_MONTH}月`,
    scope: 'month',
    periodLabel: `${initialYear()}年${CURRENT_MONTH}月`,
    selectedUserId: '',
    selectedUserIndex: 0,
    selectedUserLabel: '全部用户',
    activeDropdown: '',
    userOptions: [{ id: '', label: '全部用户' }],
    stats: null,
    monthly: [],
    users: [],
    categories: [],
    donutItems: [],
    donutGradient: '#e8efeb',
    insights: [],
    hasData: false,
  },

  async onShow() {
    const user = await requireApprovedAccess('', MODULES.BILL)
    if (!user) return
    await this.loadAnalysis()
  },

  async onPullDownRefresh() {
    try {
      await this.loadAnalysis()
    } finally {
      wx.stopPullDownRefresh()
    }
  },

  toggleFilterDropdown({ currentTarget }) {
    const name = currentTarget.dataset.name
    this.setData({ activeDropdown: this.data.activeDropdown === name ? '' : name })
  },

  closeFilterDropdown() {
    this.setData({ activeDropdown: '' })
  },

  selectYear({ currentTarget }) {
    const selectedYearIndex = Number(currentTarget.dataset.index) || 0
    const selected = YEAR_OPTIONS[selectedYearIndex] || YEAR_OPTIONS[0]
    if (selected.value === this.data.selectedYear) {
      this.closeFilterDropdown()
      return
    }
    this.setData({
      selectedYear: selected.value,
      selectedYearIndex,
      activeDropdown: '',
    }, () => this.loadAnalysis())
  },

  selectMonth({ currentTarget }) {
    const selectedMonthIndex = Number(currentTarget.dataset.index) || 0
    const selected = MONTH_OPTIONS[selectedMonthIndex] || MONTH_OPTIONS[0]
    if (selected.value === this.data.selectedMonth) {
      this.closeFilterDropdown()
      return
    }
    this.setData({
      selectedMonth: selected.value,
      selectedMonthIndex,
      selectedMonthLabel: selected.label,
      activeDropdown: '',
    }, () => this.loadAnalysis())
  },

  selectUser({ currentTarget }) {
    const selectedUserIndex = Number(currentTarget.dataset.index) || 0
    const selected = this.data.userOptions[selectedUserIndex] || this.data.userOptions[0]
    if (selectedUserIndex === this.data.selectedUserIndex) {
      this.closeFilterDropdown()
      return
    }
    this.setData({
      selectedUserIndex,
      selectedUserId: selected.id,
      selectedUserLabel: selected.label,
      activeDropdown: '',
    }, () => this.loadAnalysis())
  },

  noop() {},

  async loadAnalysis() {
    const requestSeq = (this.analysisRequestSeq || 0) + 1
    this.analysisRequestSeq = requestSeq
    // 返回当前页时保留已有内容，避免加载态让页面高度瞬间归零。
    this.setData({ loading: !this.data.stats, errorText: '' })
    try {
      const year = Number(this.data.selectedYear)
      const period = getBillPeriod(year, this.data.selectedMonth)
      const result = await BillService.stats({
        scope: period.scope,
        year,
        month: period.month || undefined,
        createdBy: this.data.selectedUserId || undefined,
      })
      const expenseTotal = Number(result.expenseTotal) || 0
      const monthlySource = Array.isArray(result.monthly) ? result.monthly : []
      const maxMonthlyExpense = Math.max(0, ...monthlySource.map((item) => Number(item.expenseAmount) || 0))
      const monthly = monthlySource.map((item) => {
        const expense = Number(item.expenseAmount) || 0
        return {
          ...item,
          expenseText: formatMoney(expense),
          barPercent: maxMonthlyExpense ? Math.max(expense ? 5 : 0, Math.round(expense / maxMonthlyExpense * 100)) : 0,
          showAmount: expense > 0 && expense === maxMonthlyExpense,
        }
      })
      const users = (Array.isArray(result.users) ? result.users : [])
        .map((item) => {
          const user = item.user || {}
          return {
            ...item,
            id: user.id || `unknown-${item.count}`,
            user,
            expense: Number(item.expenseAmount) || 0,
            expenseText: formatMoney(item.expenseAmount),
          }
        })
        .sort((a, b) => b.expense - a.expense)
      const categories = (Array.isArray(result.categories) ? result.categories : [])
        .map((item) => {
          const amount = Number(item.amount) || 0
          return {
            ...item,
            amount,
            amountText: formatMoney(amount),
            categoryIcon: getBillCategoryIcon(item.category),
            percent: expenseTotal ? Math.max(4, Math.round(amount / expenseTotal * 100)) : 0,
          }
        })
        .sort((a, b) => b.amount - a.amount)

      const donutItems = buildExpenseDonutItems(categories, expenseTotal)
      const nextData = {
        loading: false,
        stats: {
          year,
          month: period.month,
          expenseTotal: formatMoney(expenseTotal),
          incomeTotal: formatMoney(result.incomeTotal),
          billCount: Number(result.billCount) || 0,
        },
        monthly,
        users,
        categories,
        donutItems,
        donutGradient: buildDonutGradient(donutItems),
        insights: buildBillInsights({
          expenseTotal,
          categories: result.categories,
          monthly: result.monthly,
          year,
          month: period.month,
          scope: period.scope,
        }),
        scope: period.scope,
        periodLabel: period.label,
        hasData: Number(result.billCount) > 0,
      }
      if (!this.data.selectedUserId) {
        nextData.userOptions = [
          { id: '', label: '全部用户' },
          ...users.map((item) => ({ id: item.id, label: item.user.nickname || '未命名用户' })),
        ]
      }
      // 快速切换筛选时，丢弃先发后至的旧请求，避免图表回退。
      if (requestSeq !== this.analysisRequestSeq) return
      this.setData(nextData)
    } catch (error) {
      if (requestSeq !== this.analysisRequestSeq) return
      this.setData({
        loading: false,
        stats: null,
        monthly: [],
        users: [],
        categories: [],
        donutItems: [],
        donutGradient: '#e8efeb',
        insights: [],
        hasData: false,
        errorText: error.message || '消费分析加载失败',
      })
    }
  },

  retry() {
    this.loadAnalysis()
  },

  openBillCenter() {
    wx.setStorageSync('record_center_initial_mode', 'bill')
    wx.switchTab({ url: '/pages/logs/index' })
  },

  openCategoryDetails({ currentTarget }) {
    const category = currentTarget.dataset.category
    const item = this.data.categories.find((entry) => entry.category === category)
    if (!item) return
    const query = [
      `year=${encodeURIComponent(this.data.selectedYear)}`,
      `month=${encodeURIComponent(this.data.selectedMonth)}`,
      `category=${encodeURIComponent(item.category)}`,
      `amount=${encodeURIComponent(item.amountText)}`,
      `count=${encodeURIComponent(item.count)}`,
      `createdBy=${encodeURIComponent(this.data.selectedUserId)}`,
      `creatorName=${encodeURIComponent(this.data.selectedUserLabel)}`,
    ].join('&')
    wx.navigateTo({ url: `/pages/profile/bill-details?${query}` })
  },
}))
