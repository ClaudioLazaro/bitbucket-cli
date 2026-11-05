# Roadmap do bk

O objetivo é cobrir o máximo possível da API do Bitbucket sem inventar
comportamento: cada comando precisa de endpoint documentado, payload verificado
e teste contra uma API falsa antes de ser anunciado.

## Fase 1 - fundação (concluída)

- Cliente REST Cloud 2.0 com Basic, Bearer, retry e paginação por cursor.
- Camada de caminhos para Data Center e paginação por `nextPageStart`.
- Configuração por host, keyring com fallback `0600`, variáveis `BK_*`.
- Detecção de repositório por `-R`, `BK_REPO` e remotes do git.
- `auth`, `repo`, `pr`, `pipeline`, `workspace`, `snippet`, `download`,
  `webhook`, `branch` (com restrições), `ssh-key`, `gpg-key`, `search`,
  `api`, `browse`, `config`, `alias`, `completion`, `version`.
- Saída `--json`, `--json=fields`, `--template`; TSV sem TTY.
- Testes de integração e e2e com binário compilado.

## Fase 2 - Data Center completo

Mapear a API 1.0 (JSON camelCase, `start/limit`, projetos e permissões):

- Repositórios: edição, fork, tags, commits, arquivos, raw, diff.
- Pull requests: `/pull-requests`, atividades, comentários, diff, merge com
  `version`, decline, participantes, tarefas.
- Pipelines do Data Center (quando habilitado) e builds.
- Issues e Wikis (ainda existem no Data Center).
- Usuários e grupos, permissões de projeto/repositório.
- Default reviewers e branch model.

Cada operação deve passar por um adapter que converte o JSON do Data Center
para os tipos internos, mantendo os comandos iguais nos dois hosts.

## Fase 3 - autenticação e distribuição

- OAuth 2.0 web com callback local, refresh automático e armazenamento do
  refresh token.
- OAuth consumer configurável (`BK_CLIENT_ID`/`BK_CLIENT_SECRET`).
- Assinatura de binários, `.deb`/`.rpm`, Homebrew, Scoop, Winget.
- `bk upgrade`/checksum e SBOM no release.
- Documentação gerada a partir do Cobra.

## Fase 4 - cobertura Cloud restante

- Deployments e environments (`/deployments`, variáveis por ambiente).
- Merge queues.
- Reports e annotations de commit.
- Projetos de workspace (`/workspaces/{w}/projects`).
- Branching model e effective branching model.
- Pipelines: schedules, caches, runners, SSH keys/known hosts, OIDC.
- Commit comments, tags, watchers, forks.
- Source: listar arquivos, ler conteúdo, raw, diff/patch de refs.
- Default reviewers e permissões configuráveis.
- Workspace variables e runners.
- `bk repo sync`/espelhamento e `bk pr queue`.

## Fase 5 - ergonomia

- Extensões no estilo `gh extension` (`bk-<nome>` no PATH).
- Pager e renderização de Markdown.
- `--jq` embutido (hoje: pipe para `jq`).
- `bk copilot`/MCP server com ferramentas de leitura e escrita.
- Skill de agente (`skills/bk`) e receitas para CI.
- Cache local de metadados (workspaces, repositórios) com invalidação.

## Fora de escopo por decisão da Atlassian

- Issues e Wikis no Bitbucket Cloud: removidos em 20/08/2026. O comando
  `issue` não será criado para Cloud; o caminho é Jira.
- Code search no Cloud: depreciação anunciada para novembro/2026.
- App passwords: removidas em julho/2026; não haverá compatibilidade.
