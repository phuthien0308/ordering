package dto

import "fmt"

func BuildServiceAddress(appname string) string {
	return fmt.Sprintf("service_addresses:%v", appname)
}
