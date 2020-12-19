GO ?= go

.PHONY: run
run: build
	./smppgun load.yml

.PHONY: debug
debug:
	${GO} build -gcflags="all=-N -l" -v ./cmd/smppgun
	dlv exec ./smppgun load.yml

.PHONY: build
build:
	${GO} build -v ./cmd/smppgun

.DEFAULT_GOAL := build

.PHONY: deps
deps:
	${GO} mod tidy
	${GO} mod download

.PHONY: tank
tank: build
	docker run \
	-v $(CURDIR):/var/loadtest \
	-v $SSH_AUTH_SOCK:/ssh-agent -e SSH_AUTH_SOCK=/ssh-agent \
	--net host \
	-it direvius/yandex-tank \
	-c tank.yml
