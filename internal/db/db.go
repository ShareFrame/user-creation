package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sirupsen/logrus"
)

const (
	QueryTimeout = 3 * time.Second
	TableName    = "Users"
)

type DynamoDBService interface {
	CheckEmailExists(ctx context.Context, email string) (bool, error)
	StoreUser(ctx context.Context, user models.CreateUserResponse, event models.UserRequest) error
}

type DynamoDBAPI interface {
	PutItem(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, input *dynamodb.QueryInput, opts ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type DynamoDBClient struct {
	Client DynamoDBAPI
}

func NewDynamoDBClient(client DynamoDBAPI) *DynamoDBClient {
	return &DynamoDBClient{
		Client: client,
	}
}

func (d *DynamoDBClient) StoreUser(ctx context.Context, user models.CreateUserResponse, event models.UserRequest) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	themeJSON, err := json.Marshal(map[string]interface{}{})
	if err != nil {
		return fmt.Errorf("failed to marshal theme: %w", err)
	}

	item := map[string]types.AttributeValue{
		"UserId":         &types.AttributeValueMemberS{Value: user.DID},
		"Email":          &types.AttributeValueMemberS{Value: event.Email},
		"Handle":         &types.AttributeValueMemberS{Value: user.Handle},
		"CreatedAt":      &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		"ModifiedAt":     &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
		"Status":         &types.AttributeValueMemberS{Value: "active"},
		"Verified":       &types.AttributeValueMemberBOOL{Value: false},
		"Role":           &types.AttributeValueMemberS{Value: "user"},
		"DisplayName":    &types.AttributeValueMemberS{Value: user.Handle},
		"ProfilePicture": &types.AttributeValueMemberS{Value: ""},
		"ProfileBanner":  &types.AttributeValueMemberS{Value: ""},
		"Theme":          &types.AttributeValueMemberS{Value: string(themeJSON)},
		"PrimaryColor":   &types.AttributeValueMemberS{Value: "#FFFFFF"},
		"SecondaryColor": &types.AttributeValueMemberS{Value: "#000000"},
	}

	_, err = d.Client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	})
	if err != nil {
		return fmt.Errorf("failed to store user in DynamoDB: %w", err)
	}

	return nil
}


func (d *DynamoDBClient) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	result, err := d.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:              aws.String("Users"),
		IndexName:              aws.String("Email-index"),
		KeyConditionExpression: aws.String("Email = :email"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{Value: email},
		},
		Limit: aws.Int32(1),
	})

	if err != nil {
		logrus.WithError(err).Error("Error checking email existence")
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return len(result.Items) > 0, nil
}



