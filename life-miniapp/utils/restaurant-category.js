const RESTAURANT_CATEGORIES = [
  { label: '小吃', value: '小吃', icon: '🥟', icons: ['🥟', '🍜', '🥞'] },
  { label: '火锅', value: '火锅', icon: '🍲', icons: ['🍲', '🌶️', '🥘'] },
  { label: '烧烤', value: '烧烤', icon: '🔥', icons: ['🔥', '🍢', '🥩'] },
  { label: '地方菜系', value: '地方菜系', icon: '🥘', icons: ['🥘', '🍚', '🥢'] },
  { label: '地域料理', value: '地域料理', icon: '🍛', icons: ['🍛', '🍣', '🍝'] },
  { label: '鱼鲜海鲜', value: '鱼鲜海鲜', icon: '🦐', icons: ['🦐', '🐟', '🦀'] },
  { label: '面包蛋糕', value: '面包蛋糕', icon: '🥐', icons: ['🥐', '🍰', '🍞'] },
  { label: '饮品', value: '饮品', icon: '🥤', icons: ['☕', '🧋', '🥤'] },
  { label: '其他', value: '其他', icon: '🍽️', icons: ['🍽️', '🥢', '🍱'] },
]

const RESTAURANT_FILTER_CATEGORIES = [
  { label: '全部分类', value: 'all' },
  ...RESTAURANT_CATEGORIES,
]

const RESTAURANT_CATEGORY_ICONS = {}
const RESTAURANT_CATEGORY_ICON_GROUPS = {}
RESTAURANT_CATEGORIES.forEach((category) => {
  RESTAURANT_CATEGORY_ICONS[category.value] = category.icon
  RESTAURANT_CATEGORY_ICON_GROUPS[category.value] = category.icons
})

module.exports = {
  RESTAURANT_CATEGORIES,
  RESTAURANT_FILTER_CATEGORIES,
  RESTAURANT_CATEGORY_ICONS,
  RESTAURANT_CATEGORY_ICON_GROUPS,
}
