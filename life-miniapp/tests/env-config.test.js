const assert = require('node:assert/strict')
const test = require('node:test')

const { API_BASE_URL, ENV, ENV_CONFIG, resolveEnvironment, resolveApiPath, resolveApiPathWithPrefix } = require('../config/env')

test('前端只配置 local 和 prod 两套环境', () => {
  assert.deepEqual(Object.keys(ENV_CONFIG).sort(), ['local', 'prod'])
})

test('当前环境存在且接口地址不带末尾斜杠', () => {
  assert.ok(ENV_CONFIG[ENV])
  assert.equal(API_BASE_URL, ENV_CONFIG[ENV].apiBaseUrl)
  assert.equal(API_BASE_URL.endsWith('/'), false)
})

test('正式环境使用 HTTPS 域名', () => {
  assert.match(ENV_CONFIG.prod.apiBaseUrl, /^https:\/\/[a-z0-9.-]+$/i)
})

test('develop 使用本地服务，体验版和正式版使用 HTTPS 生产服务', () => {
  assert.equal(resolveEnvironment('develop'), 'local')
  assert.equal(resolveEnvironment('trial'), 'prod')
  assert.equal(resolveEnvironment('release'), 'prod')
  assert.equal(resolveEnvironment(''), 'prod')
})

test('本地去掉网关前缀，线上保留 /life-core', () => {
  assert.equal(ENV_CONFIG.local.apiPathPrefix, '')
  assert.equal(ENV_CONFIG.prod.apiPathPrefix, '/life-core')
  assert.equal(
    resolveApiPathWithPrefix('/v1/life-core/auth/login', ENV_CONFIG.local.apiPathPrefix),
    '/v1/auth/login',
  )
  assert.equal(
    resolveApiPathWithPrefix('/v1/auth/login', ENV_CONFIG.prod.apiPathPrefix),
    '/v1/life-core/auth/login',
  )
  assert.equal(
    resolveApiPathWithPrefix('/v1/life-core/auth/login', ENV_CONFIG.prod.apiPathPrefix),
    '/v1/life-core/auth/login',
  )
  if (ENV === 'local') {
    assert.equal(resolveApiPath('/v1/life-core/auth/login'), '/v1/auth/login')
  }
})

test('App 全局配置读取当前环境', () => {
  let appOptions
  global.App = (options) => {
    appOptions = options
  }
  delete require.cache[require.resolve('../app')]
  require('../app')

  assert.equal(appOptions.globalData.env, ENV)
  assert.equal(appOptions.globalData.apiBaseUrl, API_BASE_URL)
  delete global.App
})

test('小程序从后台重新进入时清除本次会话缓存', () => {
  let appOptions
  global.App = (options) => {
    appOptions = options
  }
  delete require.cache[require.resolve('../app')]
  delete require.cache[require.resolve('../utils/session-cache')]
  require('../app')
  const sessionCache = require('../utils/session-cache')

  sessionCache.saveSessionUser({ id: '1' })
  sessionCache.saveSessionPermissions('1', { task: 'write' })
  appOptions.onHide()
  appOptions.onShow()

  assert.equal(sessionCache.readSessionUser().loaded, false)
  assert.equal(sessionCache.readSessionPermissions('1'), null)
  delete global.App
})
