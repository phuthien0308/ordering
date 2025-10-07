package http

import (
	"fmt"
	"io"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestJsonReader(t *testing.T) {

	convey.Convey("given a json reader", t, func() {
		data := struct {
			FirstName string `json:"FirstName"`
		}{FirstName: "testing"}
		jsonReader, err := NewJsonReader(data)
		convey.So(err, convey.ShouldBeNil)
		convey.Convey("should return a json string", func() {
			data, err := io.ReadAll(jsonReader)
			if err != nil {
				panic(err)
			}
			fmt.Println(data)
			convey.So(err, convey.ShouldBeNil)
		})
	})
}
