FROM openeuler/openeuler:23.03 as BUILDER
RUN dnf update -y && \
    dnf install -y golang && \
    go env -w GOPROXY=https://goproxy.cn,direct

MAINTAINER zengchen1024<chenzeng765@gmail.com>

# build binary
WORKDIR /go/src/github.com/opensourceways/robot-gitlab-hook-delivery
COPY . .
RUN GO111MODULE=on CGO_ENABLED=0 go build -buildmode=pie --ldflags "-s -linkmode 'external' -extldflags '-Wl,-z,now'" -a -o robot-gitlab-hook-delivery .

# copy binary config and utils
FROM openeuler/openeuler:22.03
RUN dnf -y update && \
    dnf in -y shadow && \
    groupadd -g 5000 mindspore && \
    useradd -u 5000 -g mindspore -s /bin/bash -m mindspore

USER mindspore

WORKDIR /opt/app/

COPY  --chown=mindspore --from=BUILDER /go/src/github.com/opensourceways/robot-gitlab-hook-delivery/robot-gitlab-hook-delivery /opt/app/robot-gitlab-hook-delivery

ENTRYPOINT ["/opt/app/robot-gitlab-hook-delivery"]
