# Guia de Versionamento Monolítico

Este guia explica como usar o sistema de versionamento automático para microserviços com alinhamento monolítico.

## Como Funciona o Versionamento

O script `version_manager_pr.py` analisa os commits dos últimos 10 commits de cada repositório e determina o tipo de bump baseado nas mensagens de commit seguindo o padrão **Conventional Commits**.

### Tipos de Bump

| Tipo | Quando Ocorre | Exemplo de Commit | Versão Anterior | Nova Versão |
|------|---------------|-------------------|-----------------|-------------|
| **PATCH** | Correções de bugs | `fix: corrige erro de validação` | 1.2.3 | 1.2.4 |
| **MINOR** | Novas funcionalidades | `feat: adiciona nova API` | 1.2.3 | 1.3.0 |
| **MAJOR** | Breaking changes | `feat!: remove API v1` | 1.2.3 | 2.0.0 |

### Padrões de Commit Reconhecidos

#### PATCH (Correções)
```bash
fix: corrige bug de autenticação
fix: resolve problema de timeout
fix: ajusta validação de entrada
```

#### MINOR (Novas funcionalidades)
```bash
feat: adiciona endpoint de métricas
feat: implementa cache distribuído
feat: adiciona suporte a webhooks
```

#### MAJOR (Breaking changes)
```bash
feat!: remove suporte a API v1
feat!: altera estrutura de resposta da API
fix!: corrige comportamento que quebra compatibilidade

# Ou com BREAKING CHANGE no corpo:
feat: nova autenticação

BREAKING CHANGE: remove suporte a tokens antigos
```

## Como Estimular Cada Tipo de Versionamento

### Para PATCH (1.0.0 → 1.0.1)
```bash
# Faça commits de correção
git commit -m "fix: corrige erro de conexão com banco"
git commit -m "fix: resolve problema de encoding"
```

### Para MINOR (1.0.0 → 1.1.0)
```bash
# Faça commits de nova funcionalidade
git commit -m "feat: adiciona endpoint de health check"
git commit -m "feat: implementa retry automático"
```

### Para MAJOR (1.0.0 → 2.0.0)
```bash
# Opção 1: Use ! após o tipo
git commit -m "feat!: remove endpoint deprecated /v1/users"

# Opção 2: Use BREAKING CHANGE no corpo
git commit -m "feat: nova estrutura de autenticação

BREAKING CHANGE: tokens antigos não são mais suportados"

# Opção 3: Use palavra-chave 'major'
git commit -m "refactor: major restructure of API endpoints"
```

## Usando o version_manager_pr.py

### 1. Configuração Inicial

Edite o arquivo `version_manager_pr.py` e configure seus repositórios:

```python
REPOSITORIES: Dict[str, str] = {
    "meu-servico-auth": "./.repos/meu-servico-auth",
    "meu-servico-api": "./.repos/meu-servico-api",
    "meu-servico-worker": "./.repos/meu-servico-worker",
}
```

### 2. Simulação (Dry-run)

Sempre teste primeiro com `--dry-run`:

```bash
python version_manager_pr.py --dry-run
```

**Saída esperada:**
```
[SYNC] Sincronizando repositorios com develop...
[INFO] Inspecionando meu-servico-auth (.repos/meu-servico-auth)
   - Versao atual: 1.2.3
   - Commits (ultimos 10): 5
   - Bump sugerido por este repo: minor

================ Gerindo como monolito ================
Versao-base global: 1.2.3
Bump global: minor
=> Nova versao global: 1.3.0
======================================================
```

### 3. Execução Real

Quando estiver satisfeito com a simulação:

```bash
python version_manager_pr.py
```

### 4. Variáveis de Ambiente (Opcional)

Para criação automática de PRs no GitHub:

```bash
export GITHUB_TOKEN="seu_token_aqui"
python version_manager_pr.py
```

## Lógica de Versionamento Global

O script segue esta lógica para determinar a versão global:

1. **Analisa todos os repositórios** e determina o bump individual
2. **Prioridade de bump**: MAJOR > MINOR > PATCH
3. **Versão base**: Maior versão atual entre todos os repositórios
4. **Versão final**: Aplica o bump de maior prioridade na versão base

### ⚠️ Comportamento Importante: Commits Já Processados

O script **sempre considera os últimos 10 commits** de cada repositório, incluindo commits que já foram processados em releases anteriores.

**Cenário comum:**
1. Microserviço A tem commit `feat!:` (major) → Release v2.0.0
2. Microserviço B recebe commit `feat:` (minor)
3. Próxima execução ainda vê o `feat!:` do microserviço A → Sugere major novamente

**Isso é esperado** no modelo monolítico porque:
- ✅ Garante que todos os serviços sempre tenham a mesma versão
- ✅ Evita incompatibilidades entre versões diferentes
- ✅ Simplifica o gerenciamento de dependências

### Exemplo Prático

**Estado atual:**
- `servico-auth`: v1.2.0 (commits: 2 fix)
- `servico-api`: v1.1.5 (commits: 1 feat)  
- `servico-worker`: v1.2.1 (commits: 1 feat!)

**Resultado:**
- Bump individual: patch, minor, major
- **Bump global**: major (maior prioridade)
- **Versão base**: 1.2.1 (maior versão atual)
- **Nova versão**: 2.0.0 (major bump)

## Estratégias de Versionamento

### Opção 1: Monolítico Puro (Atual)
**Comportamento:** Todos os serviços sempre têm a mesma versão
```bash
# Primeira execução: microserviço A com feat!
python version_manager_pr.py  # → v2.0.0 para todos

# Segunda execução: microserviço B com feat
python version_manager_pr.py  # → v3.0.0 para todos (ainda vê o feat! do A)
```

### Opção 2: Baseado em Tags (Alternativa)
**Comportamento:** Considera apenas commits desde a última tag
- Requer modificação do script para usar `git describe --tags`
- Commits já "taggeados" não influenciam próximas releases
- Mais complexo, mas evita re-processamento

### Opção 3: Híbrido
**Comportamento:** Permite releases independentes com alinhamento periódico
- Releases individuais para mudanças menores
- Alinhamento monolítico para breaking changes

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
# Após aprovação do PR/MR
git checkout develop
git merge feature/nova-funcionalidade
```

### 3. Release
```bash
# Execute o versionamento
python version_manager_pr.py --dry-run  # Simule primeiro
python version_manager_pr.py            # Execute
```

### 4. Aprovação dos MRs
- Revise os MRs criados automaticamente
- Aprove e faça merge para develop
- As tags serão criadas automaticamente

## Troubleshooting

### "Commits (ultimos 10): 0"
- O repositório não tem commits ou a branch develop não existe
- Verifique se há commits na branch develop

### "Bump sugerido: patch" mesmo com feat
- Verifique se a mensagem de commit segue o padrão correto
- Use exatamente `feat:` no início da mensagem

### Erro de sincronização
- Verifique se as branches develop existem
- Confirme que os repositórios têm remotes configurados

### Versão não mudou
- Se todos os commits são irrelevantes (docs, chore), a versão não muda
- Apenas commits fix, feat, e breaking changes afetam a versão

### "Sempre sugere major mesmo após release"
- **Comportamento esperado** no modelo monolítico atual
- O script sempre analisa os últimos 10 commits, incluindo os já processados
- **Solução**: Execute releases em lotes ou considere implementar versionamento baseado em tags

### Como evitar re-processamento de commits
1. **Agrupe mudanças**: Faça releases menos frequentes com múltiplas mudanças
2. **Use branches de release**: Crie branches específicas para cada release
3. **Considere tags**: Modifique o script para considerar apenas commits desde a última tag
