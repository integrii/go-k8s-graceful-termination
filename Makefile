build:
	docker build -t integrii/go-k8s-graceful-termination:latest .
push:
	docker push integrii/go-k8s-graceful-termination:latest
m1:
	DOCKER_BUILDKIT=0 docker buildx build --platform linux/amd64 --load -t integrii/go-k8s-graceful-termination:latest -f Dockerfile .