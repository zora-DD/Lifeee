const assert = require('node:assert/strict')
const test = require('node:test')

const { withScrollRestoration } = require('../utils/page-scroll')

test('从下一级页面返回后恢复离开前的滚动位置', async (t) => {
  const originalWx = global.wx
  const calls = []
  global.wx = {
    nextTick(callback) { callback() },
    pageScrollTo(options) { calls.push(options) },
  }
  t.after(() => { global.wx = originalWx })

  const pageOptions = withScrollRestoration({
    async onShow() { this.loaded = true },
  })
  const page = {}

  pageOptions.onPageScroll.call(page, { scrollTop: 628 })
  pageOptions.onHide.call(page)
  await pageOptions.onShow.call(page)

  assert.equal(page.loaded, true)
  assert.deepEqual(calls, [{ scrollTop: 628, duration: 0 }])
})

test('首次进入页面时不主动滚动', async (t) => {
  const originalWx = global.wx
  const calls = []
  global.wx = {
    nextTick(callback) { callback() },
    pageScrollTo(options) { calls.push(options) },
  }
  t.after(() => { global.wx = originalWx })

  const pageOptions = withScrollRestoration({})
  await pageOptions.onShow.call({})

  assert.deepEqual(calls, [])
})
