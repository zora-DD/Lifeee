const STORAGE_KEY = 'life_activity_logs_v1'

const { request } = require('./request')

// 生活记录已接入后端，本地实现仅作开发回退。
const REMOTE_ACTIVITY_LOG_ENABLED = true

const API_PATHS = {
  list: '/v1/life-core/activity_log/list',
  detail: '/v1/life-core/activity_log/detail',
  create: '/v1/life-core/activity_log/create',
  stats: '/v1/life-core/activity_log/stats',
  update: '/v1/life-core/activity_log/modify',
  remove: '/v1/life-core/activity_log/delete',
  recognizeVoice: '/v1/life-core/activity_log/voice/recognize',
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function createSeedLogs() {
  const now = new Date()
  const occurredAt = new Date(now.getFullYear(), now.getMonth(), now.getDate(), 8, 30)
  const createdAt = now.toISOString()

  return [{
    id: 'seed-daily-log',
    content: '收拾客厅，洗澡，收叠衣服，倒掉客厅和卫生间垃圾，叠被子，洗水果。',
    occurredAt: occurredAt.toISOString(),
    durationMinutes: 45,
    inputType: 'text',
    creator: { id: 'me', name: '我' },
    createdAt,
    updatedAt: createdAt,
  }]
}

function loadLocalLogs() {
  const saved = wx.getStorageSync(STORAGE_KEY)
  if (Array.isArray(saved)) return saved

  const logs = createSeedLogs()
  wx.setStorageSync(STORAGE_KEY, logs)
  return logs
}

function saveLocalLogs(logs) {
  wx.setStorageSync(STORAGE_KEY, logs)
}

function buildLocalStats(logs, { scope = 'month', year, month }) {
  const filtered = logs.filter((log) => {
    const date = new Date(log.occurredAt)
    if (date.getFullYear() !== Number(year)) return false
    return scope === 'year' || date.getMonth() + 1 === Number(month)
  })
  const activeDates = new Set()
  const dailyMap = {}
  const monthlyMap = {}
  let totalMinutes = 0
  let durationRecordCount = 0
  let longestMinutes = 0

  filtered.forEach((log) => {
    const date = new Date(log.occurredAt)
    const dateKey = [date.getFullYear(), date.getMonth() + 1, date.getDate()].join('-')
    const monthKey = date.getMonth() + 1
    const minutes = Number(log.durationMinutes) || 0
    activeDates.add(dateKey)
    if (!dailyMap[dateKey]) {
      dailyMap[dateKey] = {
        key: dateKey,
        label: `${date.getMonth() + 1}/${date.getDate()}`,
        recordCount: 0,
        minutes: 0,
        timestamp: new Date(date.getFullYear(), date.getMonth(), date.getDate()).getTime(),
      }
    }
    if (!monthlyMap[monthKey]) monthlyMap[monthKey] = { recordCount: 0, minutes: 0 }
    dailyMap[dateKey].recordCount += 1
    dailyMap[dateKey].minutes += minutes
    monthlyMap[monthKey].recordCount += 1
    monthlyMap[monthKey].minutes += minutes
    if (minutes > 0) {
      totalMinutes += minutes
      durationRecordCount += 1
      longestMinutes = Math.max(longestMinutes, minutes)
    }
  })

  return {
    recordCount: filtered.length,
    activeDays: activeDates.size,
    totalMinutes,
    durationRecordCount,
    averageMinutes: durationRecordCount ? Math.round(totalMinutes / durationRecordCount) : 0,
    longestMinutes,
    dailyBreakdown: Object.values(dailyMap).sort((left, right) => left.timestamp - right.timestamp),
    monthlyBreakdown: Array.from({ length: 12 }, (_, index) => ({
      month: index + 1,
      label: `${index + 1}月`,
      recordCount: monthlyMap[index + 1] ? monthlyMap[index + 1].recordCount : 0,
      minutes: monthlyMap[index + 1] ? monthlyMap[index + 1].minutes : 0,
    })),
  }
}

function normalizeRemoteLog(log) {
  const creator = log && log.creator ? log.creator : {}
  return {
    ...log,
    creator: {
      id: creator.id || log.createdBy || '',
      name: creator.nickname || creator.name || '未知用户',
      avatarEmoji: creator.avatarEmoji || '',
      avatarColor: creator.avatarColor || '#789b8b',
    },
  }
}

const ActivityLogService = {
  list(filters = {}) {
    if (REMOTE_ACTIVITY_LOG_ENABLED) {
      return request({
        path: API_PATHS.list,
        method: 'POST',
        data: {
          from: filters.from,
          to: filters.to,
          inputType: filters.inputType,
          page: filters.page || 1,
          pageSize: filters.pageSize || 20,
        },
      }).then((result) => {
        const items = Array.isArray(result?.items) ? result.items.map(normalizeRemoteLog) : []
        return {
          items,
          page: Number(result?.page) || 1,
          pageSize: Number(result?.pageSize) || filters.pageSize || 20,
          total: Number(result?.total) || 0,
          hasMore: Boolean(result?.has_more ?? result?.hasMore),
        }
      })
    }
    const logs = loadLocalLogs()
      .slice()
      .sort((left, right) => new Date(right.occurredAt) - new Date(left.occurredAt))
    return Promise.resolve({
      items: clone(logs),
      page: 1,
      pageSize: logs.length,
      total: logs.length,
      hasMore: false,
    })
  },

  get(id) {
    if (REMOTE_ACTIVITY_LOG_ENABLED) {
      return request({ path: API_PATHS.detail, method: 'POST', data: { id } }).then(normalizeRemoteLog)
    }
    const log = loadLocalLogs().find((item) => item.id === id)
    return Promise.resolve(log ? clone(log) : null)
  },

  stats(params) {
    if (REMOTE_ACTIVITY_LOG_ENABLED) {
      return request({ path: API_PATHS.stats, method: 'POST', data: params })
    }
    return Promise.resolve(clone(buildLocalStats(loadLocalLogs(), params)))
  },

  create(input) {
    if (REMOTE_ACTIVITY_LOG_ENABLED) {
      return request({ path: API_PATHS.create, method: 'POST', data: input }).then(normalizeRemoteLog)
    }

    const now = new Date().toISOString()
    const log = {
      ...input,
      id: `activity-log-${Date.now()}`,
      inputType: input.inputType || 'text',
      creator: { id: 'me', name: '我' },
      createdAt: now,
      updatedAt: now,
    }
    saveLocalLogs([log, ...loadLocalLogs()])
    return Promise.resolve(clone(log))
  },

  update(id, input) {
    if (REMOTE_ACTIVITY_LOG_ENABLED) {
      return request({ path: API_PATHS.update, method: 'POST', data: { id, ...input } }).then(normalizeRemoteLog)
    }

    let updated = null
    const logs = loadLocalLogs().map((log) => {
      if (log.id !== id) return log
      updated = { ...log, ...input, updatedAt: new Date().toISOString() }
      return updated
    })
    saveLocalLogs(logs)
    return Promise.resolve(updated ? clone(updated) : null)
  },

  remove(id) {
    if (REMOTE_ACTIVITY_LOG_ENABLED) {
      return request({ path: API_PATHS.remove, method: 'POST', data: { id } })
    }
    saveLocalLogs(loadLocalLogs().filter((log) => log.id !== id))
    return Promise.resolve()
  },

  recognizeVoice(input) {
    return request({
      path: API_PATHS.recognizeVoice,
      method: 'POST',
      data: input,
    })
  },
}

module.exports = { API_PATHS, ActivityLogService }
