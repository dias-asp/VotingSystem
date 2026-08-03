FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/audit_service ./cmd/audit_service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/audit_service /usr/local/bin/audit_service
COPY --from=builder /src/migrations /app/migrations
WORKDIR /app
EXPOSE 8082
ENTRYPOINT ["/usr/local/bin/audit_service"]
