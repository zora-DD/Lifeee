const assert = require('node:assert/strict')
const test = require('node:test')

test('账单统计不会把空记账人作为 uint64 发送', async () => {
  let captured
  global.wx = {
    getStorageSync: () => 'test-token',
    setStorageSync: () => {},
    removeStorageSync: () => {},
    request: (options) => {
      captured = options
      options.success({
        statusCode: 200,
        data: { code: 0, data: { expenseTotal: '0.00', incomeTotal: '0.00', billCount: 0 } },
      })
    },
  }

  const { BillService } = require('../services/bill-service')
  await BillService.stats({ createdBy: '' })

  assert.deepEqual(captured.data, {})
})

test('年度账单统计会发送年份和统计范围', async () => {
  let captured
  global.wx.request = (options) => {
    captured = options
    options.success({
      statusCode: 200,
      data: { code: 0, data: { scope: 'year', year: 2026, monthly: [], users: [], categories: [] } },
    })
  }

  const { BillService } = require('../services/bill-service')
  await BillService.stats({ scope: 'year', year: 2026 })

  assert.deepEqual(captured.data, { year: 2026, scope: 'year' })
})

test('月度账单统计会发送具体月份', async () => {
  let captured
  global.wx.request = (options) => {
    captured = options
    options.success({
      statusCode: 200,
      data: { code: 0, data: { scope: 'month', year: 2026, month: 8, categories: [], users: [] } },
    })
  }

  const { BillService } = require('../services/bill-service')
  await BillService.stats({ scope: 'month', year: 2026, month: 8 })

  assert.deepEqual(captured.data, { year: 2026, month: 8, scope: 'month' })
})

test('账单列表优先展示后端返回的记账人昵称', async () => {
  global.wx.request = (options) => options.success({
    statusCode: 200,
    data: {
      code: 0,
      data: {
        items: [{
          id: '12',
          title: '午饭',
          amount: '28.00',
          createdBy: '4',
          creator: { id: '4', nickname: '云朵漫游者', avatarEmoji: '☁️', avatarColor: '#8AA6B8' },
        }],
        creators: [],
      },
    },
  })

  const { BillService } = require('../services/bill-service')
  const result = await BillService.list()

  assert.equal(result.items[0].creator.nickname, '云朵漫游者')
  assert.notEqual(result.items[0].creator.nickname, '用户 4')
})

test('创建人资料缺失时不直接展示用户 ID', async () => {
  global.wx.request = (options) => options.success({
    statusCode: 200,
    data: { code: 0, data: { id: '13', title: '晚饭', amount: '36.00', createdBy: '4' } },
  })

  const { BillService } = require('../services/bill-service')
  const bill = await BillService.get('13')

  assert.equal(bill.creator.nickname, '未命名用户')
})
