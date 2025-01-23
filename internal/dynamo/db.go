package dynamo

import (
	"context"
	"fmt"
	"time"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	DefaultStatus   = "Active"
	DefaultVerified = false
	DefaultRole     = "User"
	DefaultPicture  = ""
)

type DynamoClient struct {
	Client    DynamoDBAPI
	TableName string
	TimeZone  string
}

type DynamoDBAPI interface {
	PutItem(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.
		PutItemOutput, error)
}

func NewDynamoClient(client DynamoDBAPI, tableName string, timeZone string) *DynamoClient {
	return &DynamoClient{
		Client:    client,
		TableName: tableName,
		TimeZone:  timeZone,
	}
}

func (d *DynamoClient) StoreUser(user models.CreateUserResponse) error {
	if user.DID == "" || user.Handle == "" {
		return fmt.Errorf("user DID and handle are required")
	}

	loc, err := time.LoadLocation(d.TimeZone)
	if err != nil {
		return fmt.Errorf("failed to load time zone: %w", err)
	}

	_, err = d.Client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &d.TableName,
		Item: map[string]types.AttributeValue{
			"UserId":            &types.AttributeValueMemberS{Value: user.DID},
			"Email":             &types.AttributeValueMemberS{Value: user.Email},
			"Handle":            &types.AttributeValueMemberS{Value: user.Handle},
			"Created":           &types.AttributeValueMemberS{Value: time.Now().In(loc).Format(time.RFC3339)},
			"Status":            &types.AttributeValueMemberS{Value: DefaultStatus},
			"Verified":          &types.AttributeValueMemberBOOL{Value: DefaultVerified},
			"Role":              &types.AttributeValueMemberS{Value: DefaultRole},
			"DisplayName":       &types.AttributeValueMemberS{Value: user.Handle},
			"ProfilePictureCID": &types.AttributeValueMemberS{Value: DefaultPicture},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to store user in DynamoDB: %w", err)
	}

	return nil
}
