package main

// const credentialsFile = "credentials.json"
//
// type Config struct {
// 	types.Mail
// 	ArchiveMailbox string `json:"archive_mailbox"`
// }

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
//
// func sliceOfStringsToUint32(list []string) ([]uint32, error) {
// 	var result []uint32
// 	for _, i := range list {
// 		var num uint32
// 		_, err := fmt.Sscanf(i, "%d", &num)
// 		if err != nil {
// 			return nil, err
// 		}
// 		result = append(result, num)
// 	}
// 	return result, nil
// }

func main() {
	// config, err := getConfig()
	// if err != nil {
	// 	log.Fatalf("error getting config: %v\n", err)
	// }
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
	// sourceMailbox := flag.String("from", "INBOX", "destination mailbox")
	// destMailbox := flag.String("dest", config.ArchiveMailbox, "destination mailbox")
	// flag.Parse()
	// ids := flag.Args()
	//
	// if *destMailbox == "" {
	// 	log.Fatalln("no destination mailbox provided")
	// }
	// if len(ids) == 0 {
	// 	log.Fatalln("no ids provided")
	// }
	//
	// mailboxes, err := client.Mailboxes()
	// if err != nil {
	// 	log.Fatalf("error getting mailboxes: %v\n", err)
	// }
	//
	// if !contains(mailboxes, mailtypes.Mailbox(*sourceMailbox)) {
	// 	log.Fatalf("source mailbox '%q' does not exist\n", *sourceMailbox)
	// }
	// if !contains(mailboxes, mailtypes.Mailbox(*destMailbox)) {
	// 	log.Fatalf("destination mailbox '%q' does not exist\n", *destMailbox)
	// }
	//
	// err = client.Select(mailtypes.Mailbox(*sourceMailbox))
	// if err != nil {
	// 	log.Fatalf("error selecting mailbox: %v\n", err)
	// }
	//
	// idsAsUint32, err := sliceOfStringsToUint32(ids)
	// if err != nil {
	// 	log.Fatalf("error converting ids to uint32: %v\n", err)
	// }
	//
	// log.Printf(
	// 	"moving messages %v from %s to mailbox '%s'\n",
	// 	idsAsUint32,
	// 	*sourceMailbox,
	// 	*destMailbox,
	// )
	//
	// err = client.Move(mailtypes.Mailbox(*destMailbox), idsAsUint32...)
	// if err != nil {
	// 	log.Fatalf("error moving messages: %v\n", err)
	// }
	// log.Printf("messages moved successfully\n")

	panic("to be implemented")
}
