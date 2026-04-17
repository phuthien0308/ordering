package configuration

import (
	"os"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

func TestConfiguration(t *testing.T) {
	convey.Convey("given a file in memory", t, func() {
		f, err := os.CreateTemp(".", "sample")

		if err != nil || f == nil {
			t.Fail()
		}
		defer func() {
			os.Remove(f.Name())
		}()

		f.WriteString(`{
			"redis_host": "redishost",
			"kafka": {
				"host": "kafkahost",
				"topic": "service_registry"
			}
		}`)
		convey.Convey("parsing", func() {
			cf := loadConfiguration(f.Name())
			convey.So(cf.RedisHost, convey.ShouldEqual, "redishost")
			convey.So(cf.Kafka.Host, convey.ShouldEqual, "kafkahost")
			convey.So(cf.Kafka.Topic, convey.ShouldEqual, "service_registry")
		})
	})
}
