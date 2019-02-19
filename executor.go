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
	Execute(map[string]interface{}, chan *GoroutinePanic) error

	// Contract:
	// Implementation is responsible for this being a synchronous call.
	// Stop() should only return once all resources are cleaned up.
	//
	// Stop should only be called once by the caller.
	// Therefore, implementations should make effect to at least
	// have tried to cleanup all that it can before returning an error.
	Stop() error
}
