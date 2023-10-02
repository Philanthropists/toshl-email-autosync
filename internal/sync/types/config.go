package types

type Config struct {
	Credentials
	Timezone string `json:"timezone"`
}

type Credentials struct {
	Mail   Mail   `json:"mail"`
	Twilio Twilio `json:"twilio"`
}

type Mail struct {
	Address  string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Twilio struct {
	AccountSid string `json:"twilio-account-sid"`
	AuthToken  string `json:"twilio-auth-token"`
	FromNumber string `json:"twilio-from-number"`
}
