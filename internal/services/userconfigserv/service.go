package userconfigserv

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/patrickmn/go-cache"
	"github.com/zeebo/errs"
)

const (
	table = "toshl-users"

	defaultExpiration = cache.NoExpiration
)

type MappingConfig map[string]string

type ToshlConfig struct {
	Token string `json:"token" dynamodbav:"Token"`
}

type UserConfig struct {
	Email             string                   `json:"email"               dynamodbav:"Email"`
	SMSDeliveryNumber string                   `json:"sms_delivery_number" dynamodbav:"SMSDeliveryNumber"`
	Toshl             ToshlConfig              `json:"toshl"               dynamodbav:"Toshl"`
	Mapping           map[string]MappingConfig `json:"account_mappings"    dynamodbav:"AccountMappings"`
}

type inMemoryCache interface {
	Set(k string, v any, t time.Duration)
	Get(k string) (any, bool)
	Delete(k string)
}

type DynamoDBService struct {
	Client *dynamodb.Client

	once  sync.Once
	cache inMemoryCache
}

func (r *DynamoDBService) init(ctx context.Context) {
	r.once.Do(func() {
		r.cache = cache.New(5*time.Minute, 1*time.Minute)

		r.PreloadAllConfigs(ctx)
	})
}

func (r *DynamoDBService) PreloadAllConfigs(ctx context.Context) error {
	scanIn := &dynamodb.ScanInput{
		TableName: aws.String(table),
	}

	items := []map[string]types.AttributeValue{}
	for {
		out, err := r.Client.Scan(ctx, scanIn)
		if err != nil {
			return errs.Wrap(err)
		}

		items = append(items, out.Items...)

		if out.LastEvaluatedKey == nil {
			break
		}

		scanIn.ExclusiveStartKey = out.LastEvaluatedKey
	}

	var expTime time.Duration = defaultExpiration
	if deadline, hasDeadline := ctx.Deadline(); hasDeadline {
		expTime = time.Until(deadline)
	}

	for _, it := range items {
		var cfg UserConfig
		err := attributevalue.UnmarshalMap(it, &cfg)
		if err != nil {
			return errs.Wrap(err)
		}

		r.cache.Set(cfg.Email, cfg, expTime)
	}

	return nil
}

func (r *DynamoDBService) GetUserConfigFromEmail(
	ctx context.Context,
	email string,
) (UserConfig, error) {
	r.init(ctx)

	if val, found := r.cache.Get(email); found {
		cfg := val.(UserConfig)
		return cfg, nil
	} else {
		return UserConfig{}, errors.New("not found")
	}

	// key, err := attributevalue.MarshalMap(map[string]any{
	// 	"Email": email,
	// })
	// if err != nil {
	// 	return UserConfig{}, errs.Wrap(err)
	// }
	//
	// res, err := r.Client.GetItem(ctx, &dynamodb.GetItemInput{
	// 	Key:       key,
	// 	TableName: aws.String(table),
	// })
	// if err != nil {
	// 	return UserConfig{}, errs.Wrap(err)
	// }
	//
	// var cfg UserConfig
	// err = attributevalue.UnmarshalMap(res.Item, &cfg)
	// if err != nil {
	// 	return UserConfig{}, errs.Wrap(err)
	// }
	//
	// r.cache.Set(email, cfg, 5*time.Minute)
	//
	// return cfg, nil
}

func (r *DynamoDBService) SaveUserConfig(ctx context.Context, cfg UserConfig) error {
	it, err := attributevalue.MarshalMap(cfg)
	if err != nil {
		return errs.Wrap(err)
	}

	_, err = r.Client.PutItem(ctx, &dynamodb.PutItemInput{
		Item:      it,
		TableName: aws.String(table),
	})

	r.cache.Set(cfg.Email, cfg, 5*time.Minute)

	return errs.Wrap(err)
}
