# Configuração e credenciais

## Onde ficam os arquivos

| Arquivo | Papel | Permissão |
| --- | --- | --- |
| `$XDG_CONFIG_HOME/bk/config.yml` | hosts, defaults, aliases | `0600` |
| `$XDG_CONFIG_HOME/bk/credentials.json` | fallback de tokens quando não há keyring | `0600` |
| keyring do sistema | tokens (Secret Service/KWallet, macOS Keychain, Windows Credential Manager) | gerenciada pelo SO |

`BK_CONFIG_DIR` troca o diretório base. O `config.yml` **nunca** contém tokens:
credenciais ficam no keyring ou no `credentials.json`.

## Precedência

O valor efetivo de cada ajuste é resolvido nesta ordem:

1. Flag do comando.
2. Variável `BK_*` correspondente.
3. Entrada do host no `config.yml`.
4. Default derivado (por exemplo `https://api.bitbucket.org/2.0` no Cloud).

Para o token especificamente:

1. `BK_TOKEN`.
2. Keyring do host.
3. `credentials.json` do host.
4. Erro "not logged in".

`BK_API_URL` é sempre um override efetivo: ele vale para a execução atual e não
é gravado no `config.yml`.

## Seleção de host

Ordem: `--hostname` > `BK_HOST` > `active_host` > único host configurado.

```sh
bk --hostname bitbucket.org repo list minha-workspace
BK_HOST=https://bitbucket.example.com bk repo list PROJ
```

`bk auth login` e `bk auth status` têm um `--hostname` próprio, que vence o
global quando aparece depois do subcomando.

## config.yml

```yaml
version: 1
active_host: bitbucket.org
hosts:
  bitbucket.org:
    kind: cloud
    user: voce@empresa.com
    auth_method: basic
    workspace: minha-workspace
    project: ""
    api_url: https://api.bitbucket.org/2.0
  bitbucket.example.com:
    kind: dc
    user: ""
    auth_method: bearer
    workspace: ""
    project: PROJ
    api_url: https://bitbucket.example.com/bitbucket/rest/api/1.0
aliases:
  pl: pr list
  pv: pr view
```

| Campo | Valores | Observação |
| --- | --- | --- |
| `version` | `1` | reservado para migração |
| `active_host` | hostname | usado quando não há `BK_HOST` nem `--hostname` |
| `hosts.<hostname>.kind` | `cloud` ou `dc` | define caminhos e paginação |
| `hosts.<hostname>.user` | e-mail/usuário | necessário para Basic |
| `hosts.<hostname>.auth_method` | `basic` ou `bearer` | Bearer não usa usuário |
| `hosts.<hostname>.workspace` | slug | default do Cloud |
| `hosts.<hostname>.project` | chave | default do Data Center |
| `hosts.<hostname>.api_url` | URL | base REST completa, inclusive path de contexto |
| `aliases.<nome>` | expansão | palavras separadas por espaço |

Comandos de leitura/escrita:

```sh
bk config list
bk config get active_host
bk config get workspace
bk config set active_host bitbucket.org
bk config set workspace minha-workspace
bk config set project PROJ
```

## Normalização de host

| Entrada | Kind | Base REST |
| --- | --- | --- |
| `bitbucket.org` | cloud | `https://api.bitbucket.org/2.0` |
| `https://bitbucket.org/` | cloud | `https://api.bitbucket.org/2.0` |
| `api.bitbucket.org` | cloud | `https://api.bitbucket.org/2.0` |
| `https://bitbucket.example.com` | dc | `https://bitbucket.example.com/rest/api/1.0` |
| `https://bitbucket.example.com/bitbucket` | dc | `https://bitbucket.example.com/bitbucket/rest/api/1.0` |
| `https://bitbucket.example.com/rest/api/1.0` | dc | idem (sufixo não duplicado) |

O hostname canônico é `host[:porta]`; o path de contexto fica em `api_url`.
Consequência: **um contexto por host:porta** no `config.yml`. Para alternar
entre contextos no mesmo host, use `BK_API_URL` na execução.

## Credenciais

### Keyring

O `bk` usa o serviço `bk` e o hostname como chave. No Linux ele só tenta o
keyring quando existe sessão D-Bus (`DBUS_SESSION_BUS_ADDRESS` ou
`XDG_RUNTIME_DIR`); em macOS e Windows sempre tenta. Se o keyring falhar, cai no
arquivo e avisa no login.

### Arquivo de fallback

`credentials.json` é um objeto JSON indexado por hostname:

```json
{
  "bitbucket.org": {
    "token": "***",
    "username": "voce@empresa.com",
    "method": "basic"
  },
  "bitbucket.example.com": {
    "token": "***",
    "method": "bearer"
  }
}
```

- Gravado com `0600`; o diretório é `0700`.
- `BK_FORCE_FILE_STORE=1` ignora o keyring (CI, containers, keyring travado).
- `bk auth logout` remove a entrada dos dois backends.
- `bk auth token` imprime o token cru; trate como segredo.

### Métodos

| Cenário | Método | Header |
| --- | --- | --- |
| Cloud, API token de usuário | `basic` | `Authorization: Basic base64(email:token)` |
| Cloud, access token de repo/projeto/workspace | `bearer` | `Authorization: Bearer token` |
| Cloud, OAuth (futuro) | `bearer` | idem |
| Data Center, PAT | `bearer` (default) | idem |
| Data Center, usuário e senha | `basic` | Basic |

App passwords não são suportadas: a Atlassian as removeu em julho de 2026.

## Variáveis de ambiente

| Variável | Efeito | Exemplo |
| --- | --- | --- |
| `BK_HOST` | host ativo (nome ou URL) | `bitbucket.org` |
| `BK_TOKEN` | token da execução; ignora keyring/arquivo | `ATATT...` |
| `BK_USERNAME` | usuário do Basic | `voce@empresa.com` |
| `BK_AUTH_METHOD` | `basic` ou `bearer` | `bearer` |
| `BK_API_URL` | base REST efetiva | `http://127.0.0.1:8080/2.0` |
| `BK_WORKSPACE` | workspace default | `minha-workspace` |
| `BK_PROJECT` | projeto default (DC) | `PROJ` |
| `BK_REPO` | repositório default | `minha-workspace/api` |
| `BK_CONFIG_DIR` | diretório de configuração | `/tmp/bk-ci` |
| `BK_FORCE_FILE_STORE` | força `credentials.json` | `1` |
| `BK_HTTP_DEBUG` | loga método, URL e status | `1` |
| `HTTPS_PROXY` / `NO_PROXY` | proxy HTTP do runtime Go | padrão do ambiente |

## Defaults de repositório

Ordem: `-R OWNER/REPO` (ou posicional) > `BK_REPO` > remote do git.

Remotes aceitos:

- `https://bitbucket.org/OWNER/REPO.git`
- `git@bitbucket.org:OWNER/REPO.git`
- `ssh://git@bitbucket.org/OWNER/REPO.git`
- Data Center: `/scm/PROJ/repo.git`, `/projects/PROJ/repos/repo`,
  `ssh://git@host:7999/PROJ/repo.git`

O remote `origin` tem preferência; se ele não apontar para o host ativo, os
outros remotes são testados. Fora de um clone, sempre passe `-R`.

## Aliases

```sh
bk alias set pl "pr list"
bk alias set pv "pr view"
bk alias list
bk alias delete pv
```

A expansão usa separação simples por espaços: não há aspas, pipes ou variáveis.
Nomes de comandos embutidos não podem virar alias. Exemplo de uso:

```sh
bk pl --state open
```

## Saída para scripts

| Recurso | Efeito |
| --- | --- |
| `--json` | objeto/lista completa em JSON |
| `--json=slug,is_private` | filtra campos de topo |
| `--template '{{.slug}}'` | template Go sobre o JSON |
| stdout não-TTY | tabelas viram linhas TSV, sem cabeçalho |
| `bk pr checks` | exit 1 se algum status for `FAILED` |

Campos aninhados não são filtrados por `--json=campos`; use `--json` e `jq`.

## Códigos de saída

| Código | Significado |
| --- | --- |
| `0` | comando concluído |
| `1` | erro de validação, HTTP, autenticação ou `pr checks` com status `FAILED` |

O `bk` não usa códigos específicos por tipo de erro; scripts devem tratar `1`
como falha genérica e ler o stderr.

## Completion

```sh
# bash
bk completion bash > ~/.local/share/bash-completion/completions/bk

# zsh (com fpath configurado)
bk completion zsh > "${fpath[1]}/_bk"

# fish
bk completion fish | source

# PowerShell
bk completion powershell | Out-String | Invoke-Expression
```

Use `--no-descriptions` para shells que não exibem descrições.

## Debug

```sh
BK_HTTP_DEBUG=1 bk repo list minha-workspace
```

Imprime `> GET URL` e `< 200 URL` no stderr. Tokens não aparecem em URL nem no
log. Para diagnosticar erros HTTP, o `bk` também mostra o `x-request-id` do
Bitbucket quando presente.

## Exemplos de CI

Data Center:

```sh
export BK_HOST=https://bitbucket.example.com
export BK_TOKEN="$PAT"
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
export BK_WORKSPACE=minha-workspace
bk pipeline run --branch main
```

Cloud com access token de repositório:

```sh
export BK_HOST=bitbucket.org
export BK_TOKEN="$REPO_ACCESS_TOKEN"
export BK_AUTH_METHOD=bearer
export BK_REPO=minha-workspace/api
bk repo view
```
