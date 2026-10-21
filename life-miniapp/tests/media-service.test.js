const assert = require('node:assert/strict')
const test = require('node:test')

function loadMediaService(wxMock) {
  const originalWx = global.wx
  global.wx = wxMock
  const mediaPath = require.resolve('../services/media-service')
  delete require.cache[mediaPath]
  const media = require('../services/media-service')
  return {
    media,
    restore() {
      global.wx = originalWx
      delete require.cache[mediaPath]
    },
  }
}

test('相对图片地址按当前环境补齐域名', (t) => {
  const harness = loadMediaService({})
  t.after(harness.restore)

  assert.match(harness.media.resolveMediaURL('/media/place/a.jpg?signature=x'), /^https:\/\//)
  assert.equal(harness.media.resolveMediaURL('wxfile://tmp/a.jpg'), 'wxfile://tmp/a.jpg')
})

test('图片上传携带Token和业务类型并返回对象键', async (t) => {
  let captured
  const harness = loadMediaService({
    getStorageSync: () => 'access-token',
    uploadFile(options) {
      captured = options
      options.success({
        statusCode: 200,
        data: JSON.stringify({
          code: 0,
          data: { mediaId: '9', objectKey: 'restaurant/2026/09/a.jpg', url: '/media/restaurant/2026/09/a.jpg?signature=x' },
        }),
      })
    },
  })
  t.after(harness.restore)

  const result = await harness.media.MediaService.uploadImage('wxfile://tmp/a.jpg', 'restaurant')

  assert.equal(captured.formData.bizType, 'restaurant')
  assert.equal(captured.header.Authorization, 'Bearer access-token')
  assert.equal(result.objectKey, 'restaurant/2026/09/a.jpg')
  assert.match(result.url, /^https:\/\//)
})
