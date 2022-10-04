package notifications

type NotificationsClient interface {
	SendMsg(msg string) (string, error)
}
