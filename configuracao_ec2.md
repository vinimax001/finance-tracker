# Compile para Linux (a EC2 usa Linux)
GOOS=linux GOARCH=amd64 go build -o finance-tracker ./cmd/api

# Enviar o binário compilado
scp -i sua-chave.pem finance-tracker ec2-user@SEU-IP-EC2:/home/ec2-user/

# Enviar a migration (para criar as tabelas)
scp -i sua-chave.pem migrations/001_init.sql ec2-user@SEU-IP-EC2:/home/ec2-user/

# Atualizar sistema
sudo yum update -y

# Baixar Go 1.25.3
wget https://go.dev/dl/go1.25.3.linux-amd64.tar.gz

# Remover instalação anterior (se existir)
sudo rm -rf /usr/local/go

# Extrair para /usr/local
sudo tar -C /usr/local -xzf go1.25.3.linux-amd64.tar.gz

# Adicionar ao PATH (CAMINHO CORRETO!)
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verificar instalação
go version
# Saída esperada: go version go1.25.3 linux/amd64 (ou linux/arm64)

# Ver onde está o executável
which go
# Saída esperada: /usr/local/go/bin/go

# Limpar arquivo de instalação
rm go1.25.3.linux-amd64.tar.gz

# Dar permissão de execução (se ainda não deu)
chmod +x finance-tracker

# ABRIR PORTA DO SECURITY GROUP PARA VER APLICAÇÃO EM EXECUÇÃO
# Executar em modo memória
STORAGE=memory HTTP_ADDR=:8080 ./finance-tracker

# Você verá a execução dessa forma:
storage=memory
listening on :8080

# ============================================
# CONFIGURAÇÃO PARA PRODUÇÃO COM RDS
# ============================================

# 1. Fazer URL encode da senha (rodar na EC2)
python3 << 'EOF'
import urllib.parse
senha = "SUA_SENHA_ORIGINAL"
senha_encoded = urllib.parse.quote(senha, safe='')
print("\n=== SENHA ENCODED ===")
print(senha_encoded)
print("\n=== COPIE A LINHA ABAIXO ===")
print(f'export DATABASE_URL="postgres://postgres:{senha_encoded}@rds-finance-tracker.cqh40koaapaj.us-east-1.rds.amazonaws.com:5432/financetracker?sslmode=require"')
EOF

# 2. Criar arquivo de variáveis de ambiente
nano finance-app.env

# 3. Colar o resultado do script acima:
export STORAGE=postgres
export DATABASE_URL="postgres://postgres:SENHA_ENCODED@rds-finance-tracker.cqh40koaapaj.us-east-1.rds.amazonaws.com:5432/financetracker?sslmode=require"
export HTTP_ADDR=:8080

# Proteger o arquivo .env
chmod 600 finance-app.env

# 2. Configurar Security Groups na AWS
# RDS Security Group: Adicionar regra Inbound
#   - Type: PostgreSQL
#   - Port: 5432
#   - Source: Security Group da EC2

# EC2 Security Group: Adicionar regra Inbound
#   - Type: Custom TCP
#   - Port: 8080
#   - Source: 0.0.0.0/0

# 3. Instalar PostgreSQL Client
sudo yum install -y postgresql17
psql --version

# 4. Criar variável com senha (para comandos psql)
export PGPASSWORD='SUA_SENHA'

# 5. Testar conexão com RDS
psql -h rds-finance-tracker.cqh40koaapaj.us-east-1.rds.amazonaws.com \
     -p 5432 -U postgres -d financetracker
# Digite \q para sair

# 6. Aplicar migration (criar tabelas)
psql -h rds-finance-tracker.cqh40koaapaj.us-east-1.rds.amazonaws.com \
     -p 5432 -U postgres -d financetracker \
     -f 001_init.sql
# Saída esperada: CREATE TABLE, CREATE INDEX (3x)

# 7. Verificar tabelas criadas
psql -h rds-finance-tracker.cqh40koaapaj.us-east-1.rds.amazonaws.com \
     -p 5432 -U postgres -d financetracker
\dt
\d transactions
\q

# 8. Executar aplicação com RDS
source ~/finance-app.env
./finance-tracker
# Saída esperada: storage=postgres connected

# 9. Testar (em outro terminal)
curl http://localhost:8080/health
curl -X POST http://localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"type":"income","category":"salary","amount_cents":500000}'

# 10. Verificar dados no RDS
psql -h rds-finance-tracker.cqh40koaapaj.us-east-1.rds.amazonaws.com \
     -p 5432 -U postgres -d financetracker
SELECT * FROM transactions;
\q

# ============================================
# CONFIGURAR SERVIÇO SYSTEMD (rodar automaticamente)
# ============================================

# 1. Criar arquivo de serviço
sudo nano /etc/systemd/system/finance-tracker.service

# Cole o conteúdo:
[Unit]
Description=Finance Tracker API
After=network.target

[Service]
Type=simple
User=ec2-user
WorkingDirectory=/home/ec2-user/financetracker
EnvironmentFile=/home/ec2-user/financetracker/finance-app.env
ExecStart=/home/ec2-user/financetracker/finance-tracker
Restart=on-failure
RestartSec=5
StandardOutput=append:/home/ec2-user/financetracker/finance-tracker.log
StandardError=append:/home/ec2-user/financetracker/finance-tracker.log

[Install]
WantedBy=multi-user.target

# 2. Ativar serviço
sudo systemctl daemon-reload
sudo systemctl start finance-tracker
sudo systemctl status finance-tracker
sudo systemctl enable finance-tracker

# 3. Comandos úteis
sudo systemctl stop finance-tracker      # Parar
sudo systemctl restart finance-tracker   # Reiniciar
sudo journalctl -u finance-tracker -f    # Ver logs em tempo real
tail -f ~/finance-tracker.log            # Ver logs do arquivo

# ============================================
# ENDPOINTS DA API
# ============================================

# Health check
curl http://SEU-IP-EC2:8080/health

# Criar receita
curl -X POST http://SEU-IP-EC2:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"type":"income","category":"salary","amount_cents":500000,"description":"Salário"}'

# Criar despesa
curl -X POST http://SEU-IP-EC2:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"type":"expense","category":"rent","amount_cents":150000,"description":"Aluguel"}'

# Listar transações
curl "http://SEU-IP-EC2:8080/transactions?from=2025-10-01&to=2025-10-31"

# Resumo mensal
curl "http://SEU-IP-EC2:8080/summary/monthly?year=2025&month=10"

# Deletar transação
curl -X DELETE http://SEU-IP-EC2:8080/transactions/ID-DA-TRANSACAO