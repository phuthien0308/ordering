package retry

import (
	"fmt"
	"testing"
	"time"
)

func TestRetry(t *testing.T) {
	DoRetry(func() error {
		return fmt.Errorf("error")
	}, 3, time.Millisecond*1)
}
