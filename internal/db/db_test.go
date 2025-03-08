package db

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ShareFrame/user-management/internal/db/mocks"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func createTestUser() (models.CreateUserResponse, models.UserRequest) {
	return models.CreateUserResponse{
			DID:    "did:example:123",
			Handle: "testuser",
		}, models.UserRequest{
			Email:    "test@example.com",
			Handle:   "testuser",
			Password: "Password123!",
		}
}

func TestStoreUser(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDynamoDB := mocks.NewMockDynamoDBAPI(ctrl)
	dynamoClient := &DynamoDBClient{Client: mockDynamoDB}
	testUser, testEvent := createTestUser()

	themeJSON, _ := json.Marshal(map[string]interface{}{})

	expectedInput := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item: map[string]types.AttributeValue{
			"UserId":         &types.AttributeValueMemberS{Value: testUser.DID},
			"Did":            &types.AttributeValueMemberS{Value: testUser.DID},
			"Email":          &types.AttributeValueMemberS{Value: testEvent.Email},
			"Handle":         &types.AttributeValueMemberS{Value: testUser.Handle},
			"CreatedAt":      &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
			"ModifiedAt":     &types.AttributeValueMemberS{Value: time.Now().Format(time.RFC3339)},
			"Status":         &types.AttributeValueMemberS{Value: "active"},
			"Verified":       &types.AttributeValueMemberBOOL{Value: false},
			"Role":           &types.AttributeValueMemberS{Value: "user"},
			"DisplayName":    &types.AttributeValueMemberS{Value: testUser.Handle},
			"ProfilePicture": &types.AttributeValueMemberS{Value: ""},
			"ProfileBanner":  &types.AttributeValueMemberS{Value: ""},
			"Theme":          &types.AttributeValueMemberS{Value: string(themeJSON)},
			"PrimaryColor":   &types.AttributeValueMemberS{Value: "#FFFFFF"},
			"SecondaryColor": &types.AttributeValueMemberS{Value: "#000000"},
		},
	}

	tests := []struct {
		name        string
		mockSetup   func()
		expectError bool
	}{
		{
			name: "Successful StoreUser",
			mockSetup: func() {
				mockDynamoDB.EXPECT().PutItem(gomock.Any(), expectedInput).Return(&dynamodb.PutItemOutput{}, nil)
			},
			expectError: false,
		},
		{
			name: "Failed StoreUser (DynamoDB error)",
			mockSetup: func() {
				mockDynamoDB.EXPECT().PutItem(gomock.Any(), expectedInput).Return(nil, errors.New("DynamoDB error"))
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			err := dynamoClient.StoreUser(context.TODO(), testUser, testEvent)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCheckEmailExists(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockDynamoDB := mocks.NewMockDynamoDBAPI(ctrl)
	dynamoClient := &DynamoDBClient{Client: mockDynamoDB}

	testEmail := "test@example.com"

	tests := []struct {
		name         string
		mockSetup    func()
		expectExists bool
		expectError  bool
	}{
		{
			name: "Email exists",
			mockSetup: func() {
				mockDynamoDB.EXPECT().Query(gomock.Any(), &dynamodb.QueryInput{
					TableName:              aws.String("Users"),
					IndexName:              aws.String("Email-index"),
					KeyConditionExpression: aws.String("Email = :email"),
					ExpressionAttributeValues: map[string]types.AttributeValue{
						":email": &types.AttributeValueMemberS{Value: testEmail},
					},
					Limit: aws.Int32(1),
				}).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{
					{"Email": &types.AttributeValueMemberS{Value: testEmail}},
				}}, nil)
			},
			expectExists: true,
			expectError:  false,
		},
		{
			name: "Email does not exist",
			mockSetup: func() {
				mockDynamoDB.EXPECT().Query(gomock.Any(), gomock.Any()).Return(&dynamodb.QueryOutput{Items: []map[string]types.AttributeValue{}}, nil)
			},
			expectExists: false,
			expectError:  false,
		},
		{
			name: "DynamoDB error",
			mockSetup: func() {
				mockDynamoDB.EXPECT().Query(gomock.Any(), gomock.Any()).Return(nil, errors.New("DynamoDB error"))
			},
			expectExists: false,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.mockSetup()

			exists, err := dynamoClient.CheckEmailExists(context.TODO(), testEmail)
			assert.Equal(t, tt.expectExists, exists)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
