package db

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type DynamoClient struct {
	Client    *dynamodb.Client
	TableName string
}

func NewDynamoClient(client *dynamodb.Client, tableName string) *DynamoClient {
	return &DynamoClient{Client: client, TableName: tableName}
}

func (d *DynamoClient) StoreUser(userID, email string) error {
	_, err := d.Client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &d.TableName,
		Item: map[string]types.AttributeValue{
			"UserID": &types.AttributeValueMemberS{Value: userID},
			"Email":  &types.AttributeValueMemberS{Value: email},
		},
	})
	return err
}
