package proto

import (
	"encoding/json"
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAccessResponseMarshalJSON(t *testing.T) {
	Convey("用户与审批分页响应使用前端字段命名", t, func() {
		payload, err := json.Marshal(&ListAccessRequestsResponse{
			Items: []*UserItem{{Id: "8", AccessStatus: "pending"}},
			Page:  1, PageSize: 20, Total: 1,
		})

		So(err, ShouldBeNil)
		So(string(payload), ShouldContainSubstring, `"accessStatus":"pending"`)
		So(string(payload), ShouldContainSubstring, `"pageSize":20`)
		So(string(payload), ShouldContainSubstring, `"has_more":false`)
	})
}
