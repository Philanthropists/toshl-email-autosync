package types

import "fmt"

type Amount struct {
	Number float64
	Code   string
}

func (a Amount) String() string {
	return fmt.Sprintf("$%.2f %s", a.Number, a.Code)
}
