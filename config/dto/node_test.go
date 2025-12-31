package dto

import (
	"testing"
)

func TestApp(t *testing.T) {
	var tcs = []struct {
		desc                string
		appname             string
		ip                  string
		healthCheckEndpoint string
		expectedError       error
	}{
		{
			desc:                "happy",
			appname:             "hello",
			ip:                  "127.0.0.1",
			healthCheckEndpoint: "http://localhost/healthChedk",
			expectedError:       nil,
		},
		{
			desc:                "app name empty",
			appname:             "",
			ip:                  "127.0.0.1",
			healthCheckEndpoint: "http://localhost/healthChedk",
			expectedError:       ErrAppNameEmpty,
		},
		{
			desc:                "ip is invalid",
			appname:             "hello",
			ip:                  "",
			healthCheckEndpoint: "http://localhost/healthChedk",
			expectedError:       ErrIpInvalid,
		},
		{
			desc:                "healthcheck is empty",
			appname:             "hello",
			ip:                  "127.0.0.1",
			healthCheckEndpoint: "",
			expectedError:       ErrHealthCheckEndpointInvalid,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.desc, func(t *testing.T) {
			_, err := NewNode(tc.appname, tc.ip, tc.healthCheckEndpoint)
			if err != tc.expectedError {
				t.Fail()
			}
		})
	}
}
