package procexec

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
)

type GoroutinePanic struct {
	PanickedObject interface{}
	StackTrace     string
}

// replacement for "go".
// 1. if panicChan is not nil, captures panics in the goroutine and sends the error to the panicChan channel
// 2. if processWG is not nil, handles adding and removing this goroutine from the caller's processWG wait group
func PanicCapturingGo(f func(), panicChan chan *GoroutinePanic, processWG *sync.WaitGroup) {
	go func() {
		defer func() {
			if processWG != nil {
				processWG.Done()
			}
			if panicChan != nil {
				if r := recover(); r != nil {
					panicChan <- &GoroutinePanic{r, stackTrace()}
				}
			}
		}()

		if processWG != nil {
			processWG.Add(1)
		}
		f()
	}()
}

// Returns a stack trace
func stackTrace() string {
	buf := make([]byte, 1000000)
	n := runtime.Stack(buf, false)
	return cleanerStackTrace(string(buf[0:n]))
}

func cleanerStackTrace(st string) string {
	cleaner := ""
	var method, fileAndLine string
	lines := strings.Split(st, "\n")
	for i := 1; i+1 < len(lines); i += 2 {
		a := lines[i]
		// com.tengen/cm/util.StackTrace(0x34a9ec, 0x4120796200000021)
		// we want the 'util.StackTrace', between the last '/' and the last '('
		lastSlash := strings.LastIndex(a, "/")
		lastParen := strings.LastIndex(a, "(")
		if lastSlash < 0 || lastParen < 0 {
			method = "—"
		} else {
			method = a[lastSlash+1 : lastParen]
		}

		b := lines[i+1]
		// com.tengen/cm/util/runtime_util.go:12 +0x63
		// We want the file and line
		// between the second-to-last slash and the last "+"
		bLastSlash := strings.LastIndex(b, "/")
		if bLastSlash < 0 {
			fileAndLine = "—"
		} else {
			bSecLastSlash := strings.LastIndex(b[0:bLastSlash], "/")
			bLastPlus := strings.LastIndex(b, "+")
			if bSecLastSlash < 0 || bLastPlus < 0 {
				fileAndLine = "—"
			} else {
				fileAndLine = b[bSecLastSlash+1 : bLastPlus]
			}
		}
		s := fmt.Sprintf("%45v %v\n", method, fileAndLine)
		cleaner += s
	}
	return cleaner
}
