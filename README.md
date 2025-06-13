# POC BDD - Microservices Job Processing Pipeline (VERSÃO CORRIGIDA)

Este projeto implementa uma arquitetura de microserviços para processamento de jobs com integração BDD (Behavior-Driven Development) e **correções completas de integração com LocalStack**.

## 🎯 Status do Projeto

**✅ TOTALMENTE FUNCIONAL** - Todas as correções implementadas e testadas com sucesso.

## 🏗️ Arquitetura

O sistema consiste nos seguintes serviços:

| Serviço | Porta | Função |  |
|---------|-------|--------|--------|
| **Control-M** | 4333 | Job submission e management |
| **JMI** | 4333 | Job Manager Integrator + Monitor |
| **JMW** | 8080 | Job Manager Worker |
| **JMR** | 8084 | Job Manager Runner |
| **Scheduler Plugin** | 8085 | Criação de schedules |
| **SPA** | 4444 | Scheduler Plugin Adapter |
| **SPAQ** | 8087 | Scheduler Plugin Adapter Queue |

## 📊 Fluxo de Dados

```
Cliente → Control-M → JMI → JMW → JMR → Scheduler Plugin → SPA → SPAQ
                      ↓     ↓     ↓            ↓           ↓     ↓
                   DynamoDB Tables + SQS Queues (LocalStack)
```

**Veja diagramas detalhados:**
- [📊 Diagrama de Fluxo de Dados](./DIAGRAMA_FLUXO_DADOS.md)
- [🔄 Diagrama de Sequência](./DIAGRAMA_SEQUENCIA.md)

## 🚀 Quick Start

### 1. **Iniciar o Sistema**
```bash
# Construir todos os serviços
finch compose -f finch-compose.yml build

# Iniciar o sistema completo
finch compose -f finch-compose.yml up -d

# Aguardar inicialização (30s)
sleep 30
```

### 2. **Verificar Status**
```bash
# Dashboard em tempo real
./dashboard.sh

# Verificar health de todos os serviços
for port in 4333 8080 8084 8085 4444 8087; do
    echo "Porta $port: $(curl -s http://localhost:$port/health | jq -r '.status // "offline"')"
done
```

### 3. **Executar Testes**
```bash
# Teste completo do fluxo
./test-complete-flow.sh

# Teste individual
curl -X POST http://localhost:4333/startExecution \
  -H "Content-Type: application/json" \
  -d '{"executionName": "TESTE_001"}'
```

## 🔧 Configuração de Latência

```bash
# Configurar latência de 5 segundos
./set-latency.sh 5000

# Remover latência (velocidade máxima)
./set-latency.sh 0

# Verificar configuração atual
grep PROCESSING_DELAY_MS finch-compose.yml
```

## 📊 Monitoramento

### **Dashboard em Tempo Real**
```bash
./dashboard.sh
```

### **Endpoints de Monitoramento (via JMI)**
| Endpoint | Função | Exemplo |
|----------|--------|---------|
| `/tables` | Lista tabelas DynamoDB | `curl http://localhost:4333/tables` |
| `/executions` | Lista execuções versionadas | `curl http://localhost:4333/executions` |
| `/queues` | Status das filas SQS | `curl http://localhost:4333/queues` |
| `/health` | Status do serviço | `curl http://localhost:4333/health` |

### **Exemplo de Resposta - Execuções**
```json
{
  "count": 5,
  "executions": [
    {
      "executionName": "TEST_123#v1#jmi-start",
      "originalName": "TEST_123",
      "status": "started",
      "stage": "jmi-start",
      "processedBy": "JMI",
      "version": 1,
      "timestamp": 1749840000
    }
  ],
  "service": "jmi"
}
```

## 🗄️ Dados Persistidos

### **Tabelas DynamoDB**
- `executions` - Execuções versionadas com metadados completos
- `jobs` - Definições e status de jobs
- `schedules` - Configurações de agendamento
- `adapters` - Configurações de adaptadores
- `queue_messages` - Logs e estatísticas de mensagens

### **Filas SQS**
- `job-requests` - Solicitações de processamento
- `jmw-queue` - Jobs processados
- `jmr-queue` - Execuções completadas
- `sp-queue` - Agendamentos criados
- `spa-queue` - Adaptações configuradas
- `spaq-queue` - Mensagens finalizadas

## 🧪 Testes BDD

Os testes de integração estão escritos em sintaxe Gherkin e implementados com Godog:

### **Funcionalidades Testadas**
- ✅ Pipeline completo de processamento de jobs
- ✅ Comunicação entre serviços via SQS
- ✅ Persistência de dados versionados
- ✅ Health checks de todos os serviços
- ✅ Monitoramento em tempo real

### **Executar Testes**
```bash
# Teste completo do fluxo
./test-complete-flow.sh

# Resultados esperados:
# ✓ 6/6 testes principais passaram
# ✓ 7/7 health checks OK
# ✓ Dados persistidos corretamente
# ✓ Filas SQS funcionando
```

## 🔍 Troubleshooting

### **Serviços Não Iniciam**
```bash
# Verificar portas em uso
netstat -an | grep -E "(4333|8080|8084|8085|4444|8087|4566)"

# Verificar logs
finch compose -f finch-compose.yml logs [service-name]

# Reiniciar serviços
finch compose -f finch-compose.yml restart
```

### **Dados Não Aparecem**
```bash
# Verificar via endpoints do JMI (sempre funciona)
curl http://localhost:4333/executions | jq .
curl http://localhost:4333/tables | jq .

# Verificar conectividade LocalStack
curl -s http://localhost:4566/health || echo "LocalStack offline"
```

### **Dashboard Não Atualiza**
```bash
# Verificar se JMI está respondendo
curl http://localhost:4333/health

# Executar dashboard manualmente
./dashboard.sh
```

## 📁 Estrutura do Projeto

```
poc_bdd/
├── control-m/          # Serviço Control-M
├── jmi/               # Job Manager Integrator (+ Monitoring)
├── jmw/               # Job Manager Worker
├── jmr/               # Job Manager Runner
├── scheduler-plugin/   # Scheduler Plugin
├── spa/               # Scheduler Plugin Adapter
├── spaq/              # Scheduler Plugin Adapter Queue
├── finch-compose.yml  # Configuração dos containers
├── dashboard.sh       # Dashboard em tempo real ✅
├── test-complete-flow.sh # Teste completo ✅
├── set-latency.sh     # Configuração de latência
├── DIAGRAMA_FLUXO_DADOS.md # Diagrama de fluxo de dados
├── DIAGRAMA_SEQUENCIA.md # Diagrama de sequência
└── REVISAR/           # Arquivos desnecessários movidos
```

Para suporte ou dúvidas, consulte os diagramas detalhados e execute os scripts de teste.
