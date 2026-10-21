const assert = require('node:assert/strict')
const test = require('node:test')

const {
  RESTAURANT_CATEGORIES,
  RESTAURANT_FILTER_CATEGORIES,
  RESTAURANT_CATEGORY_ICONS,
  RESTAURANT_CATEGORY_ICON_GROUPS,
} = require('../utils/restaurant-category')

test('吃喝分类在筛选、表单和图标体系中保持一致', () => {
  const values = RESTAURANT_CATEGORIES.map((item) => item.value)

  assert.deepEqual(values, [
    '小吃', '火锅', '烧烤', '地方菜系', '地域料理',
    '鱼鲜海鲜', '面包蛋糕', '饮品', '其他',
  ])
  assert.equal(RESTAURANT_FILTER_CATEGORIES[0].value, 'all')
  assert.deepEqual(RESTAURANT_FILTER_CATEGORIES.slice(1).map((item) => item.value), values)
  values.forEach((value) => {
    assert.ok(RESTAURANT_CATEGORY_ICONS[value])
    assert.ok(RESTAURANT_CATEGORY_ICON_GROUPS[value].length >= 3)
  })
})
