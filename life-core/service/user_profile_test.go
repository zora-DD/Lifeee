package service

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"

	"life-core/model"
)

func TestGeneratedUserProfile(t *testing.T) {
	Convey("同一 OpenID 始终生成同一套资料", t, func() {
		first := new(model.User)
		second := new(model.User)

		applyGeneratedProfile(first, "openid-stable-user")
		applyGeneratedProfile(second, "openid-stable-user")

		So(second.Nickname, ShouldEqual, first.Nickname)
		So(*second.AvatarEmoji, ShouldEqual, *first.AvatarEmoji)
		So(*second.AvatarColor, ShouldEqual, *first.AvatarColor)
		So(*second.Signature, ShouldEqual, *first.Signature)
	})

	Convey("不同 OpenID 会生成不同的有趣资料组合", t, func() {
		first := new(model.User)
		second := new(model.User)

		applyGeneratedProfile(first, "openid-user-a")
		applyGeneratedProfile(second, "openid-user-b")

		So(first.Nickname, ShouldNotBeEmpty)
		So(second.Nickname, ShouldNotBeEmpty)
		So(first.Nickname+*first.AvatarEmoji+*first.AvatarColor+*first.Signature,
			ShouldNotEqual,
			second.Nickname+*second.AvatarEmoji+*second.AvatarColor+*second.Signature)
	})
}
