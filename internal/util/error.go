package util

import (
	"errors"
	"log/slog"
	"runtime"
)

func HandlePanic(e any, captureStack bool) (err error, stack string) {
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

func LogError(log *slog.Logger, err error, message string) {
	if err != nil {
		log.Error(message, "error", err)
	}
}
