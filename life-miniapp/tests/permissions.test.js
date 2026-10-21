const assert = require('node:assert/strict')
const test = require('node:test')

const {
  MODULES,
  getVisibleTabs,
  hasPermission,
  normalizePermissions,
} = require('../utils/permissions')
const { ProfileService } = require('../services/profile-service')

test('普通用户缺失的模块默认无权限', () => {
  const levels = normalizePermissions([
    { module: MODULES.RESTAURANT, level: 'read' },
    { module: MODULES.PLACE, level: 'write' },
  ], { id: '2', role: 'user' })

  assert.equal(hasPermission(levels, MODULES.TASK), false)
  assert.equal(hasPermission(levels, MODULES.RESTAURANT), true)
  assert.equal(hasPermission(levels, MODULES.RESTAURANT, 'write'), false)
  assert.equal(hasPermission(levels, MODULES.PLACE, 'write'), true)
})

test('底部导航只展示有读取权限的业务模块并始终保留主页', () => {
  const levels = normalizePermissions([
    { module: MODULES.RESTAURANT, level: 'read' },
    { module: MODULES.PLACE, level: 'write' },
  ], { id: '2', role: 'user' })

  assert.deepEqual(getVisibleTabs(levels).map((item) => item.text), ['吃喝', '地点', '主页'])
})

test('管理员始终拥有全部可写权限', () => {
  const levels = normalizePermissions([], { id: '1', role: 'admin' })
  assert.equal(Object.values(levels).every((level) => level === 'write'), true)
  assert.equal(getVisibleTabs(levels).length, 5)
})

test('审批列表兼容不同格式的用户 ID，并统一为字符串', async (t) => {
  const originalWx = global.wx
  let requestData
  global.wx = {
    getStorageSync: () => '',
    request: ({ data, success }) => {
      requestData = data
      success({
      statusCode: 200,
      data: {
        code: 0,
        data: {
          items: [{ user_id: 4, nickname: '生活体验官' }],
          page: 1,
          pageSize: 20,
          total: 1,
        },
      },
      })
    },
  }
  t.after(() => { global.wx = originalWx })

  const result = await ProfileService.listAccessRequests('approved', 1, 20)
  assert.equal(result.items[0].id, '4')
  assert.equal(requestData.page_size, 20)
  assert.equal(requestData.pageSize, undefined)
})

test('权限接口按照 Protobuf 协议发送 user_id', async (t) => {
  const originalWx = global.wx
  const requests = []
  global.wx = {
    getStorageSync: () => '',
    request: ({ url, data, success }) => {
      requests.push({ url, data })
      success({
        statusCode: 200,
        data: { code: 0, data: { user: { id: '4' }, permissions: [] } },
      })
    },
  }
  t.after(() => { global.wx = originalWx })

  await ProfileService.getUserPermissions(4)
  await ProfileService.modifyUserPermissions(4, [])
  await ProfileService.reviewAccessRequest(4, 'approved')

  assert.equal(requests.length, 3)
  requests.forEach(({ data }) => {
    assert.equal(data.user_id, '4')
    assert.equal(data.userId, undefined)
  })
})
