resource "random_pet" "random_pet" {

  keepers = {
    aws_prefix = "${var.aws_prefix}"
  }

  length    = 2
  separator = "-"
}

resource "random_pet" "random_pet_rds" {

  keepers = {
    aws_prefix = "${var.aws_prefix}"
  }

  length    = 2
  separator = ""
}

provider "aws" {
  region     = var.aws_region
  access_key = var.aws_access_key
  secret_key = var.aws_secret_key
}

resource "aws_instance" "aws_instance" {
  count                  = 2
  ami                    = var.aws_ami
  instance_type          = "t3.xlarge"
  subnet_id              = var.aws_subnet_id
  vpc_security_group_ids = [var.aws_security_group_id]
  key_name               = var.aws_pem_key_name

  root_block_device {
    volume_size = 200
    tags = {
      Name = "${random_pet.random_pet.keepers.aws_prefix}-${random_pet.random_pet.id}"
    }
  }

  tags = {
    Name = "${random_pet.random_pet.keepers.aws_prefix}-${random_pet.random_pet.id}"
  }
}


resource "aws_lb_target_group" "aws_lb_target_group_80" {
  name        = "${var.aws_prefix}-80-${random_pet.random_pet.id}"
  port        = 80
  protocol    = "TCP"
  target_type = "instance"
  vpc_id      = var.aws_vpc
  health_check {
    protocol          = "TCP"
    port              = "traffic-port"
    healthy_threshold = 3
    interval          = 10
  }
}

resource "aws_lb_target_group" "aws_lb_target_group_443" {
  name        = "${var.aws_prefix}-443-${random_pet.random_pet.id}"
  port        = 443
  protocol    = "TCP"
  target_type = "instance"
  vpc_id      = var.aws_vpc
  health_check {
    protocol          = "TCP"
    port              = 443
    healthy_threshold = 3
    interval          = 10
  }
}

# attach instances to the target group 80
resource "aws_lb_target_group_attachment" "attach_tg_80" {
  count            = length(aws_instance.aws_instance)
  target_group_arn = aws_lb_target_group.aws_lb_target_group_80.arn
  target_id        = aws_instance.aws_instance[count.index].id
  port             = 80
}

# attach instances to the target group 443
resource "aws_lb_target_group_attachment" "attach_tg_443" {
  count            = length(aws_instance.aws_instance)
  target_group_arn = aws_lb_target_group.aws_lb_target_group_443.arn
  target_id        = aws_instance.aws_instance[count.index].id
  port             = 443
}


# create a load balancer
resource "aws_lb" "aws_lb" {
  load_balancer_type = "network"
  name               = "${var.aws_prefix}-nlb-${random_pet.random_pet.id}"
  internal           = false
  ip_address_type    = "ipv4"
  subnets            = [var.aws_subnet_a, var.aws_subnet_b, var.aws_subnet_c]
}

# add a listener for port 80
resource "aws_lb_listener" "aws_lb_listener_80" {
  load_balancer_arn = aws_lb.aws_lb.arn
  port              = "80"
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.aws_lb_target_group_80.arn
  }
}

# add a listener for port 443
resource "aws_lb_listener" "aws_lb_listener_443" {
  load_balancer_arn = aws_lb.aws_lb.arn
  port              = "443"
  protocol          = "TCP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.aws_lb_target_group_443.arn
  }
}

resource "aws_rds_cluster" "aws_rds_cluster" {
  cluster_identifier      = "${var.aws_prefix}-${random_pet.random_pet_rds.id}"
  engine                  = "aurora-mysql"
  engine_version          = "5.7.mysql_aurora.2.10.2"
  availability_zones      = ["us-east-2a", "us-east-2b", "us-east-2c"]
  database_name           = "${var.aws_prefix}${random_pet.random_pet_rds.id}"
  master_username         = "tfadmin"
  master_password         = var.aws_rds_password
  backup_retention_period = 5
  preferred_backup_window = "07:00-09:00"
  skip_final_snapshot     = true
}

resource "aws_rds_cluster_instance" "aws_rds_cluster_instance" {
  count              = 1
  identifier         = "${var.aws_prefix}-${random_pet.random_pet_rds.id}-${count.index}"
  cluster_identifier = aws_rds_cluster.aws_rds_cluster.id
  instance_class     = "db.t3.small"
  engine             = aws_rds_cluster.aws_rds_cluster.engine
  engine_version     = aws_rds_cluster.aws_rds_cluster.engine_version
}

resource "aws_route53_record" "aws_route53_record" {
  zone_id = data.aws_route53_zone.zone.zone_id
  name    = "${var.aws_prefix}-${random_pet.random_pet.id}"
  type    = "CNAME"
  ttl     = "60"
  records = [aws_lb.aws_lb.dns_name]
}


data "aws_route53_zone" "zone" {
  name = var.aws_route53_fqdn
}