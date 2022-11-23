package types

type Config struct {
	Credentials
	ArchiveMailbox  string                    `json:"archive_mailbox"`
	Timezone        string                    `json:"timezone"`
	StockOptions    StockOptions              `json:"stock_options"`
	FundOptions     FundOptions               `json:"fund_options"`
	AccountMappings map[string]AccountMapping `json:"account_mappings"`
}

type Credentials struct {
	Email
	Toshl
	Rapid
	Twilio
}

type Email struct {
	Address  string `json:"mail-addr"`
	Username string `json:"mail-username"`
	Password string `json:"mail-password"`
}

type Toshl struct {
	Token string `json:"toshl-token"`
}

type Rapid struct {
	Key  string `json:"rapidapi-key"`
	Host string `json:"rapidapi-host"`
}

type Twilio struct {
	AccountSid string `json:"twilio-account-sid"`
	AuthToken  string `json:"twilio-auth-token"`
	FromNumber string `json:"twilio-from-number"`
	ToNumber   string `json:"twilio-to-number"`
}

type StockOptions struct {
	Enabled bool     `json:"enabled"`
	Stocks  []string `json:"stocks"`
	Times   []string `json:"times"`
}

type FundOptions struct {
	Enabled bool     `json:"enabled"`
	Funds   []string `json:"funds"`
	Times   []string `json:"times"`
}

type AccountMapping map[string]string
