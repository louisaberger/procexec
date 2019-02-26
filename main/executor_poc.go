package main

import (
	"github.com/louisaberger/procexec"

	"context"
	"fmt"
	"sync"
	"time"
)

/*
General Rules For When to Use a Context with a Goroutine:

* If it's in a for loop with a select statement listening on channels,
	then it should also listen for ctx.Done()
* If it’s calling I/O then it should pass a ctx
* If it’d doing CPU intensive work then it should occasionally look at ctx.Err()
(as well as yield to the scheduler)

*/

// Ensures MyExecutor implements the Executor interface
var _ procexec.Executor = (*MyExecutor)(nil)

type MyExecutor struct {
	config map[string]interface{} // config settings for my agent

	processWG *sync.WaitGroup // all spawned processes

	ctx        context.Context
	cancelFunc context.CancelFunc
}

func NewMyExecutor(conf map[string]interface{}, parentContext context.Context) *MyExecutor {
	ctx, cancelFunc := context.WithCancel(parentContext)

	return &MyExecutor{config: conf, ctx: ctx, cancelFunc: cancelFunc}
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
	procexec.PanicCapturingGo(func(ctx context.Context) { nestedFunctionToSpawn(ctx) }, panicChan, self.processWG, self.ctx)

	// Start our main function that will be running asynchronously
	procexec.PanicCapturingGo(func(ctx context.Context) { RunAgent(self.config, self.processWG, panicChan, ctx) }, panicChan, self.processWG, self.ctx)
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
	self.cancelFunc()

	// wait for all spawned go routines to return
	self.processWG.Wait()

	fmt.Printf("Stopped agent\n")

	finishedChan <- true
}

func (self *MyExecutor) Cancel() {
	self.cancelFunc()
}

func RunAgent(conf map[string]interface{}, processWG *sync.WaitGroup, panicChan chan *procexec.GoroutinePanic, ctx context.Context) {
	fmt.Printf("Started agent\n")
	// Example of a nested spawned go routine in RunAgent
	procexec.PanicCapturingGo(func(c context.Context) { nestedFunctionToSpawn(c) }, panicChan, processWG, ctx)

	i := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// do nothing / prevent blocking
		}

		// If there was an I/O-intensive call here, we would pass in the ctx
		// If the ctx is cancelled up the stack, it will make that call exit
		// straight away.
		// doIOIntensiveWork(ctx)

		fmt.Printf("Running iteration %v...\n", i)
		i++

		time.Sleep(time.Second)
	}
}

func nestedFunctionToSpawn(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
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
	StopExecutor(pe, panicChan)

	// set up to start a new instance of executor
	pe = NewMyExecutor(map[string]interface{}{"conf": map[string]interface{}{}}, parentGoCtx)

	// start a new instance of the executor
	StartExecutor(pe, panicChan)

	// someone up the stack cancels the parent context
	fmt.Printf("Cancelling parent context\n")
	parentCancelFunc()

	time.Sleep(2 * time.Second)

	// should be able to stop executor after cancelling parent function
	StopExecutor(pe, panicChan)

	// should be able to stop executor twice
	StopExecutor(pe, panicChan)

}

// What the caller writes to start/stop the executor

func StartExecutor(pe procexec.Executor, panicChan chan *procexec.GoroutinePanic) {
	if err := pe.Execute(panicChan); err != nil {
		panic(fmt.Sprintf("Error starting : %v", err))
	}
	time.Sleep(2 * time.Second)
}

func StopExecutor(executor procexec.Executor, panicChan chan *procexec.GoroutinePanic) {

	t := time.NewTimer(time.Minute)
	finishedChan := make(chan bool)
	procexec.PanicCapturingGo(
		func(context.Context) {
			executor.Stop(finishedChan)
		},
		panicChan,
		nil,
		nil,
	)

	select {
	case <-finishedChan:
		// success
	case <-t.C:
		fmt.Printf("Timed out waiting to stop executor. Cancelling and moving on.\n")
		executor.Cancel()
	}
}
