FROM golang
ADD . /src
WORKDIR /src/cmd/app
RUN go build -v
ENTRYPOINT /src/cmd/app/app