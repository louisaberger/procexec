package main

import (
	"github.com/louisaberger/procexec"

	"fmt"
	"sync"
	"time"
)

type MyExecutor struct {
	errChan   chan error
	startChan chan struct{}
	stopChan  chan struct{}

	processWG *sync.WaitGroup // all spawned processes in this goroutine

	initiatedStop bool
	isRunning     bool
}

func NewMyExecutor() *MyExecutor {

	return &MyExecutor{
		errChan:   make(chan error, 1),
		startChan: make(chan struct{}, 1),
	}
}

func (self *MyExecutor) Execute(settings map[string]interface{}, panicChan chan *procexec.GoroutinePanic) {

	// Example of how to handle errors in Execute()
	if err := someAgentSetup(); err != nil {
		self.ErrChan() <- err
		return
	}

	// Example of how to deal with nested spawned go routines
	// with wait groups.
	procexec.PanicCapturingGo(func() { nestedFunctionToSpawn(self.StopChan()) }, panicChan, self.ProcessWG())

	// Tell our executor that we're starting up
	self.StartChan() <- struct{}{}
	runAgent(self.ProcessWG(), self.StopChan(), self.ErrChan(), panicChan)
}

func someAgentSetup() error {
	return nil
}

func nestedFunctionToSpawn(stopChan chan struct{}) {
	for {
		select {
		case <-stopChan:
			return
		default:
		}
		time.Sleep(5 * time.Second)
	}
}

func runAgent(processWG *sync.WaitGroup, stopChan chan struct{}, errChan chan error, panicChan chan *procexec.GoroutinePanic) {
	procexec.PanicCapturingGo(func() { nestedFunctionToSpawn(stopChan) }, panicChan, processWG)

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

		time.Sleep(time.Second)
	}
}

// Functions to satisfy interface requirements
func (self *MyExecutor) ErrChan() chan error {
	return self.errChan
}
func (self *MyExecutor) StartChan() chan struct{} {
	return self.startChan
}
func (self *MyExecutor) StopChan() chan struct{} {
	return self.stopChan
}
func (self *MyExecutor) SetStopChan(c chan struct{}) {
	self.stopChan = c
}
func (self *MyExecutor) InitiatedStop() bool {
	return self.initiatedStop
}
func (self *MyExecutor) SetInitiatedStop(b bool) {
	self.initiatedStop = b
}
func (self *MyExecutor) IsRunning() bool {
	return self.isRunning
}
func (self *MyExecutor) SetIsRunning(b bool) {
	self.isRunning = b
}
func (self *MyExecutor) ProcessWG() *sync.WaitGroup {
	return self.processWG
}
func (self *MyExecutor) SetProcessWG(wg *sync.WaitGroup) {
	self.processWG = wg
}

func main() {
	var pe procexec.Executor = NewMyExecutor()
	panicChan := make(chan *procexec.GoroutinePanic, 128)

	settings := map[string]interface{}{}

	if err := procexec.Stop(pe); !procexec.IsAlreadyStoppedErr(err) {
		panic(err)
	}

	if err := procexec.Start(pe, settings, panicChan); err != nil {
		panic(fmt.Sprintf("Error starting : %v", err))
	}

	if err := procexec.Stop(pe); err != nil {
		panic(fmt.Sprintf("Error stopping : %v", err))
	}

	// try to stop again. should error, not panic
	if err := procexec.Stop(pe); !procexec.IsAlreadyStoppedErr(err) {
		panic(err)
	}

	// restart
	if err := procexec.Start(pe, settings, panicChan); err != nil {
		panic(fmt.Sprintf("Error starting : %v", err))
	}

	// try to start again. should error, not panic
	if err := procexec.Start(pe, settings, panicChan); !procexec.IsAlreadyStartedErr(err) {
		panic(err)
	}

	if err := procexec.Stop(pe); err != nil {
		panic(fmt.Sprintf("Error stopping : %v", err))
	}

}
