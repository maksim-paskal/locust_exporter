FROM golang:1.12 as build

COPY *.go /usr/src/locust_exporter/
COPY go.* /usr/src/locust_exporter/

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN cd /usr/src/locust_exporter \
  && go mod download \
  && go mod verify \
  && go build -v -o locust_exporter -ldflags "-X main.buildTime=$(date +"%Y%m%d%H%M%S")"

FROM alpine:latest

COPY --from=build /usr/src/locust_exporter/locust_exporter /usr/local/bin/locust_exporter

CMD locust_exporter