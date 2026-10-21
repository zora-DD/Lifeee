const assert = require('node:assert/strict')
const test = require('node:test')

test('主要列表接口默认每页只请求 20 条并保留汇总字段', async (t) => {
  const originalWx = global.wx
  const requests = []
  global.wx = {
    getStorageSync: () => 'test-token',
    setStorageSync: () => {},
    removeStorageSync: () => {},
    request(options) {
      requests.push(options)
      options.success({
        statusCode: 200,
        data: {
          code: 0,
          data: {
            items: [], creators: [], page: options.data.page || 1, pageSize: options.data.pageSize,
            total: 45, has_more: true, overallTotal: 45, totalVisitCount: 68,
            totalSpent: '1688.50', visitedCount: 18, wishlistCount: 27,
            todoCount: 30, doneCount: 15,
          },
        },
      })
    },
  }
  t.after(() => { global.wx = originalWx })

  const { ActivityLogService } = require('../services/activity-log-service')
  const { BillService } = require('../services/bill-service')
  const { RestaurantService } = require('../services/restaurant-service')
  const { PlaceService } = require('../services/place-service')
  const { TaskService } = require('../services/task-service')

  await ActivityLogService.list({ page: 2 })
  await BillService.list({ page: 2 })
  const restaurants = await RestaurantService.listPage({ page: 2 })
  const places = await PlaceService.listPage({ page: 2 })
  const tasks = await TaskService.listPage({ page: 2 })

  assert.equal(requests.length, 5)
  requests.forEach((request) => {
    assert.equal(request.data.page, 2)
    assert.equal(request.data.pageSize, 20)
  })
  assert.equal(restaurants.totalSpent, 1688.5)
  assert.equal(places.visitedCount, 18)
  assert.equal(tasks.doneCount, 15)
})
