# ---------- Build ----------
# The builder runs on the build host's own platform and Go cross-compiles for the target, so
# multi-arch builds don't run the compiler under QEMU. Only the runtime stage is per platform.
FROM --platform=$BUILDPLATFORM golang:1.26-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT
WORKDIR /app

# faster caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# GOARM comes from the variant (linux/arm/v7 -> 7) and only applies to GOARCH=arm.
RUN if [ "${TARGETARCH}" = "arm" ]; then export GOARM="${TARGETVARIANT#v}"; fi && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /app/treningheten .

# ---------- Runtime ----------
FROM alpine:3.24
ENV PUID=1000 PGID=1000 LANG=C.UTF-8 LC_ALL=C.UTF-8
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /app/treningheten /app/treningheten 
COPY --from=builder /app/entrypoint.sh /app/entrypoint.sh
COPY --from=builder /app/web/ /app/web/
RUN addgroup -g ${PGID} appgroup && \
    adduser -D -u ${PUID} -G appgroup appuser && \
    chmod +x /app/treningheten /app/entrypoint.sh && \
    chown -R appuser:appgroup /app
USER appuser
ENTRYPOINT ["/app/entrypoint.sh"]