package config

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/sirupsen/logrus"
)

type PostgresSecret struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Database     string `json:"database"`
	Host         string `json:"host"`
	Port         string `json:"port"`
	DBClusterARN string `json:"dbClusterArn"`
	SecretARN    string `json:"secretArn"`
}

type Config struct {
	DBClusterARN    string
	SecretARN       string
	DatabaseName    string
	PostgresConnStr string
	AtProtoBaseURL  string
}

type SecretsManagerAPI interface {
	GetSecretValue(ctx context.Context, input *secretsmanager.GetSecretValueInput, opts ...func(*secretsmanager.Options)) (*secretsmanager.GetSecretValueOutput, error)
}

func LoadConfig(ctx context.Context, secretsClient SecretsManagerAPI) (*Config, aws.Config, error) {
	awsCfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, aws.Config{}, fmt.Errorf("failed to load AWS config: %w", err)
	}

	secretName := os.Getenv("POSTGRES_CONN_STR")
	baseURL := os.Getenv("ATPROTO_BASE_URL")

	if secretName == "" {
		return nil, aws.Config{}, errors.New("POSTGRES_CONN_STR environment variable is required")
	}

	if baseURL == "" {
		return nil, aws.Config{}, errors.New("ATPROTO_BASE_URL environment variable is required")
	}

	secretValue, err := RetrieveSecret(ctx, secretName, secretsClient)
	if err != nil {
		return nil, aws.Config{}, fmt.Errorf("failed to retrieve PostgreSQL secret: %w", err)
	}

	var secret PostgresSecret
	if err := json.Unmarshal([]byte(secretValue), &secret); err != nil {
		return nil, aws.Config{}, fmt.Errorf("failed to parse PostgreSQL secret JSON: %w", err) // **Fix: Proper error**
	}

	if secret.Database == "" || secret.Host == "" || secret.Username == "" || secret.Password == "" {
		return nil, aws.Config{}, errors.New("parsed PostgreSQL secret is missing required fields")
	}

	formattedConnStr := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=require",
		url.QueryEscape(secret.Username), url.QueryEscape(secret.Password),
		secret.Host, secret.Port, secret.Database,
	)

	logrus.WithFields(logrus.Fields{
		"host":         secret.Host,
		"database":     secret.Database,
		"dbClusterArn": secret.DBClusterARN,
		"secretArn":    secret.SecretARN,
	}).Info("Successfully loaded PostgreSQL connection details")

	return &Config{
		DBClusterARN:    secret.DBClusterARN,
		SecretARN:       secret.SecretARN,
		DatabaseName:    secret.Database,
		PostgresConnStr: formattedConnStr,
		AtProtoBaseURL:  baseURL,
	}, awsCfg, nil
}

func RetrieveSecret(ctx context.Context, secretName string, svc SecretsManagerAPI) (string, error) {
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
