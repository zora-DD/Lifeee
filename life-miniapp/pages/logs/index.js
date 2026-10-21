const { ActivityLogService } = require('../../services/activity-log-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { BillService } = require('../../services/bill-service')
const { getCurrentUser, requireWriteAccess } = require('../../utils/access')
const { toPickerDate, toPickerTime, pickerValuesToISO } = require('../../utils/date')
const { buildCurrentYearMonthOptions, getMonthRange } = require('../../utils/month-filter')
const { getBillCategoryIcon } = require('../../utils/bill-category')
const {
  MODULES,
  hasPermission,
  loadPermissions,
  redirectToFirstAllowedTab,
  syncTabBar,
} = require('../../utils/permissions')

const WEEKDAYS = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
const BILL_CATEGORIES = [
  '餐饮', '交通', '日常采购', '家居用品', '宠物', '医疗', '娱乐',
  '住房', '购物', '智能设备', '生活缴费', '转账', '收入', '其他',
]
const RECORD_MONTH_OPTIONS = buildCurrentYearMonthOptions()

function pad(value) { return String(value).padStart(2, '0') }

function getInitialForm() {
  const now = new Date()
  return { editingId: '', content: '', dateValue: toPickerDate(now), timeValue: toPickerTime(now), durationValue: '' }
}

function readFileAsBase64(filePath) {
  return new Promise((resolve, reject) => {
    wx.getFileSystemManager().readFile({
      filePath,
      encoding: 'base64',
      success: ({ data }) => resolve(data),
      fail: ({ errMsg }) => reject(new Error(errMsg || '读取录音失败')),
    })
  })
}

function ensureRecordPermission() {
  return new Promise((resolve, reject) => {
    wx.getSetting({
      success: ({ authSetting }) => {
        if (authSetting['scope.record']) return resolve()
        wx.authorize({
          scope: 'scope.record',
          success: resolve,
          fail: () => reject(new Error('请允许使用麦克风后再录音')),
        })
      },
      fail: ({ errMsg }) => reject(new Error(errMsg || '无法读取录音权限')),
    })
  })
}

function getRecorderErrorMessage(errMsg) {
  const message = String(errMsg || '')
  if (/auth deny|authorize:fail|permission/i.test(message)) {
    return '麦克风权限未开启，请在小程序设置中允许使用麦克风。'
  }
  if (/privacy|隐私/i.test(message)) {
    return '小程序隐私保护指引尚未声明录音用途，请联系管理员配置后再试。'
  }
  if (/operateRecorder/i.test(message)) {
    return '录音启动失败，请结束其他正在使用麦克风的应用后重试。'
  }
  return message || '录音启动失败，请稍后重试。'
}

function formatTime(value) {
  const date = new Date(value)
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function formatDuration(minutes) {
  const value = Number(minutes)
  if (!value) return ''
  if (value < 60) return `${value} 分钟`
  const hours = Math.floor(value / 60)
  const rest = value % 60
  return rest ? `${hours} 小时 ${rest} 分钟` : `${hours} 小时`
}

function getDateKey(value) {
  const date = new Date(value)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function getDateLabel(value) {
  const date = new Date(value)
  return `${date.getMonth() + 1}月${date.getDate()}日 · ${WEEKDAYS[date.getDay()]}`
}

function formatMoney(value) { return Number(value || 0).toFixed(2) }

function safeDate(value) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? new Date() : date
}

function normalizeBillDraft(item, rawText, inputType) {
  const occurredAt = safeDate(item.occurredAt)
  const categoryIndex = Math.max(0, BILL_CATEGORIES.indexOf(item.category))
  return {
    type: item.type === 'income' ? 'income' : 'expense',
    title: item.title || '',
    amount: formatMoney(item.amount),
    category: BILL_CATEGORIES[categoryIndex],
    categoryIcon: getBillCategoryIcon(BILL_CATEGORIES[categoryIndex]),
    categoryIndex,
    occurredAt: occurredAt.toISOString(),
    dateValue: toPickerDate(occurredAt),
    timeValue: toPickerTime(occurredAt),
    note: item.note || '',
    rawText: rawText || '',
    inputType: inputType || 'text',
  }
}

function mergeGroups(currentGroups, nextGroups) {
  const merged = []
  const groupMap = {}
  const seen = new Set()
  ;[...(currentGroups || []), ...(nextGroups || [])].forEach((group) => {
    let target = groupMap[group.key]
    if (!target) {
      target = { ...group, items: [] }
      groupMap[group.key] = target
      merged.push(target)
    }
    ;(group.items || []).forEach((item) => {
      if (seen.has(item.id)) return
      seen.add(item.id)
      target.items.push(item)
    })
  })
  return merged
}

Page(withScrollRestoration({
  data: {
    authChecked: false,
    loggedIn: false,
    canReadActivity: false,
    canWriteActivity: false,
    canReadBill: false,
    canWriteBill: false,
    currentUserId: '',
    isAdmin: false,
    activeMode: 'bill',
    saving: false,
    parsing: false,
    recording: false,
    recognizing: false,
    recordingSeconds: 0,
    showQuickDraft: false,
    quickDraftIntent: '',
    activityDraft: null,
    billDrafts: [],
    editingBillId: '',
    rawDraftText: '',
    quickText: '',
    groups: [],
    recordCount: 0,
    logPage: 1,
    logHasMore: false,
    logLoadingMore: false,
    billGroups: [],
    billCount: 0,
    billPage: 1,
    billHasMore: false,
    billLoadingMore: false,
    billStats: { expenseTotal: '0.00', billCount: 0 },
    billCategories: BILL_CATEGORIES,
    creatorOptions: [{ id: '', nickname: '全部用户' }],
    creatorNames: ['全部用户'],
    creatorIndex: 0,
    creatorDropdownOpen: false,
    monthOptions: RECORD_MONTH_OPTIONS,
    monthIndex: Math.max(0, RECORD_MONTH_OPTIONS.length - 1),
    monthFilterYear: new Date().getFullYear(),
    monthDropdownOpen: false,
    billOverviewLabel: '本月支出',
    editingId: '',
    content: '',
    dateValue: '',
    timeValue: '',
    durationValue: '',
  },

  onLoad() {
    this.recorderManager = wx.getRecorderManager()
    this.handleRecorderStart = this.onRecorderStart.bind(this)
    this.handleRecorderStop = this.onRecorderStop.bind(this)
    this.handleRecorderError = this.onRecorderError.bind(this)
    this.recorderManager.onStart(this.handleRecorderStart)
    this.recorderManager.onStop(this.handleRecorderStop)
    this.recorderManager.onError(this.handleRecorderError)
  },

  onUnload() {
    this.discardingRecording = true
    this.setTabBarHidden(false)
    this.stopRecordingTimer()
    if (this.recorderManager) {
      if (this.data.recording) this.recorderManager.stop()
      if (typeof this.recorderManager.offStart === 'function') this.recorderManager.offStart(this.handleRecorderStart)
      if (typeof this.recorderManager.offStop === 'function') this.recorderManager.offStop(this.handleRecorderStop)
      if (typeof this.recorderManager.offError === 'function') this.recorderManager.offError(this.handleRecorderError)
    }
  },

  async onShow() {
    const initialMode = wx.getStorageSync('record_center_initial_mode')
    const hasInitialMode = initialMode === 'activity' || initialMode === 'bill'
    const canReusePrevious = this.skipNextShowRefresh && this.data.authChecked && this.data.loggedIn
    this.skipNextShowRefresh = false
    // 查看详情后直接恢复原页面，避免列表清空后重新渲染。
    if (canReusePrevious && !hasInitialMode) return
    this.setData({ authChecked: false })
    const user = await getCurrentUser()
    if (!user || user.accessStatus !== 'approved') {
      this.setData({ authChecked: true, loggedIn: false, groups: [], billGroups: [], recordCount: 0, billCount: 0, ...getInitialForm() })
      return
    }
    const levels = await loadPermissions(user)
    syncTabBar(this, 1, levels)
    const canReadActivity = hasPermission(levels, MODULES.ACTIVITY)
    const canReadBill = hasPermission(levels, MODULES.BILL)
    if (!canReadActivity && !canReadBill) {
      this.setData({ authChecked: true, loggedIn: false, groups: [], billGroups: [] })
      redirectToFirstAllowedTab(levels)
      return
    }

    let activeMode = this.data.activeMode
    if (hasInitialMode) wx.removeStorageSync('record_center_initial_mode')
    if (hasInitialMode && ((initialMode === 'bill' && canReadBill) || (initialMode === 'activity' && canReadActivity))) {
      activeMode = initialMode
    } else if (activeMode === 'bill' && !canReadBill) {
      activeMode = 'activity'
    } else if (activeMode === 'activity' && !canReadActivity) {
      activeMode = 'bill'
    }

    const reuseCurrentList = canReusePrevious &&
      !hasInitialMode &&
      activeMode === this.data.activeMode
    this.setData({
      authChecked: true,
      loggedIn: true,
      activeMode,
      canReadActivity,
      canWriteActivity: hasPermission(levels, MODULES.ACTIVITY, 'write'),
      canReadBill,
      canWriteBill: hasPermission(levels, MODULES.BILL, 'write'),
      currentUserId: String(user.id || ''),
      isAdmin: user.role === 'admin',
      ...getInitialForm(),
    })
    if (reuseCurrentList) return
    await this.loadCurrentMode()
  },

  async loadCurrentMode() {
    if (this.data.activeMode === 'bill') return this.loadBills()
    return this.loadLogs()
  },

  onReachBottom() {
    if (this.data.activeMode === 'bill') {
      this.loadBills({ append: true })
      return
    }
    this.loadLogs({ append: true })
  },

  async switchMode({ currentTarget }) {
    const mode = currentTarget.dataset.mode
    if (!mode || mode === this.data.activeMode) return
    if (mode === 'bill' && !this.data.canReadBill) return
    if (mode === 'activity' && !this.data.canReadActivity) return
    this.setData({
      activeMode: mode,
      quickText: '',
      editingBillId: '',
      monthDropdownOpen: false,
      creatorDropdownOpen: false,
      ...getInitialForm(),
    })
    await this.loadCurrentMode()
  },

  async loadLogs({ append = false } = {}) {
    if (append && (this.data.logLoadingMore || !this.data.logHasMore)) return
    const period = this.getSelectedMonthRange()
    const page = append ? this.data.logPage + 1 : 1
    if (append) this.setData({ logLoadingMore: true })
    let result
    try {
      result = await ActivityLogService.list({
        from: period ? period.from : undefined,
        to: period ? period.to : undefined,
        page,
        pageSize: 20,
      })
    } finally {
      if (append) this.setData({ logLoadingMore: false })
    }
    const grouped = []
    const groupMap = {}
    result.items.forEach((log) => {
      const key = getDateKey(log.occurredAt)
      if (!groupMap[key]) {
        groupMap[key] = { key, label: getDateLabel(log.occurredAt), items: [] }
        grouped.push(groupMap[key])
      }
      groupMap[key].items.push({
        ...log,
        timeText: formatTime(log.occurredAt),
        durationText: formatDuration(log.durationMinutes),
        creatorName: log.creator && log.creator.name ? log.creator.name : '我',
        canEdit: this.data.canWriteActivity && (this.data.isAdmin || String(log.creator?.id || '') === this.data.currentUserId),
      })
    })
    this.setData({
      groups: append ? mergeGroups(this.data.groups, grouped) : grouped,
      recordCount: result.total,
      logPage: result.page,
      logHasMore: result.hasMore,
    })
  },

  async loadBills({ append = false } = {}) {
    if (append && (this.data.billLoadingMore || !this.data.billHasMore)) return
    const creator = this.data.creatorOptions[this.data.creatorIndex] || { id: '' }
    const period = this.getSelectedMonthRange()
    const page = append ? this.data.billPage + 1 : 1
    const params = {
      createdBy: creator.id,
      from: period ? period.from : undefined,
      to: period ? period.to : undefined,
      page,
      pageSize: 20,
    }
    if (append) this.setData({ billLoadingMore: true })
    let result
    try {
      result = await BillService.list(params)
    } finally {
      if (append) this.setData({ billLoadingMore: false })
    }
    let stats = this.data.billStats
    if (!append) {
      stats = { expenseTotal: 0, billCount: result.total }
      try {
        stats = await BillService.stats({
          createdBy: creator.id,
          scope: period ? period.scope : undefined,
          year: period ? period.year : undefined,
          month: period ? period.month : undefined,
        })
      } catch (error) {
        // 汇总失败不能影响已经成功返回的账单列表。
        console.warn('加载账单汇总失败', error)
      }
    }
    const grouped = []
    const groupMap = {}
    result.items.forEach((bill) => {
      const key = getDateKey(bill.occurredAt)
      if (!groupMap[key]) {
        groupMap[key] = { key, label: getDateLabel(bill.occurredAt), items: [] }
        grouped.push(groupMap[key])
      }
      groupMap[key].items.push({
        ...bill,
        amountText: formatMoney(bill.amount),
        timeText: formatTime(bill.occurredAt),
        categoryIcon: getBillCategoryIcon(bill.category),
        creatorName: bill.creator && bill.creator.nickname ? bill.creator.nickname : '未命名用户',
        creatorEmoji: bill.creator && bill.creator.avatarEmoji ? bill.creator.avatarEmoji : '记',
        canEdit: this.data.canWriteBill && (this.data.isAdmin || String(bill.creator?.id || '') === this.data.currentUserId),
      })
    })
    const creatorOptions = [{ id: '', nickname: '全部用户' }, ...result.creators]
    const selectedID = creator.id
    let creatorIndex = creatorOptions.findIndex((item) => item.id === selectedID)
    if (creatorIndex < 0) creatorIndex = 0
    this.setData({
      billGroups: append ? mergeGroups(this.data.billGroups, grouped) : grouped,
      billCount: result.total,
      billPage: result.page,
      billHasMore: result.hasMore,
      billStats: {
        expenseTotal: formatMoney(stats && stats.expenseTotal),
        billCount: Number(stats && stats.billCount) || 0,
      },
      creatorOptions,
      creatorNames: creatorOptions.map((item) => item.nickname || '未命名用户'),
      creatorIndex,
      billOverviewLabel: period ? `${period.label}支出` : '本月支出',
    })
  },

  getSelectedMonthRange() {
    return getMonthRange(this.data.monthOptions[this.data.monthIndex])
  },

  toggleMonthDropdown() {
    this.setData({
      monthDropdownOpen: !this.data.monthDropdownOpen,
      creatorDropdownOpen: false,
    })
  },

  closeMonthDropdown() {
    this.setData({ monthDropdownOpen: false })
  },

  async selectMonth({ currentTarget }) {
    const monthIndex = Number(currentTarget.dataset.index) || 0
    if (monthIndex === this.data.monthIndex) {
      this.closeMonthDropdown()
      return
    }
    this.setData({ monthIndex, monthDropdownOpen: false })
    await this.loadCurrentMode()
  },

  toggleCreatorDropdown() {
    this.setData({
      creatorDropdownOpen: !this.data.creatorDropdownOpen,
      monthDropdownOpen: false,
    })
  },

  closeCreatorDropdown() {
    this.setData({ creatorDropdownOpen: false })
  },

  selectCreator({ currentTarget }) {
    const creatorIndex = Number(currentTarget.dataset.index) || 0
    if (creatorIndex === this.data.creatorIndex) {
      this.closeCreatorDropdown()
      return
    }
    this.setData({ creatorIndex, creatorDropdownOpen: false }, () => this.loadBills())
  },

  onContentInput({ detail }) { this.setData({ content: detail.value }) },
  onQuickTextInput({ detail }) { this.setData({ quickText: detail.value }) },
  onDateChange({ detail }) { this.setData({ dateValue: detail.value }) },
  onTimeChange({ detail }) { this.setData({ timeValue: detail.value }) },
  onDurationInput({ detail }) { this.setData({ durationValue: detail.value.replace(/\D/g, '') }) },

  async saveLog() {
    if (!await requireWriteAccess(MODULES.ACTIVITY)) return
    const content = this.data.content.trim()
    if (!content) return wx.showToast({ title: '先写下做了什么', icon: 'none' })
    if (!this.data.durationValue) return wx.showToast({ title: '请填写花费时间', icon: 'none' })
    const durationMinutes = Number(this.data.durationValue)
    if (durationMinutes < 1 || durationMinutes > 1440) {
      return wx.showToast({ title: '时长填写 1～1440 分钟', icon: 'none' })
    }
    const input = { content, occurredAt: pickerValuesToISO(this.data.dateValue, this.data.timeValue), durationMinutes, inputType: 'text' }
    this.setData({ saving: true })
    try {
      if (this.data.editingId) {
        await ActivityLogService.update(this.data.editingId, input)
        wx.showToast({ title: '记录已更新' })
      } else {
        await ActivityLogService.create(input)
        wx.showToast({ title: '已经记下' })
      }
      this.setData(getInitialForm())
      await this.loadLogs()
    } finally {
      this.setData({ saving: false })
    }
  },

  async parseQuickText() {
    if (!await requireWriteAccess(MODULES.BILL)) return
    const text = this.data.quickText.trim()
    if (!text) return wx.showToast({ title: '先写一句消费记录', icon: 'none' })
    this.setData({ parsing: true })
    try {
      const result = await BillService.parse(text)
      this.showParsedDraft(result, 'text')
    } catch (error) {
      wx.showModal({ title: '整理失败', content: error.message || '请稍后再试', showCancel: false })
    } finally {
      this.setData({ parsing: false })
    }
  },

  showParsedDraft(result, inputType) {
    if (result.intent === 'bill') {
      if (!this.data.canWriteBill) {
        this.setTabBarHidden(false)
        wx.showToast({ title: '你暂无账单记录权限', icon: 'none' })
        return
      }
      this.setData({
        activeMode: 'bill', showQuickDraft: true, quickDraftIntent: 'bill', activityDraft: null,
        billDrafts: (result.bills || []).map((item) => normalizeBillDraft(item, result.rawText, inputType)),
        editingBillId: '', rawDraftText: result.rawText || '',
      })
    } else {
      if (!this.data.canWriteActivity) {
        this.setTabBarHidden(false)
        wx.showToast({ title: '你暂无生活记录权限', icon: 'none' })
        return
      }
      this.setData({
        activeMode: 'activity', showQuickDraft: true, quickDraftIntent: 'activity_log',
        activityDraft: result.activity || null, billDrafts: [], editingBillId: '', rawDraftText: result.rawText || '',
      })
    }
    this.setTabBarHidden(true)
  },

  async toggleRecording() {
    if (this.data.recognizing) return
    if (this.data.recording) return this.finishVoiceRecording()
    const module = this.data.activeMode === 'bill' ? MODULES.BILL : MODULES.ACTIVITY
    if (!await requireWriteAccess(module)) return
    try {
      await ensureRecordPermission()
      this.discardingRecording = false
      this.beginRecordingUI()
      this.recorderManager.start({ duration: 60000, sampleRate: 16000, numberOfChannels: 1, encodeBitRate: 48000, format: 'mp3' }) //前端写死mp3格式
    } catch (error) {
      this.resetRecordingUI()
      wx.showModal({
        title: '需要麦克风权限', content: error.message || '请在设置中允许使用麦克风。', confirmText: '去设置',
        success: ({ confirm }) => { if (confirm) wx.openSetting() },
      })
    }
  },

  beginRecordingUI() {
    this.stoppingRecording = false
    this.recordStartedAt = Date.now()
    this.setData({ recording: true, recognizing: false, recordingSeconds: 0 })
    this.setTabBarHidden(true)
    this.stopRecordingTimer()
    this.recordingTimer = setInterval(() => {
      this.setData({ recordingSeconds: Math.min(60, Math.floor((Date.now() - this.recordStartedAt) / 1000)) })
    }, 500)
  },

  resetRecordingUI() {
    this.stopRecordingTimer()
    this.setData({ recording: false, recognizing: false, recordingSeconds: 0 })
    this.setTabBarHidden(false)
  },

  onRecorderStart() {
    if (!this.data.recording) this.beginRecordingUI()
  },

  async onRecorderStop({ tempFilePath, duration }) {
    this.stopRecordingTimer()
    this.stoppingRecording = false
    if (this.discardingRecording) {
      this.discardingRecording = false
      this.setData({ recording: false, recognizing: false, recordingSeconds: 0 })
      this.setTabBarHidden(false)
      return
    }
    if (!tempFilePath || Number(duration) < 800) {
      this.setData({ recording: false, recognizing: false, recordingSeconds: 0 })
      this.setTabBarHidden(false)
      return wx.showToast({ title: '录音太短，再说一次吧', icon: 'none' })
    }
    this.setData({ recording: false, recognizing: true, recordingSeconds: 0 })
    try {
      const audioBase64 = await readFileAsBase64(tempFilePath)
      const result = await BillService.recognizeVoice({ audioBase64, voiceFormat: 'mp3' })
      this.showParsedDraft(result, 'voice')
    } catch (error) {
      this.setTabBarHidden(false)
      wx.showModal({ title: '语音整理失败', content: error.message || '请稍后再试，也可以直接打字记录。', showCancel: false })
    } finally {
      this.setData({ recognizing: false })
    }
  },

  onRecorderError({ errMsg }) {
    this.stoppingRecording = false
    this.discardingRecording = false
    this.resetRecordingUI()
    wx.showModal({
      title: '无法开始录音',
      content: getRecorderErrorMessage(errMsg),
      showCancel: false,
    })
  },

  finishVoiceRecording() {
    if (!this.data.recording || !this.recorderManager || this.stoppingRecording) return
    this.stoppingRecording = true
    this.setData({ recording: false, recognizing: true })
    this.recorderManager.stop()
  },

  discardVoiceRecording() {
    if (!this.data.recording || !this.recorderManager) return
    this.discardingRecording = true
    this.stopRecordingTimer()
    this.setData({ recording: false, recordingSeconds: 0 })
    this.setTabBarHidden(false)
    this.recorderManager.stop()
  },

  stopRecordingTimer() {
    if (!this.recordingTimer) return
    clearInterval(this.recordingTimer)
    this.recordingTimer = null
  },

  setTabBarHidden(hidden) {
    if (typeof this.getTabBar !== 'function') return
    const tabBar = this.getTabBar()
    if (tabBar) tabBar.setData({ hidden })
  },

  confirmActivityDraft() {
    const draft = this.data.activityDraft
    if (!draft) return
    const occurredAt = safeDate(draft.occurredAt)
    this.setData({
      content: draft.content, dateValue: toPickerDate(occurredAt), timeValue: toPickerTime(occurredAt),
      durationValue: draft.durationMinutes ? String(draft.durationMinutes) : '',
      showQuickDraft: false, activityDraft: null, rawDraftText: '',
    })
    this.setTabBarHidden(false)
    wx.showToast({ title: '已填入，请确认' })
  },

  onBillDraftInput({ currentTarget, detail }) {
    const { index, field } = currentTarget.dataset
    this.setData({ [`billDrafts[${index}].${field}`]: detail.value })
  },

  onBillCategoryChange({ currentTarget, detail }) {
    const index = Number(currentTarget.dataset.index)
    const categoryIndex = Number(detail.value)
    this.setData({
      [`billDrafts[${index}].categoryIndex`]: categoryIndex,
      [`billDrafts[${index}].category`]: BILL_CATEGORIES[categoryIndex],
      [`billDrafts[${index}].categoryIcon`]: getBillCategoryIcon(BILL_CATEGORIES[categoryIndex]),
    })
  },

  onBillTypeTap({ currentTarget }) {
    const index = Number(currentTarget.dataset.index)
    this.setData({ [`billDrafts[${index}].type`]: currentTarget.dataset.type })
  },

  onBillDateChange({ currentTarget, detail }) {
    const index = Number(currentTarget.dataset.index)
    const draft = this.data.billDrafts[index]
    this.setData({
      [`billDrafts[${index}].dateValue`]: detail.value,
      [`billDrafts[${index}].occurredAt`]: pickerValuesToISO(detail.value, draft.timeValue),
    })
  },

  onBillTimeChange({ currentTarget, detail }) {
    const index = Number(currentTarget.dataset.index)
    const draft = this.data.billDrafts[index]
    this.setData({
      [`billDrafts[${index}].timeValue`]: detail.value,
      [`billDrafts[${index}].occurredAt`]: pickerValuesToISO(draft.dateValue, detail.value),
    })
  },

  async saveBillDrafts() {
    if (this.data.saving) return
    if (!await requireWriteAccess(MODULES.BILL)) return
    const drafts = this.data.billDrafts
    if (!drafts.length) return
    for (const draft of drafts) {
      if (!draft.title.trim() || !(Number(draft.amount) > 0)) return wx.showToast({ title: '请检查标题和金额', icon: 'none' })
    }
    this.setData({ saving: true })
    try {
      const inputs = drafts.map((draft) => ({
        type: draft.type, title: draft.title.trim(), amount: formatMoney(draft.amount), category: draft.category,
        occurredAt: draft.occurredAt, note: draft.note.trim(), inputType: draft.inputType, rawText: draft.rawText,
      }))
      if (this.data.editingBillId) {
        await BillService.update(this.data.editingBillId, inputs[0])
        wx.showToast({ title: '账单已更新' })
      } else {
        for (const input of inputs) await BillService.create(input)
        wx.showToast({ title: inputs.length > 1 ? `已记下 ${inputs.length} 笔` : '账单已记下' })
      }
      this.setData({ quickText: '', showQuickDraft: false, billDrafts: [], editingBillId: '', rawDraftText: '' })
      this.setTabBarHidden(false)
      try {
        await this.loadBills()
      } catch (refreshError) {
        wx.showModal({
          title: '账单已保存',
          content: '列表刷新失败，请切换页面后重试。',
          showCancel: false,
        })
      }
    } catch (error) {
      wx.showModal({
        title: '账单保存失败',
        content: error.message || '请稍后再试，当前确认内容仍然保留。',
        showCancel: false,
      })
    } finally {
      this.setData({ saving: false })
    }
  },

  discardQuickDraft() {
    this.setData({ showQuickDraft: false, quickDraftIntent: '', activityDraft: null, billDrafts: [], editingBillId: '', rawDraftText: '' })
    this.setTabBarHidden(false)
  },

  editBill({ currentTarget }) {
    const id = currentTarget.dataset.id
    const bills = this.data.billGroups.reduce((items, group) => items.concat(group.items), [])
    const bill = bills.find((item) => item.id === id)
    if (!bill?.canEdit) return
    this.setData({
      showQuickDraft: true, quickDraftIntent: 'bill',
      billDrafts: [normalizeBillDraft(bill, bill.rawText, bill.inputType || 'manual')],
      editingBillId: id, rawDraftText: bill.rawText || '',
    })
    this.setTabBarHidden(true)
  },

  async deleteBill({ currentTarget }) {
    const bills = this.data.billGroups.reduce((items, group) => items.concat(group.items), [])
    if (!bills.find((item) => item.id === currentTarget.dataset.id)?.canEdit) return
    if (!await requireWriteAccess(MODULES.BILL)) return
    const id = currentTarget.dataset.id
    wx.showModal({
      title: '删除账单', content: '确定删除这条账单吗？', confirmColor: '#bd5656',
      success: async ({ confirm }) => {
        if (!confirm) return
        await BillService.remove(id)
        wx.showToast({ title: '已删除' })
        this.loadBills()
      },
    })
  },

  noop() {},
  openLogDetail({ currentTarget }) {
    this.skipNextShowRefresh = true
    wx.navigateTo({
      url: `/pages/logs/detail?id=${currentTarget.dataset.id}`,
      fail: () => {
        this.skipNextShowRefresh = false
      },
    })
  },

  editLog({ currentTarget }) {
    const { id } = currentTarget.dataset
    const logs = this.data.groups.reduce((items, group) => items.concat(group.items), [])
    const log = logs.find((item) => item.id === id)
    if (!log?.canEdit) return
    this.setData({
      editingId: log.id, content: log.content, dateValue: toPickerDate(log.occurredAt),
      timeValue: toPickerTime(log.occurredAt), durationValue: log.durationMinutes ? String(log.durationMinutes) : '',
    })
    wx.pageScrollTo({ scrollTop: 0, duration: 240 })
  },

  cancelEdit() { this.setData(getInitialForm()) },

  async deleteLog({ currentTarget }) {
    const logs = this.data.groups.reduce((items, group) => items.concat(group.items), [])
    if (!logs.find((item) => item.id === currentTarget.dataset.id)?.canEdit) return
    if (!await requireWriteAccess(MODULES.ACTIVITY)) return
    const { id } = currentTarget.dataset
    wx.showModal({
      title: '删除记录', content: '确定删除这条生活记录吗？', confirmColor: '#bd5656',
      success: async ({ confirm }) => {
        if (!confirm) return
        await ActivityLogService.remove(id)
        if (this.data.editingId === id) this.setData(getInitialForm())
        wx.showToast({ title: '已删除' })
        this.loadLogs()
      },
    })
  },
}))
