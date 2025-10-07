package account

import (
	"context"
	"testing"

	"github.com/google/uuid"
	dto "github.com/phuthien0308/ordering/accountservice/repository/dto"
	"github.com/redis/go-redis/v9"
	. "github.com/smartystreets/goconvey/convey"
)

func TestAccountRepository(t *testing.T) {
	Convey("give redis", t, func() {
		rdb := redis.NewClient(&redis.Options{
			Addr:     "localhost:6379", // use default Addr
			Password: "",               // no password set
			DB:       0,                // use default DB
		})
		rep := &accountRepository{redisClient: *rdb}
		Convey("add", func() {
			idGenerator, err := uuid.NewV6()
			if err != nil {
				t.Fail()
			}
			identify := idGenerator.String()
			_, err = rep.Save(context.Background(), dto.Account{
				Id:      identify,
				Name:    idGenerator.String(),
				Address: []string{idGenerator.String()}})
			So(err, ShouldBeNil)
			Convey("load", func() {
				acc, err := rep.Load(context.Background(), identify)
				So(err, ShouldBeNil)
				So(acc.Id, ShouldEqual, identify)
			})
		})

	})
}
