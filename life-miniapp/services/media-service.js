const { API_BASE_URL, resolveApiPath } = require('../config/env')
const { getAccessToken } = require('./auth-session')
const { isUnauthorized, refreshAccessToken } = require('./request')

const API_PATHS = {
  upload: '/v1/life-core/media/upload',
}

function resolveMediaURL(value) {
  const url = String(value || '').trim()
  if (!url || !url.startsWith('/')) return url
  return `${API_BASE_URL}${url}`
}

function parseUploadResponse(statusCode, rawData) {
  let response
  try {
    response = typeof rawData === 'string' ? JSON.parse(rawData) : rawData
  } catch (error) {
    response = null
  }
  if (statusCode >= 200 && statusCode < 300 && response && response.code === 0) {
    return {
      ...response.data,
      url: resolveMediaURL(response.data && response.data.url),
    }
  }
  const uploadError = new Error(response?.msg || `图片上传失败 ${statusCode}`)
  uploadError.statusCode = statusCode
  uploadError.code = response?.code
  throw uploadError
}

function uploadOnce(filePath, bizType, accessToken) {
  return new Promise((resolve, reject) => {
    wx.uploadFile({
      url: `${API_BASE_URL}${resolveApiPath(API_PATHS.upload)}`,
      filePath,
      name: 'file',
      formData: { bizType },
      header: { Authorization: `Bearer ${accessToken}` },
      success: ({ statusCode, data }) => {
        try {
          resolve(parseUploadResponse(statusCode, data))
        } catch (error) {
          reject(error)
        }
      },
      fail: ({ errMsg }) => reject(new Error(errMsg || '图片上传失败')),
    })
  })
}

async function uploadImage(filePath, bizType) {
  let accessToken = getAccessToken()
  if (!accessToken) accessToken = await refreshAccessToken()
  try {
    return await uploadOnce(filePath, bizType, accessToken)
  } catch (error) {
    if (!isUnauthorized(error)) throw error
    accessToken = await refreshAccessToken()
    return uploadOnce(filePath, bizType, accessToken)
  }
}

module.exports = {
  API_PATHS,
  MediaService: { uploadImage },
  resolveMediaURL,
}
