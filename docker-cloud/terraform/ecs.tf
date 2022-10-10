resource "aws_ecs_cluster" "docker_cloud" {
  name = "docker-cloud"
}

resource "aws_ecs_task_definition" "docker_cloud" {
  family                   = "docker-cloud"
  network_mode             = "host"
  requires_compatibilities = ["EC2"]
  cpu                      = "256"
  memory                   = "512"
  container_definitions = jsonencode(
    [{
      name : "docker-cloud",
      image : "${aws_ecrpublic_repository.docker_cloud.repository_uri}:latest",
      portMappings : [
        {
          containerPort : 8080,
          hostPort : 8080,
          protocol : "tcp"
        },
      ]
  }])

}

resource "aws_ecs_service" "docker_cloud" {
  name            = "docker-cloud"
  cluster         = aws_ecs_cluster.docker_cloud.id
  task_definition = aws_ecs_task_definition.docker_cloud.arn
  desired_count   = 1
  launch_type     = "EC2"
}
