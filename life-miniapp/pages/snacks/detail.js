const { SnackService } = require('../../services/snack-service')
const { getModuleAccess } = require('../../utils/access')
const { MODULES } = require('../../utils/permissions')

const statusMap = {
  wishlist: { label: '想吃', icon: '♡' }, eaten: { label: '吃过', icon: '✓' },
  frequent: { label: '常买', icon: '♥' }, avoid: { label: '避雷', icon: '!' },
}
function today(){const d=new Date();return `${d.getFullYear()}-${String(d.getMonth()+1).padStart(2,'0')}-${String(d.getDate()).padStart(2,'0')}`}
function toView(snack){return{...snack,statusInfo:statusMap[snack.status]||statusMap.wishlist,lastTastedText:snack.lastTastedAt||'还没有记录'}}

Page({
  data:{id:'',snack:null,canWrite:false,showTaste:false,saving:false,record:{tastedAt:today(),note:''}},
  async onLoad(options){this.setData({id:options.id});await this.load()},
  async onShow(){if(getApp().globalData.snackChanged){getApp().globalData.snackChanged=false;await this.load()}},
  async load(){try{const [snack,access]=await Promise.all([SnackService.detail(this.data.id),getModuleAccess(MODULES.RESTAURANT,'write')]);this.setData({snack:toView(snack),canWrite:Boolean(access.allowed)})}catch(error){wx.showToast({title:error.message||'加载失败',icon:'none'})}},
  edit(){wx.navigateTo({url:`/pages/snacks/form?id=${this.data.id}`})},
  noop(){},
  toggleTaste(){this.setData({showTaste:!this.data.showTaste})},
  input({currentTarget,detail}){this.setData({[`record.${currentTarget.dataset.field}`]:detail.value})},
  dateChange({detail}){this.setData({'record.tastedAt':detail.value})},
  async saveTaste(){if(this.data.saving)return;this.setData({saving:true});try{const snack=await SnackService.taste(this.data.id,this.data.record);this.setData({snack:toView(snack),showTaste:false,record:{tastedAt:today(),note:''}});getApp().globalData.snackChanged=true;wx.showToast({title:'已记录'})}catch(error){wx.showToast({title:error.message||'记录失败',icon:'none'})}finally{this.setData({saving:false})}},
  remove(){wx.showModal({title:'删除这条零食？',content:'相关品尝记录也会一并删除。',success:async({confirm})=>{if(!confirm)return;try{await SnackService.remove(this.data.id);getApp().globalData.snackChanged=true;wx.navigateBack()}catch(error){wx.showToast({title:error.message||'删除失败',icon:'none'})}}})},
})
