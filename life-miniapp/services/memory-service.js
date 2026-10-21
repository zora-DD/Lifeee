const { request } = require('./request')
const { MediaService, resolveMediaURL } = require('./media-service')

const API_PATHS = {
  list: '/v1/life-core/memory/list',
  create: '/v1/life-core/memory/create',
}

function formatDateLabel(value) {
  const parts = String(value || '').split('-')
  if (parts.length === 1) return `${parts[0]}年`
  if (parts.length === 2) return `${Number(parts[1])}月`
  return `${Number(parts[1])}月${Number(parts[2])}日`
}

function normalizeMemory(item) {
  return {
    ...item,
    year: String(item.date || '').slice(0, 4),
    dateLabel: formatDateLabel(item.date),
    images: Array.isArray(item.images) ? item.images.map((image) => ({
      ...image,
      mediaId: String(image.mediaId || image.media_id || ''),
      objectKey: image.objectKey || image.object_key || '',
      altText: image.altText || image.alt_text || '',
      sourceUrl: resolveMediaURL(image.url || ''),
      src: '',
      mode: 'aspectFill',
    })) : [],
  }
}

function list(filters = {}) {
  return request({
    path: API_PATHS.list,
    method: 'POST',
    data: {
      year: Number(filters.year) || undefined,
      page: filters.page || 1,
      page_size: filters.pageSize || 20,
    },
  }).then((result) => ({
    items: Array.isArray(result?.items) ? result.items.map(normalizeMemory) : [],
    years: Array.isArray(result?.years) ? result.years.map(String) : [],
    page: Number(result?.page) || 1,
    pageSize: Number(result?.pageSize || result?.page_size) || 20,
    total: Number(result?.total) || 0,
    hasMore: Boolean(result?.has_more ?? result?.hasMore),
  }))
}

function create(input) {
  return request({
    path: API_PATHS.create,
    method: 'POST',
    data: {
      date: input.date,
      place: input.place,
      title: input.title,
      description: input.description,
      images: input.images.map((image) => ({
        media_id: image.mediaId,
        object_key: image.objectKey,
        width: image.width,
        height: image.height,
        alt_text: image.altText || '',
      })),
    },
  }).then(normalizeMemory)
}

function uploadImage(filePath) {
  return MediaService.uploadImage(filePath, 'memory')
}

module.exports = {
  API_PATHS,
  MemoryService: { create, list, uploadImage },
  formatDateLabel,
  normalizeMemory,
}
