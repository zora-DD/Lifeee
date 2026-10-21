const { TaskService } = require('../../services/task-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { AISummaryService } = require('../../services/ai-summary-service')
const { formatDateTime, formatDateTimeWithWeekday } = require('../../utils/date')
const { buildQuickPostponeOptions } = require('../../utils/task-schedule')
const { getCurrentUser, requireWriteAccess } = require('../../utils/access')
const {
  MODULES,
  hasPermission,
  loadPermissions,
  redirectToFirstAllowedTab,
  syncTabBar,
} = require('../../utils/permissions')

const priorityText = {
  high: '高',
  medium: '中',
  low: '低',
}

const taskTagEmoji = {
  工作: '💼',
  生活: '🌿',
  兴趣: '✨',
  学习: '📚',
  健康: '🩺',
  家务: '🧹',
  出行: '🚗',
  购物: '🛒',
  其他: '📌',
}

function buildTagViews(tags = []) {
  return tags.map((label) => ({
    label,
    icon: taskTagEmoji[label] || '🏷️',
  }))
}

const DELETE_WIDTH_RPX = 136
const SWIPE_THRESHOLD = 0.38
const TASK_PAGE_SIZE = 20
const COMPLETE_FEEDBACK_DURATION = 800

function wait(duration) {
  return new Promise((resolve) => setTimeout(resolve, duration))
}

Page(withScrollRestoration({
  data: {
    authChecked: false,
    loggedIn: false,
    canWrite: false,
    currentUserId: '',
    isAdmin: false,
    keyword: '',
    statusFilter: 'all',
    priorityFilter: 'all',
    ownerFilter: 'all',
    statusFilters: [
      { value: 'all', label: '全部' },
      { value: 'today', label: '今天' },
      { value: 'todo', label: '待做' },
      { value: 'overdue', label: '逾期' },
      { value: 'done', label: '完成' },
    ],
    ownerFilters: [
      { value: 'all', label: '全部' },
      { value: 'mine', label: '我创建的' },
    ],
    priorityFilters: [
      { value: 'all', label: '全部' },
      { value: 'high', label: '高' },
      { value: 'medium', label: '中' },
      { value: 'low', label: '低' },
    ],
    tasks: [],
    loadingTasks: false,
    loadingMore: false,
    loadError: '',
    page: 1,
    hasMore: false,
    openedTaskId: '',
    swipingTaskId: '',
    revealingTaskId: '',
    completingTaskId: '',
    postponingTaskId: '',
    completedExpanded: false,
    completedCountInList: 0,
    totalCount: 0,
    todoCount: 0,
    doneCount: 0,
    completionRate: 0,
    aiSummaryVisible: false,
    aiSummaryLoading: false,
    aiSummary: null,
    aiSummaryError: '',
    aiActionTaskId: '',
  },

  async onShow() {
    const reuseCurrentList = this.skipNextShowRefresh &&
      !this.taskListDirty &&
      this.data.authChecked &&
      this.data.loggedIn
    this.skipNextShowRefresh = false

    // 详情未发生变更时保留列表和滚动位置，不重复校验和请求。
    if (reuseCurrentList) return

    const preserveLoadedPages = this.taskListDirty && this.data.page > 1
    this.taskListDirty = false
    this.setData({ authChecked: false })
    const user = await getCurrentUser()
    if (!user || user.accessStatus !== 'approved') {
      this.setData({
        authChecked: true,
        loggedIn: false,
        loadingTasks: false,
        loadError: '',
        tasks: [],
        totalCount: 0,
        todoCount: 0,
        doneCount: 0,
        completionRate: 0,
      })
      return
    }
    const levels = await loadPermissions(user)
    syncTabBar(this, 0, levels)
    if (!hasPermission(levels, MODULES.TASK)) {
      this.setData({ authChecked: true, loggedIn: false, tasks: [] })
      redirectToFirstAllowedTab(levels)
      return
    }
    this.setData({
      authChecked: true,
      loggedIn: true,
      canWrite: hasPermission(levels, MODULES.TASK, 'write'),
      currentUserId: String(user.id || ''),
      isAdmin: user.role === 'admin',
    })
    const { windowWidth } = wx.getWindowInfo()
    this.deleteWidthPx = (DELETE_WIDTH_RPX / 750) * windowWidth
    await this.loadTasks({ preserveLoadedPages })
  },

  async openDailyRoutines() {
    if (!await requireWriteAccess(MODULES.TASK)) return
    this.skipNextShowRefresh = true
    wx.navigateTo({
      url: '/pages/tasks/routines',
      fail: () => { this.skipNextShowRefresh = false },
    })
  },

  async loadTasks({ append = false, preserveLoadedPages = false } = {}) {
    if (append && (this.data.loadingMore || !this.data.hasMore)) return
    const page = append ? this.data.page + 1 : 1
    const loadedPageCount = preserveLoadedPages ? Math.max(1, this.data.page) : 1
    this.setData(append ? { loadingMore: true } : { loadingTasks: true, loadError: '' })
    try {
      const pages = preserveLoadedPages
        ? Array.from({ length: loadedPageCount }, (_, index) => index + 1)
        : [page]
      const results = await Promise.all(pages.map((pageNumber) => TaskService.listPage({
        keyword: this.data.keyword,
        status: this.data.statusFilter,
        priority: this.data.priorityFilter,
        createdBy: this.data.ownerFilter === 'mine' ? this.data.currentUserId : undefined,
        page: pageNumber,
        pageSize: TASK_PAGE_SIZE,
      })))
      const result = results[results.length - 1]
      const summary = results[0]
      const pageTasks = results.flatMap((pageResult) => (
        pageResult.items.map((task) => ({ ...this.toViewTask(task), swipeOffset: 0 }))
      ))
      const combinedTasks = append ? [...this.data.tasks, ...pageTasks] : pageTasks
      const tasks = this.decorateTaskList(combinedTasks)
      this.setData({
        tasks,
        openedTaskId: '',
        swipingTaskId: '',
        revealingTaskId: '',
        page: preserveLoadedPages ? loadedPageCount : result.page,
        hasMore: result.hasMore,
        totalCount: summary.overallTotal,
        todoCount: summary.todoCount,
        doneCount: summary.doneCount,
        completedCountInList: tasks.filter((task) => task.status === 'done').length,
        completionRate: summary.overallTotal ? Math.round((summary.doneCount / summary.overallTotal) * 100) : 0,
      })
    } catch (error) {
      if (append) {
        wx.showToast({ title: error.message || '加载更多失败', icon: 'none' })
      } else {
        this.setData({
          tasks: [],
          page: 1,
          hasMore: false,
          totalCount: 0,
          todoCount: 0,
          doneCount: 0,
          completionRate: 0,
          loadError: error.message || '任务加载失败',
        })
      }
    } finally {
      this.setData(append ? { loadingMore: false } : { loadingTasks: false })
    }
  },

  async onPullDownRefresh() {
    await this.loadTasks()
    wx.stopPullDownRefresh()
  },

  onReachBottom() {
    this.loadTasks({ append: true })
  },

  toViewTask(task) {
    return {
      ...task,
      canEdit: this.data.canWrite && (this.data.isAdmin || String(task.creator?.id || '') === this.data.currentUserId),
      priorityText: priorityText[task.priority],
      tagViews: buildTagViews(task.tags || []),
      dueText: formatDateTimeWithWeekday(task.dueAt),
      overdue: task.status === 'todo' && new Date(task.dueAt).getTime() < Date.now(),
    }
  },

  decorateTaskList(tasks) {
    let completedFound = false
    return tasks.map((task) => {
      const firstCompleted = task.status === 'done' && !completedFound
      if (task.status === 'done') completedFound = true
      return { ...task, firstCompleted }
    })
  },

  onKeywordInput({ detail }) {
    this.setData({ keyword: detail.value })
  },

  search() {
    this.loadTasks()
  },

  retryLoad() {
    this.loadTasks()
  },

  openCalendar() {
    this.skipNextShowRefresh = true
    wx.navigateTo({
      url: '/pages/tasks/calendar',
      events: {
        taskChanged: () => { this.taskListDirty = true },
      },
    })
  },

  selectStatus({ currentTarget }) {
    this.setData({
      statusFilter: currentTarget.dataset.value,
      completedExpanded: currentTarget.dataset.value === 'done',
    }, () => this.loadTasks())
  },

  selectPriority({ currentTarget }) {
    this.setData({ priorityFilter: currentTarget.dataset.value, completedExpanded: false }, () => this.loadTasks())
  },

  selectOwner({ currentTarget }) {
    this.setData({ ownerFilter: currentTarget.dataset.value, completedExpanded: false }, () => this.loadTasks())
  },

  toggleCompleted() {
    this.setData({ completedExpanded: !this.data.completedExpanded })
  },

  async quickPostpone({ currentTarget }) {
    const id = currentTarget.dataset.id
    const task = this.data.tasks.find((item) => item.id === id)
    if (!task?.canEdit || this.data.postponingTaskId) return
    if (!await requireWriteAccess(MODULES.TASK)) return
    const options = buildQuickPostponeOptions(new Date(), task.dueAt)
    wx.showActionSheet({
      itemList: options.map((item) => item.label),
      success: async ({ tapIndex }) => {
        const selected = options[tapIndex]
        if (!selected) return
        this.setData({ postponingTaskId: id })
        try {
          await TaskService.reschedule(id, selected.dueAt, task)
          await this.loadTasks({ preserveLoadedPages: this.data.page > 1 })
          const targetText = { today: '今天', tomorrow: '明天', next_week: '一周后' }[selected.key]
          wx.showToast({ title: `已延期到${targetText}` })
        } catch (error) {
          wx.showToast({ title: error.message || '延期失败', icon: 'none' })
        } finally {
          this.setData({ postponingTaskId: '' })
        }
      },
    })
  },

  async toggleTask({ currentTarget }) {
    const target = this.data.tasks.find((task) => task.id === currentTarget.dataset.id)
    if (!this.data.canWrite || !target?.canEdit) return
    if (!await requireWriteAccess(MODULES.TASK)) return
    if (this.data.openedTaskId) {
      this.closeOpenedTask()
      return
    }
    if (this.data.completingTaskId) return

    const { id, status } = currentTarget.dataset
    const taskIndex = this.data.tasks.findIndex((task) => task.id === id)
    if (taskIndex < 0) return

    // 完成任务时先给用户明确反馈，再刷新列表让任务沉底。
    if (status === 'todo') {
      const previousOverdue = this.data.tasks[taskIndex].overdue
      const nextDoneCount = this.data.doneCount + 1
      this.setData({
        [`tasks[${taskIndex}].status`]: 'done',
        [`tasks[${taskIndex}].overdue`]: false,
        completingTaskId: id,
        todoCount: Math.max(0, this.data.todoCount - 1),
        doneCount: nextDoneCount,
        completionRate: this.data.totalCount
          ? Math.round((nextDoneCount / this.data.totalCount) * 100)
          : 0,
      })
      try {
        await Promise.all([
          TaskService.toggle(id, status),
          wait(COMPLETE_FEEDBACK_DURATION),
        ])
        await this.loadTasks()
        this.setData({ completingTaskId: '' })
      } catch (error) {
        this.setData({
          [`tasks[${taskIndex}].status`]: status,
          [`tasks[${taskIndex}].overdue`]: previousOverdue,
          completingTaskId: '',
          todoCount: this.data.todoCount + 1,
          doneCount: Math.max(0, this.data.doneCount - 1),
          completionRate: this.data.totalCount
            ? Math.round((Math.max(0, this.data.doneCount - 1) / this.data.totalCount) * 100)
            : 0,
        })
        wx.showToast({ title: error.message || '操作失败', icon: 'none' })
      }
      return
    }

    try {
      await TaskService.toggle(id, status)
      await this.loadTasks()
    } catch (error) {
      wx.showToast({ title: error.message || '操作失败', icon: 'none' })
    }
  },

  openDetail({ currentTarget }) {
    if (Date.now() < (this.ignoreTapUntil || 0)) return
    if (this.data.completingTaskId === currentTarget.dataset.id) return
    if (this.data.openedTaskId) {
      this.closeOpenedTask()
      return
    }
    this.openTaskPage(`/pages/tasks/detail?id=${currentTarget.dataset.id}`)
  },

  openTaskPage(url) {
    this.skipNextShowRefresh = true
    wx.navigateTo({
      url,
      events: {
        taskChanged: () => {
          this.taskListDirty = true
        },
        taskDeleted: () => {
          this.taskListDirty = true
        },
      },
      fail: () => {
        this.skipNextShowRefresh = false
      },
    })
  },

  onTaskTouchStart({ currentTarget, touches }) {
    const touch = touches[0]
    if (!touch) return

    const { id, index } = currentTarget.dataset
    const task = this.data.tasks[Number(index)]
    if (!task?.canEdit) return
    if (this.data.completingTaskId === id) return
    if (this.data.openedTaskId && this.data.openedTaskId !== id) {
      this.closeOpenedTask()
    }

    this.touchState = {
      id,
      index: Number(index),
      startX: touch.clientX,
      startY: touch.clientY,
      baseOffset: this.data.openedTaskId === id ? -this.deleteWidthPx : 0,
      currentOffset: this.data.openedTaskId === id ? -this.deleteWidthPx : 0,
      horizontal: false,
      cancelled: false,
    }
    this.setData({ swipingTaskId: id })
  },

  onTaskTouchMove({ touches }) {
    const touch = touches[0]
    const state = this.touchState
    if (!touch || !state || state.cancelled) return

    const deltaX = touch.clientX - state.startX
    const deltaY = touch.clientY - state.startY
    if (!state.horizontal) {
      if (Math.abs(deltaX) < 8 && Math.abs(deltaY) < 8) return
      if (Math.abs(deltaY) >= Math.abs(deltaX)) {
        state.cancelled = true
        return
      }
      state.horizontal = true
    }

    const offset = Math.max(-this.deleteWidthPx, Math.min(0, state.baseOffset + deltaX))
    state.currentOffset = offset
    this.setData({
      [`tasks[${state.index}].swipeOffset`]: offset,
      revealingTaskId: offset < 0 ? state.id : '',
    })
  },

  onTaskTouchEnd() {
    const state = this.touchState
    this.touchState = null
    if (!state) return

    if (state.cancelled || !state.horizontal) {
      this.setData({ swipingTaskId: '', revealingTaskId: '' })
      return
    }

    this.ignoreTapUntil = Date.now() + 300
    const shouldOpen = state.currentOffset <= -this.deleteWidthPx * SWIPE_THRESHOLD
    this.setData({
      [`tasks[${state.index}].swipeOffset`]: shouldOpen ? -this.deleteWidthPx : 0,
      openedTaskId: shouldOpen ? state.id : '',
      swipingTaskId: '',
      revealingTaskId: shouldOpen ? state.id : '',
    })
  },

  closeOpenedTask() {
    const index = this.data.tasks.findIndex((task) => task.id === this.data.openedTaskId)
    const changes = { openedTaskId: '', swipingTaskId: '', revealingTaskId: '' }
    if (index >= 0) changes[`tasks[${index}].swipeOffset`] = 0
    this.setData(changes)
  },

  async deleteTask({ currentTarget }) {
    if (!await requireWriteAccess(MODULES.TASK)) return
    const id = currentTarget.dataset.id
    const task = this.data.tasks.find((item) => item.id === id)
    wx.showModal({
      title: '删除任务',
      content: `确定删除“${task ? task.title : '该任务'}”吗？删除后无法恢复。`,
      confirmColor: '#bd5656',
      success: async ({ confirm }) => {
        if (!confirm) return
        await TaskService.remove(id)
        wx.showToast({ title: '已删除' })
        this.loadTasks()
      },
    })
  },

  async createTask() {
    if (!await requireWriteAccess(MODULES.TASK)) return
    this.openTaskPage('/pages/tasks/form')
  },

  async openAISummary() {
    this.setData({ aiSummaryVisible: true })
    this.setTabBarHidden(true)
    if (this.data.aiSummaryLoading) return
    await this.loadAISummary()
  },

  async refreshAISummary() {
    if (this.data.aiSummaryLoading) return
    await this.loadAISummary(true)
  },

  async loadAISummary(refresh = false) {
    this.setData({ aiSummaryLoading: true, aiSummaryError: '' })
    try {
      const summary = await AISummaryService.taskSummary({ refresh })
      this.setData({
        aiSummary: {
          ...summary,
          focusTasks: summary.focusTasks.map((item) => ({
            ...item,
            dueText: item.dueAt ? formatDateTime(item.dueAt) : '',
            metaText: item.dueAt
              ? `${item.reason} · ${formatDateTime(item.dueAt)}`
              : item.reason,
          })),
        },
      })
      if (!summary.aiGenerated) {
        wx.showToast({ title: 'AI 暂时忙，已生成基础梳理', icon: 'none' })
      }
    } catch (error) {
      this.setData({ aiSummaryError: error.message || '暂时无法生成总结' })
    } finally {
      this.setData({ aiSummaryLoading: false })
    }
  },

  closeAISummary() {
    this.setData({ aiSummaryVisible: false })
    this.setTabBarHidden(false)
  },

  openSummaryTask({ currentTarget }) {
    const id = currentTarget.dataset.id
    if (!id) return
    this.closeAISummary()
    this.openTaskPage(`/pages/tasks/detail?id=${id}`)
  },

  async completeSummaryTask({ currentTarget }) {
    await this.runSummaryTaskAction(currentTarget.dataset.id, async (id) => {
      await TaskService.setStatus(id, 'done')
      return '已完成'
    })
  },

  async postponeSummaryTask({ currentTarget }) {
    const id = currentTarget.dataset.id
    const task = this.data.aiSummary?.focusTasks.find((item) => item.id === id)
    await this.runSummaryTaskAction(id, async (taskId) => {
      const tomorrow = buildQuickPostponeOptions(new Date(), task?.dueAt)
        .find((item) => item.key === 'tomorrow')
      await TaskService.reschedule(taskId, tomorrow.dueAt)
      return '已延期到明天'
    })
  },

  async runSummaryTaskAction(id, action) {
    if (!id || this.data.aiActionTaskId) return
    if (!await requireWriteAccess(MODULES.TASK)) return
    this.setData({ aiActionTaskId: id })
    try {
      const message = await action(id)
      await Promise.all([
        this.loadTasks({ preserveLoadedPages: this.data.page > 1 }),
        this.loadAISummary(false),
      ])
      wx.showToast({ title: message })
    } catch (error) {
      wx.showToast({ title: error.message || '操作失败', icon: 'none' })
    } finally {
      this.setData({ aiActionTaskId: '' })
    }
  },

  setTabBarHidden(hidden) {
    const tabBar = typeof this.getTabBar === 'function' ? this.getTabBar() : null
    if (tabBar) tabBar.setData({ hidden })
  },

  noop() {},
}))
