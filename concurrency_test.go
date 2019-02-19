package procexec

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestPanic(t *testing.T) {
	panicChan := make(chan *GoroutinePanic, 128)
	PanicCapturingGo(func() { generateAPanic() }, panicChan, nil)
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
	stopChan := make(chan struct{}, 1)

	PanicCapturingGo(func() { ongoingFunc(stopChan) }, nil, &processWG)
	PanicCapturingGo(func() { ongoingFunc(stopChan) }, nil, &processWG)
	PanicCapturingGo(func() { ongoingFunc(stopChan) }, nil, &processWG)

	time.Sleep(time.Second)
	close(stopChan)

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

func ongoingFunc(stopChan chan struct{}) {
	i := 0
	for {
		select {
		case <-stopChan:
			numberFunctionsExited++
			return
		default:
		}

		fmt.Printf("Running iteration %v...\n", i)
		i++

		time.Sleep(time.Second)
	}
}
