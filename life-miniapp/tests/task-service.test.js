const assert = require('node:assert/strict')
const test = require('node:test')

test('任务详情展示后端返回的创建人昵称和头像', async () => {
  global.wx = {
    getStorageSync: (key) => key === 'life_access_token' ? 'test-token' : '',
    setStorageSync: () => {},
    removeStorageSync: () => {},
    request: (options) => options.success({
      statusCode: 200,
      data: {
        code: 0,
        data: {
          id: '7',
          title: '整理房间',
          createdBy: '1',
          creator: {
            id: '1',
            nickname: '晚风收藏家',
            avatarEmoji: '🌿',
            avatarColor: '#759B89',
          },
        },
      },
    }),
  }

  const { TaskService } = require('../services/task-service')
  const task = await TaskService.get('7')

  assert.equal(task.creator.name, '晚风收藏家')
  assert.equal(task.creator.avatarEmoji, '🌿')
  assert.equal(task.creator.avatarColor, '#759B89')
  assert.notEqual(task.creator.name, '用户 1')
})

test('任务列表支持今日和我创建的筛选', async (t) => {
  const originalWx = global.wx
  let requestData
  global.wx = {
    getStorageSync: () => 'test-token',
    request: (options) => {
      requestData = options.data
      options.success({
        statusCode: 200,
        data: { code: 0, data: { items: [], page: 1, pageSize: 20, total: 0, has_more: false } },
      })
    },
  }
  t.after(() => { global.wx = originalWx })
  delete require.cache[require.resolve('../services/task-service')]
  delete require.cache[require.resolve('../services/request')]
  const { TaskService } = require('../services/task-service')

  await TaskService.listPage({ status: 'today', createdBy: '7' })

  assert.equal(requestData.status, undefined)
  assert.equal(requestData.createdBy, '7')
  assert.match(requestData.dueFrom, /T00:00:00\.000/)
  assert.match(requestData.dueTo, /T23:59:59\.999/)
})

test('快速延期复用任务修改接口', async (t) => {
  const originalWx = global.wx
  const requests = []
  global.wx = {
    getStorageSync: () => 'test-token',
    request: (options) => {
      requests.push(options)
      options.success({ statusCode: 200, data: { code: 0, data: { id: '7', ...options.data } } })
    },
  }
  t.after(() => { global.wx = originalWx })
  delete require.cache[require.resolve('../services/task-service')]
  delete require.cache[require.resolve('../services/request')]
  const { TaskService } = require('../services/task-service')
  const task = {
    id: '7', title: '整理计划', description: '', priority: 'medium', tags: ['工作'],
  }

  await TaskService.reschedule('7', '2026-09-19T18:00:00+08:00', task)

  assert.match(requests[0].url, /\/v1\/task\/modify$/)
  assert.equal(requests[0].data.dueAt, '2026-09-19T18:00:00+08:00')
})
