package procexec

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

var (
	setupShouldFail      bool
	agentLoopShouldError bool
)

func TestExecutorBasic(t *testing.T) {
	var pe Executor = NewTestExecutor()
	panicChan := make(chan *GoroutinePanic, 128)

	settings := map[string]interface{}{}

	// Try to stop before starting. should error, not panic
	if err := Stop(pe); !IsAlreadyStoppedErr(err) {
		panic(err)
	}

	// start normally
	if err := Start(pe, settings, panicChan); err != nil {
		panic(fmt.Sprintf("Error starting : %v", err))
	}

	// stop normally
	if err := Stop(pe); err != nil {
		panic(fmt.Sprintf("Error stopping : %v", err))
	}

	// try to stop again. should error, not panic
	if err := Stop(pe); !IsAlreadyStoppedErr(err) {
		panic(err)
	}

	// restart
	if err := Start(pe, settings, panicChan); err != nil {
		panic(fmt.Sprintf("Error starting : %v", err))
	}

	// try to start again. should error, not panic
	if err := Start(pe, settings, panicChan); !IsAlreadyStartedErr(err) {
		panic(err)
	}

	// stop normally
	if err := Stop(pe); err != nil {
		panic(fmt.Sprintf("Error stopping : %v", err))
	}
}

func TestExecutorWithErrors(t *testing.T) {
	var pe Executor = NewTestExecutor()
	panicChan := make(chan *GoroutinePanic, 128)

	settings := map[string]interface{}{}

	setupShouldFail = true
	if err := Start(pe, settings, panicChan); !strings.Contains(err.Error(), "Erroring in setup") {
		panic(fmt.Sprintf("start should be erroring : %v\n", err))
	}

	pe = NewTestExecutor()
	setupShouldFail = false
	agentLoopShouldError = true
	if err := Start(pe, settings, panicChan); err != nil {
		panic(err)
	}
	tim := time.NewTimer(12 * time.Second)
	select {
	case err := <-pe.ErrChan():
		if !strings.Contains(err.Error(), "Erroring in runAgent") {
			panic(err)
		}
	case <-tim.C:
		panic("Timed out waiting for error to bubble up")
	}
	if err := Stop(pe); err != nil {
		panic(err)
	}
}

type TestExecutor struct {
	errChan       chan error
	startChan     chan struct{}
	stopChan      chan struct{}
	initiatedStop bool
	isRunning     bool
	processWG     *sync.WaitGroup // all spawned processes in this goroutine
	runDoneWG     *sync.WaitGroup
}

func NewTestExecutor() *TestExecutor {

	return &TestExecutor{
		errChan:   make(chan error, 1),
		startChan: make(chan struct{}, 1),
	}
}

/*
This will be executed from a goroutine.
Requires the following:
* Send errors back through the error chan
* When process is started,
*
For every spawned process in Execute, you
should add 1 to the 'doneWG'.
*/
func (self *TestExecutor) Execute(settings map[string]interface{}, panicChan chan *GoroutinePanic) {

	// Example of how to handle errors in Execute()
	if err := setupFunc(); err != nil {
		self.ErrChan() <- err
		return
	}

	// Example of how to deal with nested spawned go routines
	// with wait groups.
	PanicCapturingGo(func() { nestedFunc(self.StopChan()) }, nil, self.ProcessWG())

	// Tell our executor that we're starting up
	self.StartChan() <- struct{}{}
	runAgent(self.ProcessWG(), self.StopChan(), self.ErrChan())
}

func setupFunc() error {
	if setupShouldFail {
		return fmt.Errorf("Erroring in setup")
	}
	return nil
}

func nestedFunc(stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		default:
		}
		time.Sleep(5 * time.Second)
	}
}

func runAgent(processWG *sync.WaitGroup, stopChan chan struct{}, errChan chan error) {
	PanicCapturingGo(func() { nestedFunc(stopChan) }, nil, processWG)

	i := 0
	for {
		select {
		case <-stopChan:
			return
		default:
			// do nothing / prevent blocking
		}

		fmt.Printf("Running iteration %v...\n", i)
		i++

		// used to test how caller handles errors
		if agentLoopShouldError {
			errChan <- fmt.Errorf("Erroring in runAgent")
		}

		time.Sleep(time.Second)
	}
}

// Functions to satisfy interface requirements
func (self *TestExecutor) ErrChan() chan error {
	return self.errChan
}
func (self *TestExecutor) StartChan() chan struct{} {
	return self.startChan
}
func (self *TestExecutor) StopChan() chan struct{} {
	return self.stopChan
}
func (self *TestExecutor) SetStopChan(c chan struct{}) {
	self.stopChan = c
}
func (self *TestExecutor) InitiatedStop() bool {
	return self.initiatedStop
}
func (self *TestExecutor) SetInitiatedStop(b bool) {
	self.initiatedStop = b
}
func (self *TestExecutor) IsRunning() bool {
	return self.isRunning
}
func (self *TestExecutor) SetIsRunning(b bool) {
	self.isRunning = b
}
func (self *TestExecutor) ProcessWG() *sync.WaitGroup {
	return self.processWG
}
func (self *TestExecutor) SetProcessWG(wg *sync.WaitGroup) {
	self.processWG = wg
}
