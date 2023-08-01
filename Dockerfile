FROM golang:latest as BUILDER

MAINTAINER zengchen1024<chenzeng765@gmail.com>

# build binary
WORKDIR /go/src/github.com/opensourceways/robot-gitlab-hook-delivery
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 go build -a -o robot-gitlab-hook-delivery .

# copy binary config and utils
FROM alpine:3.14

RUN adduser mindspore -u 5000 -D
USER mindspore
WORKDIR /opt/app/

COPY  --from=BUILDER /go/src/github.com/opensourceways/robot-gitlab-hook-delivery/robot-gitlab-hook-delivery /opt/app/robot-gitlab-hook-delivery

ENTRYPOINT ["/opt/app/robot-gitlab-hook-delivery"]
