# 📊 Feature de Relatórios Mensais - Explicação Completa

## 🎯 Objetivo da Feature

Criar um endpoint que gera relatórios financeiros mensais formatados em texto, calcula saldo (receitas - despesas), e faz upload automático para o Amazon S3.

---

## 🏗️ Arquitetura da Solução

### Fluxo de Funcionamento

```
Cliente HTTP
    ↓
GET /reports/monthly?year=2025&month=11
    ↓
Handler (handlers.go)
    ↓
Service.GenerateMonthlyReport() → Gera texto formatado
    ↓
Service.MonthlySummary() → Busca dados do banco
    ↓
Upload S3 (goroutine assíncrona) ← Contexto independente
    ↓
Resposta JSON (summary + report_text + s3_file)
```

---

## 📁 Arquivos Criados/Modificados

### 1. **`internal/finance/service.go`** - Lógica de Negócio

#### Função Adicionada: `GenerateMonthlyReport()`

```go
func (s *Service) GenerateMonthlyReport(ctx context.Context, year int, month int) (string, error)
```

**O que faz:**
- Chama `MonthlySummary()` para buscar dados do banco
- Converte valores de centavos para reais (divide por 100)
- Formata relatório em texto com:
  - **Nome do mês em português** (usando map de meses)
  - Resumo financeiro (receitas, despesas, saldo)
  - Data da primeira e última transação
  - Status do saldo (POSITIVO ✓, NEGATIVO ✗, NEUTRO)

**Exemplo de saída:**
```
========================================
RELATÓRIO FINANCEIRO - Novembro/2025
========================================

Período: Novembro de 2025
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

**Conceitos Importantes:**
- **Formatação de strings com `fmt.Sprintf`**: Permite criar strings complexas com substituições
- **Map de meses em português**: Mapeia números (1-12) para nomes dos meses
- **Conversão centavos → reais**: `float64(summary.Income) / 100.0`

---

### 2. **`internal/finance/s3.go`** - Integração com AWS S3

#### Arquivo NOVO criado

```go
func UploadReportToS3(ctx context.Context, bucketName, fileName, content string) error
```

**O que faz:**
1. **Carrega configuração AWS** via `config.LoadDefaultConfig()`
   - Usa IAM Instance Profile da EC2 (sem credenciais hardcoded)
   - Define região (usa `AWS_REGION` ou fallback para `us-east-1`)

2. **Cria cliente S3** com `s3.NewFromConfig(cfg)`

3. **Faz upload** com `client.PutObject()`:
   - Bucket: `finance-tracker-releases`
   - Key: `reports/report-YYYY-MM.txt`
   - Body: Conteúdo do relatório em bytes
   - ContentType: `text/plain; charset=utf-8`

**Dependências Necessárias:**
```bash
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/s3
```

**Conceitos Importantes:**
- **AWS SDK V2**: Nova versão modular do SDK (cada serviço é um pacote separado)
- **IAM Instance Profile**: EC2 assume uma role com permissões, sem precisar de access keys
- **Context**: Permite cancelar operações longas ou definir timeouts
- **bytes.NewReader**: Converte string em `io.Reader` para upload

---

### 3. **`internal/http/handlers.go`** - Handler HTTP

#### Função Adicionada: `monthlyReport()`

```go
func monthlyReport(svc *finance.Service) http.HandlerFunc
```

**Fluxo do Handler:**

1. **Valida Query Params** (`year` e `month`)
   ```go
   yearStr := r.URL.Query().Get("year")
   monthStr := r.URL.Query().Get("month")
   ```

2. **Gera Relatório Textual**
   ```go
   reportText, err := svc.GenerateMonthlyReport(r.Context(), y, m)
   ```

3. **Constrói Nome do Arquivo S3**
   ```go
   fileName := fmt.Sprintf("reports/report-%04d-%02d.txt", y, m)
   ```

4. **Upload Assíncrono para S3** (goroutine)
   ```go
   go func() {
       ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
       defer cancel()
       
       if err := finance.UploadReportToS3(ctx, s3BucketName, fileName, reportText); err != nil {
           fmt.Printf("Error uploading report to S3: %v\n", err)
       }
   }()
   ```

5. **Retorna Resposta JSON Imediata**
   ```go
   ok(w, reportResp{
       MonthlySummary: sum,
       ReportText:     reportText,
       S3File:         fmt.Sprintf("s3://%s/%s", s3BucketName, fileName),
   })
   ```

**⚠️ PROBLEMA RESOLVIDO: Context Cancelado**

**ERRO ORIGINAL:**
```go
// ❌ ERRADO - usa r.Context() que é cancelado quando resposta HTTP é enviada
go func() {
    if err := finance.UploadReportToS3(r.Context(), ...) {
        // ERROR: request canceled, context canceled
    }
}()
```

**SOLUÇÃO:**
```go
// ✅ CORRETO - cria contexto independente com timeout
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := finance.UploadReportToS3(ctx, ...) {
        // Funciona! Contexto não é cancelado
    }
}()
```

**Por que isso acontece?**
- `r.Context()` está ligado à requisição HTTP
- Quando a resposta é enviada, Go cancela automaticamente o contexto da requisição
- A goroutine ainda está rodando, mas o contexto já foi cancelado
- **Solução**: Usar `context.Background()` cria um contexto independente

**Conceitos Importantes:**
- **Goroutines**: Executam código de forma assíncrona (não bloqueia resposta HTTP)
- **Context**: Controla tempo de vida de operações (timeout, cancelamento)
- **defer cancel()**: Garante que recursos são liberados ao finalizar a função
- **Background vs Request Context**: Background é independente, Request está ligado à requisição HTTP

---

### 4. **`terraform/main.tf`** - Permissões IAM

#### Policy S3 Adicionada

```hcl
resource "aws_iam_role_policy" "s3_access" {
  name = "finance-tracker-s3-access-v2"
  role = aws_iam_role.app.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket",
          "s3:DeleteObject"
        ]
        Resource = [
          "arn:aws:s3:::finance-tracker-releases",
          "arn:aws:s3:::finance-tracker-releases/*"
        ]
      }
    ]
  })
}
```

**O que isso faz:**
- Adiciona policy inline à IAM Role da EC2
- Permite que a aplicação faça upload/download/listagem no bucket S3
- Escopo limitado ao bucket `finance-tracker-releases`

**Conceitos Importantes:**
- **IAM Role**: Conjunto de permissões que uma entidade pode assumir
- **IAM Policy**: Documento JSON que define permissões (allow/deny)
- **Instance Profile**: Vincula IAM Role à EC2
- **ARN**: Amazon Resource Name (identificador único de recursos AWS)

---

### 5. **Registro da Rota** - `NewMux()`

```go
m.HandleFunc("GET /reports/monthly", monthlyReport(svc))
```

**Padrão HTTP Method Routing (Go 1.22+):**
- `"GET /reports/monthly"` define método HTTP + path
- Antes era preciso validar `r.Method == "GET"` manualmente
- Go 1.22+ suporta pattern matching nativo

---

## 🔍 Conceitos Técnicos Aprendidos

### 1. **AWS SDK V2 para Go**

**Diferença entre V1 e V2:**
| V1 | V2 |
|----|-----|
| Monolítico (um pacote gigante) | Modular (pacotes separados por serviço) |
| `github.com/aws/aws-sdk-go` | `github.com/aws/aws-sdk-go-v2` |
| Configuração global | Configuração por contexto |

**Autenticação via IAM Role:**
```go
cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
```
- Busca credenciais automaticamente:
  1. Variáveis de ambiente (`AWS_ACCESS_KEY_ID`)
  2. Arquivo de credenciais (`~/.aws/credentials`)
  3. **IAM Instance Profile** ← Usado em produção na EC2
  4. ECS Task Role (em containers)

### 2. **Goroutines e Concorrência**

**Goroutine:**
```go
go func() {
    // Código roda em paralelo
}()
```

**Quando usar:**
- Operações que podem rodar em background (upload S3, envio de email)
- Não bloqueia a resposta HTTP
- Melhora experiência do usuário

**⚠️ Cuidados:**
- Goroutines não retornam valores diretamente (use channels)
- Erros devem ser logados, não retornados
- Usar `defer` para garantir limpeza de recursos

### 3. **Context Pattern**

**Tipos de Context:**
```go
// 1. Context da requisição HTTP (cancelado ao finalizar request)
ctx := r.Context()

// 2. Context independente (não é cancelado)
ctx := context.Background()

// 3. Context com timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// 4. Context com deadline
ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(1*time.Hour))
defer cancel()
```

**Quando usar cada um:**
- **`r.Context()`**: Operações síncronas que devem ser canceladas com a requisição
- **`context.Background()`**: Goroutines, operações assíncronas, background jobs
- **`WithTimeout`**: Operações com tempo limite (chamadas a APIs externas)

### 4. **Formatação de Strings**

**`fmt.Sprintf` vs Concatenação:**
```go
// ❌ Difícil de ler e manter
name := "João"
age := 25
msg := "Nome: " + name + ", Idade: " + strconv.Itoa(age)

// ✅ Limpo e legível
msg := fmt.Sprintf("Nome: %s, Idade: %d", name, age)
```

**Verbos de formatação:**
- `%s`: string
- `%d`: inteiro decimal
- `%f`: float (%.2f = 2 casas decimais)
- `%v`: valor padrão (qualquer tipo)
- `%04d`: inteiro com padding de zeros (ex: 0001, 0042)

### 5. **Tratamento de Erros em Go**

**Pattern usado na aplicação:**
```go
reportText, err := svc.GenerateMonthlyReport(r.Context(), y, m)
if err != nil {
    status := http.StatusInternalServerError
    if err == finance.ErrBadRequest {
        status = http.StatusBadRequest
    }
    serr(w, err, status)
    return
}
```

**Conceitos:**
- **Erro explícito**: Go não usa try/catch, retorna error como segundo valor
- **Comparação de erros**: `err == finance.ErrBadRequest`
- **Wrap de erros**: `fmt.Errorf("failed to X: %w", err)` mantém erro original
- **Status HTTP apropriado**: 400 (bad request) vs 500 (internal error)

---

## 🧪 Testando a Feature

### 1. **Teste Local (Memória)**

```bash
# Rodar aplicação em modo memory
STORAGE=memory HTTP_ADDR=:8080 go run ./cmd/api

# Criar transações
curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"type":"income","category":"salary","amount_cents":500000}'

curl -X POST http://localhost:8080/transactions \
  -H "Content-Type: application/json" \
  -d '{"type":"expense","category":"rent","amount_cents":150000}'

# Gerar relatório
curl "http://localhost:8080/reports/monthly?year=2025&month=11" | jq
```

**Observação:** Em modo `memory`, o upload S3 vai falhar (não tem credenciais AWS), mas a resposta JSON é retornada normalmente.

### 2. **Teste em Produção (AWS)**

```bash
# Acessar instância via SSM
aws ssm start-session --target i-xxxxx

# Verificar logs da aplicação
sudo journalctl -u finance-tracker -f

# Testar endpoint
curl "http://localhost:8080/reports/monthly?year=2025&month=11"

# Verificar arquivo no S3
aws s3 ls s3://finance-tracker-releases/reports/
aws s3 cp s3://finance-tracker-releases/reports/report-2025-11.txt - | cat
```

---

## 🐛 Problemas Encontrados e Soluções

### Problema 1: "Invalid region: region was not a valid DNS name"

**Erro:**
```
Error uploading report to S3: operation error S3: PutObject, 
https response error StatusCode: 0, 
RequestID: , HostID: , Invalid region: region was not a valid DNS name.
```

**Causa:**
- SDK não conseguiu determinar a região automaticamente
- Variável `AWS_REGION` não estava configurada

**Solução:**
```go
// Antes (sem região)
cfg, err := config.LoadDefaultConfig(ctx)

// Depois (com região explícita)
region := os.Getenv("AWS_REGION")
if region == "" {
    region = "us-east-1"
}
cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
```

### Problema 2: "request canceled, context canceled"

**Erro:**
```
Error uploading report to S3: operation error S3: PutObject, 
canceled, context canceled
```

**Causa:**
- Goroutine usava `r.Context()` (contexto da requisição HTTP)
- Ao enviar resposta HTTP, Go cancela automaticamente o contexto
- Upload ainda estava rodando quando contexto foi cancelado

**Solução:**
```go
// ❌ ERRADO
go func() {
    if err := finance.UploadReportToS3(r.Context(), ...) // contexto HTTP
}()

// ✅ CORRETO
go func() {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    if err := finance.UploadReportToS3(ctx, ...) // contexto independente
}()
```

**Lição aprendida:**
- Nunca use `r.Context()` em goroutines assíncronas
- Sempre crie contexto independente para operações em background
- Use timeout para evitar goroutines "eternas"

---

## 📚 Recursos e Documentação

### AWS SDK Go V2
- [Documentação Oficial](https://aws.github.io/aws-sdk-go-v2/docs/)
- [Guia de Migração V1 → V2](https://aws.github.io/aws-sdk-go-v2/docs/migrating/)
- [Exemplos S3](https://github.com/awsdocs/aws-doc-sdk-examples/tree/main/gov2/s3)

### Go Concurrency
- [Effective Go - Goroutines](https://go.dev/doc/effective_go#goroutines)
- [Context Package](https://pkg.go.dev/context)
- [Concurrency Patterns](https://go.dev/blog/pipelines)

### Terraform AWS
- [IAM Roles](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role)
- [IAM Policies](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/iam_role_policy)
- [S3 Buckets](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/s3_bucket)

---

## ✅ Checklist de Implementação

- [x] Criar função `GenerateMonthlyReport()` no service
- [x] Criar arquivo `s3.go` com função `UploadReportToS3()`
- [x] Adicionar handler `monthlyReport()` em `handlers.go`
- [x] Registrar rota `GET /reports/monthly` no mux
- [x] Adicionar dependências AWS SDK no `go.mod`
- [x] Criar IAM policy para acesso S3 no Terraform
- [x] Criar documentação `REPORTS_API.md`
- [x] Testar localmente em modo memory
- [x] Testar em produção com upload S3
- [x] Resolver erro de região S3
- [x] Resolver erro de context cancelado
- [x] Validar arquivo no S3
- [x] Criar explicação técnica completa (este documento)

---

## 🚀 Próximos Passos (Melhorias Futuras)

1. **Logging estruturado**: Substituir `fmt.Printf` por logger apropriado (zap, logrus)
2. **Métricas**: Adicionar contador de relatórios gerados, tempo de upload, etc
3. **Retry logic**: Tentar novamente se upload S3 falhar
4. **Notificações**: Enviar email/Slack quando relatório for gerado
5. **Download direto**: Endpoint para baixar relatório do S3
6. **Relatório por categoria**: Quebrar despesas por categoria no relatório
7. **Gráficos**: Gerar gráficos PNG e fazer upload junto com o texto
8. **Histórico**: Endpoint para listar todos os relatórios já gerados

---

## 📝 Resumo para Alunos

### O que vocês aprenderam:

1. ✅ **Integração com AWS S3** usando SDK V2
2. ✅ **Goroutines e concorrência** em Go
3. ✅ **Context pattern** e seus diferentes tipos
4. ✅ **IAM Roles e policies** no Terraform
5. ✅ **Formatação de relatórios** em texto
6. ✅ **Tratamento de erros assíncronos**
7. ✅ **Debugging de problemas em produção**
8. ✅ **Boas práticas de API REST**

### Principais Conceitos:

- **Operações assíncronas** melhoram performance da API
- **Context independente** é essencial para goroutines
- **IAM Roles** são mais seguras que access keys hardcoded
- **Terraform** gerencia infraestrutura como código
- **Tratamento de erros** deve ser explícito e apropriado

---

## 🎓 Exercícios Propostos

1. **Adicionar campo de observações** no relatório (campo opcional)
2. **Criar endpoint** `GET /reports/list` que lista todos os relatórios do S3
3. **Implementar retry** com backoff exponencial no upload S3
4. **Adicionar testes unitários** para `GenerateMonthlyReport()`
5. **Criar relatório anual** agregando todos os meses
6. **Adicionar validação** de range de datas (ano > 2020, mês entre 1-12)
7. **Implementar cache** para evitar gerar relatório duplicado

---

**Dúvidas?** Entre em contato ou abra uma issue no repositório!

✨ **Feature implementada com sucesso!** ✨