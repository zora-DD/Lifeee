package service

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildHomeDish(t *testing.T) {
	Convey("家庭菜品字段校验", t, func() {
		row, err := buildHomeDish("番茄牛腩", "家常菜", "番茄、牛腩", "1. 焯水\n2. 炖煮", "normal", 45, "", "", "下次少放水", 5, 1)
		So(err, ShouldBeNil)
		So(row.Name, ShouldEqual, "番茄牛腩")
		So(row.Rating, ShouldEqual, uint32(5))
		So(row.Steps, ShouldContainSubstring, "炖煮")
		_, err = buildHomeDish("", "家常菜", "", "", "normal", 30, "", "", "", 0, 1)
		So(err, ShouldNotBeNil)
	})
}
func TestHomeDishDaysSince(t *testing.T) {
	Convey("未做过和久未制作获得更高间隔", t, func() {
		now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.Local)
		So(homeDishDaysSince(nil, now), ShouldEqual, 365)
		last := now.AddDate(0, 0, -8)
		So(homeDishDaysSince(&last, now), ShouldEqual, 8)
	})
}

func TestHomeDishStatus(t *testing.T) {
	Convey("制作次数自动对应想吃、吃过和常吃", t, func() {
		So(homeDishStatus(0), ShouldEqual, "wishlist")
		So(homeDishStatus(1), ShouldEqual, "eaten")
		So(homeDishStatus(2), ShouldEqual, "eaten")
		So(homeDishStatus(3), ShouldEqual, "frequent")
		So(validHomeDishStatus(""), ShouldBeTrue)
		So(validHomeDishStatus("unknown"), ShouldBeFalse)
	})
}
