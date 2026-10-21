let userLoaded = false
let currentUser = null
let permissionEntry = null

function readSessionUser() {
  return { loaded: userLoaded, user: currentUser }
}

function saveSessionUser(user) {
  currentUser = user || null
  userLoaded = true
}

function readSessionPermissions(userId) {
  if (!permissionEntry || String(permissionEntry.userId) !== String(userId || '')) return null
  return permissionEntry.levels
}

function saveSessionPermissions(userId, levels) {
  permissionEntry = { userId: String(userId || ''), levels }
}

function invalidateSessionCache() {
  userLoaded = false
  currentUser = null
  permissionEntry = null
}

module.exports = {
  invalidateSessionCache,
  readSessionPermissions,
  readSessionUser,
  saveSessionPermissions,
  saveSessionUser,
}
