# Contribuindo

Obrigado pelo interesse. Este documento cobre o fluxo local, o padrão de testes
e o que esperar de uma mudança.

## Requisitos

- Go 1.26 ou superior.
- `git`.
- `make` (opcional, mas os alvos abaixo assumem ele).

## Ambiente

```sh
git clone https://github.com/ClaudioLazaro/bitbucket-cli.git
cd bitbucket-cli
make build      # bin/bk com versão/commit/data
make lint       # gofmt + go vet
make test       # go test ./...
make docs       # regenera docs/commands.md
```

Para iterar rápido:

```sh
go run ./cmd/bk --help
go test ./internal/bitbucket -run TestPagination -v
```

## Estrutura

```
cmd/bk/                 processo, aliases e exit codes
internal/bitbucket/     cliente REST, caminhos, tipos, paginação, erros
internal/config/        config.yml + merge com env
internal/auth/          keyring e fallback 0600
internal/cmdutil/       Factory/App, --json, prompts, ExitError
internal/git/           remotes, branch, clone/fetch/checkout
internal/iostreams/     streams, TTY, cores
internal/tableprinter/  tabelas TTY e TSV
pkg/cmd/<grupo>/        comandos Cobra
script/gen-docs/        gerador da referência de comandos
test/                   integração com API falsa e e2e do binário
```

## Adicionando um comando

1. **Cliente** - adicione o método tipado em `internal/bitbucket`. Use `GetAll`
   para coleções, `Send` para JSON e `SendRaw` para multipart. Se o endpoint não
   existe no Data Center, retorne `UnsupportedError` em vez de chamar um caminho
   errado.
2. **Comando** - crie a função em `pkg/cmd/<grupo>`, use
   `cmdutil.Factory.App()`, `App.ResolveRepo(ctx, -R)`,
   `cmdutil.AddOutputFlags` e `tableprinter`. Toda mensagem de erro deve dizer o
   que fazer.
3. **Registro** - adicione o comando em `pkg/cmd/root/root.go`.
4. **Teste** - em `test/integration_test.go`, use `runCommand` + `httptest` e
   verifique método, caminho, query, auth e corpo. Cubra a saída textual e ao
   menos um caminho `--json`.
5. **Docs** - rode `make docs` e atualize `README.md`,
   `docs/api-coverage.md` e `docs/roadmap.md` se a cobertura mudou.

## Regras de teste

- **Nenhuma chamada real ao Bitbucket.** Use `httptest` e fixtures. Smoke com
  credencial é um passo separado e precisa de autorização explícita.
- O teste e2e compila o binário (`test/e2e_test.go`); não o substitua por um
  mock de `main`.
- Rode `go test -race ./internal/... ./test` antes de enviar mudanças de
  concorrência, retry ou armazenamento.
- Um comando novo sem teste de integração não é considerado pronto.
- `docs/commands.md` é gerado; se você adicionar um comando sem regenerar, o
  teste `TestCommandReferenceCoversEveryCommand` falha.

## Estilo

- `gofmt` obrigatório; `go vet` limpo.
- Comentários godoc em funções, tipos e constantes exportadas.
- Hífens em vez de travessões em código e documentação.
- Texto de ajuda da CLI em inglês (como o `gh`); documentação em português.
- Sem dependências novas sem justificativa: o cliente HTTP e a paginação são
  propositalmente pequenos.
- Preserve contratos de script:flags, defaults, rotas stdout/stderr, formato
  TSV sem TTY, `--json`/`--json=campos`/`--template` e o exit 1 de
  `pr checks`.

## Commits e pull requests

- Commits pequenos, no imperativo, com o porquê no corpo quando não for óbvio.
- Um PR deve explicar o comportamento antes/depois, listar testes executados e
  limitações conhecidas.
- Não misture refatoração ampla com correção funcional.
- Se a mudança alterar a cobertura de API, atualize a matriz em
  `docs/api-coverage.md`.

## Segurança

Não abra issue pública para vulnerabilidades. Siga [SECURITY.md](SECURITY.md).

## Licença

Ao contribuir, você concorda que a contribuição é licenciada sob a MIT, como o
resto do projeto.
