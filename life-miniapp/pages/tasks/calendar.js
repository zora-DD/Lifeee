const { TaskService } = require('../../services/task-service')
const { getCurrentUser } = require('../../utils/access')
const { MODULES, hasPermission, loadPermissions } = require('../../utils/permissions')
const { toPickerTime } = require('../../utils/date')
const {
  addDays,
  buildDays,
  formatDateKey,
  formatLocalRFC3339,
  formatPeriodTitle,
  formatSelectedDate,
  getDaySectionMinHeight,
  getViewRange,
  parseDateKey,
} = require('../../utils/task-calendar')

const PAGE_SIZE = 100
const priorityText = { high: '高', medium: '中', low: '低' }

function endInclusive(end) {
  return new Date(end.getTime() - 1)
}

Page({
  data: {
    ready: false,
    loading: false,
    loadError: '',
    viewMode: 'week',
    viewOptions: [
      { value: 'week', label: '周视图' },
      { value: 'month', label: '月视图' },
    ],
    weekHeaders: ['一', '二', '三', '四', '五', '六', '日'],
    periodTitle: '',
    days: [],
    selectedDate: '',
    selectedDateLabel: '',
    selectedTasks: [],
    rangeTotal: 0,
    rangeTodo: 0,
    rangeDone: 0,
    daySectionMinHeight: 250,
    canWrite: false,
  },

  async onLoad() {
    const user = await getCurrentUser()
    if (!user || user.accessStatus !== 'approved') {
      wx.showToast({ title: '登录后查看任务日历', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 500)
      return
    }
    const levels = await loadPermissions(user)
    if (!hasPermission(levels, MODULES.TASK)) {
      wx.showToast({ title: '暂无任务查看权限', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 500)
      return
    }
    const today = new Date()
    this.anchorDate = today
    this.allTasks = []
    this.openerEventChannel = typeof this.getOpenerEventChannel === 'function'
      ? this.getOpenerEventChannel()
      : null
    this.setData({
      ready: true,
      canWrite: hasPermission(levels, MODULES.TASK, 'write'),
      selectedDate: formatDateKey(today),
    })
    await this.loadCalendar()
  },

  async onShow() {
    if (!this.calendarDirty || !this.data.ready) return
    this.calendarDirty = false
    await this.loadCalendar()
  },

  async fetchRangeTasks(start, end) {
    const items = []
    let page = 1
    let hasMore = true
    while (hasMore) {
      const result = await TaskService.listPage({
        dueFrom: formatLocalRFC3339(start),
        dueTo: formatLocalRFC3339(endInclusive(end)),
        page,
        pageSize: PAGE_SIZE,
      })
      items.push(...result.items)
      hasMore = result.hasMore && result.items.length > 0
      page += 1
    }
    return items
  },

  async loadCalendar() {
    const { viewMode, selectedDate } = this.data
    const { start, end } = getViewRange(viewMode, this.anchorDate)
    this.setData({ loading: true, loadError: '' })
    try {
      this.allTasks = await this.fetchRangeTasks(start, end)
      const days = buildDays(viewMode, this.anchorDate, this.allTasks)
      const selectedStillVisible = days.some((day) => day.key === selectedDate)
      const nextSelectedDate = selectedStillVisible ? selectedDate : formatDateKey(this.anchorDate)
      const selectedTaskView = this.buildSelectedTaskView(nextSelectedDate)
      this.setData({
        loading: false,
        periodTitle: formatPeriodTitle(viewMode, this.anchorDate),
        days,
        selectedDate: nextSelectedDate,
        ...selectedTaskView,
        rangeTotal: this.allTasks.length,
        rangeTodo: this.allTasks.filter((task) => task.status !== 'done').length,
        rangeDone: this.allTasks.filter((task) => task.status === 'done').length,
        daySectionMinHeight: getDaySectionMinHeight(this.allTasks),
      })
    } catch (error) {
      this.setData({ loading: false, loadError: error.message || '任务日历加载失败' })
    }
  },

  buildSelectedTaskView(dateKey) {
    const selectedTasks = this.allTasks
      .filter((task) => task.dueAt && formatDateKey(new Date(task.dueAt)) === dateKey)
      .sort((left, right) => {
        if (left.status !== right.status) return left.status === 'done' ? 1 : -1
        return new Date(left.dueAt).getTime() - new Date(right.dueAt).getTime()
      })
      .map((task) => ({
        ...task,
        priorityText: priorityText[task.priority] || '中',
        timeText: toPickerTime(task.dueAt),
      }))
    return {
      selectedDateLabel: formatSelectedDate(dateKey),
      selectedTasks,
    }
  },

  switchView({ currentTarget }) {
    const viewMode = currentTarget.dataset.value
    if (viewMode === this.data.viewMode) return
    this.anchorDate = parseDateKey(this.data.selectedDate)
    this.setData({ viewMode }, () => this.loadCalendar())
  },

  changePeriod({ currentTarget }) {
    const direction = Number(currentTarget.dataset.direction)
    if (this.data.viewMode === 'month') {
      this.anchorDate = new Date(
        this.anchorDate.getFullYear(),
        this.anchorDate.getMonth() + direction,
        1,
      )
    } else {
      this.anchorDate = addDays(this.anchorDate, direction * 7)
    }
    this.setData({ selectedDate: formatDateKey(this.anchorDate) }, () => this.loadCalendar())
  },

  goToday() {
    this.anchorDate = new Date()
    this.setData({ selectedDate: formatDateKey(this.anchorDate) }, () => this.loadCalendar())
  },

  selectDay({ currentTarget }) {
    const selectedDate = currentTarget.dataset.date
    const taskCount = Number(currentTarget.dataset.taskCount || 0)
    this.setData({
      selectedDate,
      ...this.buildSelectedTaskView(selectedDate),
    })
    if (taskCount === 0 && this.data.canWrite) this.createTaskForDate(selectedDate)
  },

  createTaskForSelectedDate() {
    this.createTaskForDate(this.data.selectedDate)
  },

  createTaskForDate(date) {
    if (!date || !this.data.canWrite) return
    wx.navigateTo({
      url: `/pages/tasks/form?date=${date}`,
      events: {
        taskChanged: ({ task } = {}) => {
          this.calendarDirty = true
          if (this.openerEventChannel && typeof this.openerEventChannel.emit === 'function') {
            this.openerEventChannel.emit('taskChanged', { id: task?.id || '' })
          }
        },
      },
    })
  },

  openTask({ currentTarget }) {
    const id = currentTarget.dataset.id
    wx.navigateTo({
      url: `/pages/tasks/detail?id=${id}`,
      events: {
        taskChanged: () => {
          this.calendarDirty = true
          if (this.openerEventChannel && typeof this.openerEventChannel.emit === 'function') {
            this.openerEventChannel.emit('taskChanged', { id })
          }
        },
      },
    })
  },

  retryLoad() {
    this.loadCalendar()
  },
})
