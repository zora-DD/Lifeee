const assert = require('node:assert/strict')
const test = require('node:test')
const path = require('node:path')

const CACHE_MODULE = path.resolve(__dirname, '../utils/image-cache.js')

function loadImageCache() {
  const originalWx = global.wx
  const originalNow = Date.now
  const storage = {}
  const existingFiles = new Set()
  let now = 1_000_000
  let downloadCount = 0
  let saveCount = 0

  Date.now = () => now
  global.wx = {
    getStorageSync: (key) => storage[key],
    setStorageSync: (key, value) => { storage[key] = value },
    getFileSystemManager: () => ({
      accessSync(filePath) {
        if (!existingFiles.has(filePath)) throw new Error('文件不存在')
      },
      unlink({ filePath }) { existingFiles.delete(filePath) },
    }),
    downloadFile({ success }) {
      downloadCount += 1
      success({ statusCode: 200, tempFilePath: `wxfile://tmp-${downloadCount}` })
    },
    saveFile({ success }) {
      saveCount += 1
      const savedFilePath = `wxfile://saved-${saveCount}`
      existingFiles.add(savedFilePath)
      success({ savedFilePath })
    },
  }

  delete require.cache[CACHE_MODULE]
  const imageCache = require(CACHE_MODULE)

  return {
    imageCache,
    advance: (duration) => { now += duration },
    getDownloadCount: () => downloadCount,
    restore() {
      Date.now = originalNow
      global.wx = originalWx
      delete require.cache[CACHE_MODULE]
    },
  }
}

test('同一张图片在配置有效期内只下载一次', async (t) => {
  const harness = loadImageCache()
  t.after(harness.restore)
  const firstURL = 'https://api.ecsafe.cn/media/place/a.jpg?expires=1&signature=first'
  const renewedURL = 'https://api.ecsafe.cn/media/place/a.jpg?expires=2&signature=second'

  assert.equal(harness.imageCache.IMAGE_CACHE_TTL_MS, 24 * 60 * 60 * 1000)

  const first = await harness.imageCache.cacheImage(firstURL)
  harness.advance(harness.imageCache.IMAGE_CACHE_TTL_MS - 1)
  const second = await harness.imageCache.cacheImage(renewedURL)

  assert.equal(first, second)
  assert.equal(harness.getDownloadCount(), 1)
  assert.equal(harness.imageCache.getCachedImageURL(renewedURL), first)
})

test('图片缓存超过配置有效期后重新下载', async (t) => {
  const harness = loadImageCache()
  t.after(harness.restore)
  const url = 'https://api.ecsafe.cn/media/restaurant/a.jpg?expires=1&signature=first'

  const first = await harness.imageCache.cacheImage(url)
  harness.advance(harness.imageCache.IMAGE_CACHE_TTL_MS)
  const second = await harness.imageCache.cacheImage(url)

  assert.notEqual(first, second)
  assert.equal(harness.getDownloadCount(), 2)
})
