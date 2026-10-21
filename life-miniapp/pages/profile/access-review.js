const { ProfileService } = require('../../services/profile-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { requireLogin } = require('../../utils/access')
const { formatDateTime } = require('../../utils/date')

const filters = [
  { value: 'pending', label: '待审批' },
  { value: 'approved', label: '已通过' },
  { value: 'rejected', label: '已拒绝' },
]

const statusText = {
  pending: '待审批',
  approved: '已通过',
  rejected: '已拒绝',
}

Page(withScrollRestoration({
  data: {
    filters,
    activeStatus: 'pending',
    loading: true,
    reviewingId: '',
    reviewingStatus: '',
    items: [],
    total: 0,
  },

  async onShow() {
    const user = await requireLogin()
    if (!user) return
    if (user.role !== 'admin') {
      wx.showToast({ title: '仅管理员可以访问', icon: 'none' })
      wx.navigateBack()
      return
    }
    await this.loadRequests()
  },

  async loadRequests() {
    this.setData({ loading: true })
    try {
      const result = await ProfileService.listAccessRequests(this.data.activeStatus, 1, 100)
      this.setData({
        items: result.items.map((item) => ({
          ...item,
          statusText: statusText[item.accessStatus] || item.accessStatus,
          appliedAtText: formatDateTime(item.createdAt),
        })),
        total: result.total,
      })
    } catch (error) {
      wx.showToast({ title: error.message || '加载失败', icon: 'none' })
    } finally {
      this.setData({ loading: false })
    }
  },

  selectStatus({ currentTarget }) {
    const status = currentTarget.dataset.status
    if (status === this.data.activeStatus) return
    this.setData({ activeStatus: status, items: [] }, () => this.loadRequests())
  },

  review({ currentTarget }) {
    const { id, status, nickname } = currentTarget.dataset
    const approving = status === 'approved'
    wx.showModal({
      title: approving ? '通过申请？' : '拒绝申请？',
      content: approving
        ? `${nickname} 通过后可以查看和使用全部内容。`
        : `${nickname} 将无法访问业务内容。`,
      confirmText: approving ? '通过' : '拒绝',
      confirmColor: approving ? '#5f8875' : '#b8665d',
      success: async ({ confirm }) => {
        if (!confirm || this.data.reviewingId) return
        this.setData({ reviewingId: id, reviewingStatus: status })
        try {
          await ProfileService.reviewAccessRequest(id, status)
          wx.showToast({ title: approving ? '已通过' : '已拒绝', icon: 'none' })
          await this.loadRequests()
        } catch (error) {
          wx.showToast({ title: error.message || '操作失败', icon: 'none' })
        } finally {
          this.setData({ reviewingId: '', reviewingStatus: '' })
        }
      },
    })
  },

  openPermissionEditor({ currentTarget }) {
    const { userId, approve } = currentTarget.dataset
    const id = String(userId || '').trim()
    if (!/^\d+$/.test(id) || id === '0') {
      wx.showToast({ title: '用户信息异常，请刷新重试', icon: 'none' })
      return
    }
    wx.navigateTo({
      url: `/pages/profile/user-permissions?userId=${encodeURIComponent(id)}&approve=${approve ? '1' : '0'}`,
    })
  },
}))
