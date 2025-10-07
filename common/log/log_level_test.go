package log

import (
	"testing"
)

func TestLogLevel(t *testing.T) {
	tcs := []struct {
		name     string
		given    LogLevel
		expected string
	}{
		{
			name:     "happy path",
			given:    DEBUG,
			expected: "DEBUG",
		},
		{
			name:     "unhappy path",
			given:    INFO,
			expected: "INFO",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			result := tc.given.String()
			if result != tc.expected {
				t.Fail()
			}
		})
	}
}
