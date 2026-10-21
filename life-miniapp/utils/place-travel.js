function formatDriveDuration(value) {
  const minutes = Math.max(0, Math.round(Number(value) || 0))
  if (!minutes) return ''
  const hours = Math.floor(minutes / 60)
  const restMinutes = minutes % 60
  if (!hours) return `${restMinutes}分钟`
  if (!restMinutes) return `${hours}小时`
  return `${hours}小时${restMinutes}分`
}

function buildTravelMeta(place = {}) {
  const parts = ['上海出发']
  const distanceKM = Math.max(0, Number(place.distanceKm) || 0)
  const duration = formatDriveDuration(place.driveDurationMinutes)
  if (distanceKM) parts.push(`约${distanceKM}km`)
  if (duration) parts.push(`自驾约${duration}`)
  return parts.length > 1 ? parts.join(' · ') : ''
}

module.exports = { buildTravelMeta, formatDriveDuration }
