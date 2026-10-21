const { ActivityLogService } = require('../../services/activity-log-service')
const { withScrollRestoration } = require('../../utils/page-scroll')
const { AISummaryService } = require('../../services/ai-summary-service')
const { requireApprovedAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')

function pad(value) {
  return String(value).padStart(2, '0')
}

function formatDuration(minutes, emptyText = '0分钟') {
  if (!minutes) return emptyText
  if (minutes < 60) return `${minutes}分钟`
  const hours = Math.floor(minutes / 60)
  const rest = minutes % 60
  return rest ? `${hours}小时${rest}分` : `${hours}小时`
}

function initialPeriod() {
  const now = new Date()
  return {
    selectedMonth: `${now.getFullYear()}-${pad(now.getMonth() + 1)}`,
    selectedYear: String(now.getFullYear()),
  }
}

Page(withScrollRestoration({
  data: {
    loading: true,
    range: 'month',
    rangeOptions: [
      { value: 'month', label: '月度' },
      { value: 'year', label: '年度' },
    ],
    ...initialPeriod(),
    periodLabel: '',
    stats: null,
    chartItems: [],
    hasRecords: false,
    summary: '',
    aiSummary: null,
    aiLoading: false,
  },

  async onShow() {
    const user = await requireApprovedAccess('', MODULES.ACTIVITY)
    if (!user) return
    await this.loadReport()
  },

  selectRange({ currentTarget }) {
    const range = currentTarget.dataset.value
    if (range === this.data.range) return
    this.setData({ range, aiSummary: null }, () => this.loadReport())
  },

  onMonthChange({ detail }) {
    this.setData({ selectedMonth: detail.value, aiSummary: null }, () => this.loadReport())
  },

  onYearChange({ detail }) {
    this.setData({ selectedYear: detail.value, aiSummary: null }, () => this.loadReport())
  },

  async loadReport() {
    this.setData({ loading: true })
    const isMonth = this.data.range === 'month'
    const [monthYear, month] = this.data.selectedMonth.split('-').map(Number)
    const year = isMonth ? monthYear : Number(this.data.selectedYear)
    const stats = await ActivityLogService.stats({
      scope: this.data.range,
      year,
      month: isMonth ? month : undefined,
    })
    const sourceItems = isMonth ? stats.dailyBreakdown : stats.monthlyBreakdown
    const maxValue = Math.max(1, ...sourceItems.map((item) => item.recordCount))
    const chartItems = sourceItems.map((item) => {
      const value = item.recordCount
      return {
        ...item,
        barPercent: Math.max(value ? 8 : 0, Math.round((value / maxValue) * 100)),
        valueText: item.minutes
          ? `${formatDuration(item.minutes)} · ${item.recordCount}条`
          : `${item.recordCount}条`,
      }
    })
    const periodLabel = isMonth ? `${year}年${month}月` : `${year}年`
    const hasRecords = stats.recordCount > 0
    this.setData({
      loading: false,
      periodLabel,
      hasRecords,
      chartItems,
      stats: {
        ...stats,
        totalDurationText: formatDuration(stats.totalMinutes),
        averageDurationText: formatDuration(stats.averageMinutes, '暂无'),
        longestDurationText: formatDuration(stats.longestMinutes, '暂无'),
      },
      summary: hasRecords
        ? `${periodLabel}记录了 ${stats.activeDays} 天，共留下 ${stats.recordCount} 条生活片段。`
        : `${periodLabel}还没有生活记录。`,
    })
  },

  async generateAISummary(event) {
    if (this.data.aiLoading) return
    const isMonth = this.data.range === 'month'
    const [monthYear, month] = this.data.selectedMonth.split('-').map(Number)
    const year = isMonth ? monthYear : Number(this.data.selectedYear)
    const refresh = Boolean(event && event.currentTarget && event.currentTarget.dataset.refresh)
    this.setData({ aiLoading: true })
    try {
      const summary = await AISummaryService.lifeSummary({
        scope: this.data.range,
        year,
        month: isMonth ? month : undefined,
        refresh,
      })
      this.setData({ aiSummary: summary })
      if (!summary.aiGenerated) {
        wx.showToast({ title: 'AI 暂时忙，已生成基础回顾', icon: 'none' })
      }
    } catch (error) {
      wx.showToast({ title: error.message || '生活总结生成失败', icon: 'none' })
    } finally {
      this.setData({ aiLoading: false })
    }
  },
}))
