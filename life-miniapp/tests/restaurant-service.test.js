const assert = require('node:assert/strict')
const test = require('node:test')

test('店铺特点会随编辑请求提交并映射到详情字段', async (t) => {
  const originalWx = global.wx
  const captured = []
  global.wx = {
    getStorageSync: () => 'test-token',
    setStorageSync: () => {},
    removeStorageSync: () => {},
    request(options) {
      captured.push(options)
      if (options.url.endsWith('/v1/restaurant/consumption/create')) {
        options.success({
          statusCode: 200,
          data: { code: 0, data: { id: '18', consumedAt: '2026-09-18', amount: '28.50', meal: '牛肉面' } },
        })
        return
      }
      if (options.url.endsWith('/v1/restaurant/list')) {
        options.success({ statusCode: 200, data: { code: 0, data: { items: [], page: 1, pageSize: 20 } } })
        return
      }
      options.success({
        statusCode: 200,
        data: {
          code: 0,
          data: {
            id: '8', name: '马记永', category: '小吃', preference: 'must',
            features: '面好吃 · 上菜快', latitude: '31.2304000', longitude: '121.4737000', consumptions: [],
          },
        },
      })
    },
  }
  t.after(() => { global.wx = originalWx })

  const { RestaurantService } = require('../services/restaurant-service')
  const restaurant = await RestaurantService.update('8', {
    name: '马记永', category: '小吃', preference: 'must', tags: '面好吃 · 上菜快', latitude: '31.2304', longitude: '121.4737',
  })
  await RestaurantService.listPage({ status: 'frequent' })
  await RestaurantService.recommend({ category: '小吃', status: 'visited' })
  const consumption = await RestaurantService.addConsumption('8', {
    consumedAt: '2026-09-18', amount: '28.5', meal: '牛肉面', note: '',
  })

  assert.equal(captured[0].data.features, '面好吃 · 上菜快')
  assert.equal(captured[0].data.latitude, '31.2304')
  assert.equal(captured[0].data.longitude, '121.4737')
  assert.equal(captured[1].data.status, 'frequent')
  assert.equal(captured[2].data.status, 'visited')
  assert.equal(restaurant.tags, '面好吃 · 上菜快')
  assert.equal(restaurant.latitude, '31.2304000')
  assert.equal(consumption.amount, 28.5)
})
