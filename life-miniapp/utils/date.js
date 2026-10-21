function pad(value) {
  return String(value).padStart(2, '0')
}

function formatDateTime(value) {
  if (!value) return '未设置'
  const date = new Date(value)
  return `${date.getMonth() + 1}月${date.getDate()}日 ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function formatDateTimeWithWeekday(value) {
  if (!value) return '未设置'
  const date = new Date(value)
  const weekday = ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()]
  return `${date.getMonth() + 1}月${date.getDate()}日 ${weekday} · ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function formatFullDateTime(value) {
  if (!value) return '未设置'
  const date = new Date(value)
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function toPickerDate(value) {
  const date = value ? new Date(value) : new Date()
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())}`
}

function toPickerTime(value) {
  const date = value ? new Date(value) : new Date()
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

function pickerValuesToISO(dateValue, timeValue) {
  const [year, month, day] = dateValue.split('-').map(Number)
  const [hour, minute] = timeValue.split(':').map(Number)
  return new Date(year, month - 1, day, hour, minute).toISOString()
}

function getWeekdayLabel(dateValue) {
  const [year, month, day] = String(dateValue || '').split('-').map(Number)
  if (!year || !month || !day) return ''
  const date = new Date(year, month - 1, day)
  if (Number.isNaN(date.getTime())) return ''
  return ['周日', '周一', '周二', '周三', '周四', '周五', '周六'][date.getDay()]
}

module.exports = {
  formatDateTime,
  formatDateTimeWithWeekday,
  formatFullDateTime,
  toPickerDate,
  toPickerTime,
  pickerValuesToISO,
  getWeekdayLabel,
}
