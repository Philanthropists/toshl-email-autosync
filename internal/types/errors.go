package types

import "fmt"

type ErrParseFailure struct {
	Cause error
	Value any
}

func (e ErrParseFailure) Error() string {
	return fmt.Sprintf("error parsing message: %s", e.Cause)
}
