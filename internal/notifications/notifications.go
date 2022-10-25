package notifications

import (
	"fmt"
	"strings"
	"sync"

	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/queue"
	"github.com/Philanthropists/toshl-email-autosync/internal/queue/impl/mutex"
)

type notifications struct {
	Queue    queue.FIFOQueue[string]
	IsClosed bool
	Nc       NotificationsClient
}

var notifStore notifications
var createOnce sync.Once

func createNotificationsStore() {
	createOnce.Do(func() {
		notifStore.Queue = mutex.CreateQueue[string](0)
	})
}

func SetNotificationsClient(ns NotificationsClient) error {
	createNotificationsStore()

	fmt.Printf("notifStore.Nc = %p, nil = %v\n", &notifStore.Nc, nil)

	if ns == nil {
		return fmt.Errorf("notifications client cannot be nil")
	}

	if notifStore.Nc != nil {
		return fmt.Errorf("notifications client is already set: %p", &notifStore.Nc)
	}

	notifStore.Nc = ns

	return nil
}

func flushNotifications() {
	log := logger.GetLogger()

	createNotificationsStore()

	if notifStore.Nc == nil {
		log.Errorf("notifications client is not set, not sending anything")
		return
	}

	condensedMsgs := []string{}

	for !notifStore.Queue.IsEmpty() {
		msg, _ := notifStore.Queue.Pop()
		if msg != nil {
			condensedMsgs = append(condensedMsgs, *msg)
		}
	}

	completeMsg := strings.Join(condensedMsgs, "\n")
	if completeMsg == "" {
		return
	}

	resp, err := notifStore.Nc.SendMsg(completeMsg)
	if err != nil {
		log.Errorw("could not send notification",
			"message", completeMsg,
			"error", err,
			"response", resp)
	}

	log.Debugf("notifications were flushed, sent %d bytes", len(completeMsg))
}

func PushNotification(s string) error {
	if notifStore.IsClosed {
		return fmt.Errorf("notifications are closed")
	}

	createNotificationsStore()
	notifStore.Queue.PushBack(&s)

	return nil
}

func ErrorNotify(log logger.Logger, msg string) error {
	log.Error(msg)
	return PushNotification(fmt.Sprintf("ERROR: %s", msg))
}

func Close() {
	defer flushNotifications()
	notifStore.IsClosed = true
}
