FROM golang:1.23.4-bullseye as builder

ARG TARGETARCH
ARG TARGETOS

WORKDIR /app

COPY . .

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build

FROM debian:bullseye-slim as runtime

LABEL org.opencontainers.image.source=https://github.com/aunefyren/treningheten
ARG DEBIAN_FRONTEND=noninteractive
WORKDIR /app

COPY --from=builder /app .

RUN rm /var/lib/dpkg/info/libc-bin.*
RUN apt clean
RUN apt update
RUN apt install -y ca-certificates curl
RUN chmod +x /app/treningheten /app/entrypoint.sh

ENTRYPOINT /app/entrypoint.sh