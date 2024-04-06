resource "aws_ecs_cluster" "docker_cloud" {
  name = "docker-cloud"
}

resource "aws_ecs_task_definition" "docker_cloud" {
  family                   = "docker-cloud"
  network_mode             = "awsvpc"
  requires_compatibilities = ["FARGATE"]
  cpu                      = "256"
  memory                   = "512"

  container_definitions = jsonencode(
    [{
      name : "docker-cloud",
      image : "${aws_ecrpublic_repository.docker_cloud.repository_uri}:${var.image_tag}",
      essential : true,
      portMappings : [
        {
          containerPort : 80,
          hostPort : 80,
          protocol : "tcp"
        },
      ]
  }])

}
resource "aws_ecs_service" "docker_cloud" {
  name                = "docker-cloud"
  cluster             = aws_ecs_cluster.docker_cloud.id
  task_definition     = aws_ecs_task_definition.docker_cloud.arn
  scheduling_strategy = "REPLICA"
  launch_type         = "FARGATE"
  desired_count       = 1

  load_balancer {
    target_group_arn = aws_lb_target_group.docker_cloud.arn
    container_name   = "docker-cloud"
    container_port   = 80
  }

  network_configuration {
    subnets          = data.aws_subnets.public.ids
    security_groups  = [aws_security_group.lb.id]
    assign_public_ip = true
  }
}
