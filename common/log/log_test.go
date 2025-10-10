package log

import (
	"bytes"
	"strings"
	"testing"
)

func TestLog(t *testing.T) {

	tcs := []struct {
		name      string
		level     LogLevel
		inputStr  string
		expectStr string
	}{
		{
			name:      "debug",
			level:     DEBUG,
			inputStr:  "debug",
			expectStr: "debug",
		},
		{
			name:      "info",
			level:     INFO,
			inputStr:  "info",
			expectStr: "info",
		},
		{
			name:      "warm",
			level:     WARM,
			inputStr:  "warm",
			expectStr: "warm",
		},
		{
			name:      "error",
			level:     ERROR,
			inputStr:  "error",
			expectStr: "error",
		},
	}

	for _, tc := range tcs {
		t.Run(t.Name(), func(t *testing.T) {
			writer := bytes.NewBuffer([]byte{})
			logger := NewLogger(tc.level, writer)
			if tc.level == DEBUG {
				logger.Debug(t.Context(), tc.inputStr)
			}
			if tc.level == INFO {
				logger.Info(t.Context(), tc.inputStr)
			}
			if tc.level == WARM {
				logger.Warn(t.Context(), tc.inputStr)
			}
			if tc.level == ERROR {
				logger.Error(t.Context(), tc.inputStr, nil)
			}
			if !strings.Contains(writer.String(), tc.expectStr) {
				t.Fail()
			}
		})
	}
}
