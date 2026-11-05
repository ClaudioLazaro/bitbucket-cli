# Solução de problemas

Comece sempre por `BK_HTTP_DEBUG=1 bk ...` para ver método, URL e status. Os
erros HTTP incluem o `x-request-id` retornado pelo Bitbucket; informe-o ao abrir
um chamado ou issue.

## 401 Unauthorized

Causas comuns:

- Token expirado ou revogado.
- API token enviado sem o e-mail Atlassian como usuário.
- Tentativa de usar app password (removida em julho/2026).
- Access token enviado como Basic em vez de Bearer.

```sh
# API token de usuário (Basic)
bk auth login --hostname bitbucket.org --username voce@empresa.com --with-token < token.txt

# Access token (Bearer)
bk auth login --hostname bitbucket.org --access-token --with-token < access-token.txt

bk auth status
```

O `auth status` mostra `ok`, `not logged in` ou o erro HTTP de cada host. Ele
também indica se o token veio de `env`, `keyring` ou `file`.

## 403 Forbidden

O token é válido, mas não tem escopo suficiente. Confira os escopos do API
token (app Bitbucket) ou as permissões do access token:

| Área | Escopos usuais |
| --- | --- |
| repositórios | `repository`, `repository:write` |
| pull requests | `pullrequest`, `pullrequest:write` |
| pipelines | `pipeline`, `pipeline:write` |
| webhooks | `webhook` |
| snippets | `snippet`, `snippet:write` |
| SSH/GPG keys | `account` |

Tokens de acesso de repositório/projeto/workspace só enxergam o recurso ao qual
pertencem. Para listar workspaces ou o usuário atual, use um API token de
usuário.

## 404 Not Found

- Workspace/projeto ou slug errado.
- O token não tem acesso ao repositório (o Bitbucket costuma responder 404, não
  403, para não revelar a existência).
- ID de PR, pipeline ou variável inexistente.
- Data Center com path de contexto errado: confira `api_url` (`/bitbucket` antes
  de `/rest/api/1.0`).

```sh
bk repo view -R minha-workspace/api
bk api /repositories/minha-workspace/api
```

## 429 e 5xx

O cliente tenta novamente até três vezes em `429`, `502`, `503` e `504`,
respeitando `Retry-After`. Se persistir, reduza `--limit` e verifique o status
do Bitbucket. Em pipelines, evite `--limit 0` em repositórios grandes.

## "no Bitbucket host configured"

Nenhum host está configurado e `BK_HOST` não foi definido.

```sh
bk auth login
# ou
export BK_HOST=bitbucket.org BK_TOKEN=... BK_USERNAME=...
```

## "not logged in to ..."

Existe configuração de host, mas nenhum token no `BK_TOKEN`, keyring ou
`credentials.json`.

```sh
bk auth status
bk auth login --hostname bitbucket.org
```

Se o keyring estiver indisponível, o login avisa e grava
`credentials.json` com `0600`. Para forçar esse caminho:

```sh
BK_FORCE_FILE_STORE=1 bk auth login ...
```

No Linux, um keyring sem sessão D-Bus pode travar ou falhar; nesse caso o
fallback é automático. Em CI, prefira `BK_TOKEN`.

## "basic authentication requires a username"

API token do Cloud usa Basic e precisa do **e-mail da conta Atlassian**:

```sh
bk auth login --username voce@empresa.com --with-token < token.txt
# ou
export BK_USERNAME=voce@empresa.com
```

Se o token for de repositório/projeto/workspace, use Bearer:

```sh
bk auth login --access-token --with-token < access-token.txt
```

## "--json slug" vira argumento posicional

`--json` tem valor opcional. Para filtrar campos, use `=`:

```sh
bk repo list minha-workspace --json=slug,is_private
```

`--json` sozinho imprime todos os campos. Esse comportamento é coberto por
teste para evitar regressão.

## `pull request #N is already open`

O Bitbucket Cloud trata um `POST /pullrequests` para uma origem/destino que já
tem PR aberto como **atualização silenciosa** do PR existente. O `bk` consulta
antes e recusa para não sobrescrever título, descrição ou revisores de outra
pessoa.

Para alterar o PR existente de propósito:

```sh
bk pr create --update --title "Novo título" --body "Nova descrição"
```

Se a intenção era outro PR, feche/merge o existente ou use outra branch de
origem.

## "not a git repository" ou "no Bitbucket git remote"

Fora de um clone, ou com remote apontando para outro host, informe o
repositório:

```sh
bk pr list -R minha-workspace/api
# ou
export BK_REPO=minha-workspace/api
```

O remote `origin` tem preferência; depois os demais. Formatos aceitos estão em
[configuration.md](configuration.md#defaults-de-repositório).

## "HEAD is detached"

`pr create`, `pr checkout` e `browse` usam a branch atual. Em detached HEAD,
passe `--head`/`--branch` ou volte para uma branch.

## "operation not supported on Data Center hosts yet"

O comando existe para Cloud, mas ainda não foi mapeado para a API 1.0. Use o
escape hatch:

```sh
BK_HOST=https://bitbucket.example.com bk api /projects/PROJ/repos/api/pull-requests
```

A cobertura por host está em [api-coverage.md](api-coverage.md). Pull requests,
issues e pipelines do Data Center estão no roadmap.

## "pipeline build N not found in the latest 100 runs"

`bk pipeline view/stop/logs` aceita UUID ou número de build. O número é
resolvido entre os 100 builds mais recentes. Para builds antigos, use o UUID:

```sh
bk pipeline list --json | jq -r '.[] | select(.build_number==128) | .uuid'
```

## Code search retorna erro ou aviso

O endpoint de code search do Cloud foi anunciado para depreciação em
novembro/2026. O comando imprime um aviso no stderr e continua best-effort. Não
há equivalente público no Data Center.

## "pr checks" sai com código 1

É intencional: algum status está `FAILED`. Use `--json` para inspecionar, ou
trate o código no script:

```sh
if bk pr checks 42 -R minha-workspace/api; then
  echo "checks ok"
else
  echo "checks falharam"
fi
```

## Problemas de proxy ou TLS

O cliente usa o `http.Client` padrão do Go e respeita `HTTPS_PROXY`,
`HTTP_PROXY` e `NO_PROXY`. Para um host interno com CA própria, configure o
bundle do sistema (`SSL_CERT_FILE`/`SSL_CERT_DIR`). Não há flag para desabilitar
verificação TLS.

## Ainda travado?

Colete:

```sh
bk version
bk auth status
BK_HTTP_DEBUG=1 bk <comando> 2>debug.log
```

Remova tokens do log antes de compartilhar. Abra uma issue com o comando, o
status HTTP e o `x-request-id`.
