const STORAGE_KEY = 'life_tasks_v1'

const { request } = require('./request')
const { getTodayRange } = require('../utils/task-schedule')

// 任务模块已接入后端；保留本地实现便于必要时快速回退。
const REMOTE_TASK_ENABLED = true

const API_PATHS = {
  list: '/v1/life-core/task/list',
  create: '/v1/life-core/task/create',
  detail: '/v1/life-core/task/detail',
  update: '/v1/life-core/task/modify',
  status: '/v1/life-core/task/status/modify',
  remove: '/v1/life-core/task/delete',
  recognizeVoice: '/v1/life-core/task/voice/recognize',
}

function createSeedTasks() {
  const now = new Date()

  return [
    createSeedTask('整理本周计划', '梳理本周最重要的三件事，并安排执行时间。', 'high', '工作', 0, 18, now),
    createSeedTask('收藏一家想去的餐厅', '从餐厅列表里挑一家周末去体验。', 'medium', '生活', 2, 12, now),
    createSeedTask('记录一个周末地点', '整理照片和简单的游玩建议。', 'low', '兴趣', 4, 16, now, 'done'),
  ]
}

function createSeedTask(title, description, priority, tag, offsetDays, hour, now, status = 'todo') {
  const dueAt = new Date(now.getFullYear(), now.getMonth(), now.getDate() + offsetDays, hour, 0)
  return {
    id: `seed-${priority}`,
    title,
    description,
    status,
    priority,
    tags: [tag],
    creator: { id: 'me', name: '我' },
    dueAt: dueAt.toISOString(),
    createdAt: now.toISOString(),
    updatedAt: now.toISOString(),
  }
}

function clone(value) {
  return JSON.parse(JSON.stringify(value))
}

function loadLocalTasks() {
  const saved = wx.getStorageSync(STORAGE_KEY)
  if (Array.isArray(saved)) return saved

  const tasks = createSeedTasks()
  wx.setStorageSync(STORAGE_KEY, tasks)
  return tasks
}

function saveLocalTasks(tasks) {
  wx.setStorageSync(STORAGE_KEY, tasks)
}

function matchesStatus(task, status) {
  if (status === 'all') return true
  if (status === 'done') return task.status === 'done'
  if (status === 'todo') return task.status === 'todo'
  if (status === 'overdue') return task.status === 'todo' && new Date(task.dueAt).getTime() < Date.now()
  if (status === 'today') {
    const dueAt = new Date(task.dueAt).getTime()
    const range = getTodayRange()
    return dueAt >= new Date(range.dueFrom).getTime() && dueAt <= new Date(range.dueTo).getTime()
  }
  return true
}

function listLocal(filters = {}) {
  const keyword = (filters.keyword || '').trim().toLowerCase()
  return loadLocalTasks()
    .filter((task) => matchesStatus(task, filters.status || 'all'))
    .filter((task) => !filters.priority || filters.priority === 'all' || task.priority === filters.priority)
    .filter((task) => !filters.createdBy || String(task.creator?.id || task.createdBy || '') === String(filters.createdBy))
    .filter((task) => {
      if (!keyword) return true
      return [task.title, task.description, ...(task.tags || [])].join(' ').toLowerCase().includes(keyword)
    })
    .sort((left, right) => {
      if (left.status !== right.status) return left.status === 'done' ? 1 : -1
      return new Date(left.dueAt).getTime() - new Date(right.dueAt).getTime()
    })
}

function normalizeRemoteTask(task) {
  const creator = task.creator || {}
  return {
    ...task,
    tags: task.category ? [task.category] : [],
    creator: {
      id: creator.id || task.createdBy || '',
      name: creator.nickname || creator.name || '未知用户',
      avatarEmoji: creator.avatarEmoji || '',
      avatarColor: creator.avatarColor || '#789b8b',
    },
  }
}

function listRemote(filters = {}) {
  const dateRange = filters.status === 'today' ? getTodayRange() : {}
  const status = filters.status === 'overdue' ? 'todo' : filters.status === 'today' ? '' : filters.status
  const data = {
    keyword: (filters.keyword || '').trim(),
    status: status && status !== 'all' ? status : undefined,
    priority: filters.priority && filters.priority !== 'all' ? filters.priority : undefined,
    category: filters.category && filters.category !== 'all' ? filters.category : undefined,
    createdBy: filters.createdBy || undefined,
    dueFrom: filters.dueFrom || dateRange.dueFrom || undefined,
    dueTo: filters.dueTo || dateRange.dueTo || undefined,
    page: filters.page || 1,
    pageSize: filters.pageSize || 20,
  }

  return request({ path: API_PATHS.list, method: 'POST', data }).then((result) => {
    const items = Array.isArray(result?.items) ? result.items.map(normalizeRemoteTask) : []
    const filteredItems = filters.status === 'overdue'
      ? items.filter((task) => task.dueAt && new Date(task.dueAt).getTime() < Date.now())
      : items
    const hasMore = result?.has_more ?? result?.hasMore
    return {
      items: filteredItems,
      page: Number(result?.page) || data.page,
      pageSize: Number(result?.pageSize) || data.pageSize,
      total: Number(result?.total) || 0,
      hasMore: Boolean(hasMore),
      overallTotal: Number(result?.overallTotal) || 0,
      todoCount: Number(result?.todoCount) || 0,
      doneCount: Number(result?.doneCount) || 0,
    }
  })
}

function toRemoteInput(input) {
  return {
    title: input.title,
    description: input.description || '',
    priority: input.priority,
    category: input.tags?.[0] || '其他',
    dueAt: input.dueAt || '',
  }
}

const TaskService = {
  list(filters) {
    if (REMOTE_TASK_ENABLED) return listRemote(filters).then((result) => result.items)
    return Promise.resolve(clone(listLocal(filters)))
  },

  listPage(filters) {
    if (REMOTE_TASK_ENABLED) return listRemote(filters)
    const items = clone(listLocal(filters))
    return Promise.resolve({ items, page: 1, pageSize: items.length, total: items.length, hasMore: false })
  },

  get(id) {
    if (REMOTE_TASK_ENABLED) {
      return request({ path: API_PATHS.detail, method: 'POST', data: { id } }).then(normalizeRemoteTask)
    }
    const task = loadLocalTasks().find((item) => item.id === id)
    return Promise.resolve(task ? clone(task) : null)
  },

  create(input) {
    if (REMOTE_TASK_ENABLED) {
      return request({
        path: API_PATHS.create,
        method: 'POST',
        data: toRemoteInput(input),
      }).then(normalizeRemoteTask)
    }
    const now = new Date().toISOString()
    const task = {
      ...input,
      id: `task-${Date.now()}`,
      status: 'todo',
      creator: { id: 'me', name: '我' },
      createdAt: now,
      updatedAt: now,
    }
    saveLocalTasks([task, ...loadLocalTasks()])
    return Promise.resolve(clone(task))
  },

  update(id, input) {
    if (REMOTE_TASK_ENABLED) {
      return request({
        path: API_PATHS.update,
        method: 'POST',
        data: { id, ...toRemoteInput(input) },
      }).then(normalizeRemoteTask)
    }
    let updatedTask = null
    const tasks = loadLocalTasks().map((task) => {
      if (task.id !== id) return task
      updatedTask = { ...task, ...input, updatedAt: new Date().toISOString() }
      return updatedTask
    })
    saveLocalTasks(tasks)
    return Promise.resolve(updatedTask ? clone(updatedTask) : null)
  },

  remove(id) {
    if (REMOTE_TASK_ENABLED) {
      return request({ path: API_PATHS.remove, method: 'POST', data: { id } })
    }
    saveLocalTasks(loadLocalTasks().filter((task) => task.id !== id))
    return Promise.resolve()
  },

  toggle(id, currentStatus) {
    return this.setStatus(id, currentStatus === 'done' ? 'todo' : 'done')
  },

  setStatus(id, status) {
    if (REMOTE_TASK_ENABLED) {
      return request({
        path: API_PATHS.status,
        method: 'POST',
        data: { id, status },
      }).then(normalizeRemoteTask)
    }
    const task = loadLocalTasks().find((item) => item.id === id)
    if (!task) return Promise.resolve(null)
    return this.update(id, { status })
  },

  reschedule(id, dueAt, currentTask = null) {
    const loadTask = currentTask ? Promise.resolve(currentTask) : this.get(id)
    return loadTask.then((task) => this.update(id, {
      title: task.title,
      description: task.description,
      priority: task.priority,
      tags: task.tags,
      dueAt,
    }))
  },

  recognizeVoice(input) {
    return request({
      path: API_PATHS.recognizeVoice,
      method: 'POST',
      data: input,
    })
  },

  all() {
    if (REMOTE_TASK_ENABLED) return listRemote({ page: 1, pageSize: 20 }).then((result) => result.items)
    return Promise.resolve(clone(loadLocalTasks()))
  },
}

module.exports = { API_PATHS, TaskService }
