package config

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

type Config struct {
	DynamoTableName string
	AtProtoBaseURL  string
}

func LoadConfig() (*Config, aws.Config, error) {
	awsCfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, aws.Config{}, err
	}

	return &Config{
		DynamoTableName: "YourDynamoDBTableName",
		AtProtoBaseURL:  "https://example.com",
	}, awsCfg, nil
}
