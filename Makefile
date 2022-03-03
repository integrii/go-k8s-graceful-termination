build:
	docker build -t integrii/go-k8s-graceful-termination:latest .
push:
	docker push integrii/go-k8s-graceful-termination:latest