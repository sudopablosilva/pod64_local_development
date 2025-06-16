FROM golang:1.24 AS builder

ENV GOPROXY=direct
ENV GOSUMDB=off

WORKDIR /app

# Etapa 1: copiar só os arquivos de dependência
COPY go.mod go.sum ./

# Etapa 2: baixar os módulos (cacheável se go.mod/go.sum não mudarem)
RUN go mod download

# Etapa 3: copiar o restante do código
COPY . .

# Etapa 3.5: garantir que go.sum está sincronizado
RUN go mod tidy

# Etapa 4: build da aplicação
RUN CGO_ENABLED=0 GOOS=linux go build -o spaq .

FROM alpine:3.22
RUN apk --no-cache add ca-certificates && \
    adduser -D -h /app gouserapp

WORKDIR /app
COPY --from=builder /app/spaq .

USER gouserapp
EXPOSE 8080

CMD ["./spaq"]