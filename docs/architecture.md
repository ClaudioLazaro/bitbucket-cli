# Arquitetura

## Objetivos

- Uma CLI fina e previsível sobre a API do Bitbucket, com a ergonomia do `gh`:
  `-R`, descoberta pelo remote, `--json`, aliases, completion e códigos de saída
  estáveis.
- Cloud primeiro, com a camada de host preparada para Data Center sem fingir
  suporte: endpoints não mapeados falham com mensagem clara.
- Nenhum token em arquivo de configuração; nenhum teste com credencial real.

## Não-objetivos da 0.1

- Paridade 1:1 com o `gh` (issues no Cloud não existem mais).
- Reimplementar o `git`; o `bk` chama o binário `git` para clone, fetch e
  checkout.
- Renderizar Markdown, paginar saída ou rodar extensões de terceiros.

## Camadas

```
cmd/bk/main.go
  -> pkg/cmd/root (Cobra, --hostname, aliases)
      -> pkg/cmd/<grupo> (repos, prs, pipelines, ...)
          -> internal/cmdutil.Factory.App()
              -> internal/config (config.yml + env)
              -> internal/auth  (keyring / credentials.json)
              -> internal/bitbucket.Client (HTTP, auth, retry, paginação)
                  -> api.bitbucket.org/2.0  ou  host/rest/api/1.0
          -> internal/git (remote, branch, clone/fetch/checkout)
          -> internal/iostreams + tableprinter (TTY, cor, tabelas)
```

| Pacote | Responsabilidade |
| --- | --- |
| `cmd/bk` | processo, aliases, impressão de erro, exit code |
| `pkg/cmd/root` | árvore Cobra e flags globais |
| `pkg/cmd/<grupo>` | parsing de flags, orquestração, apresentação |
| `internal/cmdutil` | `Factory`, `App`, `--json`, prompts, `ExitError` |
| `internal/config` | schema, normalização de host, merge com env |
| `internal/auth` | keyring, fallback `0600`, precedência |
| `internal/bitbucket` | caminhos por host, tipos, paginação, erros |
| `internal/git` | remotes, branch atual, clone/fetch/checkout |
| `internal/tableprinter` | tabela alinhada em TTY, TSV fora de TTY |
| `internal/iostreams` | streams, detecção de TTY, cores |
| `internal/text` | truncamento, datas relativas, indentação |

## Ciclo de uma execução

1. `main` carrega aliases do `config.yml` e expande o primeiro argumento.
2. O comando raiz é executado; `PersistentPreRunE` aplica `--hostname` em
   `BK_HOST`.
3. O comando chama `f.App()`:
   - carrega `config.yml` e resolve host/kind/API URL (env vence config);
   - busca o token em `BK_TOKEN`, keyring e `credentials.json`;
   - valida que Basic tem usuário e cria o `bitbucket.Client`.
4. O comando resolve o repositório (`-R`, `BK_REPO`, remote do git).
5. O cliente monta o caminho conforme o kind, assina a requisição, aplica
   retry e decodifica o JSON em tipos.
6. O comando imprime tabela, JSON/template ou conteúdo bruto (diff, log, `api`).

## Abstração Cloud x Data Center

`Client.Kind` é `cloud` ou `dc`. A diferença fica concentrada em:

- `RepoPath`: `/repositories/{workspace}/{slug}` vs
  `/projects/{key}/repos/{slug}`.
- Coleções: `/pullrequests` vs `/pull-requests`, `/refs/branches` vs
  `/branches`.
- Paginação: cursor `next` absoluto vs `nextPageStart` com `start`.
- JSON: snake_case vs camelCase (tipos tolerantes cobrem os campos usados).

Operações ainda não mapeadas retornam `bitbucket.UnsupportedError`, que vira a
mensagem "operation not supported on Data Center hosts yet". O `bk api` cobre
esses casos manualmente.

Não há interface por endpoint: os comandos compartilham os mesmos métodos
tipados e os guards de host. Isso evita uma abstração prematura que esconderia
diferenças de payload.

## Paginação

`GetAll[T]` segue páginas até `limit` (0 = todas):

- Cloud: usa `next` absoluto quando presente; senão encerra.
- Data Center: usa `nextPageStart` até `isLastPage`.
- Tamanho de página: até 100; segurança de 1000 páginas por chamada.

Os comandos expõem `--limit` com default 30 (ou 100 em listas de comentários e
status), e `0` para todas.

## Erros e retry

- `HTTPError` guarda status, método, URL, mensagem/detalhe e request id.
- O envelope do Cloud (`{"error":{...}}`) e o do Data Center (`{"errors":[...]}`)
  são lidos; se não houver envelope, um trecho do corpo é preservado.
- `Hint()` traduz 401/403/404 em ação (token, escopo, recurso).
- Retry com backoff em 429/502/503/504, respeitando `Retry-After` (máx. 3
  tentativas).
- Conteúdo bruto (`pr diff`, `pipeline logs`, arquivos de snippet) usa
  `GetRaw`: envia `Accept: text/plain`, segue 307 sem repassar `Authorization`
  e tenta `Accept: */*` quando o host responde 406.
- `ExitError` permite código de saída específico; hoje usado por
  `bk pr checks` (1 quando há `FAILED`).

## Credenciais

- `BK_TOKEN` tem precedência e não é persistido.
- Keyring do SO é o padrão; o arquivo `credentials.json` é fallback com `0600`.
- O `config.yml` guarda apenas metadados (usuário, método, defaults).
- `BK_API_URL` é override efetivo e **não** é gravado no config pelo login.

## Resolução de repositório

`-R OWNER/REPO` (ou posicional) > `BK_REPO` > remotes do git. O remote `origin`
é preferido; os demais são testados na ordem. URLs aceitas incluem HTTPS,
`git@host:owner/repo`, `ssh://`, `/scm/PROJ/repo` e `/projects/PROJ/repos/repo`.

## Saída

- TTY: tabela alinhada com cabeçalho, truncada pela largura do terminal.
- Não-TTY: TSV sem cabeçalho, para scripts.
- `--json`, `--json=campos` e `--template` usam a mesma representação (marshal
  do tipo Go para JSON).
- Conteúdo bruto (`pr diff`, `pipeline logs`, `api` textual) vai direto para
  stdout.

## Entrega

- `.github/workflows/ci.yml` roda formatação, `go vet` e testes em push/PR.
- `.github/workflows/build.yml` compila com `CGO_ENABLED=0` para
  `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64` e
  `windows/amd64`, empacota em `.tar.gz`/`.zip` e publica um artifact por alvo.
- Em push na `main`, o job `snapshot` junta os alvos em `bk-snapshot` com
  `checksums.txt`.
- Em tag `v*`, o job `release` cria (ou atualiza) a release com todos os
  binários e checksums via `gh release`.
- Versão, commit e data entram por `-ldflags` em
  `internal/build`.

## Testes

- Unitários: cliente/paginação/erros/retry, config/normalização, git/remotes,
  aliases, multipart de snippet.
- Integração: comandos reais via `pkg/cmd/root` contra `httptest`, verificando
  método, caminho, query, header de auth e corpo JSON.
- E2E: `test/e2e_test.go` compila `bin/bk` e executa contra a API falsa,
  incluindo exit code.
- Docs: um teste falha se um comando novo não estiver em `docs/commands.md`
  (regenere com `make docs`).
- Nenhum teste acessa o Bitbucket real. Smoke com credencial é um passo
  autorizado à parte.

## Como adicionar um comando

1. Método tipado em `internal/bitbucket`, com guarda de host e paginação via
   `GetAll`.
2. Comando Cobra em `pkg/cmd/<grupo>` usando `Factory`, `ResolveRepo`,
   `AddOutputFlags` e `tableprinter`.
3. Registro em `pkg/cmd/root/root.go`.
4. Teste de integração em `test/integration_test.go` (texto + ao menos um
   `--json`).
5. `make docs`; atualize `README.md`, `docs/api-coverage.md` e
   `docs/roadmap.md` se a cobertura mudou.

## Decisões registradas

- **Não fazer fork do `gh`**: o `gh` é acoplado a endpoints e esquemas do
  GitHub; um fork criaria um diff permanente e frágil.
- **Cloud primeiro**: é o alvo com mais usuários e a API 2.0 tem OpenAPI
  estável; Data Center entra por adapters.
- **Sem comando `issue` no Cloud**: a Atlassian removeu Issues e Wikis em
  20/08/2026.
- **Sem app passwords**: removidas em julho/2026.
- **`bk api` desde a 0.1**: cobre qualquer endpoint ainda não tipado e reduz a
  pressão por paridade imediata.
- **`pr create` consulta PR aberto antes do POST**: o Bitbucket Cloud responde
  a um POST duplicado atualizando o PR em vez de recusar. O comando falha por
  padrão e `--update` faz o PUT explícito.
- **Pipelines de PR**: `target.ref_name` é nulo; a branch de origem está em
  `target.source` e a de destino em `target.destination`.
