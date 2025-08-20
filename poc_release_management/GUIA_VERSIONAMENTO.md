# Guia de Versionamento Monolitico

Este guia explica como usar o sistema de versionamento automatico para microservicos com alinhamento monolitico.

## Como Funciona o Versionamento

O script `version_manager_pr.py` analisa commits desde a ultima tag de cada repositorio e determina o tipo de bump baseado nas mensagens de commit seguindo o padrao **Conventional Commits**.

### Tipos de Bump

| Tipo | Quando Ocorre | Exemplo de Commit | Versao Anterior | Nova Versao |
|------|---------------|-------------------|-----------------|-------------|
| **PATCH** | Correcoes de bugs | `fix: corrige erro de validacao` | 1.2.3 | 1.2.4 |
| **MINOR** | Novas funcionalidades | `feat: adiciona nova API` | 1.2.3 | 1.3.0 |
| **MAJOR** | Breaking changes | `feat!: remove API v1` | 1.2.3 | 2.0.0 |

### Padroes de Commit Reconhecidos

#### PATCH (Correcoes)
```bash
fix: corrige bug de autenticacao
fix: resolve problema de timeout
fix: ajusta validacao de entrada
```

#### MINOR (Novas funcionalidades)
```bash
feat: adiciona endpoint de metricas
feat: implementa cache distribuido
feat: adiciona suporte a webhooks
```

#### MAJOR (Breaking changes)
```bash
feat!: remove suporte a API v1
feat!: altera estrutura de resposta da API
fix!: corrige comportamento que quebra compatibilidade

# Ou com BREAKING CHANGE no corpo:
feat: nova autenticacao

BREAKING CHANGE: remove suporte a tokens antigos
```

## Como Estimular Cada Tipo de Versionamento

### Para PATCH (1.0.0 → 1.0.1)
```bash
# Faca commits de correcao
git commit -m "fix: corrige erro de conexao com banco"
git commit -m "fix: resolve problema de encoding"
```

### Para MINOR (1.0.0 → 1.1.0)
```bash
# Faca commits de nova funcionalidade
git commit -m "feat: adiciona endpoint de health check"
git commit -m "feat: implementa retry automatico"
```

### Para MAJOR (1.0.0 → 2.0.0)
```bash
# Opcao 1: Use ! apos o tipo
git commit -m "feat!: remove endpoint deprecated /v1/users"

# Opcao 2: Use BREAKING CHANGE no corpo
git commit -m "feat: nova estrutura de autenticacao

BREAKING CHANGE: tokens antigos nao sao mais suportados"

# Opcao 3: Use palavra-chave 'major'
git commit -m "refactor: major restructure of API endpoints"
```

## Usando o version_manager_pr.py

### 1. Configuracao Inicial

Edite o arquivo `version_manager_pr.py` e configure seus repositorios:

```python
REPOSITORIES: Dict[str, str] = {
    "meu-servico-auth": "./.repos/meu-servico-auth",
    "meu-servico-api": "./.repos/meu-servico-api",
    "meu-servico-worker": "./.repos/meu-servico-worker",
}
```

### 2. Simulacao (Dry-run)

Sempre teste primeiro com `--dry-run`:

```bash
python version_manager_pr.py --dry-run
```

**Saida esperada:**
```
[SYNC] Sincronizando repositorios com develop...
[INFO] Inspecionando meu-servico-auth (.repos/meu-servico-auth)
   - Versao atual: 1.2.3
   - Commits desde ultima tag: 2
   - Bump sugerido por este repo: minor

================ Gerindo como monolito ================
Versao-base global: 1.2.3
Bump global: minor
=> Nova versao global: 1.3.0
======================================================
```

### 3. Execucao Real

Quando estiver satisfeito com a simulacao:

```bash
python version_manager_pr.py
```

### 4. Variaveis de Ambiente (Opcional)

Para criacao automatica de PRs no GitHub:

```bash
export GITHUB_TOKEN="seu_token_aqui"
python version_manager_pr.py
```

## Logica de Versionamento Global

O script segue esta logica para determinar a versao global:

1. **Analisa todos os repositorios** e determina o bump individual
2. **Prioridade de bump**: MAJOR > MINOR > PATCH
3. **Versao base**: Maior versao atual entre todos os repositorios
4. **Versao final**: Aplica o bump de maior prioridade na versao base

### ✅ Comportamento Inteligente: Evita Reprocessamento

O script **considera apenas commits desde a ultima tag** de cada repositorio, evitando reprocessamento de commits ja lancados.

**Cenario otimizado:**
1. Microservico A tem commit `feat!:` (major) → Release v2.0.0 → Tag criada
2. Microservico B recebe commit `feat:` (minor)
3. Proxima execucao ve apenas o novo `feat:` do microservico B → Sugere minor

**Vantagens:**
- ✅ Evita bumps desnecessarios por commits ja processados
- ✅ Permite releases incrementais mais precisos
- ✅ Mantem alinhamento monolitico quando necessario

### Exemplo Pratico

**Estado atual:**
- `servico-auth`: v1.2.0 (commits desde tag: 2 fix)
- `servico-api`: v1.1.5 (commits desde tag: 1 feat)  
- `servico-worker`: v1.2.1 (commits desde tag: 1 feat!)

**Resultado:**
- Bump individual: patch, minor, major
- **Bump global**: major (maior prioridade)
- **Versao base**: 1.2.1 (maior versao atual)
- **Nova versao**: 2.0.0 (major bump)

## Fluxo de Trabalho Recomendado

### 1. Desenvolvimento
```bash
# Trabalhe em feature branches
git checkout -b feature/nova-funcionalidade
git commit -m "feat: adiciona nova funcionalidade"
git push origin feature/nova-funcionalidade
```

### 2. Merge para Develop
```bash
# Apos aprovacao do PR/MR
git checkout develop
git merge feature/nova-funcionalidade
```

### 3. Release
```bash
# Execute o versionamento
python version_manager_pr.py --dry-run  # Simule primeiro
python version_manager_pr.py            # Execute
```

### 4. Aprovacao dos MRs
- Revise os MRs criados automaticamente
- Aprove e faca merge para develop
- As tags serao criadas e enviadas automaticamente

## Troubleshooting

### "Commits desde ultima tag: 0"
- O repositorio nao tem commits novos desde a ultima tag
- Verifique se ha commits na branch develop

### "Bump sugerido: patch" mesmo com feat
- Verifique se a mensagem de commit segue o padrao correto
- Use exatamente `feat:` no inicio da mensagem

### Erro de sincronizacao
- Verifique se as branches develop existem
- Confirme que os repositorios tem remotes configurados

### Versao nao mudou
- Se todos os commits sao irrelevantes (docs, chore), a versao nao muda
- Apenas commits fix, feat, e breaking changes afetam a versao

### "Sempre sugere major mesmo apos release"
- **Comportamento corrigido** na versao atual
- O script agora considera apenas commits desde a ultima tag
- Tags sao criadas automaticamente para marcar commits processados
