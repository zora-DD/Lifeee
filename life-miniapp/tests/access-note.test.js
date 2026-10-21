const assert = require('node:assert/strict')
const fs = require('node:fs')
const test = require('node:test')

test('申请权限支持提交身份备注', async (t) => {
  const originalWx = global.wx
  let requestOptions
  global.wx = {
    getStorageSync: () => 'token', setStorageSync: () => {}, removeStorageSync: () => {},
    request(options) {
      requestOptions = options
      options.success({ statusCode: 200, data: { code: 0, data: { id: '8', access_status: 'pending', application_note: '我是小帅' } } })
    },
  }
  t.after(() => { global.wx = originalWx })
  delete require.cache[require.resolve('../services/profile-service')]
  const { ProfileService } = require('../services/profile-service')

  const user = await ProfileService.applyAccess(' 我是小帅 ')

  assert.equal(requestOptions.url.endsWith('/v1/access/apply'), true)
  assert.equal(requestOptions.data.note, '我是小帅')
  assert.equal(user.applicationNote, '我是小帅')
})

test('主页和审批列表展示身份备注', () => {
  const home = fs.readFileSync(require.resolve('../pages/index/index.wxml'), 'utf8')
  const review = fs.readFileSync(require.resolve('../pages/profile/access-review.wxml'), 'utf8')
  assert.match(home, /告诉管理员你是谁/)
  assert.match(home, /填写身份备注/)
  assert.match(review, /身份备注/)
  assert.match(review, /未填写身份备注/)
})
