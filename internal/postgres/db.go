package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/ShareFrame/user-management/internal/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata"
	"github.com/aws/aws-sdk-go-v2/service/rdsdata/types"
	"github.com/sirupsen/logrus"
)

const (
	DefaultStatus   = "active"
	DefaultVerified = false
	DefaultRole     = "user"
	DefaultPicture  = ""
	DefaultBanner   = ""
	DefaultTheme    = "{}"
	DefaultColor1   = "#FFFFFF"
	DefaultColor2   = "#000000"
	QueryTimeout    = 3 * time.Second
)

type PostgresDBService interface {
	CheckEmailExists(ctx context.Context, email string) (bool, error)
	StoreUser(ctx context.Context, user models.CreateUserResponse, event models.UserRequest) error
}

type RDSDataAPI interface {
	ExecuteStatement(ctx context.Context, input *rdsdata.ExecuteStatementInput, opts ...func(*rdsdata.Options)) (*rdsdata.ExecuteStatementOutput, error)
}

type PostgresDB struct {
	Client       RDSDataAPI
	DBClusterARN string
	SecretARN    string
	DatabaseName string
}

func NewPostgresDB(client RDSDataAPI, dbClusterARN, secretARN, database string) *PostgresDB {
	return &PostgresDB{
		Client:       client,
		DBClusterARN: dbClusterARN,
		SecretARN:    secretARN,
		DatabaseName: database,
	}
}

func (p *PostgresDB) StoreUser(ctx context.Context, user models.CreateUserResponse, event models.UserRequest) error {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	query := `
		INSERT INTO users 
		(did, email, handle, created_at, modified_at, status, verified, role, display_name, profile_picture, profile_banner, theme, primary_color, secondary_color) 
		VALUES 
		(:did, :email, :handle, NOW(), NOW(), :status, :verified, :role, :display_name, :profile_picture, :profile_banner, CAST(:theme AS JSONB), :primary_color, :secondary_color)`

	params := []types.SqlParameter{
		newSQLParam("did", user.DID),
		newSQLParam("email", event.Email),
		newSQLParam("handle", user.Handle),
		newSQLParam("status", DefaultStatus),
		newSQLParam("verified", DefaultVerified),
		newSQLParam("role", DefaultRole),
		newSQLParam("display_name", user.Handle),
		newSQLParam("profile_picture", DefaultPicture),
		newSQLParam("profile_banner", DefaultBanner),
		newSQLParam("theme", DefaultTheme),
		newSQLParam("primary_color", DefaultColor1),
		newSQLParam("secondary_color", DefaultColor2),
	}

	result, err := p.Client.ExecuteStatement(ctx, &rdsdata.ExecuteStatementInput{
		ResourceArn: aws.String(p.DBClusterARN),
		SecretArn:   aws.String(p.SecretARN),
		Database:    aws.String(p.DatabaseName),
		Sql:         aws.String(query),
		Parameters:  params,
	})

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"email":  event.Email,
			"handle": user.Handle,
		}).Errorf("Failed to store user: %v", err)
		return fmt.Errorf("failed to store user in PostgreSQL: %w", err)
	}

	if result == nil {
		logrus.WithFields(logrus.Fields{
			"email":  event.Email,
			"handle": user.Handle,
		}).Error("ExecuteStatement returned nil response")
		return fmt.Errorf("failed to store user in PostgreSQL: unexpected nil response")
	}

	logrus.Infof("User %s successfully stored in PostgreSQL", user.Handle)
	return nil
}



func (p *PostgresDB) CheckEmailExists(ctx context.Context, email string) (bool, error) {
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	query := `SELECT 1 FROM users WHERE email = :email LIMIT 1`
	params := []types.SqlParameter{newSQLParam("email", email)}

	result, err := p.Client.ExecuteStatement(ctx, &rdsdata.ExecuteStatementInput{
		ResourceArn: aws.String(p.DBClusterARN),
		SecretArn:   aws.String(p.SecretARN),
		Database:    aws.String(p.DatabaseName),
		Sql:         aws.String(query),
		Parameters:  params,
	})

	if err != nil {
		logrus.WithFields(logrus.Fields{
			"email": email,
		}).Errorf("Error checking email existence: %v", err)
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	if result == nil {
		logrus.WithFields(logrus.Fields{
			"email": email,
		}).Error("ExecuteStatement returned nil response")
		return false, fmt.Errorf("failed to check email existence: unexpected nil response")
	}

	return len(result.Records) > 0, nil
}


func newSQLParam(name string, value interface{}) types.SqlParameter {
	switch v := value.(type) {
	case string:
		return types.SqlParameter{Name: aws.String(name), Value: &types.FieldMemberStringValue{Value: v}}
	case bool:
		return types.SqlParameter{Name: aws.String(name), Value: &types.FieldMemberBooleanValue{Value: v}}
	default:
		logrus.Warnf("Unsupported SQL parameter type for %s", name)
		return types.SqlParameter{}
	}
}
