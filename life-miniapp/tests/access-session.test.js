const assert = require('node:assert/strict')
const test = require('node:test')
const Module = require('node:module')
const path = require('node:path')

const ACCESS_MODULE = path.resolve(__dirname, '../utils/access.js')

function loadAccess({ error, cachedUser }) {
  const originalLoad = Module._load
  const originalWx = global.wx

  Module._load = function load(request, parent, isMain) {
    if (request === '../services/profile-service') {
      return { ProfileService: { getCurrent: async () => { throw error } } }
    }
    if (request === '../services/request') {
      return { isUnauthorized: (value) => value?.statusCode === 401 || value?.code === 1204 }
    }
    return originalLoad.call(this, request, parent, isMain)
  }
  global.wx = {
    getStorageSync: (key) => key === 'life_profile_v1' ? cachedUser : undefined,
  }

  delete require.cache[ACCESS_MODULE]
  const access = require(ACCESS_MODULE)
  Module._load = originalLoad

  return {
    access,
    restore() {
      global.wx = originalWx
      delete require.cache[ACCESS_MODULE]
    },
  }
}

test('用户查询临时失败时保留缓存用户，避免页面误显示退出', async (t) => {
  const cachedUser = { id: '1', nickname: '测试用户', accessStatus: 'approved' }
  const harness = loadAccess({ error: new Error('request:fail timeout'), cachedUser })
  t.after(harness.restore)

  assert.deepEqual(await harness.access.getCurrentUser(), cachedUser)
})

test('服务端明确返回 401 时不使用缓存用户', async (t) => {
  const error = Object.assign(new Error('unauthorized'), { statusCode: 401, code: 1204 })
  const harness = loadAccess({ error, cachedUser: { id: '1', accessStatus: 'approved' } })
  t.after(harness.restore)

  assert.equal(await harness.access.getCurrentUser(), null)
})
