const assert = require('node:assert/strict')
const test = require('node:test')

const {
  DEFAULT_SHARE_TITLE,
  DEFAULT_SHARE_IMAGE,
  buildShareCard,
} = require('../utils/share')

test('未登录、无权限或详情未加载时使用统一分享介绍和封面', () => {
  const card = buildShareCard({ path: '/pages/tasks/detail?id=8' })

  assert.equal(card.title, DEFAULT_SHARE_TITLE)
  assert.equal(card.imageUrl, DEFAULT_SHARE_IMAGE)
  assert.equal(card.path, '/pages/tasks/detail?id=8')
})

test('详情加载成功后保留业务标题和店铺封面', () => {
  const card = buildShareCard({
    title: '吃喝推荐｜马记永',
    path: '/pages/restaurants/detail?id=8',
    imageUrl: 'https://img.example/restaurant.jpg',
  })

  assert.equal(card.title, '吃喝推荐｜马记永')
  assert.equal(card.imageUrl, 'https://img.example/restaurant.jpg')
})
