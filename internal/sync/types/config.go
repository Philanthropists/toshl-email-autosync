package types

type Config struct {
	Credentials
	Timezone string `json:"timezone"`
}

type Credentials struct {
	Mail   Mail   `json:"mail"`
	Twilio Twilio `json:"twilio"`
	Toshl  Toshl  `json:"toshl"`
}

type Mail struct {
	Address  string `json:"addr"`
	Username string `json:"username"`
	Password string `json:"password"`
}

type Twilio struct {
	AccountSid string `json:"account-sid"`
	AuthToken  string `json:"auth-token"`
	FromNumber string `json:"from-number"`
}

type Toshl struct {
	Token string `json:"token"`
}
