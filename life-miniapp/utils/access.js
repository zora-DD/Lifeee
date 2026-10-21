const { ProfileService } = require('../services/profile-service')
const { isUnauthorized } = require('../services/request')
const { savePendingRoute } = require('./pending-navigation')
const {
  hasPermission,
  loadPermissions,
  redirectToFirstAllowedTab,
} = require('./permissions')

const PROFILE_KEY = 'life_profile_v1'
let redirecting = false

async function getCurrentUser() {
  let user = null
  try {
    user = await ProfileService.getCurrent()
  } catch (error) {
    // 网络超时、限流和服务异常不能等同于退出登录；优先保留最近一次用户信息。
    if (!isUnauthorized(error)) user = wx.getStorageSync(PROFILE_KEY) || null
  }

  return user
}

function redirectToLogin(title, returnPath = '') {
  if (redirecting) return

  if (returnPath) savePendingRoute(returnPath)
  redirecting = true
  wx.switchTab({
    url: '/pages/index/index',
    complete: () => {
      wx.showToast({ title, icon: 'none' })
      setTimeout(() => { redirecting = false }, 500)
    },
  })
}

async function requireLogin() {
  const user = await getCurrentUser()
  if (user) return user

  redirectToLogin('请先登录')
  return null
}

async function getModuleAccess(module, requiredLevel = 'read') {
  const user = await getCurrentUser()
  if (!user || user.accessStatus !== 'approved') {
    return { allowed: false, levels: null, user }
  }
  const levels = await loadPermissions(user)
  return {
    allowed: !module || hasPermission(levels, module, requiredLevel),
    levels,
    user,
  }
}

async function requireWriteAccess(module = '') {
  const access = await getModuleAccess(module, 'write')

  if (access.allowed) return access.user

  if (access.user && access.user.accessStatus === 'approved') {
    wx.showToast({ title: '当前模块为只读权限', icon: 'none' })
    return null
  }

  redirectToLogin(access.user ? '账号正在审核' : '请先登录')

  return null
}

async function requireApprovedAccess(returnPath = '', module = '') {
  const access = await getModuleAccess(module, 'read')
  if (access.allowed) return access.user

  if (access.user && access.user.accessStatus === 'approved') {
    redirectToFirstAllowedTab(access.levels)
    return null
  }

  redirectToLogin(access.user ? '账号正在审核' : '请先登录', returnPath)
  return null
}

module.exports = {
  getCurrentUser,
  getModuleAccess,
  requireApprovedAccess,
  requireLogin,
  requireWriteAccess,
}
