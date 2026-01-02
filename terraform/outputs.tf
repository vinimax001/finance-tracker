output "alb_dns_name" {
  description = "DNS name of the Application Load Balancer"
  value       = aws_lb.app.dns_name
}

output "alb_arn" {
  description = "ARN of the Application Load Balancer"
  value       = aws_lb.app.arn
}

output "target_group_arn" {
  description = "ARN of the Target Group"
  value       = aws_lb_target_group.app.arn
}

output "autoscaling_group_name" {
  description = "Name of the Auto Scaling Group"
  value       = aws_autoscaling_group.app.name
}

output "launch_template_id" {
  description = "ID of the Launch Template"
  value       = aws_launch_template.app.id
}

output "launch_template_name" {
  description = "Name of the Launch Template"
  value       = aws_launch_template.app.name
}

output "security_group_alb_id" {
  description = "ID of the ALB Security Group"
  value       = aws_security_group.alb.id
}

output "iam_role_arn" {
  description = "ARN of the IAM Role for EC2 instances"
  value       = aws_iam_role.app.arn
}

output "iam_instance_profile_arn" {
  description = "ARN of the IAM Instance Profile"
  value       = aws_iam_instance_profile.app.arn
}