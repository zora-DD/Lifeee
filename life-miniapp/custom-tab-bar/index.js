Component({
  data: {
    hidden: false,
    selected: 0,
    list: [
      { index: 0, pagePath: '/pages/tasks/index', text: '任务', key: 'task' },
      { index: 1, pagePath: '/pages/logs/index', text: '记录', key: 'log' },
      { index: 2, pagePath: '/pages/restaurants/index', text: '吃喝', key: 'food' },
      { index: 3, pagePath: '/pages/places/index', text: '地点', key: 'place' },
      { index: 4, pagePath: '/pages/index/index', text: '主页', key: 'profile' },
    ],
  },

  methods: {
    switchTab({ currentTarget }) {
      const { index, path } = currentTarget.dataset
      if (index === this.data.selected) {
        return
      }

      wx.switchTab({ url: path })
    },
  },
})
