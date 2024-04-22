FROM golang:1.22.2 AS builder

ENV GO111MODULE=on
WORKDIR /go/release
ADD . .
RUN set -x \
    && CGO_ENABLED=1 go build -trimpath -ldflags="-linkmode external -extldflags -static -s -w -buildid=" -o bepusdt ./main

FROM debian:latest

ENV DEBIAN_FRONTEND noninteractive
ENV DEBCONF_NOWARNINGS="yes"
ENV TZ=Asia/Shanghai

COPY --from=builder /go/release/bepusdt /runtime/bepusdt

ADD ./templates /runtime/templates
ADD ./static /runtime/static

RUN apt-get update && apt-get install -y --no-install-recommends tzdata ca-certificates libc6 libgcc1 libstdc++6 \
    && ln -fs /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && dpkg-reconfigure -f noninteractive tzdata \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

EXPOSE 8080
CMD ["/runtime/bepusdt"]