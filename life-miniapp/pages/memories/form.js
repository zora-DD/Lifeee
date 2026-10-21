const { MemoryService } = require('../../services/memory-service')
const { getCurrentUser } = require('../../utils/access')

function today() {
  const now = new Date()
  const month = String(now.getMonth() + 1).padStart(2, '0')
  const day = String(now.getDate()).padStart(2, '0')
  return `${now.getFullYear()}-${month}-${day}`
}

function compressImage(filePath) {
  return new Promise((resolve) => {
    if (typeof wx.compressImage !== 'function') {
      resolve(filePath)
      return
    }
    wx.compressImage({
      src: filePath,
      quality: 82,
      success: ({ tempFilePath }) => resolve(tempFilePath || filePath),
      fail: () => resolve(filePath),
    })
  })
}

Page({
  data: {
    saving: false,
    savingText: '保存时光',
    photos: [],
    form: {
      date: today(),
      title: '',
      place: '',
      description: '',
    },
  },

  async onLoad() {
    const user = await getCurrentUser()
    if (!user || user.accessStatus !== 'approved' || user.role !== 'admin') {
      wx.showToast({ title: '仅管理员可添加', icon: 'none' })
      setTimeout(() => wx.navigateBack(), 500)
    }
  },

  onInput({ currentTarget, detail }) {
    this.setData({ [`form.${currentTarget.dataset.field}`]: detail.value })
  },

  onDateChange({ detail }) {
    this.setData({ 'form.date': detail.value })
  },

  choosePhotos() {
    if (this.data.saving || this.data.photos.length >= 9) return
    wx.chooseMedia({
      count: 9 - this.data.photos.length,
      mediaType: ['image'],
      sourceType: ['album', 'camera'],
      success: ({ tempFiles }) => {
        const additions = tempFiles.map((file) => ({
          path: file.tempFilePath,
          width: Number(file.width) || 0,
          height: Number(file.height) || 0,
        }))
        this.setData({ photos: [...this.data.photos, ...additions] })
      },
    })
  },

  removePhoto({ currentTarget }) {
    if (this.data.saving) return
    const index = Number(currentTarget.dataset.index)
    this.setData({ photos: this.data.photos.filter((_, itemIndex) => itemIndex !== index) })
  },

  previewPhoto({ currentTarget }) {
    const urls = this.data.photos.map((photo) => photo.path)
    wx.previewImage({ current: currentTarget.dataset.src, urls })
  },

  async saveMemory() {
    if (this.data.saving) return
    const title = this.data.form.title.trim()
    if (!title) {
      wx.showToast({ title: '请填写这段时光的标题', icon: 'none' })
      return
    }
    if (!this.data.photos.length) {
      wx.showToast({ title: '至少选择一张照片', icon: 'none' })
      return
    }

    this.setData({ saving: true, savingText: `正在上传 0/${this.data.photos.length}` })
    try {
      const images = []
      for (let index = 0; index < this.data.photos.length; index += 1) {
        this.setData({ savingText: `正在上传 ${index + 1}/${this.data.photos.length}` })
        const path = await compressImage(this.data.photos[index].path)
        images.push(await MemoryService.uploadImage(path))
      }
      this.setData({ savingText: '正在保存...' })
      const memory = await MemoryService.create({
        ...this.data.form,
        title,
        place: this.data.form.place.trim(),
        description: this.data.form.description.trim(),
        images,
      })
      if (typeof this.getOpenerEventChannel === 'function') {
        this.getOpenerEventChannel()?.emit?.('memoryCreated', memory)
      }
      wx.showToast({ title: '时光已收藏' })
      setTimeout(() => wx.navigateBack(), 450)
    } catch (error) {
      this.setData({ saving: false, savingText: '保存时光' })
      wx.showToast({ title: error.message || '保存失败', icon: 'none' })
    }
  },
})
