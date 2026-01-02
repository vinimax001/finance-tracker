# 🪟 Guia PowerShell para Windows - Finance Tracker

## 🚀 Comandos Rápidos para Windows

### Iniciar Aplicação (Docker Compose)

```powershell
# Subir aplicação completa
docker-compose up -d

# Ver status
docker-compose ps

# Ver logs
docker-compose logs -f

# Parar
docker-compose down
```

### Testar a API

```powershell
# Health check
Invoke-WebRequest -Uri http://localhost:8080/health | Select-Object -Expand Content

# Criar receita
$body = @{
    type = "income"
    category = "salary"
    amount_cents = 500000
    description = "Salário"
} | ConvertTo-Json

Invoke-WebRequest -Method POST `
    -Uri http://localhost:8080/transactions `
    -ContentType "application/json" `
    -Body $body | Select-Object -Expand Content

# Criar despesa
$body = @{
    type = "expense"
    category = "rent"
    amount_cents = 150000
    description = "Aluguel"
} | ConvertTo-Json

Invoke-WebRequest -Method POST `
    -Uri http://localhost:8080/transactions `
    -ContentType "application/json" `
    -Body $body | Select-Object -Expand Content

# Listar transações
Invoke-WebRequest -Uri "http://localhost:8080/transactions?from=2025-10-01&to=2025-10-31" | 
    Select-Object -Expand Content | ConvertFrom-Json | ConvertTo-Json -Depth 10

# Resumo mensal
Invoke-WebRequest -Uri "http://localhost:8080/summary/monthly?year=2025&month=10" | 
    Select-Object -Expand Content | ConvertFrom-Json | ConvertTo-Json

# Deletar transação (substitua {id} pelo UUID real)
Invoke-WebRequest -Method DELETE -Uri "http://localhost:8080/transactions/{id}"
```

### Executar Sem Docker Compose

```powershell
# Modo Memory (sem banco)
cd "e:\Full Cycle\finance-tracker"
$env:STORAGE="memory"
$env:HTTP_ADDR=":8080"
go run ./cmd/api

# Modo PostgreSQL (com Docker Compose dev)
# Terminal 1: Subir banco
docker-compose -f docker-compose.dev.yml up -d

# Terminal 2: Rodar aplicação
cd "e:\Full Cycle\finance-tracker"
$env:STORAGE="postgres"
$env:DATABASE_URL="postgres://financeuser:financepass@localhost:5432/financedb?sslmode=disable"
$env:HTTP_ADDR=":8080"
go run ./cmd/api
```

### Comandos do PostgreSQL

```powershell
# Acessar o banco via Docker
docker exec -it finance-tracker-db psql -U financeuser -d financedb

# Executar query diretamente
docker exec -it finance-tracker-db psql -U financeuser -d financedb -c "SELECT COUNT(*) FROM transactions;"

# Ver todas as transações
docker exec -it finance-tracker-db psql -U financeuser -d financedb -c "SELECT * FROM transactions ORDER BY occurred_at DESC;"

# Backup do banco
docker exec finance-tracker-db pg_dump -U financeuser financedb > backup.sql

# Restaurar backup
Get-Content backup.sql | docker exec -i finance-tracker-db psql -U financeuser -d financedb
```

### Limpeza e Manutenção

```powershell
# Parar e remover tudo (incluindo dados)
docker-compose down -v

# Remover imagens antigas
docker image prune -a

# Ver uso de espaço
docker system df

# Limpeza completa do Docker
docker system prune -a --volumes

# Rebuild completo
docker-compose down -v
docker-compose build --no-cache
docker-compose up -d
```

### Verificar Logs

```powershell
# Logs gerais
docker-compose logs

# Logs da API
docker-compose logs api

# Logs do PostgreSQL
docker-compose logs postgres

# Últimas 50 linhas
docker-compose logs --tail=50

# Seguir logs em tempo real
docker-compose logs -f

# Logs com timestamp
docker-compose logs -t
```

### Troubleshooting

```powershell
# Verificar se portas estão ocupadas
netstat -ano | findstr :8080
netstat -ano | findstr :5432

# Matar processo na porta 8080 (se necessário)
# Primeiro encontrar o PID
$pid = (Get-NetTCPConnection -LocalPort 8080 -ErrorAction SilentlyContinue).OwningProcess
if ($pid) { Stop-Process -Id $pid -Force }

# Verificar containers rodando
docker ps

# Verificar todos os containers (incluindo parados)
docker ps -a

# Inspecionar container
docker inspect finance-tracker-api

# Verificar networks
docker network ls
docker network inspect finance-tracker_finance-network

# Verificar volumes
docker volume ls
docker volume inspect finance-tracker_postgres_data

# Entrar no container da API
docker exec -it finance-tracker-api sh

# Ver variáveis de ambiente do container
docker exec finance-tracker-api env
```

### Script de Teste Completo

```powershell
# Salve como test-api.ps1

# Cores para output
function Write-Success { Write-Host $args -ForegroundColor Green }
function Write-Info { Write-Host $args -ForegroundColor Cyan }
function Write-Error { Write-Host $args -ForegroundColor Red }

Write-Info "=== Testando Finance Tracker API ==="

# 1. Health Check
Write-Info "`n1. Health Check..."
try {
    $health = Invoke-WebRequest -Uri http://localhost:8080/health | Select-Object -Expand Content
    Write-Success "✓ Health: $health"
} catch {
    Write-Error "✗ Health check falhou: $_"
    exit 1
}

# 2. Criar receita
Write-Info "`n2. Criando receita..."
$income = @{
    type = "income"
    category = "salary"
    amount_cents = 500000
    description = "Salário de teste"
} | ConvertTo-Json

try {
    $result = Invoke-WebRequest -Method POST `
        -Uri http://localhost:8080/transactions `
        -ContentType "application/json" `
        -Body $income | Select-Object -Expand Content | ConvertFrom-Json
    Write-Success "✓ Receita criada: ID = $($result.id)"
    $incomeId = $result.id
} catch {
    Write-Error "✗ Erro ao criar receita: $_"
}

# 3. Criar despesa
Write-Info "`n3. Criando despesa..."
$expense = @{
    type = "expense"
    category = "rent"
    amount_cents = 150000
    description = "Aluguel de teste"
} | ConvertTo-Json

try {
    $result = Invoke-WebRequest -Method POST `
        -Uri http://localhost:8080/transactions `
        -ContentType "application/json" `
        -Body $expense | Select-Object -Expand Content | ConvertFrom-Json
    Write-Success "✓ Despesa criada: ID = $($result.id)"
    $expenseId = $result.id
} catch {
    Write-Error "✗ Erro ao criar despesa: $_"
}

# 4. Listar transações
Write-Info "`n4. Listando transações..."
$from = (Get-Date).AddDays(-1).ToString("yyyy-MM-dd")
$to = (Get-Date).AddDays(1).ToString("yyyy-MM-dd")
try {
    $transactions = Invoke-WebRequest -Uri "http://localhost:8080/transactions?from=$from&to=$to" |
        Select-Object -Expand Content | ConvertFrom-Json
    Write-Success "✓ Encontradas $($transactions.Count) transações"
} catch {
    Write-Error "✗ Erro ao listar transações: $_"
}

# 5. Resumo mensal
Write-Info "`n5. Resumo mensal..."
$year = (Get-Date).Year
$month = (Get-Date).Month
try {
    $summary = Invoke-WebRequest -Uri "http://localhost:8080/summary/monthly?year=$year&month=$month" |
        Select-Object -Expand Content | ConvertFrom-Json
    Write-Success "✓ Resumo:"
    Write-Success "  - Receitas: R$ $($summary.income_cents / 100)"
    Write-Success "  - Despesas: R$ $($summary.expense_cents / 100)"
    Write-Success "  - Saldo: R$ $($summary.net_cents / 100)"
    Write-Success "  - Total transações: $($summary.count_transactions)"
} catch {
    Write-Error "✗ Erro ao obter resumo: $_"
}

# 6. Deletar transação
if ($incomeId) {
    Write-Info "`n6. Deletando receita..."
    try {
        Invoke-WebRequest -Method DELETE -Uri "http://localhost:8080/transactions/$incomeId"
        Write-Success "✓ Receita deletada com sucesso"
    } catch {
        Write-Error "✗ Erro ao deletar receita: $_"
    }
}

Write-Success "`n=== Testes concluídos! ==="
```

Para executar o script:
```powershell
# Salvar o script
Set-Content -Path test-api.ps1 -Value (Get-Clipboard)

# Executar
.\test-api.ps1
```

---

## 🎯 Comandos Make (via PowerShell)

Se tiver `make` instalado no Windows:

```powershell
# Desenvolvimento
make test           # Rodar testes
make fmt            # Formatar código
make build          # Compilar

# Docker Compose
make compose-up     # Subir stack
make compose-down   # Parar stack
make compose-logs   # Ver logs
```

---

**Dica:** Adicione estas funções no seu `$PROFILE` do PowerShell para facilitar:

```powershell
# Ver/editar profile
notepad $PROFILE

# Adicionar estas funções:
function ft-up { docker-compose -f "e:\Full Cycle\finance-tracker\docker-compose.yml" up -d }
function ft-down { docker-compose -f "e:\Full Cycle\finance-tracker\docker-compose.yml" down }
function ft-logs { docker-compose -f "e:\Full Cycle\finance-tracker\docker-compose.yml" logs -f }
function ft-test { Invoke-WebRequest http://localhost:8080/health | Select-Object -Expand Content }
```

Depois recarregue o profile:
```powershell
. $PROFILE
```

Agora você pode usar:
```powershell
ft-up      # Subir aplicação
ft-test    # Testar
ft-logs    # Ver logs
ft-down    # Parar
```