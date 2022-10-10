resource "aws_lb_target_group" "docker_cloud" {
  name     = "docker-cloud-target-group"
  port     = 80
  protocol = "HTTP"
  vpc_id   = data.aws_vpc.default.id

  target_type = "ip"

}

resource "aws_lb" "docker_cloud" {
  name               = "docker-cloud-load-balancer"
  internal           = false
  load_balancer_type = "application"
  subnets            = data.aws_subnets.public.ids

  enable_http2 = true
}

resource "aws_lb_listener" "docker_cloud" {
  load_balancer_arn = aws_lb.docker_cloud.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.docker_cloud.arn
  }
}
