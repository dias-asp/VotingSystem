FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/anonym_service ./cmd/anonym_service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/anonym_service /usr/local/bin/anonym_service
COPY --from=builder /src/migrations /app/migrations
WORKDIR /app
EXPOSE 8081
ENTRYPOINT ["/usr/local/bin/anonym_service"]
