# Cobertura da API

Cada comando usa endpoints documentados da API do Bitbucket. O que não está
mapeado pode ser chamado com `bk api`.

Coluna de host: `C` = Cloud, `DC` = Data Center.

## Autenticação e identidade

| Comando | Endpoint | Host |
| --- | --- | --- |
| `bk auth login` | `GET /user` (verificação) | C |
| `bk auth login` | `GET /repos` (verificação) | DC |
| `bk auth status` | `GET /user` ou `GET /repos` | C/DC |
| `bk auth logout` | local (keyring/arquivo) | C/DC |
| `bk auth token` | local | C/DC |

## Repositórios e branches

| Comando | Endpoint | Host |
| --- | --- | --- |
| `bk repo list` | `GET /repositories/{workspace}` | C |
| `bk repo list` | `GET /projects/{key}/repos` ou `GET /repos` | DC |
| `bk repo view` | `GET /repositories/{w}/{slug}` | C |
| `bk repo view` | `GET /projects/{key}/repos/{slug}` | DC |
| `bk repo create` | `POST /repositories/{w}/{slug}` | C |
| `bk repo create` | `POST /projects/{key}/repos` | DC |
| `bk repo edit` | `PUT /repositories/{w}/{slug}` | C |
| `bk repo delete` | `DELETE /repositories/{w}/{slug}` | C/DC |
| `bk repo fork` | `POST /repositories/{w}/{slug}/forks` | C |
| `bk repo clone` | `GET` do repositório + `git clone` | C/DC |
| `bk branch list` | `GET /repositories/{w}/{slug}/refs/branches` | C |
| `bk branch list` | `GET /projects/{k}/repos/{slug}/branches` | DC |
| `bk browse` | local (monta URL) | C/DC |

## Pull requests

Todos Cloud (`/repositories/{w}/{slug}/pullrequests`).

| Comando | Endpoint |
| --- | --- |
| `bk pr list` | `GET .../pullrequests` (`state` + `q`) |
| `bk pr view` | `GET .../pullrequests/{id}` |
| `bk pr create` | `POST .../pullrequests` |
| `bk pr checkout` | `GET` do PR + `git fetch` da branch |
| `bk pr merge` | `POST .../pullrequests/{id}/merge` |
| `bk pr decline` | `POST .../pullrequests/{id}/decline` |
| `bk pr reopen` | `PUT .../pullrequests/{id}` (`state=OPEN`) |
| `bk pr approve` / `unapprove` | `POST` / `DELETE .../approve` |
| `bk pr request-changes` | `POST` / `DELETE .../request-changes` |
| `bk pr comment` | `POST .../comments` |
| `bk pr comments` | `GET .../comments` |
| `bk pr diff` | `GET .../diff` |
| `bk pr checks` | `GET .../statuses` |
| `bk pr commits` | `GET .../commits` |

## Pipelines

Todos Cloud (`/repositories/{w}/{slug}/pipelines...`).

| Comando | Endpoint |
| --- | --- |
| `bk pipeline list` | `GET /pipelines` |
| `bk pipeline view` | `GET /pipelines/{uuid}` + `GET /steps` |
| `bk pipeline run` | `POST /pipelines` |
| `bk pipeline stop` | `POST /pipelines/{uuid}/stopPipeline` |
| `bk pipeline logs` | `GET /pipelines/{uuid}/steps/{step}/log` |
| `bk pipeline variable list` | `GET /pipelines_config/variables` |
| `bk pipeline variable set` | `POST` ou `PUT /pipelines_config/variables/{uuid}` |
| `bk pipeline variable delete` | `DELETE /pipelines_config/variables/{uuid}` |

## Workspaces, snippets e downloads

| Comando | Endpoint | Host |
| --- | --- | --- |
| `bk workspace list` | `GET /workspaces` | C |
| `bk workspace view` | `GET /workspaces/{slug}` | C |
| `bk workspace members` | `GET /workspaces/{slug}/members` | C |
| `bk snippet list` | `GET /snippets/{workspace}` | C |
| `bk snippet view` | `GET /snippets/{w}/{id}` + `/files/{path}` | C |
| `bk snippet create` | `POST /snippets/{w}` (multipart) | C |
| `bk snippet delete` | `DELETE /snippets/{w}/{id}` | C |
| `bk snippet clone` | `git clone https://bitbucket.org/snippets/{w}/{id}.git` | C |
| `bk download list` | `GET /repositories/{w}/{s}/downloads` | C |
| `bk download upload` | `POST .../downloads` (multipart `files`) | C |
| `bk download delete` | `DELETE .../downloads/{filename}` | C |

## Webhooks, restrições e chaves

| Comando | Endpoint | Host |
| --- | --- | --- |
| `bk webhook list/view` | `GET /repositories/{w}/{s}/hooks[/{uid}]` | C |
| `bk webhook create` | `POST /repositories/{w}/{s}/hooks` | C |
| `bk webhook delete` | `DELETE .../hooks/{uid}` | C |
| `bk branch restriction list` | `GET .../branch-restrictions` | C |
| `bk branch restriction create` | `POST .../branch-restrictions` | C |
| `bk branch restriction delete` | `DELETE .../branch-restrictions/{id}` | C |
| `bk ssh-key list/add` | `GET` / `POST /users/{id}/ssh-keys` | C |
| `bk ssh-key delete` | `DELETE /users/{id}/ssh-keys/{key_id}` | C |
| `bk gpg-key list/add` | `GET` / `POST /users/{id}/gpg-keys` | C |
| `bk gpg-key delete` | `DELETE /users/{id}/gpg-keys/{fingerprint}` | C |

## Busca, API crua e locais

| Comando | Endpoint | Host |
| --- | --- | --- |
| `bk search code` | `GET /workspaces/{w}/search/code` | C |
| `bk api` | qualquer caminho sob a base REST | C/DC |
| `bk config` | local | C/DC |
| `bk alias` | local | C/DC |
| `bk completion` | local | C/DC |
| `bk version` | local | C/DC |

## Ainda não mapeado

Cloud:

- Deployments e environments.
- Merge queues.
- Reports e annotations de commit.
- Projetos de workspace (`/workspaces/{w}/projects`).
- Branching model.
- Schedules, caches, runners, SSH/known hosts e OIDC de Pipelines.
- Variáveis de workspace e de environment.
- Permissões configuráveis, default reviewers e grupos.
- Comentários de commit, tags, source/files, watchers, forks (além do comando).
- OAuth web e extensões.

Data Center:

- Pull requests, atividades, tarefas e merge com `version`.
- Pipelines e builds.
- Issues e Wikis (ainda existem no DC).
- Usuários, grupos e permissões.
- Default reviewers e branch model.

Enquanto um item não é mapeado, use `bk api` com o caminho relativo à base do
host. Exemplo:

```sh
bk api /projects/PROJ/repos/api/pull-requests
bk api /repositories/minha-workspace/api/deployments
```

## Depreciações da Atlassian

- Issues e Wikis do Cloud: removidos em 20/08/2026.
- App passwords: removidas em julho/2026.
- Code search do Cloud: depreciação anunciada para novembro/2026.
