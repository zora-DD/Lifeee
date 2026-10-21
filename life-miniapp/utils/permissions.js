const { ProfileService } = require('../services/profile-service')
const {
  readSessionPermissions,
  saveSessionPermissions,
} = require('./session-cache')

const PERMISSION_KEY = 'life_module_permissions_v1'

const MODULES = {
  TASK: 'task',
  ACTIVITY: 'activity_log',
  BILL: 'bill',
  RESTAURANT: 'restaurant',
  PLACE: 'place',
}

const MODULE_LIST = Object.values(MODULES)
const LEVEL_RANK = { none: 0, read: 1, write: 2 }

const TAB_DEFINITIONS = [
  { index: 0, pagePath: '/pages/tasks/index', text: '任务', key: 'task', modules: [MODULES.TASK] },
  { index: 1, pagePath: '/pages/logs/index', text: '记录', key: 'log', modules: [MODULES.ACTIVITY, MODULES.BILL] },
  { index: 2, pagePath: '/pages/restaurants/index', text: '吃喝', key: 'food', modules: [MODULES.RESTAURANT] },
  { index: 3, pagePath: '/pages/places/index', text: '地点', key: 'place', modules: [MODULES.PLACE] },
  { index: 4, pagePath: '/pages/index/index', text: '主页', key: 'profile', modules: [] },
]

function emptyPermissions() {
  return MODULE_LIST.reduce((result, module) => {
    result[module] = 'none'
    return result
  }, {})
}

function normalizePermissions(items, user) {
  const levels = emptyPermissions()
  if (user?.role === 'admin') {
    MODULE_LIST.forEach((module) => { levels[module] = 'write' })
    return levels
  }
  ;(Array.isArray(items) ? items : []).forEach((item) => {
    if (MODULE_LIST.includes(item?.module) && LEVEL_RANK[item?.level] !== undefined) {
      levels[item.module] = item.level
    }
  })
  return levels
}

function readPermissionCache(user) {
  if (typeof wx?.getStorageSync !== 'function') return null
  const cached = wx.getStorageSync(PERMISSION_KEY)
  if (!cached || String(cached.userId || '') !== String(user?.id || '')) return null
  return normalizePermissions(
    Object.entries(cached.levels || {}).map(([module, level]) => ({ module, level })),
    user,
  )
}

function savePermissionCache(user, levels) {
  saveSessionPermissions(user?.id, levels)
  if (typeof wx?.setStorageSync === 'function') {
    wx.setStorageSync(PERMISSION_KEY, { userId: user?.id || '', levels })
  }
}

async function loadPermissions(user, { force = false } = {}) {
  if (!user || user.accessStatus !== 'approved') return emptyPermissions()
  if (user.role === 'admin') {
    const levels = normalizePermissions([], user)
    saveSessionPermissions(user.id, levels)
    return levels
  }
  // Node 页面单测没有完整微信网络 API，沿用升级前的全部可写行为。
  if (typeof wx?.request !== 'function') return normalizePermissions([], { role: 'admin' })

  const sessionPermissions = readSessionPermissions(user.id)
  if (sessionPermissions && !force) return sessionPermissions
  const persistedPermissions = readPermissionCache(user)
  try {
    const result = await ProfileService.getMyPermissions()
    const levels = normalizePermissions(result?.permissions, user)
    savePermissionCache(user, levels)
    return levels
  } catch (error) {
    if (persistedPermissions) {
      saveSessionPermissions(user.id, persistedPermissions)
      return persistedPermissions
    }
    // 首次读取失败时按无权限处理，避免网络异常导致页面脚本中断或误开放模块。
    return emptyPermissions()
  }
}

function hasPermission(levels, module, requiredLevel = 'read') {
  return (LEVEL_RANK[levels?.[module]] || 0) >= (LEVEL_RANK[requiredLevel] || 0)
}

function hasAnyPermission(levels, modules, requiredLevel = 'read') {
  return modules.some((module) => hasPermission(levels, module, requiredLevel))
}

function getVisibleTabs(levels) {
  return TAB_DEFINITIONS.filter((tab) => !tab.modules.length || hasAnyPermission(levels, tab.modules))
}

function syncTabBar(page, selected, levels) {
  if (!page || typeof page.getTabBar !== 'function') return
  const tabBar = page.getTabBar()
  if (!tabBar) return
  tabBar.setData({ selected, list: getVisibleTabs(levels) })
}

function redirectToFirstAllowedTab(levels, title = '当前模块暂无访问权限') {
  const target = getVisibleTabs(levels).find((tab) => tab.index !== 4) || TAB_DEFINITIONS[4]
  wx.switchTab({
    url: target.pagePath,
    complete: () => wx.showToast({ title, icon: 'none' }),
  })
}

module.exports = {
  MODULES,
  PERMISSION_KEY,
  TAB_DEFINITIONS,
  emptyPermissions,
  getVisibleTabs,
  hasAnyPermission,
  hasPermission,
  loadPermissions,
  normalizePermissions,
  redirectToFirstAllowedTab,
  savePermissionCache,
  syncTabBar,
}
