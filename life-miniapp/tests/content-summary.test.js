const assert = require('node:assert/strict')
const test = require('node:test')

const { toContentCard } = require('../utils/content-summary')

test('任务摘要映射到任务详情', () => {
  const item = toContentCard({
    targetId: '9', targetType: 'task', title: '整理房间', priority: 'high', status: 'todo', dueAt: '',
  })
  assert.equal(item.url, '/pages/tasks/detail?id=9')
  assert.equal(item.typeLabel, '任务')
  assert.match(item.meta, /高/)
})

test('吃喝和地点摘要保留各自业务信息', () => {
  const restaurant = toContentCard({
    targetId: '5', targetType: 'restaurant', title: '小馆', category: '小吃', status: 'love', visitCount: 3, totalSpent: '88.00',
  })
  const place = toContentCard({
    targetId: '6', targetType: 'place', title: '公园', category: '自然风景', visited: false,
  })
  assert.equal(restaurant.url, '/pages/restaurants/detail?id=5')
  assert.match(restaurant.meta, /喜欢/)
  assert.match(restaurant.detail, /88/)
  assert.equal(place.url, '/pages/places/detail?id=6')
  assert.match(place.detail, /想去/)
})
