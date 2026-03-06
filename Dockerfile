FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o electroshop .

FROM cgr.dev/chainguard/static:latest

WORKDIR /app

COPY --from=builder /app/electroshop /app/electroshop

EXPOSE 8443

CMD ["/app/electroshop"]

