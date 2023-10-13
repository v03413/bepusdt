FROM golang:1.21.2-alpine AS builder

ENV GO111MODULE=on
WORKDIR /go/release
ADD . .
RUN set -x \
    && apk --no-cache add build-base \
    && CGO_ENABLED=1 go build -trimpath -ldflags="-s -w -buildid=" -o bepusdt ./main

FROM alpine:latest
ENV DEBIAN_FRONTEND noninteractive
ENV DEBCONF_NOWARNINGS="yes"
ENV TZ=Asia/Shanghai
COPY --from=builder /go/release/bepusdt /runtime/bepusdt
ADD ./templates /runtime/templates
ADD ./static /runtime/static
RUN apk --no-cache add tzdata ca-certificates libc6-compat libgcc libstdc++
EXPOSE 8080
CMD ["/runtime/bepusdt"]