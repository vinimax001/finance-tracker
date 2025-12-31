#!/bin/bash
set -e

# Log tudo
exec > >(tee /var/log/user-data.log)
exec 2>&1

echo "=== Iniciando configuração da aplicação Finance Tracker na EC2! ==="

# Atualizar sistema
sudo yum update -y

# Instalar dependências
sudo yum install -y postgresql17 aws-cli

# Criar diretório da aplicação
sudo mkdir -p /opt/finance-tracker
sudo chown ec2-user:ec2-user /opt/finance-tracker

# Baixar binário do S3
echo "=== Baixando binário do S3 ==="
aws s3 cp s3://BUCKET_NAME/releases/APP_VERSION/finance-tracker /opt/finance-tracker/finance-tracker
sudo chmod +x /opt/finance-tracker/finance-tracker

# Criar arquivo de variáveis de ambiente
cat > /opt/finance-tracker/finance-app.env << 'EOF'
STORAGE=postgres
RDS_SECRET_NAME=RDS_SECRET_NAME_PLACEHOLDER
DB_HOST=rds-finance-tracker.cqh40koaapaj.us-east-1.rds.amazonaws.com
DB_PORT=5432
DB_NAME=financetracker
HTTP_ADDR=:8080
EOF

sudo chmod 600 /opt/finance-tracker/finance-app.env

# Criar serviço systemd
cat > /tmp/finance-tracker.service << 'EOF'
[Unit]
Description=Finance Tracker API
After=network.target

[Service]
Type=simple
User=ec2-user
WorkingDirectory=/opt/finance-tracker
EnvironmentFile=/opt/finance-tracker/finance-app.env
ExecStart=/opt/finance-tracker/finance-tracker
Restart=on-failure
RestartSec=5
StandardOutput=append:/opt/finance-tracker/finance-tracker.log
StandardError=append:/opt/finance-tracker/finance-tracker.log

[Install]
WantedBy=multi-user.target
EOF

sudo mv /tmp/finance-tracker.service /etc/systemd/system/finance-tracker.service

# Iniciar serviço
sudo systemctl daemon-reload
sudo systemctl enable finance-tracker
sudo systemctl start finance-tracker

# Aguardar aplicação iniciar
sleep 5

# Health check
if curl -f http://localhost:8080/health; then
    echo "=== Aplicação iniciada com sucesso! ==="
else
    echo "=== ERRO: Aplicação falhou ao iniciar ==="
    exit 1
fi