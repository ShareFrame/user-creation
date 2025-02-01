package dynamo

import (
	"context"
	"fmt"
	"time"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sirupsen/logrus"
)

const (
	DefaultStatus   = "Active"
	DefaultVerified = false
	DefaultRole     = "User"
	DefaultPicture  = ""
)

type DynamoDBAPI interface {
	PutItem(ctx context.Context, input *dynamodb.PutItemInput, opts ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
	Query(ctx context.Context, input *dynamodb.QueryInput, opts ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type DynamoDBService interface {
	CheckEmailExists(ctx context.Context, email string) (bool, error)
}
var _ DynamoDBService = (*DynamoClient)(nil)

type DynamoClient struct {
	Client    DynamoDBAPI
	TableName string
	TimeZone  string
}

func NewDynamoClient(client DynamoDBAPI, tableName string, timeZone string) *DynamoClient {
	return &DynamoClient{
		Client:    client,
		TableName: tableName,
		TimeZone:  timeZone,
	}
}

func (d *DynamoClient) StoreUser(user models.CreateUserResponse, event models.UserRequest) error {
	if user.DID == "" || user.Handle == "" {
		logrus.Warnf("User DID or Handle is missing: %+v", user)
		return fmt.Errorf("user DID and handle are required")
	}

	loc, err := time.LoadLocation(d.TimeZone)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"time_zone": d.TimeZone,
			"user":      user.DID,
		}).Error("Failed to load time zone")
		return fmt.Errorf("failed to load time zone: %w", err)
	}

	item := map[string]types.AttributeValue{
		"UserId":            &types.AttributeValueMemberS{Value: user.DID},
		"Email":             &types.AttributeValueMemberS{Value: event.Email},
		"Handle":            &types.AttributeValueMemberS{Value: user.Handle},
		"Created":           &types.AttributeValueMemberS{Value: time.Now().In(loc).Format(time.RFC3339)},
		"Status":            &types.AttributeValueMemberS{Value: DefaultStatus},
		"Verified":          &types.AttributeValueMemberBOOL{Value: DefaultVerified},
		"Role":              &types.AttributeValueMemberS{Value: DefaultRole},
		"DisplayName":       &types.AttributeValueMemberS{Value: user.Handle},
		"ProfilePictureCID": &types.AttributeValueMemberS{Value: DefaultPicture},
	}

	_, err = d.Client.PutItem(context.TODO(), &dynamodb.PutItemInput{
		TableName: &d.TableName,
		Item:      item,
	})

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"user_id": user.DID,
			"email":   event.Email,
			"handle":  user.Handle,
			"item":    item,
		}).Error("Failed to store user in DynamoDB")
		return fmt.Errorf("failed to store user in DynamoDB: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"user_id": user.DID,
		"email":   event.Email,
		"handle":  user.Handle,
		"status":  DefaultStatus,
		"role":    DefaultRole,
	}).Info("User successfully stored in DynamoDB")

	return nil
}

func (d *DynamoClient) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	input := &dynamodb.QueryInput{
		TableName:              &d.TableName,
		IndexName:              aws.String("Email-index"),
		KeyConditionExpression: aws.String("Email = :email"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":email": &types.AttributeValueMemberS{Value: email},
		},
		Limit: aws.Int32(1),
	}

	result, err := d.Client.Query(ctx, input)
	if err != nil {
		logrus.WithError(err).WithField("email", email).Error("Error checking email existence in DynamoDB")
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return len(result.Items) > 0, nil
}
