IMG ?= harbor.cestc.com/public-release/go-sniffer:v0.0.1
REMOTEIMGX86 ?= 10.32.226.224:85/public-release/go-sniffer:v0.0.1_x86
REMOTEIMGARM ?= 10.32.226.224:85/public-release/go-sniffer:v0.0.1_arm
TARGET = go-sniffer
all: $(TARGET)

$(TARGET):
	go build -ldflags "-s -w" -o bin/$@

init:
	swag init

run:
	go run main.go -config config.json

docker_x86:
	docker build -t $(REMOTEIMGX86) .

docker_arm:
	docker build -t $(REMOTEIMGARM) . -f ./Dockerfile_arm

push:
	docker push $(IMG)

kind:
	kind load docker-image --name networkpolicy $(IMG)
	kind load docker-image --name networkpolicy $(IMG)

ci: docker_x86 kind

cd:
	kubectl apply -f ./build

del:
	kubectl delete -f ./build

roll:
	kubectl -n gb patch deployment go-sniffer --patch "{\"spec\":{\"template\":{\"metadata\":{\"annotations\":{\"date\":\"`date +'%s'`\"}}}}}"

pull_arm_image:
	docker pull --platform=linux/arm64 golang:1.16.15-stretch

pull_x86_image:
	docker pull --platform=linux/amd64 golang:1.16.15-stretch

tag_x86:
	docker tag $(IMG) $(REMOTEIMGX86)
	docker push $(REMOTEIMGX86)

tag_arm:
	docker push $(REMOTEIMGARM)

deploy: ci cd

upload_x86: pull_x86_image docker_x86 tag_x86

upload_arm: pull_arm_image docker_arm tag_arm