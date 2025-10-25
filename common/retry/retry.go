package retry

import (
	"fmt"
	"math/rand"
	"time"
)

type Retriable func() error

func DoRetry(f Retriable, times uint, basedBackoffTime time.Duration) error {
	random := rand.New(rand.NewSource(time.Now().UnixNano()))
	count := 1
	err := f()
	var current time.Duration
	for err != nil && count <= int(times) {
		after := basedBackoffTime + time.Millisecond*time.Duration(random.Int31n(1000)*int32(count))
		fmt.Printf("==========RetryAfter %v===========\n", after)
		current = current + after
		time.Sleep(current)
		err = f()
		count++
	}
	return err
}
