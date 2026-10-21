const assert = require('node:assert/strict')
const fs = require('node:fs')
const test = require('node:test')

test('首页登录环节提供同等明显的暂不登录按钮', () => {
  const template = fs.readFileSync(require.resolve('../pages/index/index.wxml'), 'utf8')
  const script = fs.readFileSync(require.resolve('../pages/index/index.js'), 'utf8')
  const style = fs.readFileSync(require.resolve('../pages/index/index.wxss'), 'utf8')

  assert.match(template, /暂不登录/)
  assert.match(template, /bindtap="exitMiniProgram"/)
  assert.match(template, /微信快捷登录/)
  assert.match(script, /wx\.exitMiniProgram/)
  assert.match(style, /\.login-button,.guest-exit-button\{[^}]*flex:1/s)
})

test('用户主动点击登录按钮后才执行微信登录', () => {
  const template = fs.readFileSync(require.resolve('../pages/index/index.wxml'), 'utf8')
  const script = fs.readFileSync(require.resolve('../pages/index/index.js'), 'utf8')
  assert.match(template, /bindtap="login"/)
  assert.doesNotMatch(script, /onShow\(\)\s*\{[^}]*\.login\(/s)
})
