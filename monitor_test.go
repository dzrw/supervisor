package supervisor

import (
	"context"
	"os"
	"testing"
	"time"
)

var TestExitFunc = QuietExitFunc

type TestError struct{}

func (e *TestError) Error() string { return "test error" }

var _ = error(&TestError{})

func TestGracefulShutdownOnPanic(t *testing.T) {
	m := New(context.Background())

	// setup
	m.Defer(func(ctx context.Context) error {
		if cancelled := trueIfCancelled(ctx, 100); !cancelled {
			t.Errorf("trueIfCancelled exited, but was not cancelled (increase the work factor?)")
			return &TestError{}
		}
		return nil
	}, TestExitFunc)

	m.Defer(func(ctx context.Context) error {
		time.Sleep(100 * time.Millisecond)
		panic("faulty")
	}, TestExitFunc)

	// launch
	m.Join()
}

func TestDeferredFuncsCanExitInAnyOrder(t *testing.T) {
	m := New(context.Background())

	// setup
	m.Defer(func(ctx context.Context) error {
		if trueIfCancelled(ctx, 10) {
			t.Errorf("trueIfCancelled was cancelled (increase the sleep time?)")
			return &TestError{}
		}
		return nil
	}, TestExitFunc)

	m.Defer(func(ctx context.Context) error {
		// after sleep interrupt this process to cause shutdown
		time.Sleep(1000 * time.Millisecond)
		p, _ := os.FindProcess(os.Getpid())
		_ = p.Signal(os.Interrupt)
		return nil
	}, TestExitFunc)

	// launch
	m.Join()
}

func trueIfCancelled(ctx context.Context, n int) bool {
	for i := 0; i < n; i++ {
		// cancelled?
		select {
		case <-ctx.Done():
			return true
		default:
		}

		time.Sleep(10 * time.Millisecond)
	}
	return false
}
