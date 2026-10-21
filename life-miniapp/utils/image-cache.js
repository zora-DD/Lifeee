const { CACHE_CONFIG } = require('../config/cache')

const IMAGE_CACHE_STORAGE_KEY = 'life_image_cache_v1'
const IMAGE_CACHE_TTL_MS = CACHE_CONFIG.image.ttlMs
const IMAGE_CACHE_MAX_FILES = CACHE_CONFIG.image.maxFiles

const inFlight = new Map()

function isRemoteURL(value) {
  return /^https?:\/\//i.test(String(value || '').trim())
}

function cacheKey(value) {
  return String(value || '').trim().split('?')[0]
}

function readCache() {
  if (typeof wx?.getStorageSync !== 'function') return {}
  const cached = wx.getStorageSync(IMAGE_CACHE_STORAGE_KEY)
  return cached && typeof cached === 'object' ? cached : {}
}

function saveCache(cache) {
  if (typeof wx?.setStorageSync === 'function') wx.setStorageSync(IMAGE_CACHE_STORAGE_KEY, cache)
}

function fileExists(filePath) {
  if (!filePath) return false
  try {
    const manager = wx?.getFileSystemManager?.()
    if (manager?.accessSync) manager.accessSync(filePath)
    return true
  } catch (error) {
    return false
  }
}

function removeFile(filePath) {
  if (!filePath) return
  try {
    wx?.getFileSystemManager?.().unlink({ filePath, fail: () => {} })
  } catch (error) {
    // 缓存清理失败不影响图片展示。
  }
}

function getCacheEntry(sourceURL) {
  if (!isRemoteURL(sourceURL)) return null
  const cache = readCache()
  const key = cacheKey(sourceURL)
  const entry = cache[key]
  if (!entry || !fileExists(entry.filePath)) {
    if (entry) {
      delete cache[key]
      saveCache(cache)
    }
    return null
  }
  return { ...entry, key }
}

// 缓存刚过期时继续展示旧图，同时在后台静默更新，避免界面闪烁。
function getCachedImageURL(sourceURL) {
  if (!isRemoteURL(sourceURL)) return String(sourceURL || '')
  return getCacheEntry(sourceURL)?.filePath || ''
}

function isCacheFresh(entry, now = Date.now()) {
  return Boolean(entry && now >= entry.cachedAt && now - entry.cachedAt < IMAGE_CACHE_TTL_MS)
}

function downloadImage(sourceURL) {
  return new Promise((resolve, reject) => {
    if (typeof wx?.downloadFile !== 'function') {
      reject(new Error('当前环境不支持图片缓存'))
      return
    }
    wx.downloadFile({
      url: sourceURL,
      success: ({ statusCode, tempFilePath }) => {
        if (statusCode < 200 || statusCode >= 300 || !tempFilePath) {
          reject(new Error(`图片下载失败 ${statusCode || 0}`))
          return
        }
        if (typeof wx?.saveFile !== 'function') {
          resolve(tempFilePath)
          return
        }
        wx.saveFile({
          tempFilePath,
          success: ({ savedFilePath }) => resolve(savedFilePath || tempFilePath),
          fail: reject,
        })
      },
      fail: reject,
    })
  })
}

function trimCache(cache) {
  const entries = Object.entries(cache).sort((left, right) => right[1].cachedAt - left[1].cachedAt)
  entries.slice(IMAGE_CACHE_MAX_FILES).forEach(([key, entry]) => {
    removeFile(entry.filePath)
    delete cache[key]
  })
}

async function cacheImage(sourceURL) {
  const url = String(sourceURL || '').trim()
  if (!isRemoteURL(url)) return url

  const current = getCacheEntry(url)
  if (isCacheFresh(current)) return current.filePath

  const key = cacheKey(url)
  if (inFlight.has(key)) return inFlight.get(key)

  const task = downloadImage(url)
    .then((filePath) => {
      const cache = readCache()
      const previous = cache[key]
      cache[key] = { filePath, cachedAt: Date.now() }
      trimCache(cache)
      saveCache(cache)
      if (previous?.filePath && previous.filePath !== filePath) removeFile(previous.filePath)
      return filePath
    })
    .catch(() => current?.filePath || '')
    .finally(() => inFlight.delete(key))

  inFlight.set(key, task)
  return task
}

module.exports = {
  IMAGE_CACHE_STORAGE_KEY,
  IMAGE_CACHE_TTL_MS,
  cacheImage,
  getCachedImageURL,
}
