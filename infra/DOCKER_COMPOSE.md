# 🐳 Guia Docker Compose - Finance Tracker

## Arquivos Disponíveis

### 1. `docker-compose.yml` (Produção)
Sobe toda a stack (API + PostgreSQL) pronta para uso.

### 2. `docker-compose.dev.yml` (Desenvolvimento)
Sobe apenas o PostgreSQL. A API roda localmente com `go run`.

### 3. `.env.example`
Exemplo de variáveis de ambiente (copie para `.env` se necessário).

---

## 🚀 Comandos Rápidos

### Executar Stack Completa

```bash
# Subir aplicação + banco
docker-compose up -d

# Verificar se está funcionando
curl http://localhost:8080/health

# Ver logs em tempo real
docker-compose logs -f

# Ver logs apenas da API
docker-compose logs -f api

# Ver logs apenas do PostgreSQL
docker-compose logs -f postgres

# Parar tudo
docker-compose down

# Parar e remover volumes (limpar dados)
docker-compose down -v

# Reconstruir e subir
docker-compose up -d --build
```

### Modo Desenvolvimento

```bash
# Subir apenas PostgreSQL
docker-compose -f docker-compose.dev.yml up -d

# Executar a aplicação localmente (Windows PowerShell)
$env:STORAGE="postgres"
$env:DATABASE_URL="postgres://financeuser:financepass@localhost:5432/financedb?sslmode=disable"
$env:HTTP_ADDR=":8080"
go run ./cmd/api

# Ou em uma linha
$env:STORAGE="postgres"; $env:DATABASE_URL="postgres://financeuser:financepass@localhost:5432/financedb?sslmode=disable"; $env:HTTP_ADDR=":8080"; go run ./cmd/api

# Parar PostgreSQL
docker-compose -f docker-compose.dev.yml down
```

---

## 🔍 Troubleshooting

### Verificar status dos containers

```bash
docker-compose ps
```

### Acessar o banco de dados

```bash
# Via docker exec
docker exec -it finance-tracker-db psql -U financeuser -d financedb

# Comandos SQL úteis
\dt                          # Listar tabelas
\d transactions              # Descrever tabela transactions
SELECT * FROM transactions;  # Ver todas as transações
\q                          # Sair
```

### Recriar o banco de dados

```bash
# Parar e remover volumes
docker-compose down -v

# Subir novamente (vai recriar o banco)
docker-compose up -d
```

### Verificar logs de erro

```bash
# Últimas 50 linhas de log da API
docker-compose logs --tail=50 api

# Últimas 50 linhas de log do PostgreSQL
docker-compose logs --tail=50 postgres
```

### Porta já em uso

Se a porta 8080 ou 5432 já estiver em uso, edite o `docker-compose.yml`:

```yaml
services:
  api:
    ports:
      - "8081:8080"  # Muda porta externa para 8081
  
  postgres:
    ports:
      - "5433:5432"  # Muda porta externa para 5433
```

---

## 📊 Estrutura do Docker Compose

### Rede
- `finance-network`: Rede bridge para comunicação entre containers

### Volumes
- `postgres_data`: Persiste os dados do PostgreSQL

### Health Check
O PostgreSQL tem health check configurado. A API só inicia após o banco estar pronto.

### Restart Policy
A API tem `restart: unless-stopped`, reinicia automaticamente se cair.

---

## 🔒 Segurança

Para produção, altere as credenciais padrão:

1. Edite o `docker-compose.yml`
2. Altere `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`
3. Atualize a `DATABASE_URL` correspondente
4. **Nunca commite credenciais reais no Git!**

---

## 📦 Estrutura de Volumes

```bash
# Ver volumes criados
docker volume ls

# Inspecionar volume
docker volume inspect finance-tracker_postgres_data

# Backup do banco (exemplo)
docker exec finance-tracker-db pg_dump -U financeuser financedb > backup.sql

# Restaurar backup
docker exec -i finance-tracker-db psql -U financeuser -d financedb < backup.sql
```