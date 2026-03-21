# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands
- **Build**: `go build ./...`
- **Test**: `go test ./...`
- **Single Test**: `go test -v -run <TestName> ./...`
- **Lint**: `golangci-lint run` (if available) or `go vet ./...`

## Architecture
This project is a Go module designed to add "Like" functionality to applications. As a library/module, it focuses on providing a clean API for managing likes on various entities.

- **Storage**: Likely to support multiple storage backends (SQL, Redis, etc.) via interfaces.
- **API**: Designed to be integrated into other Go services.

---

## POLÍTICA DE OPERAÇÃO INVIOLÁVEL

### 1. SPECIFICATION-FIRST MANDATORY
**NENHUMA funcionalidade pode ser desenvolvida sem especificação prévia.**

- O código NUNCA pode existir antes da especificação
- O código NUNCA pode ser escrito sem especificação conforme
- O fluxo é **SEMPRE**: `ESPECIFICAÇÃO → CÓDIGO`
- **NUNCA** o contrário: código antes de especificação é ESTRITAMENTE PROIBIDO

### 2. CONFORMIDADE OBRIGATÓRIA
**NENHUMA alteração ao código pode ser feita sem conformidade da especificação.**

- Antes de qualquer linha de código: verificar se especificação existe
- Antes de qualquer modificação: verificar se especificação está atualizada
- Especificação em `/specification/` é a ÚNICA fonte de verdade

### 3. RESOLUÇÃO DE AMBIGUIDADE
**Qualquer ambiguidade é decidida EXCLUSIVAMENTE pelo utilizador.**

- Se houver dúvida: perguntar ao utilizador
- Se houver múltiplas opções: apresentar ao utilizador decidir
- Se houver interpretação necessária: validar com utilizador primeiro
- **NUNCA** assumir/decidir em nome do utilizador em casos de ambiguidade

### 4. SEQUÊNCIA OBRIGATÓRIA
```
1. ESPECIFICAÇÃO (spec-orchestrator)
   ↓ [aprovada e conforme]
2. IMPLEMENTAÇÃO (go-elite-developer)
   ↓ [segundo especificação]
3. VERSIONAMENTO (git-flow)
```

**VIOLAÇÕES DESTA POLÍTICA SÃO ESTRITAMENTE PROIBIDAS.**

---

## REGRAS DE COMUNICACAO E DOCUMENTACAO

### 1. DOCUMENTACAO EM INGLES
Toda a documentacao tecnica deve ser escrita em ingles.

- Especificacoes em /specification/: **ingles profissional**
- Ficheiros README.md: **ingles profissional**
- Documentacao de codigo (godoc, comentarios): **ingles**
- **PROIBIDO**: emojis, caracteres ornamentais, ou formatacao decorativa
- **OBRIGATORIO**: linguagem tecnica precisa e profissional

### 2. INTERACAO COM UTILIZADOR EM PORTUGUES DE PORTUGAL
Toda a comunicacao com o utilizador deve ser em **portugues de Portugal (PT-PT)**.

- Formato de assistente tecnico profissional
- Linguagem tecnicamente relevante e precisa
- Objetivo: maximizar a eficiencia da Operacao
- **PROIBIDO**: portugues do Brasil, ingles, ou outras linguas na comunicacao direta
- **OBRIGATORIO**: estrutura clara, direta, focada na execucao

### 3. DECISAO EXCLUSIVA DO UTILIZADOR
Qualquer ambiguidade ou decisao deve ser tomada **exclusivamente pelo utilizador**.

- **NUNCA** executar acoes nao explicitamente solicitadas
- **NUNCA** assumir intencoes do utilizador
- **NUNCA** preencher gaps sem confirmacao previa
- Sempre que houver divida: **perguntar ao utilizador antes de agir**
- Sempre que houver multiplas opcoes: **apresentar ao utilizador decidir**
- Confirmacao obrigatoria antes de qualquer acao nao trivial

---

## Skill Coordination Framework

Este projeto utiliza múltiplos skills especializados que trabalham em conjunto. Abaixo está o framework de coordenação entre eles:

### 1. spec-orchestrator (ESPECIFICAÇÃO)
**Responsabilidade**: Define O QUÊ deve ser implementado
- Cria/atualiza especificações funcionais em `/specification/`
- Cada bloco funcional tem seu próprio arquivo
- README.md serve como índice
- **REGRA CRÍTICA**: Toda funcionalidade deve ter especificação ANTES da implementação

### 2. task-creator (TAREFAS)
**Responsabilidade**: Converte especificações em tarefas estruturadas
- Cria tasks e sprints no roadmap
- Coleta todos os dados obrigatórios (título, tipo, prioridade, etc.)
- Delega persistência ao roadmap-coordinator
- **Trigger**: "criar task", "nova tarefa", "criar sprint"

### 3. roadmap-coordinator (COORDENAÇÃO)
**Responsabilidade**: Orquestra execução via CLI do GoLikeit
- Utiliza CLI `rmp` como fonte da verdade
- Gerencia transições de estado (`rmp task stat`)
- Delega para especialistas
- **REGRA CRÍTICA**: NUNCA implementa diretamente, apenas coordena

### 4. go-elite-developer (IMPLEMENTAÇÃO)
**Responsabilidade**: Implementa código Go idiomático e de alta qualidade
- Implementa features, funções, packages
- Refatora e otimiza código Go
- **Trigger**: "implementar", "criar", "build", "escrever" código Go

### 5. git-flow (VERSIONAMENTO)
**Responsabilidade**: Gerencia branches e fluxo GitFlow
- Cria feature/release/hotfix branches
- Gerencia commits, merges, tags
- **REGRA CRÍTICA**: Mensagens de commit devem incluir O QUÊ mudou E POR QUÊ

### 6. red-team-hacker (SEGURANÇA)
**Responsabilidade**: Auditoria de segurança ofensiva
- Encontra vulnerabilidades em código Go
- SQL injection, command injection, race conditions
- Gera relatórios em markdown

### 7. go-performance-advisor (PERFORMANCE)
**Responsabilidade**: Análise de performance não-intrusiva
- Identifica gargalos
- Recomendações de otimização
- Análise estática e dinâmica

### 8. frontend-design (UI)
**Responsabilidade**: Cria interfaces web de alta qualidade
- Componentes, páginas, aplicações
- Design polido e não-genérico

---

## Fluxos de Trabalho Padrão

### Fluxo 1: Nova Feature
```
especificação (spec-orchestrator)
    ↓
tarefa (task-creator)
    ↓
coordenação (roadmap-coordinator)
    ↓
implementação (go-elite-developer)
    ↓
versionamento (git-flow)
```

### Fluxo 2: Security Audit
```
especificação do escopo
    ↓
tarefa de segurança
    ↓
red-team-hacker (auditoria)
    ↓
go-elite-developer (fixes)
    ↓
git-flow (commits)
```

### Fluxo 3: Performance Review
```
go-performance-advisor (análise)
    ↓
especificação de otimizações
    ↓
go-elite-developer (implementação)
    ↓
git-flow (commits)
```

---

## Regras de Coordenação

1. **Spec First**: Sempre crie/atualize especificação antes de implementar
2. **Task Tracking**: Toda implementação deve estar vinculada a uma task no roadmap
3. **Delegation Chain**: Coordenador delega → Especialista executa → GitFlow versiona
4. **No Direct Implementation**: roadmap-coordinator nunca implementa código diretamente
5. **Git Hygiene**: Toda mudança deve seguir GitFlow com mensagens descritivas
6. **Security Gate**: Features críticas devem passar por red-team-hacker
7. **Performance Gate**: Otimizações devem começar com go-performance-advisor

---

## Comandos de Coordenação Úteis

```bash
# Ver tasks pendentes
rmp task list --status=pending

# Atualizar status de task
rmp task stat <id> in_progress

# Criar feature branch via git-flow
git flow feature start <nome-da-feature>

# Finalizar feature
git flow feature finish <nome-da-feature>
```
