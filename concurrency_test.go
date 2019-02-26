package procexec

import (
	"context"
	"fmt"
	"http"
	"sync"
	"testing"
	"time"
)

func TestPanic(t *testing.T) {
	panicChan := make(chan *GoroutinePanic, 128)
	PanicCapturingGo(func() { generateAPanic() }, panicChan, nil, nil)
	tim := time.NewTimer(time.Second)
	select {
	case <-panicChan:
	case <-tim.C:
		t.Fatalf("Timed out waiting for error to bubble up")
	}
}

func TestPanicWithCtx(t *testing.T) {
	panicChan := make(chan *GoroutinePanic, 128)
	parentCtx := context.Background()
	PanicCapturingGo(func() { generateAPanic() }, panicChan, nil, parentCtx)

	tim := time.NewTimer(time.Second)
	select {
	case <-panicChan:
	case <-tim.C:
		t.Fatalf("Timed out waiting for error to bubble up")
	}

}

func generateAPanic() {
	panic("Fake panic")
}

var numberFunctionsExited int

func TestWaitGroup(t *testing.T) {
	numberFunctionsExited = 0

	var processWG sync.WaitGroup
	ctx, cancelFunc := context.WithCancel(context.Background())

	PanicCapturingGo(func(context.Context) { ongoingFunc(ctx) }, nil, &processWG, nil)
	PanicCapturingGo(func(context.Context) { ongoingFunc(ctx) }, nil, &processWG, nil)
	PanicCapturingGo(func(context.Context) { ongoingFunc(ctx) }, nil, &processWG, nil)

	time.Sleep(time.Second)
	cancelFunc()

	s := make(chan struct{})
	tim := time.NewTimer(time.Minute)

	go func() {
		processWG.Wait() // goroutine leak if never returns!
		close(s)
	}()

	select {
	case <-s:
	case <-tim.C:
		t.Fatalf("Timed out waiting for shut down")
	}

	if numberFunctionsExited != 3 {
		t.Fatalf("Expected 3 functions to fully exit, not %v", numberFunctionsExited)
	}
}

func ongoingFunc(ctx context.Context) {
	i := 0
	for {
		select {
		case <-ctx.Done():
			numberFunctionsExited++
			return
		default:
		}

		fmt.Printf("Running iteration %v...\n", i)
		i++

		time.Sleep(time.Second)
	}
}
