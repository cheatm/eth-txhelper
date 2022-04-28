package sender

import "fmt"

type AsyncTransactionError struct {
	msg       string
	baseError error
}

func (err *AsyncTransactionError) Error() string {
	return fmt.Sprintf("%s: %s", err.msg, err.baseError)
}
