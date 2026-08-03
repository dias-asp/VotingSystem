FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/voting_service ./cmd/voting_service

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/voting_service /usr/local/bin/voting_service
COPY --from=builder /src/migrations /app/migrations
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/voting_service"]
