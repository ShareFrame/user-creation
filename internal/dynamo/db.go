package dynamo

import (
	"context"
	"fmt"
	"github.com/Atlas-Mesh/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"time"
)

const (
	DefaultStatus   = "Active"
	DefaultVerified = false
	DefaultRole     = "User"
	DefaultPicture  = ""
)

type DynamoClient struct {
	Client    *dynamodb.Client
	TableName string
	Location  *time.Location
}

func NewDynamoClient(client *dynamodb.Client, tableName, location string) (*DynamoClient, error) {
	loc, err := time.LoadLocation(location)
	if err != nil {
		return nil, fmt.Errorf("failed to load location: %w", err)
	}

	return &DynamoClient{
		Client:    client,
		TableName: tableName,
		Location:  loc,
	}, nil
}

func (d *DynamoClient) StoreUser(user models.CreateUserResponse) error {
	if user.DID == "" || user.Handle == "" {
		return fmt.Errorf("user DID and handle are required")
	}

	_, err := d.Client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &d.TableName,
		Item: map[string]types.AttributeValue{
			"UserId":            &types.AttributeValueMemberS{Value: user.DID},
			"Email":             &types.AttributeValueMemberS{Value: user.Email},
			"Handle":            &types.AttributeValueMemberS{Value: user.Handle},
			"Created":           &types.AttributeValueMemberS{Value: time.Now().In(d.Location).Format(time.RFC3339)},
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
