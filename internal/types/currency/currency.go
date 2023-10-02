package currency

import "fmt"

type Amount struct {
	Code   string
	Number float64
}

func (a Amount) String() string {
	return fmt.Sprintf("$%.2f %s", a.Number, a.Code)
}
