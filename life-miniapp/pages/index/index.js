const { ProfileService } = require('../../services/profile-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { ActivityLogService } = require('../../services/activity-log-service')
const { BillService } = require('../../services/bill-service')
const { resumePendingRoute } = require('../../utils/pending-navigation')
const { buildShareCard } = require('../../utils/share')
const { MODULES, emptyPermissions, hasPermission, loadPermissions, syncTabBar } = require('../../utils/permissions')

function formatDuration(minutes) {
  if (!minutes) return '0分钟'
  if (minutes < 60) return `${minutes}分钟`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest ? `${hours}小时${rest}分` : `${hours}小时`
}

const accessCopy = {
  pending: {
    statusText: '待审核',
    title: '申请已提交',
    text: '管理员通过后，即可查看任务、记录、吃喝和地点内容。',
  },
  rejected: {
    statusText: '未通过',
    title: '申请暂未通过',
    text: '如需继续使用，请联系管理员重新确认。',
  },
  disabled: {
    statusText: '已停用',
    title: '账号已停用',
    text: '当前账号暂时无法访问，请联系管理员。',
  },
}

Page(withScrollRestoration({
  data: {
    loading: false,
    user: null,
    review: null,
    accessInfo: null,
    billSummary: null,
    canViewContentLibrary: false,
    showAccessNoteModal: false,
    accessNote: '',
    submittingAccessNote: false,
  },

  onShareAppMessage() {
    return buildShareCard()
  },

  onShow() {
    this.loadHome()
  },

  async loadHome() {
    try {
      const user = await ProfileService.getCurrent()
      if (!user) {
        syncTabBar(this, 4, emptyPermissions())
        this.setData({ user: null, review: null, accessInfo: null, billSummary: null, canViewContentLibrary: false })
        return
      }
      if (user.accessStatus !== 'approved') {
        syncTabBar(this, 4, emptyPermissions())
        const accessInfo = accessCopy[user.accessStatus] || accessCopy.pending
        this.setData({
          user: { ...user, accessStatusText: accessInfo.statusText },
          review: null,
          accessInfo,
          billSummary: null,
          canViewContentLibrary: false,
          accessNote: user.applicationNote || '',
          showAccessNoteModal: user.accessStatus === 'pending' && !user.applicationNote,
        })
        return
      }
      const levels = await loadPermissions(user)
      syncTabBar(this, 4, levels)
      this.setData({
        user: { ...user, accessStatusText: '已通过' },
        accessInfo: null,
        review: hasPermission(levels, MODULES.ACTIVITY) ? this.data.review : null,
        billSummary: hasPermission(levels, MODULES.BILL) ? this.data.billSummary : null,
        canViewContentLibrary: [MODULES.TASK, MODULES.RESTAURANT, MODULES.PLACE]
          .some((module) => hasPermission(levels, module)),
      })
      // 从分享卡片进入时，登录或审批通过后优先回到原详情页。
      if (resumePendingRoute()) return

      const loaders = []
      if (hasPermission(levels, MODULES.ACTIVITY)) loaders.push(this.loadReview())
      if (hasPermission(levels, MODULES.BILL)) loaders.push(this.loadBillSummary())
      await Promise.allSettled(loaders)
    } catch (error) {
      this.setData({ user: null, review: null, accessInfo: null, billSummary: null, canViewContentLibrary: false })
    }
  },

  async loadReview() {
    const now = new Date()
    const stats = await ActivityLogService.stats({
      scope: 'month',
      year: now.getFullYear(),
      month: now.getMonth() + 1,
    })
    this.setData({
      review: {
        monthLabel: `${now.getMonth() + 1}月`,
        recordCount: stats.recordCount,
        activeDays: stats.activeDays,
        durationText: formatDuration(stats.totalMinutes),
        summary: stats.recordCount
          ? `这个月已记录 ${stats.activeDays} 天，继续收集生活里的小事。`
          : '这个月还没有记录，从今天的一件小事开始吧。',
      },
    })
  },

  async loadBillSummary() {
    const now = new Date()
    const stats = await BillService.stats({ year: now.getFullYear(), month: now.getMonth() + 1 })
    const expenseTotal = Number(stats.expenseTotal) || 0
    this.setData({
      billSummary: {
        monthLabel: `${now.getMonth() + 1}月`,
        expenseTotal: expenseTotal.toFixed(2),
        billCount: Number(stats.billCount) || 0,
      },
    })
  },

  async login() {
    if (this.data.loading) return

    this.setData({ loading: true })
    try {
      const code = await this.getLoginCode()
      const user = await ProfileService.login(code)
      this.setData({ loading: false })
      await this.loadHome()
      wx.showToast({ title: user.accessStatus === 'approved' ? '登录成功' : '请填写身份备注', icon: 'none' })
    } catch (error) {
      this.setData({ loading: false })
      wx.showToast({ title: error.message || '登录失败', icon: 'none' })
    }
  },

  exitMiniProgram() {
    if (this.data.loading) return
    wx.exitMiniProgram({
      fail: () => wx.showToast({ title: '可点击左上角返回微信', icon: 'none' }),
    })
  },

  async refreshAccessStatus() {
    if (this.data.loading) return
    this.setData({ loading: true })
    try {
      const user = await ProfileService.getAccessStatus()
      await this.loadHome()
      wx.showToast({
        title: user.accessStatus === 'approved' ? '审核已通过' : '状态已刷新',
        icon: 'none',
      })
    } catch (error) {
      wx.showToast({ title: error.message || '刷新失败', icon: 'none' })
    } finally {
      this.setData({ loading: false })
    }
  },

  openAccessNoteModal() {
    this.setData({
      showAccessNoteModal: true,
      accessNote: this.data.user?.applicationNote || '',
    })
  },

  closeAccessNoteModal() {
    if (this.data.submittingAccessNote) return
    this.setData({ showAccessNoteModal: false })
  },

  onAccessNoteInput({ detail }) {
    this.setData({ accessNote: detail.value })
  },

  async submitAccessNote() {
    if (this.data.submittingAccessNote) return
    const note = this.data.accessNote.trim()
    if (!note) {
      wx.showToast({ title: '请填写你是谁或与管理员的关系', icon: 'none' })
      return
    }
    this.setData({ submittingAccessNote: true })
    try {
      const user = await ProfileService.applyAccess(note)
      this.setData({
        user: { ...user, accessStatusText: (accessCopy[user.accessStatus] || accessCopy.pending).statusText },
        accessInfo: accessCopy[user.accessStatus] || accessCopy.pending,
        showAccessNoteModal: false,
      })
      wx.showToast({ title: '申请备注已提交', icon: 'none' })
    } catch (error) {
      wx.showToast({ title: error.message || '提交失败', icon: 'none' })
    } finally {
      this.setData({ submittingAccessNote: false })
    }
  },

  getLoginCode() {
    return new Promise((resolve, reject) => {
      wx.login({
        success: ({ code }) => code ? resolve(code) : reject(new Error('未获取到登录凭证')),
        fail: () => reject(new Error('微信登录失败')),
      })
    })
  },

  noop() {},

  openProfilePage({ currentTarget }) {
    wx.navigateTo({ url: currentTarget.dataset.url })
  },

  openAccount() {
    wx.navigateTo({ url: '/pages/profile/account' })
  },

  openReports() {
    wx.navigateTo({ url: '/pages/profile/reports' })
  },

  openBills() {
    wx.navigateTo({ url: '/pages/profile/bill-analysis' })
  },

}))
