const assert = require('node:assert/strict')
const test = require('node:test')

const { getBillCategoryIcon } = require('../utils/bill-category')

test('账单分类返回统一图标', () => {
  assert.equal(getBillCategoryIcon('交通'), '🚗')
  assert.equal(getBillCategoryIcon('餐饮'), '🍜')
  assert.equal(getBillCategoryIcon('家居用品'), '🪑')
  assert.equal(getBillCategoryIcon('宠物'), '🐱')
  assert.equal(getBillCategoryIcon('未知分类'), '💸')
})
