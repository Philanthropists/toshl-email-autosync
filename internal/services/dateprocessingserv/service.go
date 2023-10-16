package dateprocessingserv

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/zeebo/errs"

	"github.com/Philanthropists/toshl-email-autosync/v2/internal/logging"
)

const (
	OverrideLasProcessedDateEnvName = "OVERRIDE_LAST_PROC_DATE"

	table  = "toshl-data"
	field  = "LastProcessedDate"
	itemId = 1

	dateFormat = time.RFC822Z
)

var sinceFallbackDate time.Time = time.Now().Add(-30 * 24 * 60 * 60 * time.Second) // 30 days

type dynamoClient interface {
	Scan(
		context.Context,
		*dynamodb.ScanInput,
		...func(*dynamodb.Options),
	) (*dynamodb.ScanOutput, error)

	GetItem(
		context.Context,
		*dynamodb.GetItemInput,
		...func(*dynamodb.Options),
	) (*dynamodb.GetItemOutput, error)

	UpdateItem(
		context.Context,
		*dynamodb.UpdateItemInput,
		...func(*dynamodb.Options),
	) (*dynamodb.UpdateItemOutput, error)
}

type DynamoDBService struct {
	Client dynamoClient
}

func (r DynamoDBService) GetLastProcessedDate(
	ctx context.Context,
) (time.Time, error) {
	log := logging.New()
	defer func() { _ = log.Sync() }()

	log = log.With(logging.Time("fallbackDate", sinceFallbackDate))

	if overrideDate, ok := r.getLastProcessedDateOverride(); ok {
		return overrideDate, nil
	}

	if r.Client == nil {
		return time.Time{}, errs.New("dynamoDB client is nil")
	}

	since, err := r.getDateFromStorage(ctx)
	if err != nil {
		log.Error("could not get date from dynamodb", logging.Error(err))
		return sinceFallbackDate, nil
	}

	const oneDayBefore time.Duration = -24 * time.Hour
	since = since.Add(oneDayBefore)

	return since, nil
}

func (r DynamoDBService) getLastProcessedDateOverride() (time.Time, bool) {
	log := logging.New()
	defer func() { _ = log.Sync() }()

	if dateStr := os.Getenv(OverrideLasProcessedDateEnvName); dateStr != "" {
		const dateFormat = "2006-01-02"

		selectedDate, err := time.Parse(dateFormat, dateStr)
		if err == nil {
			log.Info("date is overriden",
				logging.Time("date_override", selectedDate),
			)
			return selectedDate, true
		} else {
			log.Error("override is set, but it is invalid")
		}
	}

	return time.Time{}, false
}

func (r DynamoDBService) SaveProcessedDate(
	ctx context.Context,
	t time.Time,
) error {
	key, err := attributevalue.MarshalMap(map[string]any{
		"Id": itemId,
	})
	if err != nil {
		return errs.Wrap(err)
	}

	value := t.Format(dateFormat)
	expAttrValues, err := attributevalue.MarshalMap(map[string]any{
		":r": value,
	})
	if err != nil {
		return errs.Wrap(err)
	}

	exp := fmt.Sprintf("set %s = :r", field)
	ps := &dynamodb.UpdateItemInput{
		Key:                       key,
		ExpressionAttributeValues: expAttrValues,
		TableName:                 aws.String(table),
		ReturnValues:              types.ReturnValueUpdatedNew,
		UpdateExpression:          aws.String(exp),
	}

	_, err = r.Client.UpdateItem(ctx, ps)
	if err != nil {
		return errs.New("could not update processing date: %w", err)
	}

	return nil
}

type ProcessedDate time.Time

func (d ProcessedDate) MarshalDynamoDBAttributeValue() (types.AttributeValue, error) {
	s := time.Time(d).Format(dateFormat)
	return &types.AttributeValueMemberS{
		Value: s,
	}, nil
}

func (d *ProcessedDate) UnmarshalDynamoDBAttributeValue(v types.AttributeValue) error {
	str, ok := v.(*types.AttributeValueMemberS)
	if !ok {
		return errs.New("field is not a string type: %v", v)
	}

	s := str.Value
	selectedDate, err := time.Parse(dateFormat, s)
	if err != nil {
		return errs.New(
			"%q is not a string representing a date: %w",
			s, err,
		)
	}

	*d = ProcessedDate(selectedDate)

	return nil
}

type DateObj struct {
	ProcessedDate ProcessedDate `dynamodbav:"LastProcessedDate"`
}

func (r DynamoDBService) getDateFromStorage(
	ctx context.Context,
) (time.Time, error) {
	key, err := attributevalue.MarshalMap(map[string]any{
		"Id": itemId,
	})
	if err != nil {
		return time.Time{}, err
	}

	res, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(table),
	})
	if err != nil {
		return time.Time{}, errs.New(
			"could not get item with id [%d] from dynamodb table [%s]: %w",
			itemId, table, err,
		)
	}

	var val DateObj
	err = attributevalue.UnmarshalMap(res.Item, &val)
	if err != nil {
		return time.Time{}, errs.Wrap(err)
	}

	return time.Time(val.ProcessedDate), nil
}
