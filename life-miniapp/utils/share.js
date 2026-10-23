const DEFAULT_SHARE_TITLE = '狐小助｜一起记录任务、账单、吃喝与生活'
const DEFAULT_SHARE_IMAGE = '/assets/share/default.jpg'

function buildShareCard({ title = '', path = '/pages/index/index', imageUrl = '' } = {}) {
  const normalizedTitle = String(title || '').trim()
  return {
    title: normalizedTitle || DEFAULT_SHARE_TITLE,
    path,
    imageUrl: imageUrl || DEFAULT_SHARE_IMAGE,
  }
}

module.exports = {
  DEFAULT_SHARE_TITLE,
  DEFAULT_SHARE_IMAGE,
  buildShareCard,
}
