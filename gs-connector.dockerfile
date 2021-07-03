# Builder
FROM golang:alpine AS builder
LABEL stage=builder
RUN apk update && apk add --no-cache git ca-certificates && update-ca-certificates
ENV USER=gulfstream
ENV UID=10001
RUN adduser \
    --disabled-password \
    --gecos "" \
    --home "/nonexistent" \
    --shell "/sbin/nologin" \
    --no-create-home \
    --uid "${UID}" \
    "${USER}"
WORKDIR /app
COPY . .
RUN GOOS=linux GOARCH=amd64 go build -mod=mod -ldflags="-w -s" ./cmd/gs-connector

# Service
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group
COPY --from=builder /app/gs-connector /app/gs-connector
USER gulfstream:gulfstream
ENTRYPOINT ["/app/gs-connector"]