const { BillService } = require('../../services/bill-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { requireApprovedAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')
const { formatMoney, getBillPeriod } = require('../../utils/bill-analysis')
const { getBillCategoryIcon } = require('../../utils/bill-category')

const PAGE_SIZE = 20
const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
function pad(value) { return String(value).padStart(2, '0') }

function formatTime(value) {
  const date = new Date(value)
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function getDateKey(value) {
  const date = new Date(value)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function getDateLabel(value) {
  const date = new Date(value)
  return `${date.getMonth() + 1}月${date.getDate()}日 · ${WEEKDAYS[date.getDay()]}`
}

function appendToGroups(groups, bills) {
  const nextGroups = groups.map((group) => ({ ...group, items: [...group.items] }))
  const groupMap = Object.fromEntries(nextGroups.map((group) => [group.key, group]))
  bills.forEach((bill) => {
    const key = getDateKey(bill.occurredAt)
    if (!groupMap[key]) {
      groupMap[key] = { key, label: getDateLabel(bill.occurredAt), items: [] }
      nextGroups.push(groupMap[key])
    }
    groupMap[key].items.push({
      ...bill,
      timeText: formatTime(bill.occurredAt),
      amountText: formatMoney(bill.amount),
      creatorName: bill.creator?.nickname || '未命名用户',
    })
  })
  return nextGroups
}

Page(withScrollRestoration({
  data: {
    year: new Date().getFullYear(),
    month: 0,
    periodLabel: '',
    category: '',
    categoryIcon: '记',
    creatorName: '全部用户',
    totalAmount: '0.00',
    totalCount: 0,
    groups: [],
    page: 1,
    hasMore: false,
    loading: true,
    loadingMore: false,
    errorText: '',
  },

  async onLoad(options) {
    const year = Number(options.year) || new Date().getFullYear()
    const month = Number(options.month) || 0
    const period = getBillPeriod(year, month)
    const category = decodeURIComponent(options.category || '')
    this.createdBy = decodeURIComponent(options.createdBy || '')
    this.setData({
      year,
      month: period.month,
      periodLabel: period.label,
      category,
      categoryIcon: getBillCategoryIcon(category),
      creatorName: decodeURIComponent(options.creatorName || '全部用户'),
      totalAmount: decodeURIComponent(options.amount || '0.00'),
      totalCount: Number(options.count) || 0,
    })
    wx.setNavigationBarTitle({ title: `${category || '消费'}明细` })
    const user = await requireApprovedAccess('', MODULES.BILL)
    if (!user) return
    await this.loadBills(true)
  },

  async onPullDownRefresh() {
    try {
      await this.loadBills(true)
    } finally {
      wx.stopPullDownRefresh()
    }
  },

  onReachBottom() {
    if (this.data.hasMore && !this.data.loadingMore) this.loadBills(false)
  },

  async loadBills(reset) {
    if (reset) {
      this.setData({ loading: true, errorText: '', page: 1, groups: [] })
    } else {
      this.setData({ loadingMore: true })
    }
    const page = reset ? 1 : this.data.page + 1
    const range = getBillPeriod(this.data.year, this.data.month)
    try {
      const result = await BillService.list({
        type: 'expense',
        category: this.data.category,
        createdBy: this.createdBy || undefined,
        from: range.from,
        to: range.to,
        page,
        pageSize: PAGE_SIZE,
      })
      this.setData({
        groups: appendToGroups(reset ? [] : this.data.groups, result.items),
        page,
        hasMore: result.hasMore,
        loading: false,
        loadingMore: false,
      })
    } catch (error) {
      this.setData({
        loading: false,
        loadingMore: false,
        errorText: error.message || '消费明细加载失败',
      })
    }
  },

  retry() {
    this.loadBills(true)
  },
}))
