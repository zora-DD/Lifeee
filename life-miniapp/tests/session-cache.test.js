const assert = require('node:assert/strict')
const test = require('node:test')
const path = require('node:path')

const PROFILE_MODULE = path.resolve(__dirname, '../services/profile-service.js')
const REQUEST_MODULE = path.resolve(__dirname, '../services/request.js')
const AUTH_MODULE = path.resolve(__dirname, '../services/auth-session.js')
const SESSION_MODULE = path.resolve(__dirname, '../utils/session-cache.js')
const PERMISSION_MODULE = path.resolve(__dirname, '../utils/permissions.js')

function clearModules() {
  ;[PROFILE_MODULE, REQUEST_MODULE, AUTH_MODULE, SESSION_MODULE, PERMISSION_MODULE]
    .forEach((modulePath) => { delete require.cache[modulePath] })
}

function createHarness() {
  const originalWx = global.wx
  const storage = new Map([
    ['access_token', 'valid-token'],
    ['life_profile_v1', { id: '2', accessStatus: 'approved', role: 'user' }],
  ])
  let currentUserCalls = 0
  let permissionCalls = 0

  global.wx = {
    getStorageSync: (key) => storage.get(key),
    setStorageSync: (key, value) => storage.set(key, value),
    removeStorageSync: (key) => storage.delete(key),
    request({ url, success }) {
      if (url.endsWith('/v1/user/current')) {
        currentUserCalls += 1
        success({
          statusCode: 200,
          data: { code: 0, data: { id: '2', access_status: 'approved', role: 'user' } },
        })
        return
      }
      if (url.endsWith('/v1/user/permission/list')) {
        permissionCalls += 1
        success({
          statusCode: 200,
          data: { code: 0, data: { permissions: [{ module: 'restaurant', level: 'read' }] } },
        })
      }
    },
  }

  clearModules()
  const { ProfileService } = require(PROFILE_MODULE)
  const { loadPermissions } = require(PERMISSION_MODULE)
  const { invalidateSessionCache } = require(SESSION_MODULE)

  return {
    ProfileService,
    invalidateSessionCache,
    loadPermissions,
    currentUserCalls: () => currentUserCalls,
    permissionCalls: () => permissionCalls,
    restore() {
      global.wx = originalWx
      clearModules()
    },
  }
}

test('同一次小程序会话只请求一次用户和权限', async (t) => {
  const harness = createHarness()
  t.after(harness.restore)

  const firstUser = await harness.ProfileService.getCurrent()
  const secondUser = await harness.ProfileService.getCurrent()
  await harness.loadPermissions(firstUser)
  await harness.loadPermissions(secondUser)

  assert.equal(harness.currentUserCalls(), 1)
  assert.equal(harness.permissionCalls(), 1)
})

test('小程序重新进入后重新校验用户和权限', async (t) => {
  const harness = createHarness()
  t.after(harness.restore)

  const firstUser = await harness.ProfileService.getCurrent()
  await harness.loadPermissions(firstUser)
  harness.invalidateSessionCache()
  const nextUser = await harness.ProfileService.getCurrent()
  await harness.loadPermissions(nextUser)

  assert.equal(harness.currentUserCalls(), 2)
  assert.equal(harness.permissionCalls(), 2)
})
