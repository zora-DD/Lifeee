const assert = require('node:assert/strict')
const fs = require('node:fs')
const test = require('node:test')

test('零食接口映射、筛选与记录参数正确', async (t) => {
  const originalWx = global.wx
  const captured = []
  global.wx = {
    getStorageSync: () => 'token', setStorageSync: () => {}, removeStorageSync: () => {},
    request(options) {
      captured.push(options)
      options.success({ statusCode: 200, data: { code: 0, data: { items: [{ id: '3', name: '大豫竹方便面', purchase_place: '便利店', status: 'eaten', tasting_count: 2 }], page: 1, page_size: 20, total: 1, has_more: false } } })
    },
    uploadFile() {},
  }
  t.after(() => { global.wx = originalWx })
  delete require.cache[require.resolve('../services/snack-service')]
  const { SnackService } = require('../services/snack-service')
  const result = await SnackService.list({ category: '方便速食', status: 'eaten' })
  assert.equal(captured[0].data.status, 'eaten')
  assert.equal(result.items[0].purchasePlace, '便利店')
  assert.equal(result.items[0].tastingCount, 2)
})

test('零食柜页面、筛选、推荐和记录入口已接入', () => {
  const app = require('../app.json')
  const js = fs.readFileSync(require.resolve('../pages/restaurants/index.js'), 'utf8')
  const wxml = fs.readFileSync(require.resolve('../pages/restaurants/index.wxml'), 'utf8')
  assert.ok(app.pages.includes('pages/snacks/detail'))
  assert.ok(app.pages.includes('pages/snacks/form'))
  assert.match(wxml, /零食柜/)
  assert.match(wxml, /showSnackCategoryMenu/)
  assert.match(wxml, /showSnackStatusMenu/)
  assert.match(js, /recommendSnack/)
  assert.match(js, /loadSnacks/)
})

test('零食详情支持今天吃了、编辑和删除', () => {
  const wxml = fs.readFileSync(require.resolve('../pages/snacks/detail.wxml'), 'utf8')
  assert.match(wxml, /今天吃了/)
  assert.match(wxml, /购买地点/)
  assert.match(wxml, /关于它的记忆/)
  assert.match(wxml, /品尝记录/)
})
