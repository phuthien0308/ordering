package monitoring

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func Init() {

}

func Handler() http.Handler {
	return promhttp.Handler()
}
