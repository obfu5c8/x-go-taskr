
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))

DOCKER_COMPOSE_FILE="docker-compose.devenv.yml"
LOCALSTACK_VOLUME_DIR=$(realpath $(ROOT_DIR)/../.localstack)
SQS_TOOLS=$(realpath $(ROOT_DIR)/../../sqs)

.EXPORT_ALL_VARIABLES:


ready: install-devtools

install-devtools:
	pip3 install awscli-local

devenv.start:
	mkdir -p $(LOCALSTACK_VOLUME_DIR)
	docker compose -f $(DOCKER_COMPOSE_FILE) up --detach --wait
	$(SQS_TOOLS) setup

devenv.stop:
	docker compose -f $(DOCKER_COMPOSE_FILE) down 
