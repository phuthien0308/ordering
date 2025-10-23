package helper

import (
	"fmt"
	"os"
)

var AppName = "orderservice"
var POD_ID = func() string {
	if os.Getenv("POD_IP") != "" {
		return os.Getenv("POD_IP")
	}
	return "localhost"
}()
var HealthCheckEndpoint = func(port uint) string {
	return fmt.Sprintf("http://%v:%v/healthz", POD_ID, port)
}
