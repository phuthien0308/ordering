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
	return "localhost:8081"
}()
var HealthCheckEndpoint = func() string {
	return fmt.Sprintf("http://%v/healthz", POD_ID)
}()
