const assert = require('node:assert/strict')
const test = require('node:test')

const REQUEST_MODULE = require.resolve('../services/request')
const AUTH_SESSION_MODULE = require.resolve('../services/auth-session')

function loadRequestWithMock(requestHandler, initialStorage = {}) {
  const storage = new Map(Object.entries(initialStorage))
  const calls = []
  let loginCount = 0

  global.wx = {
    getStorageSync: (key) => storage.get(key),
    setStorageSync: (key, value) => storage.set(key, value),
    removeStorageSync: (key) => storage.delete(key),
    login: ({ success }) => {
      loginCount += 1
      success({ code: 'mock-wx-code' })
    },
    request: (options) => {
      calls.push(options)
      requestHandler(options)
    },
  }

  delete require.cache[REQUEST_MODULE]
  delete require.cache[AUTH_SESSION_MODULE]
  const { request } = require('../services/request')
  return { calls, getLoginCount: () => loginCount, request, storage }
}

test('有效 Token 会放入 Authorization，且不发送空字段', async () => {
  const harness = loadRequestWithMock((options) => {
    options.success({ statusCode: 200, data: { code: 0, data: { items: [] } } })
  }, { access_token: 'valid-token' })

  const result = await harness.request({
    path: '/v1/life-core/task/list', method: 'POST', data: { page: 1, keyword: undefined, category: null },
  })

  assert.deepEqual(result, { items: [] })
  assert.equal(harness.calls[0].header.Authorization, 'Bearer valid-token')
  assert.deepEqual(harness.calls[0].data, { page: 1 })
})

test('接口返回 401 时静默登录并使用新 Token 重试一次', async () => {
  let businessCalls = 0
  const harness = loadRequestWithMock((options) => {
    if (options.url.endsWith('/v1/auth/login')) {
      options.success({
        statusCode: 200,
        data: { code: 0, data: { access_token: 'new-token', expires_at: '2099-01-01T00:00:00+08:00' } },
      })
      return
    }
    businessCalls += 1
    if (businessCalls === 1) {
      options.success({ statusCode: 401, data: { code: 1204, msg: 'unauthorized' } })
      return
    }
    options.success({ statusCode: 200, data: { code: 0, data: { id: '1' } } })
  }, { access_token: 'expired-token' })

  const result = await harness.request({ path: '/v1/life-core/task/detail', method: 'POST', data: { id: '1' } })

  assert.deepEqual(result, { id: '1' })
  assert.equal(harness.getLoginCount(), 1)
  assert.equal(businessCalls, 2)
  assert.equal(harness.calls.at(-1).header.Authorization, 'Bearer new-token')
  assert.equal(harness.storage.get('access_token'), 'new-token')
})

test('接口返回 403 时不触发重新登录且保留会话', async () => {
  const harness = loadRequestWithMock((options) => {
    options.success({ statusCode: 403, data: { code: 1205, msg: 'forbidden' } })
  }, { access_token: 'approved-format-token' })

  await assert.rejects(
    harness.request({ path: '/v1/life-core/task/list', method: 'POST' }),
    (error) => error.statusCode === 403,
  )
  assert.equal(harness.getLoginCount(), 0)
  assert.equal(harness.storage.get('access_token'), 'approved-format-token')
})

test('并发请求同时收到 401 时只发起一次静默登录', async () => {
  let initialCalls = 0
  let retryCalls = 0
  const harness = loadRequestWithMock((options) => {
    if (options.url.endsWith('/v1/auth/login')) {
      setImmediate(() => options.success({
        statusCode: 200,
        data: { code: 0, data: { access_token: 'shared-new-token', expires_at: '2099-01-01T00:00:00+08:00' } },
      }))
      return
    }
    if (options.header.Authorization === 'Bearer old-token') {
      initialCalls += 1
      setImmediate(() => options.success({ statusCode: 401, data: { code: 1204, msg: 'unauthorized' } }))
      return
    }
    retryCalls += 1
    setImmediate(() => options.success({ statusCode: 200, data: { code: 0, data: { ok: true } } }))
  }, { access_token: 'old-token' })

  const [first, second] = await Promise.all([
    harness.request({ path: '/v1/life-core/task/list', method: 'POST' }),
    harness.request({ path: '/v1/life-core/place/list', method: 'POST' }),
  ])

  assert.deepEqual(first, { ok: true })
  assert.deepEqual(second, { ok: true })
  assert.equal(initialCalls, 2)
  assert.equal(retryCalls, 2)
  assert.equal(harness.getLoginCount(), 1)
})

test('静默登录临时失败时保留原登录状态供后续重试', async () => {
  const harness = loadRequestWithMock((options) => {
    if (options.url.endsWith('/v1/auth/login')) {
      options.fail({ errMsg: 'request:fail timeout' })
      return
    }
    options.success({ statusCode: 401, data: { code: 1204, msg: 'unauthorized' } })
  }, {
    access_token: 'old-token',
    life_profile_v1: { id: '1', nickname: '测试用户' },
  })

  await assert.rejects(
    harness.request({ path: '/v1/life-core/place/create', method: 'POST' }),
    /request:fail timeout/,
  )

  assert.equal(harness.storage.get('access_token'), 'old-token')
  assert.deepEqual(harness.storage.get('life_profile_v1'), { id: '1', nickname: '测试用户' })
})
