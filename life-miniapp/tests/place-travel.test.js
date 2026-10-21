const assert = require('node:assert/strict')
const test = require('node:test')

const { buildTravelMeta, formatDriveDuration } = require('../utils/place-travel')

test('自驾分钟转换为易读时间', () => {
  assert.equal(formatDriveDuration(45), '45分钟')
  assert.equal(formatDriveDuration(240), '4小时')
  assert.equal(formatDriveDuration(270), '4小时30分')
})

test('地点列表组合上海出发的紧凑信息', () => {
  assert.equal(
    buildTravelMeta({ distanceKm: 320, driveDurationMinutes: 270 }),
    '上海出发 · 约320km · 自驾约4小时30分',
  )
  assert.equal(buildTravelMeta({}), '')
})
