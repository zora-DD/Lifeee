const { request } = require('./request')
const { MediaService, resolveMediaURL } = require('./media-service')

const API_PATHS = {
  list: '/v1/life-core/restaurant/home/list', detail: '/v1/life-core/restaurant/home/detail',
  create: '/v1/life-core/restaurant/home/create', update: '/v1/life-core/restaurant/home/modify',
  remove: '/v1/life-core/restaurant/home/delete', cook: '/v1/life-core/restaurant/home/cook',
  recommend: '/v1/life-core/restaurant/home/recommend',
}

function normalizeRecord(item = {}) { return { ...item, id:String(item.id||''),dishId:String(item.dishId||item.dish_id||''),cookedBy:String(item.cookedBy||item.cooked_by||''),cookedAt:item.cookedAt||item.cooked_at||'',durationMinutes:Number(item.durationMinutes??item.duration_minutes)||0,photoUrl:resolveMediaURL(item.photoUrl||item.photo_url||''),photoObjectKey:item.photoObjectKey||item.photo_object_key||'' } }
function normalizeDish(item = {}) { const cookCount=Number(item.cookCount??item.cook_count)||0;return { ...item,id:String(item.id||''),estimatedMinutes:Number(item.estimatedMinutes??item.estimated_minutes)||0,coverUrl:resolveMediaURL(item.coverUrl||item.cover_url||''),coverObjectKey:item.coverObjectKey||item.cover_object_key||'',steps:item.steps||'',cookCount,status:item.status||(cookCount===0?'wishlist':cookCount>=3?'frequent':'eaten'),lastCookedAt:item.lastCookedAt||item.last_cooked_at||'',createdBy:String(item.createdBy||item.created_by||''),records:(item.records||[]).map(normalizeRecord) } }
function payload(input={}){return{name:input.name||'',category:input.category||'',ingredients:input.ingredients||'',steps:input.steps||'',difficulty:input.difficulty||'normal',estimated_minutes:Number(input.estimatedMinutes)||0,cover_url:input.coverUrl||'',cover_object_key:input.coverObjectKey||'',experience:input.experience||'',rating:0}}

const HomeDishService={
  list(filters={}){return request({path:API_PATHS.list,method:'POST',data:{category:filters.category||'',difficulty:filters.difficulty||'',max_minutes:Number(filters.maxMinutes)||0,keyword:filters.keyword||'',status:filters.status||'',page:filters.page||1,page_size:filters.pageSize||20}}).then(r=>({...r,items:(r.items||[]).map(normalizeDish),pageSize:Number(r.pageSize||r.page_size)||20,hasMore:Boolean(r.hasMore??r.has_more)}))},
  detail(id){return request({path:API_PATHS.detail,method:'POST',data:{id:String(id)}}).then(normalizeDish)},
  create(input){return request({path:API_PATHS.create,method:'POST',data:payload(input)}).then(normalizeDish)},
  update(id,input){return request({path:API_PATHS.update,method:'POST',data:{id:String(id),...payload(input)}}).then(normalizeDish)},
  remove(id){return request({path:API_PATHS.remove,method:'POST',data:{id:String(id)}})},
  cook(id,input={}){return request({path:API_PATHS.cook,method:'POST',data:{dish_id:String(id),cooked_at:input.cookedAt||'',duration_minutes:Number(input.durationMinutes)||0,rating:Number(input.rating)||0,note:input.note||'',photo_url:input.photoUrl||'',photo_object_key:input.photoObjectKey||''}}).then(normalizeDish)},
  recommend(filters={}){return request({path:API_PATHS.recommend,method:'POST',data:{category:filters.category||'',difficulty:filters.difficulty||'',max_minutes:Number(filters.maxMinutes)||0,ingredients:filters.ingredients||'',status:filters.status||''}}).then(normalizeDish)},
  uploadCover(filePath){return MediaService.uploadImage(filePath,'home_dish')},
}
module.exports={API_PATHS,HomeDishService,normalizeDish}
