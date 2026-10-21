const { ProfileService } = require('../../services/profile-service')
const { requireLogin } = require('../../utils/access')

const levelOptions = [
  { value: 'none', label: '无权限', note: '页面隐藏，接口拒绝访问' },
  { value: 'read', label: '只读', note: '可以查看，不能新增或修改' },
  { value: 'write', label: '可参与', note: '可新增，只能修改自己的内容' },
]

const moduleDefinitions = [
  { module: 'task', icon: '✓', name: '任务', description: '任务列表、详情和 AI 梳理' },
  { module: 'activity_log', icon: '记', name: '生活记录', description: '生活流水和月度回顾' },
  { module: 'bill', icon: '¥', name: '账单', description: '记账、统计和消费分析' },
  { module: 'restaurant', icon: '吃', name: '吃喝', description: '店铺、消费记录和随机推荐' },
  { module: 'place', icon: '⌖', name: '地点', description: '地点清单、详情和出行参考' },
]

const templates = {
  food_place: { task: 'none', activity_log: 'none', bill: 'none', restaurant: 'write', place: 'write' },
  readonly: { task: 'read', activity_log: 'read', bill: 'read', restaurant: 'read', place: 'read' },
  full: { task: 'write', activity_log: 'write', bill: 'write', restaurant: 'write', place: 'write' },
  paused: { task: 'none', activity_log: 'none', bill: 'none', restaurant: 'none', place: 'none' },
}

function buildModules(items = []) {
  const values = {}
  items.forEach((item) => { values[item.module] = item.level })
  return moduleDefinitions.map((item) => ({ ...item, level: values[item.module] || 'none' }))
}

Page({
  data: {
    userId: '',
    approveAfterSave: false,
    loading: true,
    saving: false,
    user: null,
    levelOptions,
    modules: buildModules(),
  },

  onLoad(options = {}) {
    const userId = String(options.userId || options.user_id || options.id || '').trim()
    this.setData({ userId, approveAfterSave: options.approve === '1' })
  },

  async onShow() {
    const currentUser = await requireLogin()
    if (!currentUser) return
    if (currentUser.role !== 'admin') {
      wx.showToast({ title: '仅管理员可以访问', icon: 'none' })
      wx.navigateBack()
      return
    }
    await this.loadDetail()
  },

  async loadDetail() {
    if (!/^\d+$/.test(this.data.userId) || this.data.userId === '0') {
      this.setData({ loading: false })
      wx.showToast({ title: '缺少有效用户信息', icon: 'none' })
      return
    }
    this.setData({ loading: true })
    try {
      const result = await ProfileService.getUserPermissions(this.data.userId)
      this.setData({ user: result.user, modules: buildModules(result.permissions) })
    } catch (error) {
      wx.showToast({ title: error.message || '权限加载失败', icon: 'none' })
    } finally {
      this.setData({ loading: false })
    }
  },

  selectLevel({ currentTarget }) {
    const { index, level } = currentTarget.dataset
    this.setData({ [`modules[${index}].level`]: level })
  },

  applyTemplate({ currentTarget }) {
    const template = templates[currentTarget.dataset.template]
    if (!template) return
    this.setData({
      modules: this.data.modules.map((item) => ({ ...item, level: template[item.module] })),
    })
  },

  async save() {
    if (this.data.saving) return
    this.setData({ saving: true })
    try {
      const permissions = this.data.modules.map(({ module, level }) => ({ module, level }))
      await ProfileService.modifyUserPermissions(this.data.userId, permissions)
      if (this.data.approveAfterSave && this.data.user?.accessStatus !== 'approved') {
        await ProfileService.reviewAccessRequest(this.data.userId, 'approved')
      }
      wx.showToast({ title: this.data.approveAfterSave ? '已配置并通过' : '权限已保存' })
      setTimeout(() => wx.navigateBack(), 450)
    } catch (error) {
      wx.showToast({ title: error.message || '保存失败', icon: 'none' })
      this.setData({ saving: false })
    }
  },
})
