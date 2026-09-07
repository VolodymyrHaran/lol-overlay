FROM golang:1.26 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /app/bin/lol-timer ./cmd/server

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /app/bin/champion-service ./cmd/champion-service


FROM alpine:3.22 AS runtime

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app


FROM runtime AS champion-service

COPY --from=builder \
    /app/bin/champion-service \
    ./champion-service

EXPOSE 50051

ENTRYPOINT ["./champion-service"]


FROM runtime AS app

COPY --from=builder \
    /app/bin/lol-timer \
    ./lol-timer

EXPOSE 8080

ENTRYPOINT ["./lol-timer"]