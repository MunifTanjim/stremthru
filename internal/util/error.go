package util

import (
	"errors"
	"runtime"
)

func RecoverPanic(captureStack bool) (err error, stack string) {
	e := recover()
	if e == nil {
		return nil, stack
	}

	if perr, ok := e.(error); ok {
		err = perr
	} else {
		err = errors.New("something went wrong")
	}

	if !captureStack {
		return err, stack
	}

	buf := make([]byte, 2048)
	n := runtime.Stack(buf, false)
	buf = buf[:n]
	stack = string(buf)
	return err, stack
}
