const BILL_CATEGORY_ICONS = {
  '交通': '🚗',
  '餐饮': '🍜',
  '家居用品': '🪑',
  '医疗': '💊',
  '宠物': '🐱',
  '娱乐': '📱',
  '日常采购': '🛒',
  '住房': '🏠',
  '转账': '💸',
  '购物': '🛒',
  '智能设备': '💡',
  '生活缴费': '🧾',
  '收入': '💰',
  '其他': '💸',
  '其余': '💸',
}

function getBillCategoryIcon(category) {
  return BILL_CATEGORY_ICONS[category] || BILL_CATEGORY_ICONS.其他
}

module.exports = { BILL_CATEGORY_ICONS, getBillCategoryIcon }
