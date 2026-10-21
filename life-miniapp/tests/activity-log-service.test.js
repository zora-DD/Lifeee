const assert = require('node:assert/strict')
const test = require('node:test')

test('生活记录展示后端返回的创建人昵称和头像', async () => {
  global.wx = {
    getStorageSync: (key) => key === 'life_access_token' ? 'test-token' : '',
    setStorageSync: () => {},
    removeStorageSync: () => {},
    request: (options) => options.success({
      statusCode: 200,
      data: {
        code: 0,
        data: {
          id: '9',
          content: '整理房间',
          createdBy: '4',
          creator: {
            id: '4',
            nickname: '云朵漫游者',
            avatarEmoji: '☁️',
            avatarColor: '#8AA6B8',
          },
        },
      },
    }),
  }

  const { ActivityLogService } = require('../services/activity-log-service')
  const log = await ActivityLogService.get('9')

  assert.equal(log.creator.name, '云朵漫游者')
  assert.equal(log.creator.avatarEmoji, '☁️')
  assert.equal(log.creator.avatarColor, '#8AA6B8')
  assert.notEqual(log.creator.name, '用户 4')
})
