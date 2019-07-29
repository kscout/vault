package errors

import (
	"fmt"
	"github.com/Noah-Huppert/golog"
)

// ErrorHandler logs an error and exits the process. If nil does nothing
type ErrorHandler func(err error, msg string, a ...interface{})

// Returns an ErrorHandler which logs errors and exits using golog
func GetErrorHandler(logger golog.Logger) ErrorHandler {
	return func(err error, msg string, a ...interface{}) {
		if err == nil {
			return
		}

		logger.Fatalf("%s: %s", fmt.Sprintf(msg, a), err.Error())
	}
}
