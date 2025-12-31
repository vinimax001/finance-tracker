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