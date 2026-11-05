# bk - Bitbucket CLI

[![CI](https://github.com/ClaudioLazaro/bitbucket-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/ClaudioLazaro/bitbucket-cli/actions/workflows/ci.yml)
[![Build](https://github.com/ClaudioLazaro/bitbucket-cli/actions/workflows/build.yml/badge.svg)](https://github.com/ClaudioLazaro/bitbucket-cli/actions/workflows/build.yml)

`bk` é uma interface de linha de comando para o Bitbucket, escrita em Go e
inspirada na ergonomia do GitHub CLI (`gh`): `-R OWNER/REPO`, detecção do
repositório pelo remote do git, saída `--json`, aliases, completion e
autenticação por host.

O foco da versão 0.1 é **Bitbucket Cloud**. A camada de API já é plugável e o
Data Center funciona para autenticação, repositórios e o escape hatch `bk api`;
os demais comandos retornam um erro explícito de "não suportado" nesse tipo de
host.

Este projeto é independente do `cli/cli` e não é um fork do `gh`. O `gh` é
acoplado aos endpoints e esquemas do GitHub; o `bk` tem cliente HTTP, tipos e
paginação próprios para a API 2.0 do Bitbucket.

## Instalação

Requisitos: Go 1.26 ou superior e `git`.

```sh
git clone https://github.com/ClaudioLazaro/bitbucket-cli.git
cd bitbucket-cli
make build
./bin/bk version
```

Ou, sem clonar:

```sh
go install github.com/ClaudioLazaro/bitbucket-cli/cmd/bk@latest
```

O `make build` injeta versão, commit e data via `-ldflags`.

### Build e releases

- `make build` gera `bin/bk` localmente.
- `.github/workflows/ci.yml` roda `gofmt`, `go vet` e a suíte de testes.
- `.github/workflows/build.yml` compila para `linux/amd64`, `linux/arm64`,
  `darwin/amd64`, `darwin/arm64` e `windows/amd64`, empacota em `.tar.gz`/`.zip`
  e publica:
  - `bk-snapshot` como artifact em cada push na `main`;
  - uma release com todos os binários e `checksums.txt` quando uma tag `v*` é
    enviada.

Para baixar o snapshot mais recente:

```sh
gh run download --repo ClaudioLazaro/bitbucket-cli --name bk-snapshot
```

Para publicar uma release:

```sh
git tag v0.1.0
git push origin v0.1.0
```

## Autenticação

O Bitbucket Cloud **removeu as app passwords em julho de 2026**. Hoje existem
três caminhos:

### API token (recomendado para pessoas e scripts)

1. Abra Atlassian account -> Security -> **Create API token with scopes**.
2. Escolha o app **Bitbucket** e conceda os escopos necessários (por exemplo
   `Account: Read`, `Repositories: Read/Write`, `Pull requests: Read/Write`,
   `Pipelines`, `Webhooks`).
3. Faça login usando o **e-mail da conta Atlassian** e o token:

```sh
bk auth login --hostname bitbucket.org --username voce@empresa.com --with-token < token.txt
bk auth status
```

### Access token (repositório, projeto ou workspace)

Tokens de acesso pertencem ao recurso, não a um usuário, e usam Bearer:

```sh
bk auth login --hostname bitbucket.org --access-token --with-token < access-token.txt
```

### Bitbucket Data Center (PAT)

```sh
bk auth login --hostname https://bitbucket.example.com --with-token < pat.txt
```

O padrão no Data Center é Bearer. Se você informar `--username`, o `bk` usa
Basic auth; também é possível forçar com `--auth-method basic|bearer`.

### Onde a credencial fica

O token vai para o keyring do sistema (Secret Service/KWallet, macOS Keychain,
Windows Credential Manager). Sem keyring, o `bk` grava
`$XDG_CONFIG_HOME/bk/credentials.json` com permissão `0600` e avisa no login.
Para forçar o arquivo:

```sh
BK_FORCE_FILE_STORE=1 bk auth login ...
```

Nenhum token é gravado no `config.yml`.

### CI e containers

```sh
export BK_HOST=https://bitbucket.example.com
export BK_TOKEN="$BITBUCKET_PAT"
export BK_PROJECT=ABC
export BK_REPO=meu-servico
bk pr list
```

Cloud com API token:

```sh
export BK_HOST=bitbucket.org
export BK_TOKEN="$API_TOKEN"
export BK_USERNAME=voce@empresa.com
export BK_AUTH_METHOD=basic
```

## Uso rápido

```sh
# Repositórios
bk repo list minha-workspace
bk repo view -R minha-workspace/api
bk repo create minha-workspace/novo-servico --description "Serviço novo" --clone
bk repo clone minha-workspace/api ./api

# Pull requests
bk pr list --state open --author @me
bk pr view 42 --comments
bk pr create --title "feat: cache" --body "Adiciona cache" --reviewer colega
bk pr checkout 42
bk pr checks 42
bk pr merge 42 --strategy squash --close-source-branch

# Pipelines
bk pipeline list --branch main
bk pipeline run --branch main --var DEPLOY_ENV=staging
bk pipeline logs 128 --step 1
bk pipeline variable set API_URL https://api.example.com

# Escape hatch para qualquer endpoint
bk api /repositories/minha-workspace/api/pullrequests -f state=OPEN
```

## Documentação

| Documento | Conteúdo |
| --- | --- |
| [docs/commands.md](docs/commands.md) | referência de todos os comandos e flags (gerada por `make docs`) |
| [docs/configuration.md](docs/configuration.md) | `config.yml`, credenciais, precedência, env e defaults |
| [docs/troubleshooting.md](docs/troubleshooting.md) | 401/403/404, keyring, Data Center, proxy e depuração |
| [docs/architecture.md](docs/architecture.md) | camadas, fluxo de request, paginação, erros e testes |
| [docs/api-coverage.md](docs/api-coverage.md) | comando -> endpoint, cobertura Cloud/DC e lacunas |
| [docs/roadmap.md](docs/roadmap.md) | próximas fases e fora de escopo |
| [CONTRIBUTING.md](CONTRIBUTING.md) | build, testes, estilo e como adicionar um comando |
| [CHANGELOG.md](CHANGELOG.md) | mudanças por versão |
| [SECURITY.md](SECURITY.md) | reporte privado de vulnerabilidades |

A referência de comandos é gerada a partir da árvore Cobra:

```sh
make docs
```

Um teste falha quando um comando novo não aparece nela.

## Comandos

| Grupo | Comandos |
| --- | --- |
| `auth` | `login`, `logout`, `status`, `token` |
| `repo` | `list`, `view`, `create`, `clone`, `fork`, `edit`, `delete` |
| `pr` | `list`, `view`, `create`, `checkout`, `merge`, `decline`, `reopen`, `approve`, `unapprove`, `request-changes`, `comment`, `comments`, `diff`, `checks`, `commits` |
| `pipeline` | `list`, `view`, `run`, `stop`, `logs`, `variable list/set/delete` |
| `workspace` | `list`, `view`, `members` |
| `snippet` | `list`, `view`, `create`, `clone`, `delete` |
| `download` | `list`, `upload`, `delete` |
| `webhook` | `list`, `view`, `create`, `delete` |
| `branch` | `list`, `restriction list/create/delete` |
| `ssh-key` | `list`, `add`, `delete` |
| `gpg-key` | `list`, `add`, `delete` |
| `search` | `code` |
| `api` | requisição REST crua, com `--paginate` |
| `browse` | abre repositório, arquivo, PR, commit ou pipeline |
| `config` | `list`, `get`, `set` |
| `alias` | `list`, `set`, `delete` |
| `completion` | `bash`, `zsh`, `fish`, `powershell` |
| `version` | versão, commit, data e runtime |

Todos os comandos que aceitam um repositório aceitam `-R OWNER/REPO` (ou
`HOST/OWNER/REPO`). Dentro de um clone, o repositório é inferido do remote
`origin` (ou de outro remote que aponte para o host configurado).

## Saída para scripts

- `--json` imprime o objeto/lista completo.
- `--json=slug,is_private` filtra campos de topo.
- `--template '{{.slug}}'` renderiza um template Go sobre o JSON.
- Sem TTY, tabelas viram linhas separadas por tabulação, sem cabeçalho.
- `bk pr checks` sai com código 1 se algum status estiver `FAILED` (como o `gh`).

```sh
bk repo list minha-workspace --json | jq -r '.[].slug'
bk pr list -R minha-workspace/api --json=id,title,state
```

## Configuração e aliases

Arquivo: `$XDG_CONFIG_HOME/bk/config.yml` (ou `$BK_CONFIG_DIR/config.yml`).

```yaml
version: 1
active_host: bitbucket.org
hosts:
  bitbucket.org:
    kind: cloud
    user: voce@empresa.com
    auth_method: basic
    workspace: minha-workspace
    api_url: https://api.bitbucket.org/2.0
aliases:
  pl: pr list
  pv: pr view
```

```sh
bk config list
bk config set workspace minha-workspace
bk config get workspace
bk alias set pl "pr list"
bk alias list
```

## Variáveis de ambiente

| Variável | Efeito |
| --- | --- |
| `BK_HOST` | Host ativo (nome ou URL, Cloud ou Data Center) |
| `BK_TOKEN` | Token da requisição; tem precedência sobre keyring/arquivo |
| `BK_USERNAME` | Usuário do Basic auth (e-mail Atlassian no Cloud) |
| `BK_AUTH_METHOD` | `basic` ou `bearer` |
| `BK_API_URL` | Sobrescreve a base REST (útil para proxy, testes e GHES-like) |
| `BK_WORKSPACE` | Workspace padrão no Cloud |
| `BK_PROJECT` | Projeto padrão no Data Center |
| `BK_REPO` | Repositório padrão (`OWNER/REPO`) |
| `BK_CONFIG_DIR` | Diretório de configuração |
| `BK_FORCE_FILE_STORE` | Usa o arquivo `0600` em vez do keyring |
| `BK_HTTP_DEBUG` | Imprime método, URL e status de cada requisição |

## Bitbucket Cloud x Data Center

| Área | Cloud | Data Center |
| --- | --- | --- |
| Auth (API token, access token, PAT) | sim | sim |
| `repo list/view/create/clone` + `branch list` | sim | sim |
| `pr`, `pipeline`, `snippet`, `download`, `webhook`, `branch restriction`, `ssh-key`, `gpg-key`, `search` | sim | ainda não |
| `api`, `browse`, `config`, `alias`, `completion` | sim | `api`/`browse`/configuração sim |

Comandos indisponíveis no Data Center retornam um erro claro em vez de chamar
um endpoint errado.

## Limitações conhecidas

- **Issues e Wikis do Cloud foram removidos pela Atlassian em 20/08/2026.**
  Não existe comando `issue` para Cloud; use Jira/Confluence. O Data Center
  ainda tem esses recursos na API e eles estão no roadmap.
- **Data Center**: pull requests, pipelines, issues e demais grupos ainda não
  foram mapeados. O `bk api` cobre esses endpoints manualmente.
- **OAuth web** (`bk auth login --web`) ainda não está implementado. Use API
  token, access token ou PAT.
- **Code search** do Cloud foi anunciado para depreciação em novembro/2026; o
  comando avisa no stderr.
- `bk download upload` monta o multipart em arquivo temporário para enviar
  `Content-Length`; não é streaming puro de memória.
- Sem extensões/plugins, sem renderização Markdown e sem pager.
- O texto de ajuda é em inglês (como no `gh`); esta documentação é em português.

## Arquitetura

```
cmd/bk/                 entrada do processo, aliases e códigos de saída
internal/bitbucket/     cliente REST, paginação, erros, tipos e endpoints
internal/config/        config.yml + resolução de host/env
internal/auth/          keyring e fallback de credenciais
internal/cmdutil/       factory, output --json, prompts, erros
internal/git/           remotes, branch atual, clone/fetch/checkout
internal/iostreams/     streams, TTY, cor
internal/tableprinter/  tabelas alinhadas ou TSV
pkg/cmd/<grupo>/        comandos Cobra por família
test/                   integração com API falsa + e2e do binário
```

Decisões relevantes:

- O cliente monta caminhos por tipo de host (`/repositories/...` no Cloud,
  `/projects/{key}/repos/...` no Data Center).
- Paginação trata o cursor `next` do Cloud e `nextPageStart` do Data Center.
- Erros 401/403/404 trazem dica acionável e o `x-request-id` quando presente.
- Requisições têm retry com backoff para 429/502/503/504 e respeitam
  `Retry-After`.

## Testes

```sh
make test
make lint
```

A suíte cobre configuração, normalização de hosts, autenticação Basic/Bearer,
paginação dos dois tipos de host, retry, erros, parsing de remotes, aliases,
armazenamento de credenciais e comandos de repo, PR, pipeline, API, auth,
browse e config. `test/e2e_test.go` compila o binário de verdade e o executa
contra uma API Bitbucket falsa, incluindo o código de saída 1 de
`pr checks` com status `FAILED`.

Os testes **não usam credenciais reais** nem acessam o Bitbucket.

## Roadmap

Veja [docs/roadmap.md](docs/roadmap.md). Próximas fases: Data Center completo
(PRs, pipelines, issues), OAuth web, extensões, deployments, merge queues,
reports, projetos, branch model, schedules/caches/runners e um servidor MCP.

## Licença

MIT. Veja [LICENSE](LICENSE).
