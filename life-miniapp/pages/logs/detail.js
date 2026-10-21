const { ActivityLogService } = require('../../services/activity-log-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { requireApprovedAccess } = require('../../utils/access')
const { formatFullDateTime } = require('../../utils/date')
const { buildShareCard } = require('../../utils/share')
const { MODULES } = require('../../utils/permissions')

function formatDuration(minutes) {
  const value = Number(minutes) || 0
  if (!value) return '未记录时长'
  if (value < 60) return `${value} 分钟`
  const hours = Math.floor(value / 60)
  const rest = value % 60
  return rest ? `${hours} 小时 ${rest} 分钟` : `${hours} 小时`
}

function shareSummary(content) {
  const text = String(content || '').replace(/\s+/g, ' ').trim()
  return text.length > 24 ? `${text.slice(0, 24)}…` : text
}

Page(withScrollRestoration({
  data: {
    id: '',
    log: null,
    loadError: '',
  },

  onLoad({ id }) {
    this.setData({ id: id || '' })
  },

  async onShow() {
    this.setData({ log: null, loadError: '' })
    const returnPath = `/pages/logs/detail?id=${encodeURIComponent(this.data.id)}`
    if (!await requireApprovedAccess(returnPath, MODULES.ACTIVITY)) return
    await this.loadLog()
  },

  async loadLog() {
    if (!this.data.id) {
      this.setData({ loadError: '记录参数不完整' })
      return
    }
    try {
      const log = await ActivityLogService.get(this.data.id)
      if (!log) {
        this.setData({ loadError: '这条记录不存在或已删除' })
        return
      }
      this.setData({
        log: {
          ...log,
          occurredText: formatFullDateTime(log.occurredAt),
          createdText: formatFullDateTime(log.createdAt),
          durationText: formatDuration(log.durationMinutes),
          creatorName: log.creator && log.creator.name ? log.creator.name : '未知用户',
        },
      })
    } catch (error) {
      this.setData({ loadError: error.message || '记录加载失败' })
    }
  },

  onShareAppMessage() {
    const log = this.data.log
    return buildShareCard({
      title: log ? `生活记录｜${shareSummary(log.content)}` : '',
      path: `/pages/logs/detail?id=${encodeURIComponent(this.data.id)}`,
    })
  },

  goBack() {
    if (getCurrentPages().length > 1) {
      wx.navigateBack()
      return
    }
    wx.switchTab({ url: '/pages/logs/index' })
  },
}))
