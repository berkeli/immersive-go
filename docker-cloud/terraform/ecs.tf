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

  network_configuration {
    subnets = data.aws_subnets.public.ids
  }
}
