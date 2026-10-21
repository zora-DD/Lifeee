package service

import (
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
)

func TestBuildSnack(t *testing.T) {
	Convey("零食字段和状态校验", t, func() {
		row, err := buildSnack("大豫竹方便面", "大豫竹", "便利店", "方便速食", "eaten", "", "", "童年的味道", "童年味道", 1)
		So(err, ShouldBeNil)
		So(row.Name, ShouldEqual, "大豫竹方便面")
		So(row.Status, ShouldEqual, "eaten")
		So(value(row.MemoryTag), ShouldEqual, "童年味道")

		_, err = buildSnack("棒棒娃", "", "山姆", "肉干卤味", "unknown", "", "", "", "", 1)
		So(err, ShouldNotBeNil)
	})
}

func TestSnackDaysSince(t *testing.T) {
	Convey("从未吃过和很久没吃的零食具有更长间隔", t, func() {
		now := time.Date(2026, 10, 4, 0, 0, 0, 0, time.Local)
		So(snackDaysSince(nil, now), ShouldEqual, 365)
		last := now.AddDate(0, 0, -12)
		So(snackDaysSince(&last, now), ShouldEqual, 12)
	})
}

func TestSnackStatusValidation(t *testing.T) {
	Convey("零食状态仅允许想吃、吃过、常买和避雷", t, func() {
		So(validSnackStatus("wishlist", false), ShouldBeTrue)
		So(validSnackStatus("eaten", false), ShouldBeTrue)
		So(validSnackStatus("frequent", false), ShouldBeTrue)
		So(validSnackStatus("avoid", false), ShouldBeTrue)
		So(validSnackStatus("", true), ShouldBeTrue)
		So(validSnackStatus("unknown", true), ShouldBeFalse)
	})
}
