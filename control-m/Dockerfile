FROM golang:1.24 AS builder

# Configurar proxy Go direto para os repositórios
ENV GOPROXY=direct
ENV GOSUMDB=off

WORKDIR /app
COPY go.mod go.sum ./
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux go build -o control-m .

FROM alpine:3.22
RUN apk --no-cache add ca-certificates && \
    adduser -D -h /app gouserapp

WORKDIR /app
COPY --from=builder /app/control-m .

USER gouserapp
EXPOSE 8080

CMD ["./control-m"]