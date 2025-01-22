package config

import (
	"context"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"os"
)

type Config struct {
	DynamoTableName string
	AtProtoBaseURL  string
}

func LoadConfig(ctx context.Context) (*Config, aws.Config, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	tableName := os.Getenv("DYNAMO_TABLE_NAME")
	baseURL := os.Getenv("ATPROTO_BASE_URL")

	if tableName == "" {
		return nil, aws.Config{}, errors.New("DYNAMO_TABLE_NAME environment variable is required")
	}

	if baseURL == "" {
		return nil, aws.Config{}, errors.New("ATPROTO_BASE_URL environment variable is required")
	}

	return &Config{
		DynamoTableName: tableName,
		AtProtoBaseURL:  baseURL,
	}, awsCfg, nil
}

func RetrieveAdminCreds(ctx context.Context) (string, error) {
	secretName := os.Getenv("PDS_ADMIN_SECRET_NAME")
	region := os.Getenv("AWS_REGION")

	if secretName == "" {
		return "", errors.New("PDS_ADMIN_SECRET_NAME environment variable is required")
	}

	if region == "" {
		region = "us-east-1"
	}

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return "", fmt.Errorf("failed to load AWS config: %w", err)
	}

	svc := secretsmanager.NewFromConfig(cfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"),
	}

	result, err := svc.GetSecretValue(ctx, input)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve secret: %w", err)
	}

	if result.SecretString == nil {
		return "", fmt.Errorf("secret string is nil")
	}

	return *result.SecretString, nil
}
