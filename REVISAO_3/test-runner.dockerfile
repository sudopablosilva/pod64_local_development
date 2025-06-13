FROM golang:1.23

WORKDIR /app

# Instalar ferramentas necessárias
RUN apt-get update && apt-get install -y \
    curl \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Copiar arquivos de teste
COPY integration-tests/ ./integration-tests/

# Configurar ambiente
ENV AWS_ENDPOINT=http://localstack:4566
ENV AWS_REGION=us-east-1
ENV AWS_ACCESS_KEY_ID=test
ENV AWS_SECRET_ACCESS_KEY=test

# Script para executar os testes
COPY docker-test-runner.sh ./
RUN chmod +x ./docker-test-runner.sh

CMD ["./docker-test-runner.sh"]