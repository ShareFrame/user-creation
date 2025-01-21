package config

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"log"
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
		DynamoTableName: "Users",
		AtProtoBaseURL:  "https://shareframe.social",
	}, awsCfg, nil
}

func RetrieveAdminCreds() (string, error) {
	secretName := "pds_admin"
	region := "us-east-1"

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(region))
	if err != nil {
		log.Fatal(err)
	}

	svc := secretsmanager.NewFromConfig(cfg)

	input := &secretsmanager.GetSecretValueInput{
		SecretId:     aws.String(secretName),
		VersionStage: aws.String("AWSCURRENT"), // VersionStage defaults to AWSCURRENT if unspecified
	}

	result, err := svc.GetSecretValue(context.TODO(), input)
	if err != nil {
		log.Fatal(err.Error())
	}

	return *result.SecretString, nil
}
