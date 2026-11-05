# Segurança

## Versões suportadas

| Versão | Suporte |
| --- | --- |
| 0.1.x | correções de segurança |

Versões de desenvolvimento (`dev`) não têm suporte formal.

## Como reportar

Não abra issue, PR ou discussão pública com detalhes de vulnerabilidade.

Use o canal privado do GitHub:

1. Acesse `https://github.com/ClaudioLazaro/bitbucket-cli/security/advisories/new`.
2. Descreva o impacto, a versão, o sistema e os passos para reproduzir.
3. Se possível, inclua um PoC mínimo e a saída de `bk version`.

Se o canal privado não estiver disponível, abra uma issue sem detalhes apenas
pedindo um contato privado.

## Escopo

Exemplos de problemas tratados como segurança:

- Vazamento de token em logs, mensagens de erro, URLs ou arquivos com permissão
  incorreta.
- Execução de comando ou injeção via nomes de repositório, branch, webhook,
  snippet, nome de arquivo de download ou template.
- Path traversal ao baixar/gravar arquivos.
- Verificação TLS desabilitada ou proxy mal configurado.
- Falha do keyring que grave credenciais em local legível por outros usuários.
- Dependência vulnerável com impacto explorável no `bk`.

Não são tratados como vulnerabilidade:

- Exigir token com escopo correto para acessar um recurso.
- Limitações já documentadas em `docs/api-coverage.md` ou `README.md`.
- Comportamento do Bitbucket, incluindo endpoints deprecados pela Atlassian.

## O que incluir

- Versão (`bk version`).
- Sistema operacional e arquitetura.
- Comando e flags usados (com tokens removidos).
- Status HTTP e `x-request-id`, se houver.
- Impacto esperado e observado.

## Divulgação

O relatório é tratado em privado até a correção ser publicada. Combine o prazo
de divulgação com o mantenedor; não publique detalhes antes disso.

## Boas práticas para usuários

- Prefira keyring; use `BK_FORCE_FILE_STORE` só quando necessário.
- Não passe tokens em linha de comando (ficam no histórico); use
  `--with-token` com stdin ou `BK_TOKEN`.
- Rotacione tokens periodicamente; API tokens têm expiração máxima de um ano.
- Revogue tokens de CI ao remover o job.
- Rode `bk auth logout` em máquinas compartilhadas.
