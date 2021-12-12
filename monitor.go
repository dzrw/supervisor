package supervisor

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"

	log "github.com/sirupsen/logrus"
)

type ExitFunc = func(panicked bool, err interface{})

func QuietExitFunc(panicked bool, err interface{}) {}

// A Monitor runs long-lived functions and responds to OS signals.
type Monitor interface {
	Defer(fn func(ctx context.Context) error, exitfn ExitFunc)
	Join()
}

type item struct {
	fn   func(ctx context.Context) error
	exit ExitFunc
}

type monitor struct {
	wg     sync.WaitGroup
	ctx    context.Context
	cancel context.CancelFunc
	coll   []*item
}

// New creates a Monitor.
func New(ctx context.Context) Monitor {
	child, cancel := context.WithCancel(ctx)
	return &monitor{
		wg:     sync.WaitGroup{},
		ctx:    child,
		cancel: cancel,
		coll:   make([]*item, 0),
	}
}

func (m *monitor) Defer(fn func(ctx context.Context) error, exit ExitFunc) {
	m.coll = append(m.coll, &item{fn, exit})
}

func (m *monitor) Join() {
	for _, i := range m.coll {
		ii := i
		go m.dispatch(ii)
	}

	// Goroutine activations are randomized, so we inline the signal handler
	// and pre-increment the WaitGroup to prevent Wait from returning
	// immediately when the scheduler defers all goroutine activation.
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()

		ch := make(chan os.Signal, 2)
		signal.Notify(ch, os.Interrupt, syscall.SIGTERM)
		select {
		case u := <-ch:
			log.WithField("signal", u).Info("shutdown (caught signal)")
			m.cancel() // cancel top-level context on signal
		case <-m.ctx.Done():
			log.Info("signal handler cancelled (context)")
			return
		}
	}()

	log.Debug("joined")

	m.wg.Wait() // await cancellation
}

func (m *monitor) dispatch(i *item) {
	m.wg.Add(1)
	defer m.wg.Done() // cancel monitor

	var panicked bool
	var e interface{}

	defer func() {
		if i.exit != nil {
			i.exit(panicked, e)
		}
	}()

	defer func() {
		if err := recover(); err != nil {
			panicked = true
			e = err
			m.cancel() // cancel top-level context on panic
			return
		}
	}() // trap panics

	if err := i.fn(m.ctx); err != nil {
		panicked = false
		e = err
	}
}
