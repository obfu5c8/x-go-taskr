
ROOT_DIR:=$(shell dirname $(realpath $(firstword $(MAKEFILE_LIST))))



start-localstack:
	mkdir -p $(ROOT_DIR)/.localstack
	LOCALSTACK_VOLUME_DIR="$(ROOT_DIR)/.localstack" \
	localstack start