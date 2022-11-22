package dynamodb

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type dynamoClient interface {
	Scan(context.Context, *dynamodb.ScanInput, ...func(*dynamodb.Options)) (*dynamodb.ScanOutput, error)
	GetItem(context.Context, *dynamodb.GetItemInput, ...func(*dynamodb.Options)) (*dynamodb.GetItemOutput, error)
	UpdateItem(context.Context, *dynamodb.UpdateItemInput, ...func(*dynamodb.Options)) (*dynamodb.UpdateItemOutput, error)
}

type Client struct {
	DynamoDBClient dynamoClient
}

func NewDynamoDBClient(ctx context.Context, region string) (Client, error) {
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return Client{}, err
	}

	dynamoClient := dynamodb.NewFromConfig(cfg)

	return Client{
		DynamoDBClient: dynamoClient,
	}, nil
}

func (c Client) dynamodb() (dynamoClient, error) {
	if c.DynamoDBClient == nil {
		return nil, errors.New("there is no DynamoDBClient defined")
	}

	return c.DynamoDBClient, nil
}

func (c Client) Scan(ctx context.Context, table string) ([]map[string]interface{}, error) {
	dynamo, err := c.dynamodb()
	if err != nil {
		return nil, err
	}

	p := dynamodb.ScanInput{
		TableName: aws.String(table),
	}

	res, err := dynamo.Scan(ctx, &p)
	if err != nil {
		return nil, err
	}

	resConv := make([]map[string]interface{}, 0)
	err = attributevalue.UnmarshalListOfMaps(res.Items, &resConv)
	if err != nil {
		return nil, err
	}

	return resConv, nil
}
func (c Client) GetItem(ctx context.Context, table string, k map[string]interface{}) (map[string]interface{}, error) {
	dynamo, err := c.dynamodb()
	if err != nil {
		return nil, err
	}

	key, err := attributevalue.MarshalMap(k)
	if err != nil {
		return nil, err
	}

	ps := &dynamodb.GetItemInput{
		Key:       key,
		TableName: aws.String(table),
	}

	res, err := dynamo.GetItem(ctx, ps)
	if err != nil {
		return nil, err
	}

	resConv := make(map[string]interface{})
	err = attributevalue.UnmarshalMap(res.Item, &resConv)
	if err != nil {
		return nil, err
	}

	return resConv, nil
}

func (c Client) UpdateItem(ctx context.Context, table string, k map[string]interface{}, expressionAttributeValues map[string]interface{}, updateExpression string) error {
	dynamo, err := c.dynamodb()
	if err != nil {
		return err
	}

	key, err := attributevalue.MarshalMap(k)
	if err != nil {
		return err
	}

	expAttrValConv, err := attributevalue.MarshalMap(expressionAttributeValues)
	if err != nil {
		return err
	}

	ps := &dynamodb.UpdateItemInput{
		Key:                       key,
		ExpressionAttributeValues: expAttrValConv,
		TableName:                 aws.String(table),
		ReturnValues:              types.ReturnValueUpdatedNew,
		UpdateExpression:          aws.String(updateExpression),
	}

	_, err = dynamo.UpdateItem(ctx, ps)
	return err
}
