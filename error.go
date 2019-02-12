package procexec

import (
	"errors"
	"strings"
)

const alreadyStoppedMarker = "Process is not running, cannot stop."

var AlreadyStoppedErr = errors.New(alreadyStoppedMarker)

func IsAlreadyStoppedErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), alreadyStoppedMarker)
}

const alreadyStartedMarker = "Process is already started, cannot start."

var AlreadyStartedErr = errors.New(alreadyStartedMarker)

func IsAlreadyStartedErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), alreadyStartedMarker)
}
