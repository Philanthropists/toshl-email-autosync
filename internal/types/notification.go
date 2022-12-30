package types

import (
	"fmt"
	"time"
)

type NotificationType uint8

const (
	Success NotificationType = iota
	Failed
	Parse
)

func (n NotificationType) String() string {
	switch n {
	case Success:
		return "success"
	case Failed:
		return "failed"
	case Parse:
		return "parse"
	default:
		return "unknown"
	}
}

type Notification struct {
	Type NotificationType
	Date time.Time
	Msg  string
}

const (
	txsFormat  = `%s || %s || %s`
	dateFormat = "2006-01-02"
)

func (n Notification) String() string {
	return fmt.Sprintf(txsFormat,
		n.Date.Format(dateFormat),
		n.Msg,
		n.Type,
	)
}
