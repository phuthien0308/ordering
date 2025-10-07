package log

import (
	"context"
	"errors"
	"testing"
)

func TestLog(t *testing.T) {
	logger := NewLogger(DEBUG, AppNameTag("ordering service"), NewTag("a", "b"))
	logger.Error(context.Background(), "hello", errors.New("error"))
}
