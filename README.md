# Finance Tracker API (Go)

API REST para registrar transações financeiras (entradas/saídas), listar por período e gerar resumo mensal.

## 🚀 Como Executar a Aplicação

### Pré-requisitos
- Go 1.22 ou superior instalado ([Download](https://go.dev/dl/))
- Docker (opcional, apenas se quiser usar PostgreSQL)

### Passo 1: Instalar Dependências

```bash
# Clone ou navegue até o diretório do projeto
cd finance-tracker

# Baixe as dependências
go mod download

# Ou organize as dependências automaticamente
go mod tidy
```

### Passo 2: Executar os Testes

```bash
# Rodar todos os testes
go test ./... -v

# Rodar testes sem verbose
go test ./...
```

### Passo 3: Executar a Aplicação

#### Opção A: Modo Memory (Desenvolvimento - Sem Banco de Dados)

**Linux/Mac:**
```bash
STORAGE=memory HTTP_ADDR=:8080 go run ./cmd/api
```

**Windows (PowerShell):**
```powershell
$env:STORAGE="memory"; $env:HTTP_ADDR=":8080"; go run ./cmd/api
```

**Ou usando Makefile:**
```bash
make run
```

#### Opção B: Modo PostgreSQL (Produção)

**1. Subir o PostgreSQL com Docker:**
```bash
docker run --name finance-pg \
  -e POSTGRES_USER=user \
  -e POSTGRES_PASSWORD=pass \
  -e POSTGRES_DB=financedb \
  -p 5432:5432 \
  -d postgres:15
```

**2. Executar as migrations:**
```bash
# Linux/Mac
docker exec -i finance-pg psql -U user -d financedb < migrations/001_init.sql

# Windows (PowerShell)
Get-Content migrations/001_init.sql | docker exec -i finance-pg psql -U user -d financedb
```

**3. Executar a aplicação:**

**Linux/Mac:**
```bash
STORAGE=postgres \
DATABASE_URL="postgres://user:pass@localhost:5432/financedb?sslmode=disable" \
HTTP_ADDR=:8080 \
go run ./cmd/api
```

**Windows (PowerShell):**
```powershell
$env:STORAGE="postgres"; $env:DATABASE_URL="postgres://user:pass@localhost:5432/financedb?sslmode=disable"; $env:HTTP_ADDR=":8080"; go run ./cmd/api
```

**Ou usando Makefile:**
```bash
make run-pg
```

### Passo 4: Verificar se está funcionando

```bash
# Health check
curl http://localhost:8080/health

# Deve retornar: {"status":"ok"}
```

### 🐳 Executar com Docker

```bash
# Build da imagem
docker build -t finance-tracker:latest .
# ou
make docker-build

# Executar o container
docker run --rm -p 8080:8080 -e STORAGE=memory finance-tracker:latest
# ou
make docker-run
```

### 📊 Variáveis de Ambiente

| Variável | Descrição | Padrão | Obrigatório |
|----------|-----------|--------|-------------|
| `STORAGE` | Tipo de armazenamento: `memory` ou `postgres` | `memory` | Não |
| `HTTP_ADDR` | Endereço do servidor HTTP | `:8080` | Não |
| `DATABASE_URL` | Connection string do PostgreSQL | - | Sim (se `STORAGE=postgres`) |

### 🛠️ Comandos Úteis (Makefile)

```bash
make fmt        # Formatar código
make vet        # Análise estática
make tidy       # Organizar dependências
make test       # Rodar testes
make build      # Compilar binário
make run        # Executar em modo memory
make run-pg     # Executar com PostgreSQL
```

## Endpoints
- `GET /health`
- `POST /transactions`
- `GET /transactions?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `DELETE /transactions/{id}`
- `GET /summary/monthly?year=YYYY&month=MM`

### Exemplo de uso (curl)
```bash
# Health
curl -s localhost:8080/health

# Criar transação (income)
curl -s -X POST localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"type":"income","category":"salary","amount_cents":500000,"description":"Salário de outubro"}'

# Criar transação (expense)
curl -s -X POST localhost:8080/transactions \
  -H 'Content-Type: application/json' \
  -d '{"type":"expense","category":"rent","amount_cents":150000,"description":"Aluguel"}'

# Listar período
curl -s "localhost:8080/transactions?from=2025-10-01&to=2025-10-31"

# Resumo do mês
curl -s "localhost:8080/summary/monthly?year=2025&month=10"
```

---

## 📝 Testando com Postman

### 1. Health Check
- **Método:** GET
- **URL:** `http://localhost:8080/health`

### 2. Criar Receita (Income)
- **Método:** POST
- **URL:** `http://localhost:8080/transactions`
- **Headers:** `Content-Type: application/json`
- **Body:**
```json
{
  "type": "income",
  "category": "salary",
  "amount_cents": 500000,
  "description": "Salário de Outubro"
}
```

### 3. Criar Despesa (Expense)
- **Método:** POST
- **URL:** `http://localhost:8080/transactions`
- **Headers:** `Content-Type: application/json`
- **Body:**
```json
{
  "type": "expense",
  "category": "rent",
  "amount_cents": 150000,
  "description": "Aluguel"
}
```

### 4. Listar Transações
- **Método:** GET
- **URL:** `http://localhost:8080/transactions?from=2025-10-01&to=2025-10-31`

### 5. Resumo Mensal
- **Método:** GET
- **URL:** `http://localhost:8080/summary/monthly?year=2025&month=10`

### 6. Deletar Transação
- **Método:** DELETE
- **URL:** `http://localhost:8080/transactions/{id}`
- *Substitua `{id}` pelo UUID da transação*

---

## 💡 Observações Importantes

- **Valores monetários:** Sempre em centavos (ex: `500000` = R$ 5.000,00)
- **Data da transação:** Automaticamente definida como data/hora atual no servidor
- **Storage Memory:** Dados são perdidos ao reiniciar a aplicação
- **Storage PostgreSQL:** Dados são persistidos no banco de dados

---

## 📦 Estrutura do Projeto

```
finance-tracker/
├── cmd/
│   └── api/
│       └── main.go           # Entry point da aplicação
├── internal/
│   ├── finance/
│   │   ├── model.go          # Modelos de domínio
│   │   ├── service.go        # Lógica de negócio
│   │   ├── repo_memory.go    # Repositório em memória
│   │   ├── repo_postgres.go  # Repositório PostgreSQL
│   │   └── repo_memory_test.go
│   └── http/
│       ├── handlers.go       # HTTP handlers
│       └── handlers_test.go
├── migrations/
│   └── 001_init.sql          # Schema do banco de dados
├── Dockerfile
├── Makefile
├── go.mod
└── README.md
```