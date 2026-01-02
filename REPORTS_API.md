# API de Relatórios Mensais

## Endpoint: `GET /reports/monthly`

Gera um relatório financeiro mensal com cálculo de receitas, despesas e saldo final.

### Funcionalidades

1. **Retorna JSON** com resumo financeiro e relatório formatado
2. **Gera arquivo .txt** e faz upload automático para S3
3. **Cálculo automático** de saldo (receitas - despesas)

### Parâmetros Query String

| Parâmetro | Tipo | Obrigatório | Descrição |
|-----------|------|-------------|-----------|
| `year` | int | Sim | Ano do relatório (ex: 2025) |
| `month` | int | Sim | Mês do relatório (1-12) |

### Exemplo de Requisição

```bash
curl "http://localhost:8080/reports/monthly?year=2025&month=11"
```

### Exemplo de Resposta JSON

```json
{
  "year": 2025,
  "month": 11,
  "income_cents": 500000,
  "expense_cents": 320000,
  "net_cents": 180000,
  "count_transactions": 15,
  "first_tx": "2025-11-01T10:30:00Z",
  "last_tx": "2025-11-28T18:45:00Z",
  "report_text": "========================================\nRELATÓRIO FINANCEIRO - November/2025\n========================================\n\nPeríodo: November de 2025\nTotal de Transações: 15\n\nRESUMO FINANCEIRO:\n------------------------------------------\nReceitas:       R$ 5000.00\nDespesas:       R$ 3200.00\n------------------------------------------\nSaldo Final:    R$ 1800.00\n------------------------------------------\n\nPrimeira Transação: 2025-11-01T10:30:00Z\nÚltima Transação:   2025-11-28T18:45:00Z\n\nStatus: POSITIVO ✓\n========================================\n",
  "s3_file": "s3://finance-tracker-releases/reports/report-2025-11.txt"
}
```

### Estrutura do Relatório em Texto

O relatório `.txt` gerado no S3 contém:

```
========================================
RELATÓRIO FINANCEIRO - November/2025
========================================

Período: November de 2025
Total de Transações: 15

RESUMO FINANCEIRO:
------------------------------------------
Receitas:       R$ 5000.00
Despesas:       R$ 3200.00
------------------------------------------
Saldo Final:    R$ 1800.00
------------------------------------------

Primeira Transação: 2025-11-01T10:30:00Z
Última Transação:   2025-11-28T18:45:00Z

Status: POSITIVO ✓
========================================
```

### Arquivo S3

- **Bucket:** `finance-tracker-releases`
- **Caminho:** `reports/report-YYYY-MM.txt`
- **Exemplo:** `s3://finance-tracker-releases/reports/report-2025-11.txt`
- **Content-Type:** `text/plain; charset=utf-8`

### Upload Assíncrono

O upload para S3 é feito **em background** (goroutine) para não bloquear a resposta HTTP. Se houver erro no upload:
- A resposta HTTP ainda é retornada com sucesso
- Erro é logado no console da aplicação
- Não afeta a experiência do usuário

### Permissões IAM Necessárias

A IAM Role da EC2 (`finance-tracker-ec2-role-v2`) já possui permissões para:
- `s3:PutObject` no bucket `finance-tracker-releases`
- `s3:GetObject`
- `s3:ListBucket`
- `s3:DeleteObject`

### Status do Saldo

O relatório indica o status financeiro:
- **POSITIVO ✓**: Receitas > Despesas
- **NEGATIVO ✗**: Receitas < Despesas
- **NEUTRO**: Receitas = Despesas

### Códigos de Resposta HTTP

| Código | Descrição |
|--------|-----------|
| 200 | Relatório gerado com sucesso |
| 400 | Parâmetros inválidos (ano/mês ausente ou mês fora do range 1-12) |
| 500 | Erro interno do servidor (problema ao acessar banco de dados) |

### Exemplo de Uso Completo

```bash
# 1. Criar algumas transações
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"type":"income","category":"salary","amount_cents":500000,"description":"Salário Novembro"}'

curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"type":"expense","category":"rent","amount_cents":150000,"description":"Aluguel"}'

curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"type":"expense","category":"food","amount_cents":80000,"description":"Supermercado"}'

# 2. Gerar relatório do mês
curl "http://localhost:8080/reports/monthly?year=2025&month=11" | jq

# 3. Verificar arquivo no S3 (via AWS CLI)
aws s3 cp s3://finance-tracker-releases/reports/report-2025-11.txt - | cat
```

### Diferença entre `/summary/monthly` e `/reports/monthly`

| Endpoint | Retorno | Arquivo S3 | Formatação |
|----------|---------|------------|------------|
| `/summary/monthly` | JSON resumido | ❌ Não | Apenas dados estruturados |
| `/reports/monthly` | JSON + texto formatado | ✅ Sim | Relatório completo em texto |

### Notas Técnicas

1. **AWS SDK V2**: Usa `github.com/aws/aws-sdk-go-v2` para integração com S3
2. **IAM Role**: Autenticação via IAM Instance Profile (sem credenciais hardcoded)
3. **Thread-safe**: Upload em goroutine separada não bloqueia outras requisições
4. **Encoding**: Arquivo salvo com UTF-8 para suportar caracteres especiais
5. **Idempotência**: Chamar múltiplas vezes sobrescreve o arquivo do mesmo mês