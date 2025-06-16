# POC BDD - Microservices Job Processing Pipeline (VERSÃO CORRIGIDA)

Este projeto implementa uma arquitetura de microserviços para processamento de jobs com integração BDD (Behavior-Driven Development) e **correções completas de integração com LocalStack**.

## 🎯 Status do Projeto

**✅ TOTALMENTE FUNCIONAL** - Todas as correções implementadas e testadas com sucesso.

## 🏗️ Arquitetura

O sistema consiste nos seguintes serviços:

| Serviço | Porta | Função |  |
|---------|-------|--------|--------|
| **Web Dashboard** | 3000 | Modern UX monitoring dashboard |
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
                                    ↑
                            Web Dashboard (Real-time monitoring)
```

**Veja diagramas detalhados:**
- [📊 Diagrama de Fluxo de Dados](./DIAGRAMA_FLUXO_DADOS.md)
- [🔄 Diagrama de Sequência](./DIAGRAMA_SEQUENCIA.md)

## 🚀 Quick Start

### 1. **Configuração Inicial**
```bash
# Autenticação Amazon
mwinit

# Clonar o repositório
git clone git@ssh.gitlab.aws.dev:pcsilva/pod64_local_development.git
cd pod64_local_development

# Instalar e configurar Finch
brew install finch
sudo finch vm init
```

### 2. **Iniciar o Sistema (Opção Recomendada - Web Dashboard)**
```bash
# Iniciar sistema completo com dashboard web moderno
./start-web-dashboard.sh

# Acesse o dashboard em: http://localhost:3000
```

### 3. **Iniciar o Sistema (Opção Tradicional)**
```bash
# Iniciar o sistema completo
finch compose up -d

# Aguardar inicialização (2 minutos)
sleep 120

# Iniciar o dashboard de monitoramento em terminal
./dashboard.sh
```

### 4. **Executar Testes**
```bash
# Em outro terminal, execute o teste completo
./test-complete-flow.sh
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
# Teste completo do fluxo (via Control-M → JMI)
./test-complete-flow.sh

# Teste individual via Control-M
curl -X POST http://localhost:8081/startExecution \
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

### **Web Dashboard Moderno (Recomendado)**
```bash
# Iniciar com dashboard web
./start-web-dashboard.sh

# Acessar dashboard: http://localhost:3000
```

**Funcionalidades do Web Dashboard:**
- ✅ **Interface Moderna**: Design responsivo e acessível
- ✅ **Tempo Real**: Atualizações via WebSocket
- ✅ **Multi-dispositivo**: Funciona em desktop, tablet e mobile
- ✅ **Interativo**: Clique para acessar serviços diretamente
- ✅ **Filtros Avançados**: Busca e filtros para execuções
- ✅ **Modo Escuro**: Suporte automático ao modo escuro
- ✅ **Acessibilidade**: Navegação por teclado e leitores de tela

### **Dashboard em Terminal (Tradicional)**
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

## 🔧 Guia de Instalação Completo

### **Pré-requisitos**
- Conta AWS com acesso configurado
- Git instalado
- Homebrew (para macOS)
- Finch (alternativa ao Docker para macOS)

### **Passo a Passo**

1. **Autenticação na Amazon**
   ```bash
   mwinit
   ```
   Este comando inicia o processo de autenticação com a AWS.

2. **Clonar o Repositório**
   ```bash
   git clone git@ssh.gitlab.aws.dev:pcsilva/pod64_local_development.git
   cd pod64_local_development
   ```

3. **Instalar e Configurar Finch**
   ```bash
   brew install finch
   sudo finch vm init
   ```
   Finch é uma alternativa ao Docker para ambientes macOS.

4. **Iniciar os Serviços**
   ```bash
   finch compose up -d
   ```
   Este comando inicia todos os microserviços em containers.

5. **Aguardar Inicialização**
   ```bash
   sleep 120
   ```
   Aguarde 2 minutos para que todos os serviços estejam prontos.

6. **Iniciar o Dashboard**
   ```bash
   ./dashboard.sh
   ```
   O dashboard mostra o status em tempo real de todos os serviços.

7. **Executar Testes (em outro terminal)**
   ```bash
   ./test-complete-flow.sh
   ```
   Este script executa um fluxo completo de testes para validar a integração.

## 📁 Estrutura do Projeto

```
pod64_local_development/
├── control-m/          # Serviço Control-M
├── jmi/               # Job Manager Integrator (+ Monitoring)
├── jmw/               # Job Manager Worker
├── jmr/               # Job Manager Runner
├── scheduler-plugin/   # Scheduler Plugin
├── spa/               # Scheduler Plugin Adapter
├── spaq/              # Scheduler Plugin Adapter Queue
├── docker-compose.yml  # Configuração dos containers
├── dashboard.sh       # Dashboard em tempo real ✅
├── test-complete-flow.sh # Teste completo ✅
├── set-latency.sh     # Configuração de latência
├── DIAGRAMA_FLUXO_DADOS.md # Diagrama de fluxo de dados
├── DIAGRAMA_SEQUENCIA.md # Diagrama de sequência
└── REVISAO_2/         # Arquivos de revisão
```

Para suporte ou dúvidas, consulte os diagramas detalhados e execute os scripts de teste.
