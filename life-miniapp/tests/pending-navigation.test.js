const assert = require('node:assert/strict')
const test = require('node:test')

let storage = {}
let navigatedTo = ''

global.wx = {
  getStorageSync(key) {
    return storage[key]
  },
  setStorageSync(key, value) {
    storage[key] = value
  },
  removeStorageSync(key) {
    delete storage[key]
  },
  navigateTo({ url, success }) {
    navigatedTo = url
    success()
  },
}

const {
  getPendingRoute,
  resumePendingRoute,
  savePendingRoute,
} = require('../utils/pending-navigation')

test.beforeEach(() => {
  storage = {}
  navigatedTo = ''
})

test('分享目标页可保存并在登录后恢复', () => {
  const path = '/pages/tasks/detail?id=123'
  savePendingRoute(path)

  assert.equal(getPendingRoute(), path)
  assert.equal(resumePendingRoute(), true)
  assert.equal(navigatedTo, path)
  assert.equal(getPendingRoute(), '')
})

test('不接受外部地址作为回跳目标', () => {
  savePendingRoute('https://example.com')

  assert.equal(resumePendingRoute(), false)
  assert.equal(navigatedTo, '')
})
