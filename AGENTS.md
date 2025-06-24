# AGENTS.md

## Guide for Agents and Contributors

This document defines standards and practices for automated agents and humans contributing to this project.

---

### 1. Commit Standards
- Use descriptive commit messages in the format [type]: short description (e.g., `fix: fix transaction validation`).
- Commits should be atomic: each commit addresses a single intent or issue.
- **Para adicionar todos os arquivos modificados no stage, sempre use `git add -A` antes de commitar.**

### 2. Tests
- Every relevant change must be covered by automated tests.
- Always run the tests before committing or opening a PR.
- **CRÍTICO: Enquanto QUALQUER teste automatizado (incluindo snapshots) não passar, o código está considerado QUEBRADO e não deve ser considerado pronto para produção ou merge.**
- The standard command to run the tests is:
  ```sh
  npm run test
  ```
- If a test fails after a change, adjust the code or the test to ensure all tests pass.

### 3. Traceability and Metadata
- Whenever possible, include traceability information (e.g., error location, origin context) in processed data.
- Use fields like `meta.location` to indicate the origin of data or errors.

### 4. Interoperability
- New agents, validators, or parsers must follow the data structure and metadata standards already established in the project.
- Avoid reinventing existing conventions; consult this document and the codebase for standards.

### 5. General Principles
- Prefer clarity and traceability over "magic" or premature optimizations.
- When in doubt, run the tests and follow the commit and metadata standards.
- Document non-trivial decisions in this file.

---

## Diretrizes de Commits (Git Guidelines)

Para manter o histórico do projeto organizado, utilize os tipos de commit convencionais:

- **feat**: Adição de nova funcionalidade (ex: feat: adicionar upload de múltiplos arquivos)
- **fix**: Correção de bug (ex: fix: corrigir erro de parsing de datas)
- **chore**: Tarefas de manutenção, sem impacto direto no código de produção (ex: chore: atualizar dependências)
- **docs**: Mudanças apenas em documentação (ex: docs: atualizar README)
- **refactor**: Refatoração de código, sem alterar comportamento (ex: refactor: mover lógica para função utilitária)
- **style**: Mudanças de formatação, sem alterar lógica (ex: style: padronizar indentação)
- **test**: Adição ou ajuste de testes (ex: test: adicionar teste para parser de tags)
- **perf**: Melhorias de performance (ex: perf: otimizar laço de parsing)
- **build**: Mudanças que afetam o sistema de build (ex: build: ajustar configuração do Vite)
- **ci**: Mudanças em arquivos/configuração de integração contínua (ex: ci: adicionar workflow do GitHub Actions)
- **revert**: Reversão de commit anterior (ex: revert: feat: adicionar upload de múltiplos arquivos)

**Exemplo de mensagem de commit:**

```
feat: permitir seleção de múltiplos arquivos no upload
```

> Sempre escreva mensagens de commit claras e objetivas, preferencialmente em português.

> This document should be kept up to date as the project evolves. 

**Observação:** Não utilizar CSS inline. Sempre prefira componentes Sveltestrap e classes utilitárias. 

## Notas de automação

- O agente (assistente) sempre pode rodar os testes automatizados para validar alterações no código. 