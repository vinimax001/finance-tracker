# 🚀 Deploy com Auto Scaling Group - Guia Completo

## 📋 Pré-requisitos na AWS

### 1. Criar S3 Bucket para releases

```bash
aws s3 mb s3://finance-tracker-releases --region us-east-1

# Habilitar versionamento
aws s3api put-bucket-versioning \
  --bucket finance-tracker-releases \
  --versioning-configuration Status=Enabled
```

---

### 2. Configurar OIDC Provider no IAM

#### 2.1. Criar Identity Provider

```bash
# Via Console:
IAM → Identity Providers → Add Provider

Provider Type: OpenID Connect
Provider URL: https://token.actions.githubusercontent.com
Audience: sts.amazonaws.com
```

#### 2.2. Criar IAM Role para GitHub Actions

**Tipo de Role**: `Custom trust policy` (ou `Web identity`)

**Trust Policy** (Trust Relationship):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::SUA-CONTA-AWS:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
          "token.actions.githubusercontent.com:sub": "repo:danielgundim/finance-tracker:ref:refs/heads/main"
        }
      }
    }
  ]
}
```

**Como criar via Console AWS:**

1. **IAM → Roles → Create role**
2. **Trusted entity type**: `Web identity`
3. **Identity provider**: `token.actions.githubusercontent.com`
4. **Audience**: `sts.amazonaws.com`
5. **GitHub organization**: `danielgundim`
6. **GitHub repository**: `finance-tracker`
7. **GitHub branch**: `main`
8. **Role name**: `GitHubActionsDeployRole`

**Ou via AWS CLI:**

```bash
# Substitua SUA-CONTA-AWS pelo seu Account ID (ex: 123456789012)
cat > trust-policy.json << 'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::SUA-CONTA-AWS:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com",
          "token.actions.githubusercontent.com:sub": "repo:danielgundim/finance-tracker:ref:refs/heads/main"
        }
      }
    }
  ]
}
EOF

# Criar a role
aws iam create-role \
  --role-name GitHubActionsDeployRole \
  --assume-role-policy-document file://trust-policy.json \
  --description "Role for GitHub Actions to deploy finance-tracker"

# Anotar o ARN que será retornado:
# arn:aws:iam::SUA-CONTA-AWS:role/GitHubActionsDeployRole
```

**Importante**: No `sub`, o formato é sempre `repo:OWNER/REPO:ref:refs/heads/BRANCH`
- `OWNER` = `danielgundim` (seu username do GitHub)
- `REPO` = `finance-tracker`
- `BRANCH` = `main`

#### 2.3. Anexar políticas ao Role

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::finance-tracker-releases",
        "arn:aws:s3:::finance-tracker-releases/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:CreateLaunchTemplateVersion",
        "ec2:ModifyLaunchTemplate",
        "ec2:DescribeLaunchTemplates",
        "ec2:DescribeLaunchTemplateVersions"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "autoscaling:UpdateAutoScalingGroup",
        "autoscaling:DescribeAutoScalingGroups",
        "autoscaling:StartInstanceRefresh",
        "autoscaling:DescribeInstanceRefreshes",
        "autoscaling:CancelInstanceRefresh"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "elasticloadbalancing:DescribeTargetHealth",
        "elasticloadbalancing:DescribeTargetGroups"
      ],
      "Resource": "*"
    }
  ]
}
```

---

### 3. Criar VPC e Subnets (se não tiver)

```bash
# Usar VPC padrão ou criar nova VPC
# Precisa de pelo menos 2 subnets públicas em AZs diferentes
```

---

### 4. Criar Security Groups

#### 4.1. Security Group da EC2

```bash
aws ec2 create-security-group \
  --group-name finance-tracker-ec2-sg \
  --description "Security group para EC2 do finance-tracker" \
  --vpc-id vpc-xxxxx

# Permitir tráfego do ALB na porta 8080
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxx \
  --protocol tcp \
  --port 8080 \
  --source-group sg-ALB-ID

# Permitir SSH (opcional, para debug)
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxx \
  --protocol tcp \
  --port 22 \
  --cidr 0.0.0.0/0
```

#### 4.2. Security Group do RDS

```bash
# Permitir tráfego das EC2 na porta 5432
aws ec2 authorize-security-group-ingress \
  --group-id sg-RDS-ID \
  --protocol tcp \
  --port 5432 \
  --source-group sg-EC2-ID
```

#### 4.3. Security Group do ALB

```bash
aws ec2 create-security-group \
  --group-name finance-tracker-alb-sg \
  --description "Security group para ALB do finance-tracker" \
  --vpc-id vpc-xxxxx

# Permitir HTTP
aws ec2 authorize-security-group-ingress \
  --group-id sg-ALB-ID \
  --protocol tcp \
  --port 80 \
  --cidr 0.0.0.0/0

# Permitir HTTPS (se tiver certificado)
aws ec2 authorize-security-group-ingress \
  --group-id sg-ALB-ID \
  --protocol tcp \
  --port 443 \
  --cidr 0.0.0.0/0
```

---

### 5. Criar IAM Instance Profile para EC2

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::finance-tracker-releases",
        "arn:aws:s3:::finance-tracker-releases/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    }
  ]
}
```

```bash
# Criar role
aws iam create-role \
  --role-name finance-tracker-ec2-role \
  --assume-role-policy-document '{
    "Version": "2012-10-17",
    "Statement": [{
      "Effect": "Allow",
      "Principal": {"Service": "ec2.amazonaws.com"},
      "Action": "sts:AssumeRole"
    }]
  }'

# Anexar política
aws iam put-role-policy \
  --role-name finance-tracker-ec2-role \
  --policy-name finance-tracker-ec2-policy \
  --policy-document file://ec2-policy.json

# Criar instance profile
aws iam create-instance-profile \
  --instance-profile-name finance-tracker-ec2-profile

# Associar role ao instance profile
aws iam add-role-to-instance-profile \
  --instance-profile-name finance-tracker-ec2-profile \
  --role-name finance-tracker-ec2-role
```

---

### 6. Criar Application Load Balancer

```bash
# Criar ALB
aws elbv2 create-load-balancer \
  --name finance-tracker-alb \
  --subnets subnet-xxxxx subnet-yyyyy \
  --security-groups sg-ALB-ID \
  --scheme internet-facing \
  --type application

# Criar Target Group
aws elbv2 create-target-group \
  --name finance-tracker-tg \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-xxxxx \
  --health-check-path /health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3

# Criar Listener
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:... \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:...
```

---

### 7. Criar Launch Template

```bash
aws ec2 create-launch-template \
  --launch-template-name finance-tracker-lt \
  --version-description "Initial version" \
  --launch-template-data '{
    "ImageId": "ami-0c55b159cbfafe1f0",
    "InstanceType": "t3.micro",
    "KeyName": "SUA-KEY-PAIR",
    "IamInstanceProfile": {
      "Name": "finance-tracker-ec2-profile"
    },
    "SecurityGroupIds": ["sg-EC2-ID"],
    "TagSpecifications": [{
      "ResourceType": "instance",
      "Tags": [
        {"Key": "Name", "Value": "finance-tracker"},
        {"Key": "ManagedBy", "Value": "AutoScaling"}
      ]
    }],
    "UserData": ""
  }'
```

---

### 8. Criar Auto Scaling Group

```bash
aws autoscaling create-auto-scaling-group \
  --auto-scaling-group-name finance-tracker-asg \
  --launch-template "LaunchTemplateName=finance-tracker-lt,Version=$Latest" \
  --min-size 2 \
  --max-size 4 \
  --desired-capacity 2 \
  --vpc-zone-identifier "subnet-xxxxx,subnet-yyyyy" \
  --target-group-arns arn:aws:elasticloadbalancing:... \
  --health-check-type ELB \
  --health-check-grace-period 300 \
  --tags "Key=Name,Value=finance-tracker-asg,PropagateAtLaunch=true"
```

---

## 🔐 Configurar Secrets no GitHub

```
GitHub → Repositório → Settings → Secrets and variables → Actions

Adicionar:
- AWS_ROLE_ARN: arn:aws:iam::SUA-CONTA:role/GitHubActionsRole
- DATABASE_URL: postgres://postgres:senha@rds-endpoint:5432/financetracker?sslmode=require
```

---

## 🚀 Como funciona o Pipeline

### Fluxo completo:

1. **Build**: Compila binário Go
2. **Upload**: Envia para S3 versionado
3. **User Data**: Prepara script com configurações
4. **Launch Template**: Cria nova versão
5. **ASG Update**: Atualiza Auto Scaling Group
6. **Instance Refresh**: Rolling update (50% por vez)
7. **Health Check**: Verifica Target Group
8. **Rollback**: Automático se falhar

### Rolling Update:

```
Antes:  [EC2-v1] [EC2-v1]
        ↓ Instance Refresh
Meio:   [EC2-v1] [EC2-v2] (50% saudável)
        ↓
Depois: [EC2-v2] [EC2-v2] (100% saudável)
```

---

## 🎯 Testar o Deploy

```bash
# 1. Fazer commit
git add .
git commit -m "feat: deploy com auto scaling"
git push origin main

# 2. Acompanhar no GitHub Actions
# GitHub → Actions → Deploy to Auto Scaling Group

# 3. Verificar ALB
curl http://ALB-DNS-NAME/health

# 4. Ver instâncias
aws autoscaling describe-auto-scaling-groups \
  --auto-scaling-group-names finance-tracker-asg
```

---

## 🔄 Rollback Manual

```bash
# Listar versões do Launch Template
aws ec2 describe-launch-template-versions \
  --launch-template-name finance-tracker-lt

# Voltar para versão anterior
aws ec2 modify-launch-template \
  --launch-template-name finance-tracker-lt \
  --default-version 1

# Fazer instance refresh
aws autoscaling start-instance-refresh \
  --auto-scaling-group-name finance-tracker-asg
```

---

## 📊 Monitoramento

### CloudWatch Logs

```bash
# Ver logs das instâncias
aws logs tail /var/log/user-data.log --follow
```

### Métricas importantes

- Target Group Healthy Host Count
- Auto Scaling Group In Service Instances
- Application Load Balancer Request Count
- EC2 CPU Utilization

---

## 💰 Custos Estimados (us-east-1)

- **ALB**: ~$16/mês
- **EC2 t3.micro x2**: ~$15/mês
- **RDS db.t3.micro**: ~$13/mês
- **S3**: ~$1/mês
- **Total**: ~$45/mês

---

## ✅ Vantagens desta Arquitetura

- ✅ **Zero downtime** nos deploys
- ✅ **Auto scaling** baseado em métricas
- ✅ **Alta disponibilidade** (multi-AZ)
- ✅ **Rollback rápido** (só mudar versão do LT)
- ✅ **Imutabilidade** (cada deploy é uma nova versão)
- ✅ **Health checks** automáticos
- ✅ **Versionamento** de releases no S3