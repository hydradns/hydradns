FROM golang:1.25-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /bin/scanner ./cmd/scanner

FROM alpine:3.22
WORKDIR /app

RUN apk add --no-cache ca-certificates
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

COPY --from=builder /bin/scanner /app/scanner

USER appuser

EXPOSE 8080

CMD ["/app/scanner"]
