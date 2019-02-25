package main

import (
	"github.com/louisaberger/procexec"

	"context"
	"fmt"
	"sync"
	"time"
)

// Ensures MyExecutor implements the Executor interface
var _ procexec.Executor = (*MyExecutor)(nil)

type MyExecutor struct {
	config map[string]interface{} // config settings for my agent

	processWG *sync.WaitGroup // all spawned processes

	goctx context.Context
	// Read/write from anywhere, as long as you grab cancelFuncLock first
	cancelFunc     context.CancelFunc
	cancelFuncLock sync.Mutex
}

func NewMyExecutor(conf map[string]interface{}, parentContext context.Context) *MyExecutor {
	goctx, cancelFunc := context.WithCancel(parentContext)

	return &MyExecutor{config: conf, goctx: goctx, cancelFunc: cancelFunc, cancelFuncLock: sync.Mutex{}}
}

func (self *MyExecutor) Execute(panicChan chan *procexec.GoroutinePanic) error {

	// set up Process Wait Group
	var wg sync.WaitGroup
	self.processWG = &wg

	// Do some setup before starting the agent
	if err := executeSetup(); err != nil {
		return err
	}

	// Some other setup for the agent is spawned off in a separate goroutine
	procexec.PanicCapturingGo(func() { nestedFunctionToSpawn(self.goctx) }, panicChan, self.processWG)

	// Start our main function that will be running asynchronously
	procexec.PanicCapturingGo(func() { RunAgent(self.config, self.processWG, panicChan, self.goctx) }, panicChan, self.processWG)
	return nil
}

func executeSetup() error {
	if false {
		return fmt.Errorf("Some error setting up for execute")
	}
	return nil
}

func (self *MyExecutor) Stop(finishedChan chan bool) {
	// send a signal to all spawned go routines to stop
	self.cancelFuncLock.Lock()
	defer self.cancelFuncLock.Unlock()
	self.cancelFunc()

	// wait for all spawned go routines to return
	self.processWG.Wait()

	fmt.Printf("Stopped agent\n")

	finishedChan <- true
}

func RunAgent(conf map[string]interface{}, processWG *sync.WaitGroup, panicChan chan *procexec.GoroutinePanic, goctx context.Context) {
	fmt.Printf("Started agent\n")
	// Example of a nested spawned go routine in RunAgent
	procexec.PanicCapturingGo(func() { nestedFunctionToSpawn(goctx) }, panicChan, processWG)

	i := 0
	for {
		select {
		case <-goctx.Done():
			return
		default:
			// do nothing / prevent blocking
		}

		// If there was an I/O-intensive call here, we would pass in the goctx
		// If the goctx is cancelled up the stack, it will make that call exit
		// straight away.
		// doIOIntensiveWork(goctx)

		fmt.Printf("Running iteration %v...\n", i)
		i++

		time.Sleep(time.Second)
	}
}

func nestedFunctionToSpawn(goctx context.Context) {
	for {
		select {
		case <-goctx.Done():
			return
		default:
		}
		time.Sleep(5 * time.Second)
	}
}

func main() {
	panicChan := make(chan *procexec.GoroutinePanic, 128)

	// set up to start executor
	parentGoCtx, parentCancelFunc := context.WithCancel(context.Background())
	var pe procexec.Executor = NewMyExecutor(map[string]interface{}{"conf": map[string]interface{}{}}, parentGoCtx)

	// start and stop executor
	StartExecutor(pe, panicChan)
	StopExecutor(pe, panicChan, parentCancelFunc)

	// set up to start a new instance of executor
	pe = NewMyExecutor(map[string]interface{}{"conf": map[string]interface{}{}}, parentGoCtx)

	// start a new instance of the executor
	StartExecutor(pe, panicChan)

	fmt.Printf("Cancelling parent context\n")
	parentCancelFunc()

	time.Sleep(2 * time.Second)

	// should be able to stop executor after cancelling parent function
	StopExecutor(pe, panicChan, parentCancelFunc)

	// should be able to stop executor twice
	StopExecutor(pe, panicChan, parentCancelFunc)

}

// What the caller writes to start/stop the executor

func StartExecutor(pe procexec.Executor, panicChan chan *procexec.GoroutinePanic) {
	if err := pe.Execute(panicChan); err != nil {
		panic(fmt.Sprintf("Error starting : %v", err))
	}
	time.Sleep(2 * time.Second)
}

func StopExecutor(executor procexec.Executor, panicChan chan *procexec.GoroutinePanic, parentCancelFunc context.CancelFunc) {

	t := time.NewTimer(time.Minute)
	finishedChan := make(chan bool)
	procexec.PanicCapturingGo(
		func() {
			executor.Stop(finishedChan)
		},
		panicChan,
		nil,
	)

	select {
	case <-finishedChan:
		// success
	case <-t.C:
		fmt.Printf("Timed out waiting to stop executor. Cancelling parent context and moving on.\n")
		parentCancelFunc()
	}
}
