package entities

type BankDelegate interface {
	// FilterMessage(message entities.Message) bool
	// ExtractTransactionInfoFromMessage(message entities.Message) (*TransactionInfo, error)
	Name() string
}
