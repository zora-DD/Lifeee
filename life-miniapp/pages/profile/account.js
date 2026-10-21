const { ProfileService } = require('../../services/profile-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { requireLogin } = require('../../utils/access')

const statusText = {
  approved: '已通过',
  pending: '待审核',
  rejected: '未通过',
  disabled: '已停用',
}

Page(withScrollRestoration({
  data: {
    user: null,
    pendingAccessCount: 0,
  },

  async onShow() {
    const user = await requireLogin()
    if (!user) return
    this.setData({
      user: {
        ...user,
        accessStatusText: statusText[user.accessStatus] || '未知',
      },
    })
    if (user.role === 'admin') {
      try {
        const result = await ProfileService.listAccessRequests('pending', 1, 1)
        this.setData({ pendingAccessCount: result.total })
      } catch (error) {
        this.setData({ pendingAccessCount: 0 })
      }
    }
  },

  openAccessReview() {
    wx.navigateTo({ url: '/pages/profile/access-review' })
  },

  logout() {
    wx.showModal({
      title: '退出登录？',
      content: '仅退出当前设备，不会注销账号，也不会删除任何数据。',
      confirmText: '退出登录',
      confirmColor: '#b8665d',
      success: async ({ confirm }) => {
        if (!confirm) return
        await ProfileService.logout()
        wx.switchTab({
          url: '/pages/index/index',
          success: () => wx.showToast({ title: '已退出登录', icon: 'none' }),
        })
      },
    })
  },
}))
