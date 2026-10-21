const assert = require('node:assert/strict')
const test = require('node:test')

const {
  buildLogInspiration,
  buildPlaceInspiration,
  buildRestaurantInspiration,
  buildTaskInspiration,
  getDailySourceIndex,
} = require('../utils/daily-inspiration')

const now = new Date('2026-09-04T09:00:00+08:00')

test('任务灵感优先选择最近截止的未完成任务', () => {
  const result = buildTaskInspiration([
    { id: '2', title: '下周任务', status: 'todo', priority: 'low', dueAt: '2026-09-10T10:00:00+08:00' },
    { id: '1', title: '今日任务', status: 'todo', priority: 'high', dueAt: '2026-09-04T18:00:00+08:00' },
    { id: '3', title: '已完成', status: 'done', priority: 'high', dueAt: '2026-09-03T18:00:00+08:00' },
  ], now)

  assert.equal(result.title, '今日任务')
  assert.equal(result.path, '/pages/tasks/detail?id=1')
  assert.match(result.description, /今天 18:00/)
})

test('吃喝、地点和生活记录生成可直达的灵感', () => {
  const restaurant = buildRestaurantInspiration({ id: '8', name: '小馆子', category: '小吃', preference: 'love', visitCount: 2, coverUrl: 'https://img.example/food.jpg' }, now)
  const place = buildPlaceInspiration([{ id: '9', name: '湖边', category: '自然风景', visited: false, coverUrl: 'https://img.example/place.jpg' }], now)
  const log = buildLogInspiration([{ id: '10', content: '收拾房间', occurredAt: now.toISOString(), durationMinutes: 30 }], now)

  assert.equal(restaurant.path, '/pages/restaurants/detail?id=8')
  assert.equal(restaurant.imageUrl, 'https://img.example/food.jpg')
  assert.equal(place.path, '/pages/places/detail?id=9')
  assert.equal(place.imageUrl, 'https://img.example/place.jpg')
  assert.equal(log.path, '/pages/logs/detail?id=10')
  assert.match(restaurant.layout, /^(left|right)$/)
  assert.match(place.motion, /^(rise|slide|zoom)$/)
})

test('同一天的默认灵感类型保持稳定', () => {
  assert.equal(getDailySourceIndex(now), getDailySourceIndex(new Date('2026-09-04T22:30:00+08:00')))
})
