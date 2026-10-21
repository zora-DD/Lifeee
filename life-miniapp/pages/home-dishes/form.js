const { HomeDishService } = require('../../services/home-dish-service')
const { requireWriteAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')

const difficulties = [
  { value: 'easy', label: '简单' },
  { value: 'normal', label: '适中' },
  { value: 'hard', label: '挑战' },
]

function compressImage(filePath) {
  return new Promise((resolve) => {
    if (typeof wx.compressImage !== 'function') {
      resolve(filePath)
      return
    }
    wx.compressImage({
      src: filePath,
      quality: 65,
      success: ({ tempFilePath }) => resolve(tempFilePath || filePath),
      fail: () => resolve(filePath),
    })
  })
}

Page({
  data: {
    id: '',
    editing: false,
    saving: false,
    uploading: false,
    previewUrl: '',
    difficulties,
    difficultyIndex: 1,
    form: {
      name: '',
      category: '家常菜',
      ingredients: '',
      steps: '',
      difficulty: 'normal',
      estimatedMinutes: '30',
      coverUrl: '',
      coverObjectKey: '',
      experience: '',
    },
  },

  async onLoad(options) {
    const id = options.id || ''
    this.setData({ id, editing: Boolean(id) })
    wx.setNavigationBarTitle({ title: id ? '编辑菜品' : '新增家常菜' })
    const user = await requireWriteAccess(MODULES.RESTAURANT)
    if (!user) return
    if (id) await this.load()
  },

  async load() {
    try {
      const dish = await HomeDishService.detail(this.data.id)
      const difficultyIndex = Math.max(0, difficulties.findIndex((item) => item.value === dish.difficulty))
      this.setData({
        difficultyIndex,
        previewUrl: dish.coverUrl,
        form: {
          name: dish.name,
          category: dish.category,
          ingredients: dish.ingredients,
          steps: dish.steps || '',
          difficulty: dish.difficulty,
          estimatedMinutes: String(dish.estimatedMinutes || ''),
          coverUrl: dish.coverUrl,
          coverObjectKey: dish.coverObjectKey,
          experience: dish.experience || '',
        },
      })
    } catch (error) {
      wx.showToast({ title: error.message || '加载失败', icon: 'none' })
    }
  },

  input({ currentTarget, detail }) {
    this.setData({ [`form.${currentTarget.dataset.field}`]: detail.value })
  },

  difficultyChange({ detail }) {
    const difficultyIndex = Number(detail.value)
    this.setData({
      difficultyIndex,
      'form.difficulty': difficulties[difficultyIndex].value,
    })
  },

  chooseImage() {
    if (this.data.uploading) return
    wx.chooseMedia({
      count: 1,
      mediaType: ['image'],
      success: async ({ tempFiles }) => {
        const originalPath = tempFiles?.[0]?.tempFilePath
        if (!originalPath) return
        this.setData({ uploading: true })
        try {
          const imagePath = await compressImage(originalPath)
          this.setData({ previewUrl: imagePath })
          const result = await HomeDishService.uploadCover(imagePath)
          this.setData({
            'form.coverUrl': result.url,
            'form.coverObjectKey': result.objectKey,
          })
          wx.showToast({ title: '图片已上传' })
        } catch (error) {
          this.setData({ previewUrl: this.data.form.coverUrl })
          wx.showToast({ title: error.message || '上传失败', icon: 'none' })
        } finally {
          this.setData({ uploading: false })
        }
      },
    })
  },

  async save() {
    if (this.data.saving) return
    if (this.data.uploading) {
      wx.showToast({ title: '请等待图片上传完成', icon: 'none' })
      return
    }
    if (!this.data.form.name.trim() || !this.data.form.category.trim()) {
      wx.showToast({ title: '请填写菜名和分类', icon: 'none' })
      return
    }
    this.setData({ saving: true })
    try {
      if (this.data.editing) await HomeDishService.update(this.data.id, this.data.form)
      else await HomeDishService.create(this.data.form)
      wx.showToast({ title: '已保存' })
      getApp().globalData.homeDishChanged = true
      setTimeout(() => wx.navigateBack(), 350)
    } catch (error) {
      wx.showToast({ title: error.message || '保存失败', icon: 'none' })
    } finally {
      this.setData({ saving: false })
    }
  },
})
