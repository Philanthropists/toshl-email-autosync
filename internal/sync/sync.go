package sync

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/text/language"
	"golang.org/x/text/message"

	"github.com/Philanthropists/toshl-email-autosync/internal/bank"
	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap"
	"github.com/Philanthropists/toshl-email-autosync/internal/logger"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/common"
	"github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/Philanthropists/toshl-email-autosync/internal/toshl"
)

func ExtractTransactionInfoFromMessages(msgs []types.BankMessage) ([]*types.TransactionInfo, int64) {
	log := logger.GetLogger()
	var failures int64

	var transactions []*types.TransactionInfo
	for _, bankMsg := range msgs {
		t, err := bankMsg.Bank.ExtractTransactionInfoFromMessage(bankMsg.Message)
		if err == nil {
			transactions = append(transactions, t)
		} else {
			log.Errorw("Error processing message",
				"error", err,
				"msgId", bankMsg.SeqNum,
			)
			failures++
		}
	}

	return transactions, failures
}

func getEarliestDateFromTxs(txs []*types.TransactionInfo) time.Time {
	earliestDate := time.Now().Add(-24 * time.Hour)
	for _, tx := range txs {
		date := tx.Date
		if date.Before(earliestDate) {
			earliestDate = date
		}
	}

	return earliestDate
}

const notificationFormat = `%s Transactions: s:%d / f:%d / parse:%d`

type txsStatus struct {
	SuccessfulTxs []*types.TransactionInfo
	FailedTxs     []*types.TransactionInfo
	ParseFailures int64
}

func notificationString(success, failures []*types.TransactionInfo, parseFails int64) string {
	versionInfo := common.GetVersion()[:4]
	msg := fmt.Sprintf(notificationFormat, versionInfo, len(success), len(failures), parseFails)

	p := message.NewPrinter(language.English)

	const txsFormat = `%s || $ %.2f %s|| %s`
	const dateFormat = "2006-01-02"
	var status []string
	status = append(status, msg)

	for _, txs := range success {
		status = append(status,
			p.Sprintf(txsFormat,
				txs.Date.Format(dateFormat),
				*txs.Value.Rate,
				txs.Place,
				"SUCCESS"))
	}

	for _, txs := range failures {
		status = append(status,
			p.Sprintf(txsFormat,
				txs.Date.Format(dateFormat),
				*txs.Value.Rate,
				txs.Place,
				"FAILED"))
	}

	return strings.Join(status, "\n")
}

func Run(ctx context.Context, auth types.Auth) error {
	var status txsStatus

	log := logger.GetLogger()
	defer log.Sync()

	defer func() {
		log.Infow("Synced transactions",
			"successful", len(status.SuccessfulTxs),
			"failed", len(status.FailedTxs),
			"failed_to_parse", status.ParseFailures,
		)

		shouldNotify := status.ParseFailures > 0
		shouldNotify = shouldNotify || len(status.FailedTxs) > 0
		shouldNotify = shouldNotify || len(status.SuccessfulTxs) > 0

		if shouldNotify && auth.TwilioAccountSid != "" {
			msg := notificationString(status.SuccessfulTxs, status.FailedTxs, status.ParseFailures)
			SendNotifications(auth, msg)
		}
	}()

	log.Infow("Timezone Locale",
		"timezone", auth.Timezone)
	SetTimezoneLocale(auth.Timezone)

	banks := bank.GetBanks()

	mailClient, err := imap.GetMailClient(auth.Addr, auth.Username, auth.Password)
	if err != nil {
		return err
	}
	defer mailClient.Logout()

	msgs, err := GetEmailFromInbox(mailClient, banks)
	if err != nil {
		return err
	}

	var transactions []*types.TransactionInfo
	transactions, status.ParseFailures = ExtractTransactionInfoFromMessages(msgs)

	if status.ParseFailures > 0 {
		log.Infow("Had failures extracting information from messages",
			"failures", status.ParseFailures,
		)
	}

	if len(transactions) == 0 {
		log.Info("no transactions to process, exiting ... ")
		return nil
	}

	log.Debug("Transactions to process")
	for i, t := range transactions {
		log.Debugf("%d: %+v", i, t)
	}

	toshlClient := toshl.NewApiClient(auth.ToshlToken)

	internalCategoryIds := createInternalCategoryIdsIfAbsent(toshlClient)

	accounts, err := toshlClient.GetAccounts()
	if err != nil {
		return err
	}

	log.Debug("Account")
	for i, a := range accounts {
		log.Debugf("%d: %s", i, a.Name)
	}

	mappableAccounts := GetMappableAccounts(accounts)

	log.Debug("Mappable accounts")
	for name, account := range mappableAccounts {
		log.Debugf("%s: %s", name, account.Name)
	}

	cleanTransactionsAccount(transactions, mappableAccounts, auth.AccountMappings)

	status.SuccessfulTxs, status.FailedTxs = CreateEntries(toshlClient, transactions,
		mappableAccounts, internalCategoryIds)

	ArchiveEmailsFromSuccessfulTransactions(mailClient, auth.ArchiveMailbox, status.SuccessfulTxs)
	if err := UpdateLastProcessedDate(status.FailedTxs); err != nil {
		return fmt.Errorf("failed to update last processed date: %s", err)
	}

	return nil
}

func cleanTransactionsAccount(transactions []*types.TransactionInfo, mappableAccounts map[string]*toshl.Account, accountMappings map[string]types.AccountMapping) {
	log := logger.GetLogger()
	defer log.Sync()

	for _, tx := range transactions {
		accountName := tx.Account
		_, ok := mappableAccounts[accountName]
		if !ok {
			log.Warnw("transaction is not directly mappable to a toshl account",
				"accountName", accountName)

			tryToFixTransactionAccountMapping(accountMappings, accountName, tx)
		}
	}
}

func tryToFixTransactionAccountMapping(accountMappings map[string]types.AccountMapping, accountName string, tx *types.TransactionInfo) {
	log := logger.GetLogger()

	bank := tx.Bank.String()
	bankMapping, found := accountMappings[bank]
	if !found {
		log.Errorw("mapping not found for transaction bank",
			"accountName", accountName, "bank", bank)
		return
	}

	fmt.Println(bankMapping)

	if mapping, found := bankMapping[accountName]; found {
		log.Infow("mapping found for transaction account",
			"accountName", accountName, "mapping", mapping)
		tx.Account = mapping
	} else {
		log.Errorw("mapping not found for transaction",
			"accountName", accountName)
	}
}

func createInternalCategoryIdsIfAbsent(toshlClient toshl.ApiClient) map[types.TransactionType]string {
	internalCategoryIds := map[types.TransactionType]string{}

	internalCategoryIds[types.Expense] = CreateInternalCategoryIfAbsent(toshlClient, types.Expense)
	internalCategoryIds[types.Income] = CreateInternalCategoryIfAbsent(toshlClient, types.Income)

	return internalCategoryIds
}
