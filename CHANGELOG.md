# Changelog

Todas as mudanças relevantes deste projeto são documentadas aqui. O formato
segue [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e o projeto
usa [Semantic Versioning](https://semver.org/lang/pt-BR/).

## [0.1.2] - 2026-11-05

### Corrigido

- `pr create` não sobrescreve mais um pull request aberto. Quando já existe PR
  aberto da mesma origem para o mesmo destino, o Bitbucket Cloud responde ao
  POST **atualizando o PR existente** (título, descrição, revisores) em vez de
  recusar; o comando agora confere antes e falha mostrando o PR. Para alterar
  de propósito, `--update` faz um PUT explícito no PR existente.
- `pr create --fill --head <branch>` lê o assunto e o corpo do último commit
  **dessa branch** (local ou `origin/<branch>`), e não do HEAD do clone.
- `pipeline list` mostra a branch de origem nas pipelines de pull request
  (`target.source`), que antes saíam com a coluna vazia.

## [0.1.1] - 2026-11-05

### Corrigido

- `pr list` e `pr view` não quebram mais quando a API devolve `summary` como
  objeto `{raw, markup, html}`. O texto é extraído e `--json` continua emitindo
  uma string.
- `pipeline logs` envia `Accept: text/plain`, segue o redirecionamento 307 para
  o storage sem vazar o `Authorization` e tenta `Accept: */*` quando o host
  responde 406.
- `pipeline logs --step` aceita índice 1-based, nome do step ou UUID; a tabela
  de `pipeline view` agora mostra o índice.

## [0.1.0] - 2026-11-05

Primeira versão funcional. Foco em Bitbucket Cloud, com camada de host
preparada para Data Center.

### Adicionado

- Cliente REST Cloud 2.0 com Basic auth (API token), Bearer (access token) e
  PAT de Data Center, retry com backoff e dicas para 401/403/404.
- Paginação por cursor `next` no Cloud e `nextPageStart` no Data Center.
- Configuração por host, `BK_*`, keyring do SO e fallback `credentials.json`
  com permissão `0600`.
- Descoberta de repositório por `-R`, `BK_REPO` e remotes HTTPS/SSH do git.
- Comandos `auth` (`login`, `logout`, `status`, `token`).
- Comandos `repo` (`list`, `view`, `create`, `clone`, `fork`, `edit`, `delete`).
- Comandos `pr` (`list`, `view`, `create`, `checkout`, `merge`, `decline`,
  `reopen`, `approve`, `unapprove`, `request-changes`, `comment`, `comments`,
  `diff`, `checks`, `commits`).
- Comandos `pipeline` (`list`, `view`, `run`, `stop`, `logs`, `variable`).
- Comandos `workspace` (`list`, `view`, `members`).
- Comandos `snippet` (`list`, `view`, `create`, `clone`, `delete`).
- Comandos `download` (`list`, `upload`, `delete`).
- Comandos `webhook` (`list`, `view`, `create`, `delete`).
- Comandos `branch` (`list`, `restriction list/create/delete`).
- Comandos `ssh-key` e `gpg-key` (`list`, `add`, `delete`).
- `search code` (best-effort, com aviso de depreciação).
- `api` com `--field`, `--raw-field`, `--header`, `--input`, `--paginate`,
  `--include` e `--silent`.
- `browse`, `config`, `alias`, `completion`, `version`.
- Saída `--json`, `--json=campos` e `--template`; TSV sem TTY.
- Aliases de comando e completion para bash, zsh, fish e PowerShell.
- Referência de comandos gerada (`docs/commands.md`) e teste que detecta
  documentação desatualizada.
- Suíte de testes com API falsa, e2e do binário compilado e `-race`.
- CI de lint/testes (`ci.yml`) e build multi-plataforma (`build.yml`) com
  snapshot na `main` e release por tag `v*`, incluindo checksums.

### Limitações conhecidas

- Data Center: somente autenticação, repositórios, branches, browse e `api`.
- Cloud: sem issues/wikis (removidos pela Atlassian em 20/08/2026).
- OAuth web ainda não implementado.
- Code search do Cloud anunciado para depreciação em novembro/2026.
- Sem extensões, pager ou renderização de Markdown.
- `download upload` usa arquivo temporário para multipart, não streaming puro.

[0.1.2]: https://github.com/ClaudioLazaro/bitbucket-cli/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/ClaudioLazaro/bitbucket-cli/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/ClaudioLazaro/bitbucket-cli/releases/tag/v0.1.0
