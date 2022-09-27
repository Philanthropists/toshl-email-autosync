package bancolombia

import (
	"testing"
	"time"

	"github.com/Philanthropists/toshl-email-autosync/internal/datasource/imap/types"
	synctypes "github.com/Philanthropists/toshl-email-autosync/internal/sync/types"
	"github.com/Philanthropists/toshl-go"
	_imap "github.com/emersion/go-imap"
	"github.com/stretchr/testify/assert"
)

const CopCode = "COP"

type transInfo struct {
	TransactionType synctypes.TransactionType
	Place           string
	Account         string
	Value           synctypes.Currency
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
				TransactionType: synctypes.Expense,
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
				TransactionType: synctypes.Expense,
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
				TransactionType: synctypes.Expense,
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
				TransactionType: synctypes.Expense,
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
				TransactionType: synctypes.Expense,
				Place:           "MF_OCEANMA2",
				Account:         "5021",
				Value:           generateCurrency(CopCode, 150000.0),
			},
		},
	},
}

func generateCurrency(code string, rate float64) synctypes.Currency {
	var ratePtr *float64 = new(float64)
	*ratePtr = rate

	return synctypes.Currency{
		Currency: toshl.Currency{
			Code: code,
			Rate: ratePtr,
		},
	}
}

func generateTestMessage(body string) types.Message {
	return types.Message{
		Message: &_imap.Message{
			SeqNum: 1,
			Envelope: &_imap.Envelope{
				Date: time.Now(),
			},
		},
		RawBody: []byte(body),
	}
}

func Test_RegexpExpressions(t *testing.T) {
	bank := Bancolombia{}

	for _, r := range regexpExpressions {
		msg := generateTestMessage(r.Body)

		res, err := bank.ExtractTransactionInfoFromMessage(msg)

		if r.Result.Err == nil {
			assert.NotNil(t, res)
			assert.Equal(t, res.TransactionType, r.Result.Result.TransactionType)
			assert.Equal(t, res.Place, r.Result.Result.Place)
			assert.Equal(t, res.Account, r.Result.Result.Account)
			assert.Equal(t, res.Value, r.Result.Result.Value)
		} else {
			assert.Error(t, err)
		}
	}

}
