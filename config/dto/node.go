package dto

import (
	"errors"
	"net"
)

var ErrAppNameEmpty = errors.New("app name is empty")
var ErrIpInvalid = errors.New("the ip is not valid")
var ErrHealthCheckEndpointInvalid = errors.New("the healthCheckEndpoint is empty")

type Node struct {
	AppName string   `json:"-"`
	Ips     []string `json:"ips"`
}

func NewNode(appName, ip, healCheckEndpoint string) (*Node, error) {
	if appName == "" {
		return nil, ErrAppNameEmpty
	}
	if valiateIp(ip) != nil {
		return nil, ErrIpInvalid
	}
	if healCheckEndpoint == "" {
		return nil, ErrHealthCheckEndpointInvalid
	}
	return &Node{AppName: appName,
		Ips: []string{ip},
	}, nil
}

func valiateIp(ip string) error {
	parsedIp := net.ParseIP(ip)
	if parsedIp == nil {
		return errors.New("the ip is not valid")
	}
	ip4 := parsedIp.To4()
	if ip4 == nil {
		return errors.New("the ip is not an ipv4")
	}
	return nil
}
