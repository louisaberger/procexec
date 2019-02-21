package procexec

type Executor interface {
	// Contract:
	// Implementation is responsible for this being a synchonous call.
	// Execute() should only return with a nil error if the process has successfully
	// started.
	//
	// When it has returned, it is expected that the "main work" in Execute
	// is now running asynchronously.
	//
	// Execute() should only be called once by the caller.
	//
	// The GoroutinePanic chan is used by the caller to surface panics in spawned
	// goroutines up to the calling function.
	// All go routines spawned off within Execute (or its called functions)
	// should be wrapped in a 'PanicCapturingGo', with this panic chan passed in.
	// Any panics from a spawned go routine will automatically be recorded to the
	// channel.
	// The implementation should not touch this channel for any other purposes.
	Execute(map[string]interface{}, chan *GoroutinePanic) error

	// Contract:
	// Implementation is responsible for this being a synchronous call.
	// Stop() should only return once all resources are cleaned up.
	//
	// Stop should only be called once by the caller.
	// Therefore, implementation should cleanup all that it can before returning an error.
	Stop() error
}
