package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ShareFrame/user-management/config"
	ATProtocol "github.com/ShareFrame/user-management/internal/atproto"
	"github.com/ShareFrame/user-management/internal/helper"
	"github.com/ShareFrame/user-management/internal/models"
	"github.com/ShareFrame/user-management/internal/postgres"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/sirupsen/logrus"
)

func UserHandler(ctx context.Context, event models.UserRequest) (*models.CreateUserResponse, error) {
	logrus.WithField("handle", event.Handle).Info("Processing create account request")

	cfg, awsCfg, err := config.LoadConfig(ctx)
	if err != nil {
		logrus.WithError(err).Error("Failed to load application configuration")
		return nil, fmt.Errorf("internal error: failed to load application configuration: %w", err)
	}

	rdsClient := rdsdata.NewFromConfig(awsCfg)
	secretsManagerClient := secretsmanager.NewFromConfig(awsCfg)

	dbClient := postgres.NewPostgresDB(rdsClient, cfg.DBClusterARN, cfg.SecretARN, cfg.DatabaseName)

	updatedEvent, err := helper.ValidateAndFormatUser(ctx, event, dbClient)
	if err != nil {
		logrus.WithError(err).Warn("Validation error")
		return nil, fmt.Errorf("validation error: %w", err)
	}
	event = updatedEvent

	adminCreds, err := helper.RetrieveAdminCredentials(ctx, secretsManagerClient)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve admin credentials")
		return nil, fmt.Errorf("internal error: could not retrieve admin credentials: %w", err)
	}

	atProtoClient := ATProtocol.NewATProtocolClient(cfg.AtProtoBaseURL, &http.Client{})
	logrus.WithField("base_url", cfg.AtProtoBaseURL).Info("Initializing ATProtocol client")

	inviteCode, err := atProtoClient.CreateInviteCode(adminCreds)
	if err != nil {
		logrus.WithError(err).Error("Failed to generate invite code using AT Protocol")
		return nil, fmt.Errorf("internal error: failed to generate invite code: %w", err)
	}

	utilAccountCreds, err := helper.RetrieveUtilAccountCreds(ctx, secretsManagerClient)
	if err != nil {
		logrus.WithError(err).Error("Failed to retrieve util account credentials")
		return nil, fmt.Errorf("internal error: could not retrieve authentication credentials: %w", err)
	}

	session, err := atProtoClient.CreateSession(utilAccountCreds.Username, utilAccountCreds.Password)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"username": utilAccountCreds.Username,
			"error":    err.Error(),
		}).Error("Failed to authenticate with AT Protocol")
		return nil, fmt.Errorf("authentication failed for user %s: %w", utilAccountCreds.Username, err)
	}

	logrus.Info("Session created successfully")

	exists, err := atProtoClient.CheckUserExists(event.Handle, session.AccessJwt)
	if err != nil {
		logrus.WithError(err).WithField("handle", event.Handle).Error("Failed to check user existence")
		return nil, fmt.Errorf("internal error: failed to check if user exists: %w", err)
	}

	if exists {
		logrus.WithField("handle", event.Handle).Warn("User already exists on PDS")
		return nil, fmt.Errorf("user already exists with handle: %s", event.Handle)
	}

	user, err := atProtoClient.RegisterUser(event.Handle, event.Email, inviteCode.Code, event.Password)
	if err != nil {
		logrus.WithError(err).WithFields(logrus.Fields{
			"handle": event.Handle,
			"email":  event.Email,
		}).Error("Failed to register user via AT Protocol")
		return nil, fmt.Errorf("failed to register user: %w", err)
	}

	if err = dbClient.StoreUser(ctx, user, updatedEvent); err != nil {
		logrus.WithError(err).Error("Failed to store user in PostgreSQL")
		return nil, fmt.Errorf("internal error: failed to store user data: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"did":    user.DID,
		"handle": user.Handle,
	}).Info("Successfully created and stored user")

	return &user, nil
}
