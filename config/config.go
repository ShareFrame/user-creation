package config

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
)

type Config struct {
	DynamoTableName string
	AtProtoBaseURL  string
}

func LoadConfig() (*Config, error) {
	_, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}

	return &Config{
		DynamoTableName: "YourDynamoDBTableName",
		AtProtoBaseURL:  "https://example.com",
	}, nil
}
