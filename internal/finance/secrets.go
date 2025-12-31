package finance

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
)

// RDSSecret representa a estrutura do secret do RDS no Secrets Manager
// Contém apenas username e password (formato simplificado)
type RDSSecret struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// GetRDSCredentials busca as credenciais do RDS no AWS Secrets Manager
func GetRDSCredentials(ctx context.Context, secretName string) (*RDSSecret, error) {
	// Carregar configuração AWS (usa IAM role da EC2)
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Criar cliente Secrets Manager
	client := secretsmanager.NewFromConfig(cfg)

	// Buscar secret
	result, err := client.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	// Parse JSON do secret
	var secret RDSSecret
	if err := json.Unmarshal([]byte(*result.SecretString), &secret); err != nil {
		return nil, fmt.Errorf("failed to parse secret JSON: %w", err)
	}

	return &secret, nil
}

// BuildDatabaseURL constrói a connection string do PostgreSQL a partir do secret
// Requer host, port e dbname como parâmetros adicionais (vindos de env vars)
func BuildDatabaseURL(secret *RDSSecret, host string, port int, dbname string) string {
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=require",
		host,
		port,
		secret.Username,
		secret.Password,
		dbname,
	)
}