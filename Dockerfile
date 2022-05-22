FROM golang:1.15-alpine3.14 as builder

WORKDIR /app

COPY . .

RUN CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct go build -o ingress-manager main.go

FROM alpine:3.14

WORKDIR /app

COPY --from=builder /app/ingress-manager .

CMD ["./ingress-manager"]



