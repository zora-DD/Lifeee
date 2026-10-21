const { TaskService } = require('../../services/task-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const {
  getWeekdayLabel,
  pickerValuesToISO,
  toPickerDate,
  toPickerTime,
} = require('../../utils/date')
const { requireWriteAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')

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
        if (authSetting['scope.record']) {
          resolve()
          return
        }
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

Page(withScrollRestoration({
  data: {
    id: '',
    isEdit: false,
    title: '',
    description: '',
    priority: 'medium',
    priorityOptions: [
      { value: 'high', label: '高' },
      { value: 'medium', label: '中' },
      { value: 'low', label: '低' },
    ],
    categoryIndex: 0,
    categories: ['工作', '生活', '兴趣', '其他'],
    dueDate: '',
    dueWeekday: '',
    dueTime: '18:00',
    saving: false,
    recording: false,
    recognizing: false,
    recordingSeconds: 0,
    showVoiceDraft: false,
    voiceDraft: null,
  },

  async onLoad({ id, date }) {
    this.recorderManager = wx.getRecorderManager()
    this.handleRecorderStart = this.onRecorderStart.bind(this)
    this.handleRecorderStop = this.onRecorderStop.bind(this)
    this.handleRecorderError = this.onRecorderError.bind(this)
    this.recorderManager.onStart(this.handleRecorderStart)
    this.recorderManager.onStop(this.handleRecorderStop)
    this.recorderManager.onError(this.handleRecorderError)

    const user = await requireWriteAccess(MODULES.TASK)
    if (!user) return
    wx.setNavigationBarTitle({ title: id ? '编辑任务' : '创建任务' })
    const defaultDueDate = !id && /^\d{4}-\d{2}-\d{2}$/.test(date || '')
      ? date
      : toPickerDate()
    this.setData({
      id: id || '',
      isEdit: Boolean(id),
      dueDate: defaultDueDate,
      dueWeekday: getWeekdayLabel(defaultDueDate),
      dueTime: '18:00',
    })

    if (!id) return

    const task = await TaskService.get(id)
    if (!task) return
    if (user.role !== 'admin' && String(task.creator?.id || '') !== String(user.id || '')) {
      wx.showToast({ title: '只能编辑自己创建的任务', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 400)
      return
    }
    const categoryIndex = Math.max(0, this.data.categories.indexOf(task.tags?.[0]))
    const dueDate = toPickerDate(task.dueAt)
    this.setData({
      title: task.title,
      description: task.description,
      priority: task.priority,
      categoryIndex,
      dueDate,
      dueWeekday: getWeekdayLabel(dueDate),
      dueTime: toPickerTime(task.dueAt),
    })
  },

  onUnload() {
    this.discardingRecording = true
    this.stopRecordingTimer()
    if (!this.recorderManager) return
    if (this.data.recording) this.recorderManager.stop()
    if (typeof this.recorderManager.offStart === 'function') this.recorderManager.offStart(this.handleRecorderStart)
    if (typeof this.recorderManager.offStop === 'function') this.recorderManager.offStop(this.handleRecorderStop)
    if (typeof this.recorderManager.offError === 'function') this.recorderManager.offError(this.handleRecorderError)
  },

  onTitleInput({ detail }) {
    this.setData({ title: detail.value })
  },

  onDescriptionInput({ detail }) {
    this.setData({ description: detail.value })
  },

  selectPriority({ currentTarget }) {
    this.setData({ priority: currentTarget.dataset.value })
  },

  onCategoryChange({ detail }) {
    this.setData({ categoryIndex: Number(detail.value) })
  },

  onDateChange({ detail }) {
    this.setData({
      dueDate: detail.value,
      dueWeekday: getWeekdayLabel(detail.value),
    })
  },

  onTimeChange({ detail }) {
    this.setData({ dueTime: detail.value })
  },

  async startVoiceRecording() {
    if (this.data.recording || this.data.recognizing || this.data.isEdit) return
    if (!await requireWriteAccess(MODULES.TASK)) return
    try {
      await ensureRecordPermission()
      this.discardingRecording = false
      this.beginRecordingUI()
      this.recorderManager.start({
        duration: 60000,
        sampleRate: 16000,
        numberOfChannels: 1,
        encodeBitRate: 48000,
        format: 'mp3',
      })
    } catch (error) {
      this.resetRecordingUI()
      wx.showModal({
        title: '需要麦克风权限',
        content: error.message || '请在设置中允许使用麦克风。',
        confirmText: '去设置',
        success: ({ confirm }) => {
          if (confirm) wx.openSetting()
        },
      })
    }
  },

  beginRecordingUI() {
    this.stoppingRecording = false
    this.recordStartedAt = Date.now()
    this.setData({ recording: true, recognizing: false, recordingSeconds: 0 })
    this.stopRecordingTimer()
    this.recordingTimer = setInterval(() => {
      const seconds = Math.min(60, Math.floor((Date.now() - this.recordStartedAt) / 1000))
      this.setData({ recordingSeconds: seconds })
    }, 500)
  },

  resetRecordingUI() {
    this.stopRecordingTimer()
    this.setData({ recording: false, recognizing: false, recordingSeconds: 0 })
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
      return
    }
    if (!tempFilePath || Number(duration) < 800) {
      this.setData({ recording: false, recognizing: false, recordingSeconds: 0 })
      wx.showToast({ title: '录音太短，再说一次吧', icon: 'none' })
      return
    }

    this.setData({ recording: false, recognizing: true, recordingSeconds: 0 })
    try {
      const audioBase64 = await readFileAsBase64(tempFilePath)
      const result = await TaskService.recognizeVoice({
        audioBase64,
        voiceFormat: 'mp3',
      })
      this.setData({
        showVoiceDraft: true,
        voiceDraft: {
          rawText: result.rawText || '',
          title: result.title || '',
          description: result.description || '',
          category: result.category || '其他',
        },
      })
    } catch (error) {
      wx.showModal({
        title: '语音整理失败',
        content: error.message || '请稍后再试，也可以直接输入任务。',
        showCancel: false,
      })
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
    this.recorderManager.stop()
  },

  stopRecordingTimer() {
    if (!this.recordingTimer) return
    clearInterval(this.recordingTimer)
    this.recordingTimer = null
  },

  confirmVoiceDraft() {
    const draft = this.data.voiceDraft
    if (!draft) return
    const categoryIndex = Math.max(0, this.data.categories.indexOf(draft.category))
    this.setData({
      title: draft.title,
      description: draft.description,
      categoryIndex,
      showVoiceDraft: false,
      voiceDraft: null,
    })
    wx.showToast({ title: '已填入，请确认' })
  },

  discardVoiceDraft() {
    this.setData({ showVoiceDraft: false, voiceDraft: null })
  },

  noop() {},

  async save() {
    const title = this.data.title.trim()
    if (!title) {
      wx.showToast({ title: '请输入任务名称', icon: 'none' })
      return
    }

    this.setData({ saving: true })
    const input = {
      title,
      description: this.data.description.trim(),
      priority: this.data.priority,
      tags: [this.data.categories[this.data.categoryIndex]],
      dueAt: pickerValuesToISO(this.data.dueDate, this.data.dueTime),
    }

    try {
      let savedTask
      if (this.data.isEdit) {
        savedTask = await TaskService.update(this.data.id, input)
      } else {
        savedTask = await TaskService.create(input)
      }
      if (typeof this.getOpenerEventChannel === 'function') {
        const eventChannel = this.getOpenerEventChannel()
        if (eventChannel && typeof eventChannel.emit === 'function') {
          eventChannel.emit('taskChanged', { task: savedTask })
        }
      }
      wx.showToast({ title: this.data.isEdit ? '修改成功' : '创建成功' })
      setTimeout(() => wx.navigateBack(), 500)
    } catch (error) {
      this.setData({ saving: false })
      wx.showToast({ title: error.message || '保存失败', icon: 'none' })
    }
  },
}))
