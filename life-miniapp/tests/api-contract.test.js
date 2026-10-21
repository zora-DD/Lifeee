const assert = require('node:assert/strict')
const test = require('node:test')

const { API_PATHS: taskPaths } = require('../services/task-service')
const { API_PATHS: dailyRoutinePaths } = require('../services/daily-routine-service')
const { API_PATHS: activityPaths } = require('../services/activity-log-service')
const { API_PATHS: restaurantPaths } = require('../services/restaurant-service')
const { API_PATHS: placePaths } = require('../services/place-service')
const { API_PATHS: profilePaths } = require('../services/profile-service')
const { API_PATHS: favoritePaths } = require('../services/favorite-service')
const { API_PATHS: billPaths } = require('../services/bill-service')
const { API_PATHS: aiSummaryPaths } = require('../services/ai-summary-service')
const { API_PATHS: mediaPaths } = require('../services/media-service')
const { API_PATHS: homeDishPaths } = require('../services/home-dish-service')
const { API_PATHS: snackPaths } = require('../services/snack-service')

test('业务接口路径以 /v1/life-core 为规范写法，由环境配置决定是否保留网关前缀', () => {
  const implementedPaths = [
    ...Object.values(taskPaths),
    ...Object.values(dailyRoutinePaths),
    ...Object.values(activityPaths),
    ...Object.values(restaurantPaths),
    ...Object.values(placePaths),
    profilePaths.login,
    profilePaths.me,
    profilePaths.contributions,
    profilePaths.accessStatus,
    profilePaths.applyAccess,
    profilePaths.adminRequests,
    profilePaths.reviewAccess,
    ...Object.values(favoritePaths),
    ...Object.values(billPaths),
    ...Object.values(aiSummaryPaths),
    ...Object.values(mediaPaths),
    ...Object.values(homeDishPaths),
    ...Object.values(snackPaths),
  ]

  assert.ok(implementedPaths.length > 0)
  implementedPaths.forEach((path) => assert.match(path, /^\/v1\/life-core\//))
  // 后端另有健康检查接口，不由业务页面调用。
  assert.equal(new Set(implementedPaths).size, 69)
  assert.equal(taskPaths.recognizeVoice, '/v1/life-core/task/voice/recognize')
  assert.equal(billPaths.recognizeVoice, '/v1/life-core/record/voice/recognize')
  assert.equal(aiSummaryPaths.taskSummary, '/v1/life-core/task/ai_summary')
  assert.equal(aiSummaryPaths.lifeSummary, '/v1/life-core/report/life_summary')
})
