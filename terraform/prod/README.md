# Finance Tracker - Infraestrutura V2

Esta pasta contém a configuração Terraform para provisionar a infraestrutura **V2** do Finance Tracker na AWS.

## 📦 Recursos Criados

A infraestrutura V2 cria os seguintes recursos:

### 1. **Launch Template** (`finance-tracker-launch-template-v2`)
- AMI: Amazon Linux configurável via variável
- Tipo de instância: Configurável (padrão: t3.micro)
- **IP Público: DESABILITADO** (apenas IP privado)
- IAM Instance Profile com permissões S3
- Metadata IMDSv2 obrigatório

### 2. **Auto Scaling Group** (`finance-tracker-auto-scaling-v2`)
- Min: 2 instâncias (configurável)
- Max: 6 instâncias (configurável)
- Desired: 2 instâncias (configurável)
- Health Check: ELB (verifica `/health` na porta 8080)
- Rolling Update: 50% instâncias healthy durante deploy

### 3. **Target Group** (`finance-tracker-target-group-v2`)
- Porta: **8080** (porta da aplicação nas EC2)
- Protocolo: HTTP
- Health Check:
  - Endpoint: `/health`
  - Porta: `8080`
  - Intervalo: 30s
  - Timeout: 5s

### 4. **Application Load Balancer** (`finance-tracker-load-balancer-v2`)
- Tipo: Application Load Balancer
- Internet-facing (público)
- Listener HTTP:
  - Recebe na porta **80**
  - Encaminha para Target Group (porta **8080**)

### 5. **Security Group ALB** (`finance-tracker-load-balancer-sg-v2`)
- Ingress: Porta 80 (HTTP) de 0.0.0.0/0
- Egress: Todo tráfego permitido

### 6. **IAM Role EC2** (`finance-tracker-ec2-role-v2`)
Permissões S3:
```json
{
  "Effect": "Allow",
  "Action": [
    "s3:PutObject",
    "s3:GetObject",
    "s3:ListBucket",
    "s3:DeleteObject"
  ],
  "Resource": [
    "arn:aws:s3:::finance-tracker-releases",
    "arn:aws:s3:::finance-tracker-releases/*"
  ]
}
```

## 🔧 Variáveis Obrigatórias

Você deve fornecer no arquivo `terraform.tfvars`:

| Variável | Descrição | Exemplo |
|----------|-----------|---------|
| `vpc_id` | ID da VPC | `vpc-0123456789abcdef` |
| `subnet_ids` | Lista de subnets (mín. 2) | `["subnet-xxx", "subnet-yyy"]` |
| `ami_id` | AMI do Amazon Linux | `ami-0c02fb55b7b5b1ea1` |
| `ec2_security_group_id` | Security Group das EC2 | `sg-0123456789abcdef` |
| `rds_security_group_id` | Security Group do RDS | `sg-fedcba9876543210` |

## 📋 Pré-requisitos

### 1. Criar Backend S3 e DynamoDB

```bash
# Criar bucket S3 para state
aws s3 mb s3://finance-tracker-terraform-state --region us-east-1
aws s3api put-bucket-versioning --bucket finance-tracker-terraform-state --versioning-configuration Status=Enabled
aws s3api put-bucket-encryption --bucket finance-tracker-terraform-state --server-side-encryption-configuration '{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"}}]}'

# Criar tabela DynamoDB para locks
aws dynamodb create-table \
  --table-name finance-tracker-terraform-locks \
  --attribute-definitions AttributeName=LockID,AttributeType=S \
  --key-schema AttributeName=LockID,KeyType=HASH \
  --billing-mode PAY_PER_REQUEST \
  --region us-east-1
```

### 2. Obter AMI ID mais recente

```bash
aws ec2 describe-images \
  --owners amazon \
  --filters "Name=name,Values=al2023-ami-2023*-x86_64" \
  --query 'sort_by(Images, &CreationDate)[-1].ImageId' \
  --output text
```

### 3. Obter IDs dos recursos existentes

```bash
# Listar VPCs
aws ec2 describe-vpcs --query 'Vpcs[*].[VpcId,Tags[?Key==`Name`].Value|[0]]' --output table

# Listar Subnets
aws ec2 describe-subnets --query 'Subnets[*].[SubnetId,AvailabilityZone,Tags[?Key==`Name`].Value|[0]]' --output table

# Listar Security Groups
aws ec2 describe-security-groups --query 'SecurityGroups[*].[GroupId,GroupName,Description]' --output table
```

## 🚀 Como Usar

### 1. Preencher variáveis

Edite `terraform.tfvars` e preencha os valores reais:

```hcl
vpc_id                = "vpc-0abc123def456"
subnet_ids            = ["subnet-111", "subnet-222"]
ami_id                = "ami-0c02fb55b7b5b1ea1"
ec2_security_group_id = "sg-ec2abc123"
rds_security_group_id = "sg-rdsdef456"
```

### 2. Inicializar Terraform

```bash
cd terraform
terraform init -backend-config=prod/backend.tfvars
```

### 3. Planejar mudanças

```bash
terraform plan -var-file=prod/terraform.tfvars
```

### 4. Aplicar infraestrutura

```bash
terraform apply -var-file=prod/terraform.tfvars
```

### 5. Verificar outputs

```bash
terraform output
```

## 📤 Outputs Disponíveis

Após aplicar, você terá acesso a:

- `alb_dns_name` - URL do Load Balancer
- `launch_template_id` - ID do Launch Template
- `launch_template_name` - Nome do Launch Template (para usar no pipeline)
- `autoscaling_group_name` - Nome do Auto Scaling Group
- `target_group_arn` - ARN do Target Group
- `iam_role_arn` - ARN da IAM Role EC2

## 🔒 Segurança

- ✅ **IP Público desabilitado** nas EC2
- ✅ **IMDSv2 obrigatório** (proteção contra SSRF)
- ✅ **Health checks** na porta 8080
- ✅ **Permissões S3 específicas** (não usa wildcards)
- ✅ **Terraform state criptografado** no S3
- ✅ **State locking** via DynamoDB

## 🆚 Diferenças da V1

| Aspecto | V1 | V2 |
|---------|----|----|
| Nome dos recursos | `finance-tracker-*` | `finance-tracker-*-v2` |
| IP Público EC2 | Habilitado | **Desabilitado** |
| Permissões S3 | Apenas Get/List | **Get/List/Put/Delete** |
| Security Groups | Criados pelo Terraform | **Fornecidos via variável** |
| Subnets | Públicas/Privadas separadas | **Lista única configurável** |

## 🧹 Cleanup

Para destruir toda a infraestrutura V2:

```bash
terraform destroy -var-file=prod/terraform.tfvars
```

## 📞 Troubleshooting

### Erro: "No declaration found for var.X"
- Verifique se preencheu todas as variáveis obrigatórias no `terraform.tfvars`

### Erro: "Error launching source instance"
- Verifique se o AMI ID está correto para a região us-east-1
- Verifique se o Security Group permite tráfego na porta 8080

### Instâncias não ficam healthy
- Verifique se a aplicação está rodando na porta 8080
- Verifique se o endpoint `/health` retorna HTTP 200
- Verifique logs em CloudWatch ou via SSH nas instâncias