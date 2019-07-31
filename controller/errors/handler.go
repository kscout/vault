package errors

import (
	"fmt"
	"log"
)

// HandleErr logs and exits if error is not nil
func HandleErr(err error, msg string, a ...interface{}) {
	if err == nil {
		return
	}

	fmtMsg := msg
	if len(a) > 0 {
		fmtMsg = fmt.Sprintf(msg, a)
	}

	log.Fatalf("%s: %s\n", fmtMsg, err.Error())
}
