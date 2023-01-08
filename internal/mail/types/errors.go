package types

import "fmt"

type ErrInternal struct {
	CauseErr error
	Msg      string
}

func (e ErrInternal) Error() string {
	return fmt.Sprintf("imap client error: %s: %v", e.Msg, e.CauseErr)
}

func (e ErrInternal) Cause() error {
	return e.CauseErr
}
