# AGENTS.md

## Padrões de Parsing, Validação e Relato de Erros

Este documento descreve os padrões adotados para parsing, validação e relato de erros no projeto, conforme discutido e implementado.

---

## 1. Parsing de Entradas

- O parser principal (`createParser`) recebe um `ParserConfig` e opcionalmente o nome do arquivo a ser processado.
- Se o nome do arquivo não for informado, assume-se `stdin`.
- Cada diretiva (entry) parseada recebe um campo `meta` que SEMPRE inclui a localização de origem no formato:
  - `meta.location = "$file:linha"` (ex: `meuarquivo.beancount:12`)
  - Se não houver arquivo, `meta.location = "stdin:linha"`
- O campo `meta` também pode conter outros metadados extraídos do YAML ou do próprio arquivo.

### Exemplo de uso do parser
```ts
const parser = createParser(config, "meuarquivo.beancount");
const entries = parser(texto);
// entries[0].meta.location === "meuarquivo.beancount:1"
```

---

## 2. Validação de Transações

- A função de validação (`validateAndFillTransactions`) recebe uma lista de entries parseadas.
- Ela retorna um objeto `{ entries, errors }`:
  - `entries`: lista de entries ajustada (com postings preenchidos quando possível)
  - `errors`: lista de erros encontrados, cada um com:
    - `source`: o campo `meta` do entry (inclui localização)
    - `message`: mensagem de erro
    - `entry`: o próprio entry problemático
- O validador preenche postings faltantes apenas se houver exatamente um sem valor e todas as moedas baterem. Caso contrário, gera erro.

### Exemplo de uso da validação
```ts
const { entries: balancedEntries, errors } = validateAndFillTransactions(entries);
errors.forEach(err => {
  console.error(`Erro em ${err.source.location}: ${err.message}`);
});
```

---

## 3. Padrão de Erros

- Todos os erros de validação seguem a estrutura:
  ```ts
  {
    source: entry.meta, // sempre inclui .location
    message: string,
    entry: BaseEntry
  }
  ```
- Isso permite rastrear precisamente a origem do erro no arquivo de entrada.

---

## 4. Testes

- Os testes devem aceitar tanto o formato `$file:linha` quanto `stdin:linha` em `meta.location`.
- Para validar metadados, use `toMatchObject` ao invés de `toEqual` para ignorar campos extras como `location`.

---

## 5. Extensibilidade

- O parser e o validador são agnósticos ao conteúdo do arquivo, desde que sigam o padrão de entries e meta descrito acima.
- Para novos tipos de diretivas, basta garantir que o campo `meta.location` seja preenchido conforme o padrão.

---

**Este documento deve ser atualizado sempre que houver mudanças nos padrões de parsing, validação ou relato de erros.** 