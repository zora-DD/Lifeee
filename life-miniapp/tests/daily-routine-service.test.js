const assert = require('node:assert/strict')
const fs = require('node:fs')
const path = require('node:path')
const test = require('node:test')

test('习惯打卡接口支持频率、列表、编辑和幂等打卡', async (t) => {
  const originalWx = global.wx
  const requests = []
  global.wx = {
    getStorageSync: () => 'test-token',
    setStorageSync: () => {},
    removeStorageSync: () => {},
    request(options) {
      requests.push(options)
      options.success({
        statusCode: 200,
        data: {
          code: 0,
          data: options.url.endsWith('/list')
            ? {
                date: '2026-09-20',
                items: [{
                  id: '9',
                  title: '抹药',
                  time_of_day: '08:00',
                  repeat_unit: 'day',
                  repeat_interval: 2,
                  start_date: '2026-09-18',
                  enabled: true,
                  scheduled_today: true,
                  checked_today: false,
                  current_streak: 3,
                  total_checkins: 12,
                  last_seven_day_rate: 71,
                  history: [],
                  month_history: [{ date: '2026-09-20', completed: true, scheduled: true }],
                }],
              }
            : { id: '9', title: '抹药', time_of_day: '08:00', enabled: true, checked_today: true, history: [] },
        },
      })
    },
  }
  t.after(() => { global.wx = originalWx })

  const { DailyRoutineService } = require('../services/daily-routine-service')
  const list = await DailyRoutineService.list({ date: '2026-09-20' })
  const checked = await DailyRoutineService.checkin('9', '2026-09-20', true)
  await DailyRoutineService.update('9', {
    title: '抹药', timeOfDay: '08:30', repeatUnit: 'week', repeatInterval: 1,
    startDate: '2026-09-20', enabled: false,
  })

  assert.equal(list.items[0].id, '9')
  assert.equal(list.items[0].timeOfDay, '08:00')
  assert.equal(list.items[0].currentStreak, 3)
  assert.equal(list.items[0].totalCheckins, 12)
  assert.equal(list.items[0].last7DayRate, 71)
  assert.equal(list.items[0].repeatUnit, 'day')
  assert.equal(list.items[0].repeatInterval, 2)
  assert.equal(list.items[0].startDate, '2026-09-18')
  assert.equal(list.items[0].scheduledToday, true)
  assert.equal(list.items[0].monthHistory[0].scheduled, true)
  assert.equal(list.items[0].monthHistory.length, 1)
  assert.equal(checked.checkedToday, true)
  assert.deepEqual(requests[0].data, { date: '2026-09-20', include_disabled: false })
  assert.deepEqual(requests[1].data, { id: '9', date: '2026-09-20', completed: true })
  assert.equal(requests[2].data.time_of_day, '08:30')
  assert.equal(requests[2].data.repeat_unit, 'week')
  assert.equal(requests[2].data.repeat_interval, 1)
  assert.equal(requests[2].data.start_date, '2026-09-20')
  assert.equal(requests[2].data.timeOfDay, undefined)
  assert.equal(requests[2].data.enabled, false)
})

test('任务首页将习惯打卡和任务日历作为同级入口', () => {
  const taskPage = fs.readFileSync(path.join(__dirname, '../pages/tasks/index.wxml'), 'utf8')
  const routinePageStyle = fs.readFileSync(path.join(__dirname, '../pages/tasks/routines.wxss'), 'utf8')

  assert.match(taskPage, /task-heading-actions[\s\S]*任务日历[\s\S]*习惯打卡/)
  assert.doesNotMatch(taskPage, /class="routine-card/)
  assert.match(routinePageStyle, /\.empty-card/)
  assert.match(routinePageStyle, /\.empty-title/)
})

test('习惯打卡页面支持灵活频率、今日筛选与整体总览', () => {
  const routinePage = fs.readFileSync(path.join(__dirname, '../pages/tasks/routines.wxml'), 'utf8')
  const routineScript = fs.readFileSync(path.join(__dirname, '../pages/tasks/routines.js'), 'utf8')

  assert.match(routinePage, /class="routine-checkin/)
  assert.match(routinePage, /bindtap="toggleCheckin"/)
  assert.match(routineScript, /DailyRoutineService\.checkin/)
  assert.match(routineScript, /今天已完成/)
  assert.match(routinePage, /连续次数/)
  assert.match(routinePage, /累计打卡/)
  assert.match(routinePage, /查看本月打卡日历/)
  assert.match(routinePage, /重复频率/)
  assert.match(routinePage, /个活跃习惯/)
  assert.match(routinePage, /次本月打卡/)
  assert.match(routinePage, /本月完成率/)
  assert.match(routinePage, /data-scope="today"/)
  assert.match(routinePage, /data-scope="all"/)
  assert.match(routineScript, /每 2 天/)
  assert.match(routineScript, /每周/)
  assert.match(routineScript, /每月/)
})
