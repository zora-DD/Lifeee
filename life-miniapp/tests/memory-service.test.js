const assert = require('node:assert/strict')
const test = require('node:test')
const fs = require('node:fs')
const path = require('node:path')

function loadService(wxMock) {
  global.wx = wxMock
  const modules = [
    '../services/memory-service',
    '../services/media-service',
    '../services/request',
    '../services/auth-session',
    '../config/env',
  ]
  modules.forEach((modulePath) => delete require.cache[require.resolve(modulePath)])
  return require('../services/memory-service')
}

test('时光列表按年月日生成适合时间轴的日期文案', () => {
  const { formatDateLabel } = loadService({})
  assert.equal(formatDateLabel('2026-09-06'), '9月6日')
  assert.equal(formatDateLabel('2013-05'), '5月')
  assert.equal(formatDateLabel('2012'), '2012年')
})

test('时光列表分页请求并补齐受保护图片域名', async () => {
  let requestOptions
  const { MemoryService } = loadService({
    getStorageSync() { return 'token' },
    request(options) {
      requestOptions = options
      options.success({
        statusCode: 200,
        data: {
          code: 0,
          data: {
            items: [{ id: '1', date: '2026-09-06', images: [{ objectKey: 'memory/a.jpg', url: '/media/memory/a.jpg?signature=x' }] }],
            years: [2026], page: 1, pageSize: 20, total: 1, has_more: false,
          },
        },
      })
    },
  })
  const result = await MemoryService.list({ year: 2026, page: 1, pageSize: 20 })
  assert.equal(requestOptions.url.endsWith('/v1/memory/list'), true)
  assert.equal(requestOptions.data.year, 2026)
  assert.match(result.items[0].images[0].sourceUrl, /^https:\/\//)
  assert.deepEqual(result.years, ['2026'])
})

test('时光页面渲染接口写入的 memories 列表', () => {
  const template = fs.readFileSync(
    path.join(__dirname, '../pages/memories/index.wxml'),
    'utf8',
  )
  assert.match(template, /wx:for="{{memories}}"/)
  assert.doesNotMatch(template, /visibleMemories/)
})

test('生活小程序主页不再展示相册入口或预加载相册图片', () => {
  const homeTemplate = fs.readFileSync(path.join(__dirname, '../pages/index/index.wxml'), 'utf8')
  const homeScript = fs.readFileSync(path.join(__dirname, '../pages/index/index.js'), 'utf8')
  assert.doesNotMatch(homeTemplate, /memory-entry/)
  assert.doesNotMatch(homeScript, /MemoryService|loadMemoryPreview|openMemories|memoryCovers/)
})

test('时光页面单图完整展示，多图保持裁切拼图', () => {
  const template = fs.readFileSync(
    path.join(__dirname, '../pages/memories/index.wxml'),
    'utf8',
  )
  const stylesheet = fs.readFileSync(
    path.join(__dirname, '../pages/memories/index.wxss'),
    'utf8',
  )
  assert.match(template, /memory\.images\.length === 1 \? 'widthFix' : 'aspectFill'/)
  assert.match(stylesheet, /\.memory-images\.image-count-1\s*{[^}]*height:\s*auto;/s)
  assert.match(stylesheet, /\.image-count-1 \.memory-image\s*{[^}]*height:\s*auto;/s)
})
