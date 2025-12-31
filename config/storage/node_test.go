package storage

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/phuthien0308/ordering/config/dto"
	"github.com/redis/go-redis/v9"

	. "github.com/smartystreets/goconvey/convey"
)

func TestAddNode(t *testing.T) {
	Convey("give miniredis", t, func() {
		miniR := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: miniR.Addr()})

		Convey("add new node", func() {
			storage := NewAddressStorage(client)
			storage.Add(context.Background(), "ut", "127.0.0.1")
			storage.Add(context.Background(), "ut", "127.0.0.2")
			result, err := miniR.SMembers("ut")
			Convey("the result should be as expected", func() {
				So(err, ShouldBeNil)
				So(len(result), ShouldEqual, 2)
			})
			Convey("remove node", func() {
				err = storage.Remove(context.Background(), "ut", "127.0.0.1")
				Convey("the key should be empty", func() {
					So(err, ShouldBeNil)
					result, _ := miniR.SMembers("ut")
					So(len(result), ShouldEqual, 1)
				})
			})
		})

	})

}

func TestGetAllService(t *testing.T) {
	Convey("given redis client", t, func() {
		miniR := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: miniR.Addr()})
		storage := NewAddressStorage(client)
		Convey("and 3 services have registered", func() {
			storage.Add(context.Background(), dto.BuildServiceAddress("sv1"), "123.0.0.1")
			storage.Add(context.Background(), dto.BuildServiceAddress("sv2"), "123.0.0.2")
			storage.Add(context.Background(), dto.BuildServiceAddress("sv3"), "123.0.0.3")
			Convey("get all service method should return all nodes", func() {
				data, err := storage.GetAddressOfAllServices(context.Background())
				So(err, ShouldBeNil)
				So(len(data), ShouldEqual, 3)
			})
		})
	})
}
