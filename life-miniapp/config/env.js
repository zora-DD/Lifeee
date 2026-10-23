// 可临时指定 'local' 或 'prod'；留空时按微信当前小程序版本自动选择。
// develop 环境走本机后端，trial/release 和非微信测试环境走 HTTPS 生产域名。
const ENV_OVERRIDE = 'prod'

function getWeChatEnvVersion() {
  if (typeof wx === 'undefined' || typeof wx.getAccountInfoSync !== 'function') return ''
  try {
    return wx.getAccountInfoSync()?.miniProgram?.envVersion || ''
  } catch (error) {
    return ''
  }
}

function resolveEnvironment(envVersion) {
  return envVersion === 'develop' ? 'local' : 'prod'
}

const ENV = ENV_OVERRIDE || resolveEnvironment(getWeChatEnvVersion())

// lifeee.zoradu.cn 完成 DNS、备案和 HTTPS 配置后用于正式环境。
const PROD_API_BASE_URL = 'https://lifeee.zoradu.cn'
const GATEWAY_PATH_PREFIX = '/life-core'
const API_VERSION_PREFIX = '/v1'

const ENV_CONFIG = {
  local: {
    apiBaseUrl: 'http://127.0.0.1:8081',
    // 本地服务也使用 /v1/life-core/... 路径。
    apiPathPrefix: '/life-core',
  },
  prod: {
    apiBaseUrl: PROD_API_BASE_URL,
    // 线上网关按服务区分，实际请求 /v1/life-core/auth/login。
    apiPathPrefix: GATEWAY_PATH_PREFIX,
  },
}

const currentConfig = ENV_CONFIG[ENV]

if (!currentConfig) {
  throw new Error(`未知的运行环境：${ENV}`)
}

function resolveApiPathWithPrefix(path, prefix) {
  const normalized = String(path || '')
  const gatewayPrefixed = `${API_VERSION_PREFIX}${GATEWAY_PATH_PREFIX}/`
  const versionPrefixed = `${API_VERSION_PREFIX}/`

  if (prefix) {
    if (normalized.startsWith(gatewayPrefixed) || !normalized.startsWith(versionPrefixed)) {
      return normalized
    }
    return `${API_VERSION_PREFIX}${prefix}${normalized.slice(API_VERSION_PREFIX.length)}`
  }

  if (normalized.startsWith(gatewayPrefixed)) {
    return `${versionPrefixed}${normalized.slice(gatewayPrefixed.length)}`
  }
  return normalized
}

function resolveApiPath(path) {
  return resolveApiPathWithPrefix(path, currentConfig.apiPathPrefix)
}

module.exports = {
  ENV,
  ENV_CONFIG,
  API_BASE_URL: currentConfig.apiBaseUrl,
  API_PATH_PREFIX: currentConfig.apiPathPrefix,
  resolveEnvironment,
  resolveApiPath,
  resolveApiPathWithPrefix,
}
