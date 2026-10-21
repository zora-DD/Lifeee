const assert = require('node:assert/strict')
const test = require('node:test')

test('家常菜接口字段映射与推荐参数正确', async (t) => {
  const originalWx=global.wx;const captured=[];global.wx={getStorageSync:()=> 'token',setStorageSync:()=>{},removeStorageSync:()=>{},request(o){captured.push(o);o.success({statusCode:200,data:{code:0,data:{id:'7',name:'番茄牛腩',estimated_minutes:45,cook_count:3,last_cooked_at:'2026-10-01',records:[]}}})},uploadFile(){}}
  t.after(()=>{global.wx=originalWx})
  delete require.cache[require.resolve('../services/home-dish-service')]
  const {HomeDishService}=require('../services/home-dish-service')
  const dish=await HomeDishService.recommend({category:'家常菜',difficulty:'normal',maxMinutes:60,ingredients:'番茄 牛腩'})
  assert.equal(captured[0].url.endsWith('/v1/restaurant/home/recommend'),true)
  assert.equal(captured[0].data.ingredients,'番茄 牛腩')
  assert.equal(dish.estimatedMinutes,45)
  assert.equal(dish.cookCount,3)
})

test('家常菜图片相对地址会补齐 API 域名', async (t) => {
  const originalWx=global.wx;global.wx={getStorageSync:()=> 'token',setStorageSync:()=>{},removeStorageSync:()=>{},request(o){o.success({statusCode:200,data:{code:0,data:{id:'7',name:'番茄牛腩',cover_url:'/media/home_dish/demo.jpg?expires=1&signature=test',records:[{id:'9',photo_url:'/media/home_dish/record.jpg?expires=1&signature=test'}]}}})},uploadFile(){}}
  t.after(()=>{global.wx=originalWx})
  delete require.cache[require.resolve('../services/home-dish-service')]
  const {HomeDishService}=require('../services/home-dish-service')
  const dish=await HomeDishService.detail('7')
  assert.match(dish.coverUrl,/^https:\/\//)
  assert.match(dish.coverUrl,/\/media\/home_dish\/demo\.jpg/)
  assert.match(dish.records[0].photoUrl,/^https:\/\//)
})

test('家常菜创建请求包含人工填写的制作步骤且不再保存综合评价', async (t) => {
  const originalWx=global.wx;const captured=[];global.wx={getStorageSync:()=> 'token',setStorageSync:()=>{},removeStorageSync:()=>{},request(o){captured.push(o);o.success({statusCode:200,data:{code:0,data:{id:'7',name:'番茄炒蛋',steps:'1. 切番茄\n2. 炒鸡蛋'}}})},uploadFile(){}}
  t.after(()=>{global.wx=originalWx})
  delete require.cache[require.resolve('../services/home-dish-service')]
  const {HomeDishService}=require('../services/home-dish-service')
  await HomeDishService.create({name:'番茄炒蛋',category:'家常菜',steps:'1. 切番茄\n2. 炒鸡蛋',rating:5})
  assert.equal(captured[0].data.steps,'1. 切番茄\n2. 炒鸡蛋')
  assert.equal(captured[0].data.rating,0)
})

test('家庭菜品页面已注册',()=>{const app=require('../app.json');assert.ok(app.pages.includes('pages/home-dishes/detail'));assert.ok(app.pages.includes('pages/home-dishes/form'))})

test('没有家常菜时使用友好引导而不是原始接口错误',()=>{const fs=require('node:fs');const source=fs.readFileSync(require.resolve('../pages/restaurants/index.js'),'utf8');assert.match(source,/还没有家常菜/);assert.match(source,/新增一道/);assert.match(source,/当前筛选下没有合适的菜/)})

test('家里做的分类和状态使用同一行页面内下拉框',()=>{const fs=require('node:fs');const wxml=fs.readFileSync(require.resolve('../pages/restaurants/index.wxml'),'utf8');assert.match(wxml,/home-filter-row/);assert.match(wxml,/home-category-menu/);assert.match(wxml,/home-status-menu/);assert.match(wxml,/showHomeCategoryMenu\|\|showHomeStatusMenu/);assert.doesNotMatch(wxml,/home-status-tabs/);assert.doesNotMatch(wxml,/最多用时/);assert.doesNotMatch(wxml,/现有食材/);assert.doesNotMatch(wxml,/全部难度/)})

test('家里做支持想吃、吃过和常吃状态筛选',()=>{const fs=require('node:fs');const js=fs.readFileSync(require.resolve('../pages/restaurants/index.js'),'utf8');const wxml=fs.readFileSync(require.resolve('../pages/restaurants/index.wxml'),'utf8');assert.match(js,/wishlist.*想吃/);assert.match(js,/eaten.*吃过/);assert.match(js,/frequent.*常吃/);assert.match(js,/status:homeStatuses\[this\.data\.homeStatusIndex\]\.value/);assert.match(js,/toggleHomeStatusMenu/);assert.match(wxml,/home-status-menu/);assert.match(wxml,/status-\{\{item\.statusInfo\.value\}\}/)})

test('家里做帮我选使用弹窗展示推荐结果',()=>{const fs=require('node:fs');const js=fs.readFileSync(require.resolve('../pages/restaurants/index.js'),'utf8');const wxml=fs.readFileSync(require.resolve('../pages/restaurants/index.wxml'),'utf8');assert.match(js,/showHomeRecommendation:true/);assert.match(js,/closeHomeRecommendation/);assert.match(js,/viewHomeRecommendation/);assert.match(wxml,/wx:if="\{\{showHomeRecommendation\}\}"/);assert.match(wxml,/再选一次/);assert.match(wxml,/查看详情/);assert.doesNotMatch(wxml,/class="home-result card"/)})

test('家常菜表单支持人工步骤且移除 AI 识别和综合评价',()=>{const fs=require('node:fs');const js=fs.readFileSync(require.resolve('../pages/home-dishes/form.js'),'utf8');const form=fs.readFileSync(require.resolve('../pages/home-dishes/form.wxml'),'utf8');const detail=fs.readFileSync(require.resolve('../pages/home-dishes/detail.wxml'),'utf8');assert.match(js,/steps:\s*''/);assert.match(form,/data-field="steps"/);assert.match(detail,/step-list/);assert.doesNotMatch(form,/AI 识别/);assert.doesNotMatch(js,/recognizeImage/);assert.doesNotMatch(form,/综合评价/);assert.doesNotMatch(detail,/综合评价/)})
