# Go Gitflow

Status: **OTIMIZADA** ✅

Data original: 2026-03-21
Data otimização: 2026-03-21

## Resumo

Skill para Gitflow seguro em projetos Go, com aprovação obrigatória do utilizador, mensagens de commit técnicas detalhadas, e requisitos estritos de testes antes de push.

## Otimizações Realizadas

### 1. Descrição Melhorada para Triggering

**Antes:**
```yaml
description: Safe Gitflow for Go with mandatory human approval...
```

**Depois:**
```yaml
description: |
  Safe Gitflow for Go...

  TRIGGER when:
  - User mentions "git", "branch", "merge", "commit", "push", "pull"...
  - User asks to "create a feature", "start a feature", "finish a feature"...
  - User mentions "hotfix", "release", "version", "tag", "deploy"...
  ...
  ALWAYS use this skill for ANY git operation in a Go project...
```

**Melhorias:**
- Lista abrangente de triggers (mais de 20 palavras-chave)
- Instrução explícita para "sempre usar" em operações git
- Cobertura de contextos: gitflow, branching, merging, tags, releases

### 2. Table of Contents Adicionada

Nova estrutura com 16 secções organizadas:
1. Approval Policy (MANDATORY)
2. Detailed Commit Standards
3. Feature Start
4. Feature Finish
5. Hotfix Procedure
6. Release Procedure
7. Release Finish
8. **Visual Workflow Diagrams** (NOVO)
9. Rollback Procedures
10. Collaborative Ecosystem
11. Roadmap Integration
12. Error Handling
13. System Instruction
14. Quick Command Reference
15. Basic Git Commands
16. Commands Reference Table

### 3. Diagramas Visuais Adicionados (Secção 8)

**Novos diagramas ASCII:**
- Feature Branch Workflow
- Hotfix Workflow (Urgent)
- Release Workflow
- Complete Gitflow Overview

**Valor:** Permite visualizar rapidamente como os workflows funcionam antes de executar.

### 4. Todas as Diretrizes Preservadas

✅ **Approval Policy (MANDATORY)**
- Nenhuma operação destrutiva sem confirmação explícita
- Todos os testes devem passar antes de push
- Apresentação de Plano de Ação obrigatória
- Mensagens de commit detalhadas

✅ **Git Push Requirements (ZERO EXCEPTIONS)**
- `go test ./...` sem falhas
- Autorização explícita do utilizador
- ABORT imediato se testes falharem

✅ **Commit Standards**
- Proibição de referências a Claude/AI
- Formato: `type(scope): subject`
- Body com explicação técnica detalhada
- Contexto Go específico

✅ **Workflows Gitflow Completos**
- Feature Start/Finish
- Hotfix Procedure
- Release Start/Finish
- Rollback Procedures

✅ **Collaborative Ecosystem**
- Integração com spec-orchestrator
- Coordenação com go-elite-developer
- Validação com go-performance-advisor
- Roadmap integration

✅ **Basic Git Commands**
- /git-status, /git-fetch, /git-pull
- /git-push, /git-commit
- /git-log, /git-diff, /git-branch
- /git-checkout, /git-merge, /git-reset
- /git-stash, /git-rebase, /git-cherry-pick

## Localização

- SKILL.md: `.claude/skills/go-gitflow/SKILL.md`
- STATUS.md: `.claude/skills/go-gitflow/STATUS.md`

## Uso

A skill é automaticamente ativada quando o utilizador menciona qualquer operação git ou gitflow em projetos Go.

### Comandos Principais

| Comando | Descrição |
|---------|-----------|
| `/feature-start <name>` | Inicia feature branch |
| `/feature-finish` | Finaliza e merge feature |
| `/hotfix <version>` | Cria e aplica hotfix |
| `/release-start <version>` | Inicia release |
| `/release-finish <version>` | Finaliza release |
| `/git-push` | Push com validação obrigatória |
| `/git-commit` | Commit com mensagem técnica |
