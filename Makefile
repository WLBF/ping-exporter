VERSION?=v$(shell date +%Y%m%d)-$(shell git rev-parse --short HEAD)
REGISTRY?=localhost:5000
IMAGE:=ping-exporter:$(VERSION)

build.amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build .

build.arm64v8:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build .

image.amd64:
	docker build --build-arg ARCH="amd64" -t $(IMAGE)-amd64 .

image.arm64v8:
	docker build --build-arg ARCH="arm64v8" -t $(IMAGE)-arm64v8 .

push.amd64: image.amd64
	docker tag $(IMAGE)-amd64 $(REGISTRY)/$(IMAGE)-amd64
	docker push $(REGISTRY)/$(IMAGE)-amd64

push.arm64v8: image.arm64v8
	docker tag $(IMAGE)-arm64v8 $(REGISTRY)/$(IMAGE)-arm64v8
	docker push $(REGISTRY)/$(IMAGE)-arm64v8
