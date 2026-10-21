package service

import (
	"crypto/sha256"
	"strings"

	"life-core/model"
)

const legacyDefaultNickname = "生活体验官"

var profileNamePrefixes = []string{
	"生活", "晚风", "星光", "云朵", "日落", "清风", "四季", "山野",
	"月亮", "银河", "晨光", "松林", "海边", "橘子", "雨后", "小岛",
}

var profileNameTitles = []string{
	"梦想家", "收藏家", "漫游者", "记录员", "探路人", "观察员",
	"行动派", "生活家", "寻光者", "计划师", "拾趣人", "散步者",
}

var profileEmojis = []string{
	"🌿", "🐳", "🍵", "☁️", "🐾", "🌙", "🍊", "🪴",
	"🐈", "🌻", "🐚", "🥐", "🎈", "🏕️", "🍃", "🐋",
}

var profileColors = []string{
	"#759B89", "#7394A4", "#8C9D79", "#88969F", "#AA8F75", "#7F91AD",
	"#A17F86", "#6F9A91", "#9A8F70", "#758FA0", "#8F829F", "#A08468",
}

var profileSignatures = []string{
	"心向往的地方，慢慢走也会抵达。",
	"把普通日子，过成喜欢的模样。",
	"认真生活，也记得偶尔发发呆。",
	"收藏烟火气，也收藏小欢喜。",
	"今天也去发现一点新鲜。",
	"慢慢记录，慢慢成为喜欢的自己。",
	"去见想见的人，也去看想看的风景。",
	"愿每个平常日子，都有值得记住的瞬间。",
	"保持好奇，生活总会冒出新故事。",
	"不赶时间，只认真感受生活。",
	"沿途有风景，日常也有答案。",
	"把今天过好，就是很棒的计划。",
}

// applyGeneratedProfile 根据微信 OpenID 稳定生成默认资料。
// 同一用户重复登录保持不变，不同字段使用不同摘要片段降低组合重复概率。
func applyGeneratedProfile(user *model.User, openID string) {
	if user == nil {
		return
	}
	digest := sha256.Sum256([]byte(strings.TrimSpace(openID)))
	prefixIndex := profileIndex(digest[0], digest[1], len(profileNamePrefixes))
	titleIndex := profileIndex(digest[2], digest[3], len(profileNameTitles))
	emojiIndex := profileIndex(digest[4], digest[5], len(profileEmojis))
	colorIndex := profileIndex(digest[6], digest[7], len(profileColors))
	signatureIndex := profileIndex(digest[8], digest[9], len(profileSignatures))

	user.Nickname = profileNamePrefixes[prefixIndex] + profileNameTitles[titleIndex]
	user.AvatarEmoji = optionalString(profileEmojis[emojiIndex])
	user.AvatarColor = optionalString(profileColors[colorIndex])
	user.Signature = optionalString(profileSignatures[signatureIndex])
}

func profileIndex(high, low byte, size int) int {
	return (int(high)<<8 | int(low)) % size
}
