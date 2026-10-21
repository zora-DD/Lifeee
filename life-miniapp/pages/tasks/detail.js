const { TaskService } = require('../../services/task-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { FavoriteService } = require('../../services/favorite-service')
const { formatFullDateTime } = require('../../utils/date')
const { requireApprovedAccess, requireWriteAccess } = require('../../utils/access')
const { buildShareCard } = require('../../utils/share')
const { MODULES, hasPermission, loadPermissions } = require('../../utils/permissions')

const priorityText = {
  high: '高',
  medium: '中',
  low: '低',
}

Page(withScrollRestoration({
  data: {
    id: '',
    task: null,
    favorited: false,
    favoriteSaving: false,
    canWrite: false,
    canEdit: false,
  },

  onLoad({ id }) {
    this.setData({ id })
  },

  async onShow() {
    this.setData({ task: null })
    const user = await requireApprovedAccess(`/pages/tasks/detail?id=${encodeURIComponent(this.data.id)}`, MODULES.TASK)
    if (!user) return
    this.currentUser = user
    this.canWrite = hasPermission(await loadPermissions(user), MODULES.TASK, 'write')
    await this.loadTask(user)
  },

  onShareAppMessage() {
    const task = this.data.task
    return buildShareCard({
      title: task ? `任务｜${task.title}` : '',
      path: `/pages/tasks/detail?id=${encodeURIComponent(this.data.id)}`,
    })
  },

  async loadTask(currentUser) {
    const task = await TaskService.get(this.data.id)
    if (!task) {
      wx.showToast({ title: '任务不存在', icon: 'none' })
      return
    }
    const overdue = task.status === 'todo' && new Date(task.dueAt).getTime() < Date.now()
    this.setData({
      task: {
        ...task,
        overdue,
        stateText: task.status === 'done' ? '已完成' : overdue ? '已逾期' : '待完成',
        creatorInitial: task.creator.avatarEmoji || task.creator.name.slice(0, 1),
        priorityText: priorityText[task.priority],
        dueText: formatFullDateTime(task.dueAt),
        createdText: formatFullDateTime(task.createdAt),
        updatedText: formatFullDateTime(task.updatedAt),
      },
      canWrite: this.canWrite,
      canEdit: this.canWrite && (currentUser?.role === 'admin' || String(task.creator?.id || '') === String(currentUser?.id || '')),
    })
    this.loadFavoriteStatus()
  },

  async loadFavoriteStatus() {
    try {
      const favorited = await FavoriteService.status('task', this.data.id)
      this.setData({ favorited })
    } catch (error) {
      this.setData({ favorited: false })
    }
  },

  async toggleFavorite() {
    if (this.data.favoriteSaving || !await requireWriteAccess(MODULES.TASK)) return
    const next = !this.data.favorited
    this.setData({ favoriteSaving: true })
    try {
      const favorited = await FavoriteService.modify('task', this.data.id, next)
      this.setData({ favorited })
      wx.showToast({ title: favorited ? '已收藏' : '已取消收藏', icon: 'none' })
    } catch (error) {
      wx.showToast({ title: error.message || '操作失败', icon: 'none' })
    } finally {
      this.setData({ favoriteSaving: false })
    }
  },

  async toggleStatus() {
    if (!this.data.canEdit || !await requireWriteAccess(MODULES.TASK)) return
    await TaskService.toggle(this.data.id, this.data.task.status)
    await this.loadTask(this.currentUser)
    this.emitToTaskList('taskChanged', { task: this.data.task })
  },

  async editTask() {
    if (!this.data.canEdit || !await requireWriteAccess(MODULES.TASK)) return
    wx.navigateTo({
      url: `/pages/tasks/form?id=${this.data.id}`,
      events: {
        taskChanged: ({ task } = {}) => {
          this.emitToTaskList('taskChanged', { task })
        },
      },
    })
  },

  async deleteTask() {
    if (!this.data.canEdit || !await requireWriteAccess(MODULES.TASK)) return
    wx.showModal({
      title: '删除任务',
      content: '删除后无法恢复，确定继续吗？',
      confirmColor: '#d55656',
      success: async ({ confirm }) => {
        if (!confirm) return
        await TaskService.remove(this.data.id)
        this.emitToTaskList('taskDeleted', { id: this.data.id })
        wx.showToast({ title: '已删除' })
        setTimeout(() => wx.navigateBack(), 500)
      },
    })
  },

  emitToTaskList(eventName, payload) {
    if (typeof this.getOpenerEventChannel !== 'function') return
    const eventChannel = this.getOpenerEventChannel()
    if (eventChannel && typeof eventChannel.emit === 'function') {
      eventChannel.emit(eventName, payload)
    }
  },
}))
