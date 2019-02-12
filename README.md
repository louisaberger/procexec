# procexec

This package is an API for processes that will be spawned as separate goroutines by a parent function. 

Main Interface:
A 'Executor' interface is defined in executor.go. This should be implemented with a custom executor that defines an "Execute" function (for the actual process to run), and defines the various getter/setter functions needed to satisfy the interface requirements. 

An example of an implementation and how it would be used by the caller is in main/executor_poc.go.

Other Functions: 
* concurrency.go handles managing and cleaning up after spawned goroutines. The helpers will capture panics, and keep track of which goroutines are still running.