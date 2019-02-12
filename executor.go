package procexec

import (
	"fmt"
	"sync"
	"time"
)

type Executor interface {
	/* The function that we will run in a separate go routine.

	settings: Configuration map for spawned process.
	panicChan: Channel to pass back panics from the spawned go routine to the caller.

	* Responsible for managing its own errors by sending them back over ErrChan channel.
		These will get consumed and passed back to the caller in "Start()"
	* Any goroutines spawned from within Execute() should be started with PanicCapturingGo().
		This ensures that all spawned go routines are properly handled with panics and wait groups.

	*/
	Execute(map[string]interface{}, chan *GoroutinePanic)
	// Channel for the spawned process to send errors back
	// to the executor.
	ErrChan() chan error
	// Channel for the spawned process to send a message
	// to the executor that it has finished starting up.
	StartChan() chan struct{}
	// Channel for the executor to send a message
	// to the spawned process that it should stop.
	StopChan() chan struct{}
	SetStopChan(chan struct{})
	// True if we have sent a message on StopChan
	// Used to ensure we do not try to close
	// the stop channel more than once
	InitiatedStop() bool
	SetInitiatedStop(bool)
	// True if the process has started up
	// Used to ensure we do not try to Start
	// or Stop multiple times.
	IsRunning() bool
	SetIsRunning(bool)
	// WaitGroup for all go routines spawned by Start().
	// Used to ensure we shut down all spawned go routines
	// before exiting.
	ProcessWG() *sync.WaitGroup
	SetProcessWG(wg *sync.WaitGroup)
}

func Start(self Executor, settings map[string]interface{}, panicChan chan *GoroutinePanic) error {
	if self.IsRunning() {
		return AlreadyStartedErr
	}

	// set up Process Wait Group
	var wg sync.WaitGroup
	self.SetProcessWG(&wg)

	// refresh Stop Channel
	self.SetInitiatedStop(false)
	self.SetStopChan(make(chan struct{}, 1))

	PanicCapturingGo(func() { self.Execute(settings, panicChan) }, panicChan, self.ProcessWG())

	return waitForStartup(self)
}

func waitForStartup(self Executor) error {
	t := time.NewTimer(2 * time.Minute)
	select {
	case <-self.StartChan():
		self.SetIsRunning(true)
		return nil
	case err := <-self.ErrChan():
		return err
	case <-t.C:
		return fmt.Errorf("Timed out waiting for process to start")
	}
}

func Stop(self Executor) error {
	if !self.IsRunning() {
		return AlreadyStoppedErr
	}

	// We may call Stop multiple times, but we want to
	// make sure we don't try to close the channel multiple times.
	if !self.InitiatedStop() {
		close(self.StopChan())
		self.SetInitiatedStop(true)
	}

	if err := waitForShutdown(self); err != nil {
		return fmt.Errorf("Error waiting for process to shut down: %v", err)
	}

	self.SetIsRunning(false)
	self.SetInitiatedStop(false) // reset for future stops

	return nil
}

func waitForShutdown(self Executor) error {
	s := make(chan struct{})
	t := time.NewTimer(2 * time.Minute)

	go func() {
		self.ProcessWG().Wait() // goroutine leak if never returns!
		close(s)
	}()

	select {
	case <-s:
		return nil
	case <-t.C:
		return fmt.Errorf("Timed out waiting for shut down")
	}
}
