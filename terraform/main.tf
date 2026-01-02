terraform {
  required_version = ">= 1.0"
  
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  # State local - não usa S3 nem DynamoDB
}

provider "aws" {
  region = var.aws_region
}

# Security Group para Application Load Balancer
resource "aws_security_group" "alb" {
  name        = "finance-tracker-load-balancer-sg-v2"
  description = "Security group for finance-tracker ALB"
  vpc_id      = var.vpc_id

  # Permitir tráfego HTTP da internet
  ingress {
    description = "HTTP from internet"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Permitir todo tráfego de saída
  egress {
    description = "Allow all outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "finance-tracker-load-balancer-sg-v2"
  }
}

# Regra de ingress no Security Group da EC2 para aceitar tráfego do ALB
resource "aws_security_group_rule" "ec2_allow_alb" {
  type                     = "ingress"
  from_port                = 8080
  to_port                  = 8080
  protocol                 = "tcp"
  source_security_group_id = aws_security_group.alb.id
  security_group_id        = var.ec2_security_group_id
  description              = "Allow traffic from ALB on port 8080"
}

# IAM Role para EC2 instances
resource "aws_iam_role" "app" {
  name = "finance-tracker-ec2-role-v2"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = {
    Name = "finance-tracker-ec2-role-v2"
  }
}

# IAM Policy para acesso ao S3
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

# Attach AWS managed policy para SSM (Session Manager)
resource "aws_iam_role_policy_attachment" "ssm_policy" {
  role       = aws_iam_role.app.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

# IAM Policy para acesso ao Secrets Manager (RDS credentials)
resource "aws_iam_role_policy" "secrets_manager_access" {
  name = "finance-tracker-secrets-manager-access-v2"
  role = aws_iam_role.app.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
          "secretsmanager:DescribeSecret"
        ]
        Resource = "arn:aws:secretsmanager:us-east-1:*:secret:rds!*"
      }
    ]
  })
}

# Instance Profile
resource "aws_iam_instance_profile" "app" {
  name = "finance-tracker-instance-profile-v2"
  role = aws_iam_role.app.name

  tags = {
    Name = "finance-tracker-instance-profile-v2"
  }
}

# Target Group
resource "aws_lb_target_group" "app" {
  name     = "finance-tracker-target-group-v2"
  port     = 8080
  protocol = "HTTP"
  vpc_id   = var.vpc_id

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 3
    timeout             = 5
    interval            = 30
    path                = "/health"
    port                = "8080"
    matcher             = "200"
  }

  deregistration_delay = 30

  tags = {
    Name = "finance-tracker-target-group-v2"
  }
}

# Application Load Balancer
resource "aws_lb" "app" {
  name               = "finance-tracker-load-balancer-v2"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb.id]
  subnets            = var.subnet_ids

  enable_deletion_protection = false

  tags = {
    Name = "finance-tracker-load-balancer-v2"
  }
}

# ALB Listener (HTTP na porta 80 -> Target Group na porta 8080)
resource "aws_lb_listener" "http" {
  load_balancer_arn = aws_lb.app.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.app.arn
  }

  tags = {
    Name = "finance-tracker-listener-v2"
  }
}

# Launch Template
resource "aws_launch_template" "app" {
  name        = "finance-tracker-launch-template-v2"
  image_id    = var.ami_id
  instance_type = var.instance_type

  iam_instance_profile {
    arn = aws_iam_instance_profile.app.arn
  }

  # Habilitar IP público para acesso à internet
  network_interfaces {
    associate_public_ip_address = true
    security_groups             = [var.ec2_security_group_id]
    delete_on_termination       = true
  }

  monitoring {
    enabled = true
  }

  metadata_options {
    http_endpoint               = "enabled"
    http_tokens                 = "required"
    http_put_response_hop_limit = 1
  }

  tag_specifications {
    resource_type = "instance"
    tags = {
      Name = "finance-tracker-instance-v2"
    }
  }

  tag_specifications {
    resource_type = "volume"
    tags = {
      Name = "finance-tracker-volume-v2"
    }
  }

  lifecycle {
    create_before_destroy = true
  }

  tags = {
    Name = "finance-tracker-launch-template-v2"
  }
}

# Auto Scaling Group
resource "aws_autoscaling_group" "app" {
  name                = "finance-tracker-auto-scaling-v2"
  vpc_zone_identifier = var.subnet_ids
  target_group_arns   = [aws_lb_target_group.app.arn]
  health_check_type   = "ELB"
  health_check_grace_period = 300

  min_size         = var.asg_min_size
  max_size         = var.asg_max_size
  desired_capacity = var.asg_desired_capacity

  launch_template {
    id      = aws_launch_template.app.id
    version = "$Latest"
  }

  instance_refresh {
    strategy = "Rolling"
    preferences {
      min_healthy_percentage = 50
      instance_warmup        = 60
    }
  }

  tag {
    key                 = "Name"
    value               = "finance-tracker-asg-instance-v2"
    propagate_at_launch = true
  }

  lifecycle {
    create_before_destroy = true
    ignore_changes        = [desired_capacity]
  }
}