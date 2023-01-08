package bancolombia

import (
	"testing"
	"time"

	mail "github.com/Philanthropists/toshl-email-autosync/v2/internal/mail/types"
	"github.com/Philanthropists/toshl-email-autosync/v2/internal/types"
	_imap "github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
)

const CopCode = "COP"

type transInfo struct {
	TransactionType types.TransactionType
	Place           string
	Account         string
	Value           types.Amount
}

type result struct {
	Result transInfo
	Err    error
}

type regexpTest struct {
	Body   string
	Result result
}

var regexpExpressions []regexpTest = []regexpTest{
	{
		Body: "Bancolombia le informa compra por $30,000.00 a Prueba desde cta *0000.",
		Result: result{
			Result: transInfo{
				TransactionType: types.Expense,
				Place:           "Prueba",
				Account:         "0000",
				Value:           generateCurrency(CopCode, 30000.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Compra por $3.150,00 en EXITO EXPRESS AVE 19 19:02. 20/09/2022 T.Deb *5021.",
		Result: result{
			Result: transInfo{
				TransactionType: types.Expense,
				Place:           "EXITO EXPRESS AVE 19 19:02",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 3150.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Retiro por $70.000,00 en CALLE100-2. Hora 16:44 01/09/2022 T.Deb *5021.",
		Result: result{
			Result: transInfo{
				TransactionType: types.Expense,
				Place:           "CALLE100-2",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 70000.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Retiro por $100.000,00 en EXITO_FUSAG. Hora 18:20 14/08/2022 T.Deb *5021.",
		Result: result{
			Result: transInfo{
				TransactionType: types.Expense,
				Place:           "EXITO_FUSAG",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 100000.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Retiro por $150.000,00 en MF_OCEANMA2. Hora 10:30 13/07/2022 T.Deb *5021.",
		Result: result{
			Result: transInfo{
				TransactionType: types.Expense,
				Place:           "MF_OCEANMA2",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 150000.0),
			},
		},
	},
	{
		Body: "Bancolombia te informa Pago por $112,400.00 a A Toda Hora SA desde producto *9785.",
		Result: result{
			Result: transInfo{
				TransactionType: types.Expense,
				Place:           "A Toda Hora SA",
				Account:         "9785",
				Value:           generateCurrency(CopCode, 112400.0),
			},
		},
	},
	{
		Body: "Bancolombia le informa Compra por $23.050,00 en RAPPI RESTAURANTE 11:25. 04/01/2023 T.Cred *3616.",
		Result: result{
			Result: transInfo{
				TransactionType: types.Expense,
				Place:           "RAPPI RESTAURANTE 11:25",
				Account:         "3616",
				Value:           generateCurrency(CopCode, 23050.0),
			},
		},
	},
}

func generateCurrency(code string, rate float64) types.Amount {
	return types.Amount{
		Number: rate,
		Code:   code,
	}
}

func generateTestMessage(body string) types.Message {
	return types.Message{
		Message: &mail.Message{
			Message: &_imap.Message{
				SeqNum: 1,
				Envelope: &_imap.Envelope{
					Date: time.Now(),
				},
			},
			RawBody: []byte(body),
		},
	}
}

func Test_RegexpExpressions(t *testing.T) {
	bank := Bancolombia{}

	for _, r := range regexpExpressions {
		msg := generateTestMessage(r.Body)

		res, err := bank.ExtractTransactionInfoFromMessage(msg)

		if r.Result.Err == nil {
			assert.NotNil(t, res)
			assert.Equal(t, res.Type, r.Result.Result.TransactionType)
			assert.Equal(t, res.Place, r.Result.Result.Place)
			assert.Equal(t, res.Account, r.Result.Result.Account)
			assert.Equal(t, res.Value, r.Result.Result.Value)
		} else {
			assert.Error(t, err)
		}
	}

}
