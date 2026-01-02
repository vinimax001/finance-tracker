# Como Testar o Endpoint de Relatórios Localmente

## Pré-requisitos

1. **PostgreSQL** rodando (ou use `STORAGE=memory` para teste rápido)
2. **AWS Credentials** configuradas (se quiser testar upload S3)
   - Ou rode com IAM role na EC2
   - Ou configure `~/.aws/credentials` localmente

## Opção 1: Teste Local SEM S3 (Apenas Relatório)

```bash
# 1. Rodar aplicação em modo memory (sem banco)
export STORAGE=memory
export HTTP_ADDR=:8080
go run ./cmd/api/main.go
```

Em outro terminal:

```bash
# 2. Criar transações de teste
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"type":"income","category":"salary","amount_cents":500000,"description":"Salário"}'

curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"type":"expense","category":"rent","amount_cents":180000,"description":"Aluguel"}'

# 3. Gerar relatório
curl "http://localhost:8080/reports/monthly?year=2025&month=11" | jq

# Você verá o report_text formatado e uma mensagem de erro no console
# (erro ao fazer upload S3, pois não tem credenciais configuradas - é esperado!)
```

## Opção 2: Teste Local COM S3

```bash
# 1. Configure AWS credentials localmente
aws configure
# Insira suas credenciais AWS

# 2. Verifique se tem acesso ao bucket
aws s3 ls s3://finance-tracker-releases/

# 3. Rode a aplicação
export STORAGE=memory
export HTTP_ADDR=:8080
go run ./cmd/api/main.go
```

Em outro terminal:

```bash
# 4. Gere relatório
curl "http://localhost:8080/reports/monthly?year=2025&month=11" | jq

# 5. Verifique o arquivo no S3
aws s3 ls s3://finance-tracker-releases/reports/

# 6. Baixe e visualize o relatório
aws s3 cp s3://finance-tracker-releases/reports/report-2025-11.txt - | cat
```

## Opção 3: Teste Automatizado com Script

```bash
# Execute o script de teste (requer jq instalado)
bash test_reports_endpoint.sh
```

## Opção 4: Teste em Produção (EC2 com ASG)

Após fazer deploy via GitHub Actions:

```bash
# 1. Obter DNS do ALB
cd terraform
ALB_DNS=$(terraform output -raw alb_dns_name)

# 2. Criar transações
curl -X POST "http://$ALB_DNS/transactions" \
  -H "Content-Type: application/json" \
  -d '{"type":"income","category":"salary","amount_cents":500000}'

# 3. Gerar relatório
curl "http://$ALB_DNS/reports/monthly?year=2025&month=11" | jq

# 4. Verificar arquivo no S3
aws s3 ls s3://finance-tracker-releases/reports/
aws s3 cp s3://finance-tracker-releases/reports/report-2025-11.txt - | cat
```

## Exemplo de Resposta Esperada

```json
{
  "year": 2025,
  "month": 11,
  "income_cents": 500000,
  "expense_cents": 180000,
  "net_cents": 320000,
  "count_transactions": 2,
  "first_tx": "2025-11-17T15:30:00Z",
  "last_tx": "2025-11-17T15:31:00Z",
  "report_text": "========================================\nRELATÓRIO FINANCEIRO - November/2025\n========================================\n\nPeríodo: November de 2025\nTotal de Transações: 2\n\nRESUMO FINANCEIRO:\n------------------------------------------\nReceitas:       R$ 5000.00\nDespesas:       R$ 1800.00\n------------------------------------------\nSaldo Final:    R$ 3200.00\n------------------------------------------\n\nPrimeira Transação: 2025-11-17T15:30:00Z\nÚltima Transação:   2025-11-17T15:31:00Z\n\nStatus: POSITIVO ✓\n========================================\n",
  "s3_file": "s3://finance-tracker-releases/reports/report-2025-11.txt"
}
```

## Verificando Logs da Aplicação

**Sucesso no upload S3:**
```
Report uploaded to S3: s3://finance-tracker-releases/reports/report-2025-11.txt
```

**Erro no upload S3 (esperado localmente sem credenciais):**
```
Error uploading report to S3: operation error S3: PutObject, failed to sign request: failed to retrieve credentials: ...
```

## Troubleshooting

### Erro: "No credentials found"
**Solução:** Configure AWS credentials ou rode na EC2 com IAM role

### Erro: "Access Denied" no S3
**Solução:** Verifique se a IAM role/user tem permissão `s3:PutObject` no bucket

### Relatório vazio (count_transactions = 0)
**Solução:** Crie transações no mês/ano especificado antes de gerar o relatório

### Erro: "month out of range"
**Solução:** Mês deve estar entre 1 e 12