const assert = require('node:assert/strict')
const fs = require('node:fs')
const test = require('node:test')

test('审核版本不声明或调用 wx.chooseLocation', () => {
  const app = require('../app.json')
  const formScript = fs.readFileSync(require.resolve('../pages/restaurants/form.js'), 'utf8')
  const formTemplate = fs.readFileSync(require.resolve('../pages/restaurants/form.wxml'), 'utf8')

  assert.equal(app.requiredPrivateInfos, undefined)
  assert.equal(app.permission?.['scope.userLocation'], undefined)
  assert.doesNotMatch(formScript, /^\s*wx\.chooseLocation\s*\(/m)
  assert.doesNotMatch(formTemplate, /bindtap="chooseLocation"/)
})

test('生产环境使用 lifeee.zoradu.cn', () => {
  const source = fs.readFileSync(require.resolve('../config/env.js'), 'utf8')
  assert.match(source, /const PROD_API_BASE_URL = 'https:\/\/lifeee\.zoradu\.cn'/)
  assert.doesNotMatch(source, /api\.ecsafe\.cn/)
})
