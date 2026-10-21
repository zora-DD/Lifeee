const { SnackService } = require('../../services/snack-service')
const { requireWriteAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')

const categories = ['肉干卤味', '坚果炒货', '方便速食', '糖果巧克力', '饼干糕点', '饮料乳品', '其他']
const statuses = [
  { value: 'wishlist', label: '想吃' },
  { value: 'eaten', label: '吃过' },
  { value: 'frequent', label: '常买' },
  { value: 'avoid', label: '避雷' },
]

function compressImage(filePath) {
  return new Promise((resolve) => {
    if (typeof wx.compressImage !== 'function') return resolve(filePath)
    wx.compressImage({ src: filePath, quality: 72, success: ({ tempFilePath }) => resolve(tempFilePath || filePath), fail: () => resolve(filePath) })
  })
}

Page({
  data: {
    id: '', editing: false, saving: false, uploading: false,
    categories, statuses, categoryIndex: 0, statusIndex: 0,
    showCategoryMenu: false, showStatusMenu: false, previewUrl: '',
    form: { name: '', brand: '', purchasePlace: '', category: categories[0], status: statuses[0].value, coverUrl: '', coverObjectKey: '', note: '', memoryTag: '' },
  },
  async onLoad(options) {
    const id = options.id || ''
    this.setData({ id, editing: Boolean(id) })
    wx.setNavigationBarTitle({ title: id ? '编辑零食' : '新增零食' })
    if (!await requireWriteAccess(MODULES.RESTAURANT)) return
    if (id) await this.load()
  },
  async load() {
    try {
      const snack = await SnackService.detail(this.data.id)
      const categoryIndex = Math.max(0, categories.indexOf(snack.category))
      const statusIndex = Math.max(0, statuses.findIndex((item) => item.value === snack.status))
      this.setData({ categoryIndex, statusIndex, previewUrl: snack.coverUrl, form: { name: snack.name, brand: snack.brand, purchasePlace: snack.purchasePlace, category: snack.category, status: snack.status, coverUrl: snack.coverUrl, coverObjectKey: snack.coverObjectKey, note: snack.note || '', memoryTag: snack.memoryTag || '' } })
    } catch (error) {
      wx.showToast({ title: error.message || '加载失败', icon: 'none' })
    }
  },
  input({ currentTarget, detail }) { this.setData({ [`form.${currentTarget.dataset.field}`]: detail.value }) },
  toggleCategoryMenu() { this.setData({ showCategoryMenu: !this.data.showCategoryMenu, showStatusMenu: false }) },
  toggleStatusMenu() { this.setData({ showStatusMenu: !this.data.showStatusMenu, showCategoryMenu: false }) },
  closeMenus() { this.setData({ showCategoryMenu: false, showStatusMenu: false }) },
  selectCategory({ currentTarget }) { const categoryIndex = Number(currentTarget.dataset.index); this.setData({ categoryIndex, showCategoryMenu: false, 'form.category': categories[categoryIndex] }) },
  selectStatus({ currentTarget }) { const statusIndex = Number(currentTarget.dataset.index); this.setData({ statusIndex, showStatusMenu: false, 'form.status': statuses[statusIndex].value }) },
  chooseImage() {
    if (this.data.uploading) return
    wx.chooseMedia({ count: 1, mediaType: ['image'], success: async ({ tempFiles }) => {
      const originalPath = tempFiles?.[0]?.tempFilePath
      if (!originalPath) return
      this.setData({ uploading: true })
      try {
        const imagePath = await compressImage(originalPath)
        this.setData({ previewUrl: imagePath })
        const result = await SnackService.uploadCover(imagePath)
        this.setData({ 'form.coverUrl': result.url, 'form.coverObjectKey': result.objectKey })
        wx.showToast({ title: '图片已上传' })
      } catch (error) {
        this.setData({ previewUrl: this.data.form.coverUrl })
        wx.showToast({ title: error.message || '上传失败', icon: 'none' })
      } finally { this.setData({ uploading: false }) }
    } })
  },
  async save() {
    if (this.data.saving) return
    if (this.data.uploading) return wx.showToast({ title: '请等待图片上传完成', icon: 'none' })
    if (!this.data.form.name.trim()) return wx.showToast({ title: '请填写零食名称', icon: 'none' })
    this.setData({ saving: true })
    try {
      if (this.data.editing) await SnackService.update(this.data.id, this.data.form)
      else await SnackService.create(this.data.form)
      getApp().globalData.snackChanged = true
      wx.showToast({ title: '已保存' })
      setTimeout(() => wx.navigateBack(), 350)
    } catch (error) {
      wx.showToast({ title: error.message || '保存失败', icon: 'none' })
    } finally { this.setData({ saving: false }) }
  },
})
