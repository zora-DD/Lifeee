const { MemoryService } = require('../../services/memory-service')
const { cacheImage, getCachedImageURL } = require('../../utils/image-cache')
const { getCurrentUser } = require('../../utils/access')

const PAGE_SIZE = 20

Page({
  data: {
    loading: true,
    loadingMore: false,
    selectedYear: '全部',
    years: ['全部'],
    memories: [],
    page: 1,
    total: 0,
    hasMore: false,
  },

  async onLoad() {
    const user = await getCurrentUser()
    if (!user || user.accessStatus !== 'approved' || user.role !== 'admin') {
      wx.showToast({ title: '仅管理员可查看', icon: 'none' })
      setTimeout(() => wx.switchTab({ url: '/pages/index/index' }), 500)
      return
    }
    await this.loadMemories()
  },

  onReachBottom() {
    this.loadMemories({ append: true })
  },

  selectYear({ currentTarget }) {
    const selectedYear = currentTarget.dataset.year
    if (selectedYear === this.data.selectedYear) return
    this.setData({ selectedYear }, () => this.loadMemories())
  },

  async loadMemories({ append = false } = {}) {
    if (append && (this.data.loadingMore || !this.data.hasMore)) return
    const page = append ? this.data.page + 1 : 1
    this.setData(append ? { loadingMore: true } : { loading: true })
    try {
      const result = await MemoryService.list({
        year: this.data.selectedYear === '全部' ? 0 : Number(this.data.selectedYear),
        page,
        pageSize: PAGE_SIZE,
      })
      const items = result.items.map((memory) => ({
        ...memory,
        images: memory.images.map((image) => ({
          ...image,
          src: getCachedImageURL(image.sourceUrl),
        })),
      }))
      this.setData({
        loading: false,
        loadingMore: false,
        page: result.page,
        total: result.total,
        hasMore: result.hasMore,
        years: ['全部', ...result.years],
        memories: append ? [...this.data.memories, ...items] : items,
      })
      this.refreshImages(items)
    } catch (error) {
      this.setData({ loading: false, loadingMore: false })
      wx.showToast({ title: error.message || '加载相册失败', icon: 'none' })
    }
  },

  async refreshImages(items) {
    const tasks = []
    items.forEach((memory) => {
      memory.images.forEach((image) => {
        tasks.push(cacheImage(image.sourceUrl).then((localURL) => ({
          memoryId: String(memory.id),
          objectKey: image.objectKey,
          sourceURL: image.sourceUrl,
          localURL,
        })))
      })
    })
    const resolved = await Promise.all(tasks)
    const urls = new Map(resolved.filter((item) => item.localURL).map((item) => [`${item.memoryId}:${item.objectKey}`, item]))
    if (!urls.size) return
    this.setData({
      memories: this.data.memories.map((memory) => ({
        ...memory,
        images: memory.images.map((image) => {
          const cached = urls.get(`${memory.id}:${image.objectKey}`)
          if (!cached || image.sourceUrl !== cached.sourceURL) return image
          return { ...image, src: cached.localURL }
        }),
      })),
    })
  },

  previewImage({ currentTarget }) {
    const { id, src } = currentTarget.dataset
    const memory = this.data.memories.find((item) => String(item.id) === String(id))
    if (!memory) return
    const urls = memory.images.map((image) => image.src || image.sourceUrl).filter(Boolean)
    if (urls.length) wx.previewImage({ current: src, urls })
  },

  addMemory() {
    wx.navigateTo({
      url: '/pages/memories/form',
      events: {
        memoryCreated: () => this.loadMemories(),
      },
    })
  },
})
