package main

// const credentialsFile = "credentials.json"
//
// type Config struct {
// 	types.Mail
// }
//
// func getConfig() (Config, error) {
// 	credFile, err := os.Open(credentialsFile)
// 	if err != nil {
// 		return Config{}, err
// 	}
// 	defer credFile.Close()
//
// 	raw, err := io.ReadAll(credFile)
// 	if err != nil {
// 		return Config{}, err
// 	}
//
// 	var config Config
// 	err = json.Unmarshal(raw, &config)
// 	if err != nil {
// 		return Config{}, err
// 	}
//
// 	return config, nil
// }
//
// func contains(list []mailtypes.Mailbox, item mailtypes.Mailbox) bool {
// 	for _, i := range list {
// 		if i == item {
// 			return true
// 		}
// 	}
// 	return false
// }

func main() {
	// config, err := getConfig()
	// if err != nil {
	// 	panic(err)
	// }
	//
	// sourceMailbox := flag.String("source", "INBOX", "Source mailbox")
	//
	// flag.Parse()
	//
	// client := mail.Client{
	// 	Addr:     config.Address,
	// 	Username: config.Username,
	// 	Password: config.Password,
	// }
	// defer func() {
	// 	err := client.Logout()
	// 	if err != nil {
	// 		log.Printf("error logging out of client: %v\n", err)
	// 	}
	// }()
	//
	// mailboxes, err := client.Mailboxes()
	// if err != nil {
	// 	panic(err)
	// }
	//
	// fmt.Println("Mailboxes ----")
	// for _, m := range mailboxes {
	// 	fmt.Println(m)
	// }
	//
	// if !contains(mailboxes, mailtypes.Mailbox(*sourceMailbox)) {
	// 	log.Fatalf("source mailbox %q does not exist\n", *sourceMailbox)
	// }
	//
	// // --------------
	// since := time.Now().Add(-2 * 30 * 24 * time.Hour)
	// msgs, err := client.Messages(context.Background(), mailtypes.Mailbox(*sourceMailbox), since)
	// if err != nil {
	// 	panic(err)
	// }
	//
	// fmt.Println("Messages from INBOX ----")
	// for m := range msgs {
	// 	msg := m.Value
	// 	if m.Error == nil {
	// 		fmt.Printf(
	// 			"%d - %s - size: %d - %s\n",
	// 			msg.SeqNum,
	// 			msg.Envelope.Subject,
	// 			len(msg.RawBody),
	// 			msg.Envelope.Date.Format(time.RFC822),
	// 		)
	// 	} else {
	// 		fmt.Printf("%d - %s - size: %d - %s - Error: %v\n", msg.SeqNum, msg.Envelope.Subject, len(msg.RawBody), msg.Envelope.Date.Format(time.RFC822), m.Error)
	// 	}
	// }

	panic("to be implemented")
}
