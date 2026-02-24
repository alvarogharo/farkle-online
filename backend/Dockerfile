FROM golang:1.23 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o farkle-server .

FROM alpine:3.20

WORKDIR /app

ENV FARKLE_PORT=8080

COPY --from=builder /app/farkle-server .

EXPOSE 8080

CMD ["./farkle-server"]

