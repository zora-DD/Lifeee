const { DailyRoutineService } = require('../../services/daily-routine-service')
const { requireWriteAccess } = require('../../utils/access')
const { getWeekdayLabel } = require('../../utils/date')
const { MODULES } = require('../../utils/permissions')

const repeatOptions = [
  { label: '每天', unit: 'day', interval: 1 },
  { label: '每 2 天', unit: 'day', interval: 2 },
  { label: '每周', unit: 'week', interval: 1 },
  { label: '每月', unit: 'month', interval: 1 },
]

function localDateText(date = new Date()) {
  const pad = (value) => String(value).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function getRepeatLabel(unit, interval) {
  const matched = repeatOptions.find((item) => item.unit === unit && item.interval === interval)
  return matched?.label || `每 ${interval} ${unit === 'month' ? '个月' : unit === 'week' ? '周' : '天'}`
}

function toRoutineView(item) {
  const title = item.title || ''
  let icon = '🌱'
  if (/药|维生素|保健/.test(title)) icon = '💊'
  else if (/书|阅读|学习|知识/.test(title)) icon = '📖'
  else if (/水|喝水/.test(title)) icon = '💧'
  else if (/跑|运动|健身|走路/.test(title)) icon = '🏃'
  else if (/睡|早起/.test(title)) icon = '🌙'
  return {
    ...item,
    icon,
    repeatLabel: getRepeatLabel(item.repeatUnit, item.repeatInterval),
    history: item.history.map((day) => ({
      ...day,
      weekday: getWeekdayLabel(day.date).replace('周', ''),
      dayText: String(Number(day.date.slice(-2))),
    })),
  }
}

Page({
  data: {
    loading: true,
    routines: [],
    visibleRoutines: [],
    routineScope: 'today',
    currentDate: '',
    checkingId: '',
    enabledCount: 0,
    doneCount: 0,
    todayRate: 0,
    todayTotal: 0,
    monthCheckins: 0,
    monthRate: 0,
    showCalendar: false,
    calendarTitle: '',
    calendarRoutineTitle: '',
    calendarCompleted: 0,
    calendarDays: [],
    calendarWeekdays: ['一', '二', '三', '四', '五', '六', '日'],
    repeatOptions,
    showEditor: false,
    saving: false,
    editingId: '',
    form: {
      title: '', timeOfDay: '09:00', repeatUnit: 'day', repeatInterval: 1,
      repeatIndex: 0, startDate: localDateText(), enabled: true,
    },
  },

  async onLoad() {
    const user = await requireWriteAccess(MODULES.TASK)
    if (!user) return
    this.loadRoutines()
  },

  async loadRoutines({ silent = false } = {}) {
    if (!silent) this.setData({ loading: true })
    try {
      const result = await DailyRoutineService.list({ includeDisabled: true })
      const routines = result.items.map(toRoutineView)
      this.applyRoutineData(routines, result.date)
    } catch (error) {
      wx.showToast({ title: error.message || '加载失败', icon: 'none' })
    } finally {
      if (!silent) this.setData({ loading: false })
    }
  },

  applyRoutineData(routines, currentDate, scope = this.data.routineScope) {
    const enabledRoutines = routines.filter((item) => item.enabled)
    const todayRoutines = enabledRoutines.filter((item) => item.scheduledToday)
    const doneCount = todayRoutines.filter((item) => item.checkedToday).length
    const scheduledMonthDays = enabledRoutines.flatMap((item) => item.monthHistory)
      .filter((day) => day.scheduled && day.date <= currentDate)
    const monthCheckins = scheduledMonthDays.filter((day) => day.completed).length
    this.setData({
      currentDate,
      routines,
      visibleRoutines: scope === 'today' ? todayRoutines : routines,
      routineScope: scope,
      enabledCount: enabledRoutines.length,
      doneCount,
      todayTotal: todayRoutines.length,
      todayRate: todayRoutines.length ? Math.round((doneCount / todayRoutines.length) * 100) : 0,
      monthCheckins,
      monthRate: scheduledMonthDays.length
        ? Math.round((monthCheckins / scheduledMonthDays.length) * 100)
        : 0,
    })
  },

  selectRoutineScope({ currentTarget }) {
    this.applyRoutineData(this.data.routines, this.data.currentDate, currentTarget.dataset.scope)
  },

  async toggleCheckin({ currentTarget }) {
    const id = currentTarget.dataset.id
    const routine = this.data.routines.find((item) => item.id === id)
    if (!routine || this.data.checkingId) return
    if (!routine.enabled) {
      wx.showToast({ title: '请先启用该习惯', icon: 'none' })
      return
    }
    if (!routine.scheduledToday) {
      wx.showToast({ title: '今天不需要打卡', icon: 'none' })
      return
    }
    this.setData({ checkingId: id })
    try {
      await DailyRoutineService.checkin(
        id,
        this.data.currentDate,
        !routine.checkedToday,
      )
      await this.loadRoutines({ silent: true })
      wx.showToast({ title: routine.checkedToday ? '已撤销' : '今天已完成', icon: 'none' })
    } catch (error) {
      wx.showToast({ title: error.message || '打卡失败', icon: 'none' })
    } finally {
      this.setData({ checkingId: '' })
    }
  },

  createRoutine() {
    this.setData({
      showEditor: true,
      editingId: '',
      form: {
        title: '', timeOfDay: '09:00', repeatUnit: 'day', repeatInterval: 1,
        repeatIndex: 0, startDate: this.data.currentDate || localDateText(), enabled: true,
      },
    })
  },

  editRoutine({ currentTarget }) {
    const routine = this.data.routines.find((item) => item.id === currentTarget.dataset.id)
    if (!routine) return
    this.startEditingRoutine(routine)
  },

  startEditingRoutine(routine) {
    const repeatIndex = Math.max(0, repeatOptions.findIndex((item) => (
      item.unit === routine.repeatUnit && item.interval === routine.repeatInterval
    )))
    this.setData({
      showEditor: true,
      editingId: routine.id,
      form: {
        title: routine.title,
        timeOfDay: routine.timeOfDay,
        repeatUnit: routine.repeatUnit,
        repeatInterval: routine.repeatInterval,
        repeatIndex,
        startDate: routine.startDate || this.data.currentDate || localDateText(),
        enabled: routine.enabled,
      },
    })
  },

  openRoutineActions({ currentTarget }) {
    const routine = this.data.routines.find((item) => item.id === currentTarget.dataset.id)
    if (!routine) return
    wx.showActionSheet({
      itemList: ['编辑习惯', routine.enabled ? '暂停习惯' : '重新启用', '删除习惯'],
      success: ({ tapIndex }) => {
        if (tapIndex === 0) this.startEditingRoutine(routine)
        if (tapIndex === 1) this.toggleRoutineEnabled(routine)
        if (tapIndex === 2) this.confirmDeleteRoutine(routine.id)
      },
    })
  },

  async toggleRoutineEnabled(routine) {
    try {
      await DailyRoutineService.update(routine.id, {
        title: routine.title,
        timeOfDay: routine.timeOfDay,
        repeatUnit: routine.repeatUnit,
        repeatInterval: routine.repeatInterval,
        startDate: routine.startDate,
        enabled: !routine.enabled,
      })
      await this.loadRoutines({ silent: true })
      wx.showToast({ title: routine.enabled ? '已暂停' : '已启用', icon: 'none' })
    } catch (error) {
      wx.showToast({ title: error.message || '操作失败', icon: 'none' })
    }
  },

  closeEditor() {
    if (!this.data.saving) this.setData({ showEditor: false })
  },

  onTitleInput({ detail }) {
    this.setData({ 'form.title': detail.value })
  },

  onTimeChange({ detail }) {
    this.setData({ 'form.timeOfDay': detail.value })
  },

  onRepeatChange({ detail }) {
    const repeatIndex = Number(detail.value) || 0
    const selected = repeatOptions[repeatIndex]
    this.setData({
      'form.repeatIndex': repeatIndex,
      'form.repeatUnit': selected.unit,
      'form.repeatInterval': selected.interval,
    })
  },

  onStartDateChange({ detail }) {
    this.setData({ 'form.startDate': detail.value })
  },

  onEnabledChange({ detail }) {
    this.setData({ 'form.enabled': detail.value })
  },

  async saveRoutine() {
    if (this.data.saving) return
    const title = this.data.form.title.trim()
    if (!title) {
      wx.showToast({ title: '请填写习惯名称', icon: 'none' })
      return
    }
    this.setData({ saving: true })
    try {
      const input = { ...this.data.form, title }
      if (this.data.editingId) {
        await DailyRoutineService.update(this.data.editingId, input)
      } else {
        await DailyRoutineService.create(input)
      }
      this.setData({ showEditor: false })
      await this.loadRoutines()
      this.notifyChanged()
      wx.showToast({ title: this.data.editingId ? '已更新' : '已添加' })
    } catch (error) {
      wx.showToast({ title: error.message || '保存失败', icon: 'none' })
    } finally {
      this.setData({ saving: false })
    }
  },

  deleteRoutine({ currentTarget }) {
    this.confirmDeleteRoutine(currentTarget.dataset.id)
  },

  confirmDeleteRoutine(id) {
    const routine = this.data.routines.find((item) => item.id === id)
    if (!routine) return
    wx.showModal({
      title: '删除习惯',
      content: `确定删除“${routine.title}”吗？`,
      confirmColor: '#bd625d',
      success: async ({ confirm }) => {
        if (!confirm) return
        try {
          await DailyRoutineService.remove(id)
          await this.loadRoutines()
          this.notifyChanged()
          wx.showToast({ title: '已删除' })
        } catch (error) {
          wx.showToast({ title: error.message || '删除失败', icon: 'none' })
        }
      },
    })
  },

  openMonthCalendar({ currentTarget }) {
    const routine = this.data.routines.find((item) => item.id === currentTarget.dataset.id)
    if (!routine || !routine.monthHistory.length) return
    const firstDate = new Date(`${routine.monthHistory[0].date}T00:00:00`)
    const leading = (firstDate.getDay() + 6) % 7
    const calendarDays = [
      ...Array.from({ length: leading }, (_, index) => ({ key: `blank-${index}`, blank: true })),
      ...routine.monthHistory.map((day) => ({
        ...day,
        key: day.date,
        dayText: String(Number(day.date.slice(-2))),
        today: day.date === this.data.currentDate,
        future: day.date > this.data.currentDate,
      })),
    ]
    this.setData({
      showCalendar: true,
      calendarTitle: `${firstDate.getFullYear()}年${firstDate.getMonth() + 1}月`,
      calendarRoutineTitle: routine.title,
      calendarCompleted: routine.monthHistory.filter((day) => day.completed).length,
      calendarDays,
    })
  },

  closeMonthCalendar() {
    this.setData({ showCalendar: false })
  },

  notifyChanged() {
    if (typeof this.getOpenerEventChannel !== 'function') return
    const channel = this.getOpenerEventChannel()
    if (channel && typeof channel.emit === 'function') channel.emit('routineChanged')
  },

  noop() {},
})

module.exports = { toRoutineView }
