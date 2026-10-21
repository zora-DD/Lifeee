const assert = require('node:assert/strict')
const test = require('node:test')

test('地点创建会传递微信公众号攻略链接', async (t) => {
  const originalWx = global.wx
  let requestOptions
  global.wx = {
    getStorageSync: () => 'test-token',
    setStorageSync: () => {},
    removeStorageSync: () => {},
    request(options) {
      requestOptions = options
      options.success({
        statusCode: 200,
        data: {
          code: 0,
          data: {
            id: '4',
            name: options.data.name,
            feature: options.data.feature,
            guideUrl: options.data.guideUrl,
          },
        },
      })
    },
  }
  t.after(() => { global.wx = originalWx })

  const { PlaceService } = require('../services/place-service')
  const guideUrl = 'https://mp.weixin.qq.com/s?__biz=test&mid=1'
  const place = await PlaceService.create({
    name: '皖浙天路', feature: '自驾风景线', guideUrl,
    distanceKm: 330, driveDurationMinutes: 270,
  })

  assert.match(requestOptions.url, /\/v1\/place\/create$/)
  assert.equal(requestOptions.data.guideUrl, guideUrl)
  assert.equal(requestOptions.data.distanceKm, 330)
  assert.equal(requestOptions.data.driveDurationMinutes, 270)
  assert.equal(place.guideUrl, guideUrl)
})
