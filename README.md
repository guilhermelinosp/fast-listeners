# fast-listeners

Serviço **listeners** da plataforma de corridas [fast-platform](https://github.com/guilhermelinosp/fast-platform): lê o outbox do PostgreSQL (acordado por `LISTEN/NOTIFY`, com reconciliação periódica) e publica os eventos no Kafka. O runtime e os eventos de pedido vêm da biblioteca [fast-platform](https://github.com/guilhermelinosp/fast-platform) (`platform`, `env` e `events`).

[![pipeline](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/pipeline.yml/badge.svg)](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/pipeline.yml)
[![pr-check](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/pr-check.yml/badge.svg)](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/pr-check.yml)
[![CodeQL](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/codeql.yml/badge.svg)](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/codeql.yml)

## Início rápido

O serviço lê o `.env` da própria pasta (`cmd/listeners/.env`, ignorado pelo git): copie o `cmd/listeners/.env.example` e ajuste. São necessários PostgreSQL, Redis e Kafka.

```bash
cp cmd/listeners/.env.example cmd/listeners/.env
cd cmd/listeners && go run -race main.go
```

## Configuração

| Variável | Descrição |
|---|---|
| `HELLNET_SERVICE`, `HELLNET_ENVIRONMENT` | Nome do serviço e ambiente |
| `HELLNET_TELEMETRY_ENDPOINT` | Endpoint OTLP/HTTP (Alloy) |
| `DATABASE_HOST`, `DATABASE_PORT`, `DATABASE_NAME`, `DATABASE_USERNAME`, `DATABASE_PASSWORD`, `DATABASE_POOL_MAX_SIZE` | Conexão PostgreSQL |
| `KAFKA_BROKERS`, `KAFKA_SECURITY_PROTOCOL` | Conexão Kafka |
| `KAFKA_TOPIC_ORDER_REQUESTED`, `KAFKA_TOPIC_ORDER_ACCEPTED` | Tópicos dos eventos |
| `KAFKA_MATCHING_CONSUMER_GROUP` | Grupo do consumer de matching (em standby) |
| `CACHE_CONNECTION`, `CACHE_ENABLE_L2`, `CACHE_DEFAULT_TTL` | Cache L1 (memória) e L2 (Redis) |

Variáveis já definidas no ambiente têm prioridade sobre o `.env`. Faltando uma variável obrigatória, o processo falha com um erro claro.

## Arquitetura

```text
PostgreSQL ── NOTIFY outbox_events ──► fast-listeners ──► Kafka ──► fast-sockets
```

Quem grava o pedido (a API do [fast-platform](https://github.com/guilhermelinosp/fast-platform)) insere o evento no outbox na mesma escrita atômica. O `fast-listeners` é acordado por `LISTEN/NOTIFY` e publica os eventos no Kafka; como rede de segurança, reconcilia a cada 5 s os eventos da última hora (e faz uma varredura completa na partida e de hora em hora). A publicação é registrada em uma linha nova, em `outbox_publications` (o banco é append-only).

O consumer de matching (`internal/matching`) escolhe um motorista disponível para cada pedido. Está em **standby**: o bloco está comentado em `cmd/listeners/main.go`, e nada grava a disponibilidade dos motoristas ainda.

## Estrutura

```text
cmd/listeners          outbox -> Kafka (matching em standby)
internal/listeners      outbox, NOTIFY e reconciliação
internal/producers      producer Kafka que publica os eventos do outbox
internal/matching       escolha de motorista (standby)

importa github.com/guilhermelinosp/fast-platform/{platform,env,events}  (runtime, variáveis de ambiente e eventos de pedido)
```

## Desenvolvimento

```bash
go test -race ./...
go vet ./...
golangci-lint run ./...
```

Os hooks do [Lefthook](.lefthook.yml) rodam `gofmt`, `vet`, testes (com e sem `-race`), build, `go mod tidy`, lint, `govulncheck` e o scan de segredos; instale-os uma vez com `lefthook install`. Commits seguem [Conventional Commits](https://www.conventionalcommits.org/).

## CI/CD

| Workflow | Gatilho | O que faz |
|---|---|---|
| `pr-check` | pull request | shellcheck, estratégia de merge e Conventional Commits (`merge-check`), Gitleaks, labels e o gate de qualidade Go (integridade do módulo, vet, testes com race e cobertura, lint, build, dependency review). O `pr-gate` reúne tudo e é o check obrigatório |
| `pipeline` | push na `main` (ignora `.github/**`) ou manual | guarda de semver (bloqueia major automático), tag imutável + GitHub Release, imagem de container |
| `codeql` | diário ou manual | análise estática (CodeQL) |
| `security` | diário ou manual | scans de Gitleaks e Trivy |
| `auto-pr` | push em `feat/**` ou `fix/**` | abre o pull request automaticamente |
| `dependabot-actions-auto-merge` | pull requests do Dependabot | faz auto-merge das atualizações de GitHub Actions |

Os workflows chamam workflows reutilizáveis de [templates](https://github.com/guilhermelinosp/templates) em `@latest`. Não há secret: o release e os demais jobs trocam o token OIDC por um token do [Octo STS](https://github.com/apps/octo-sts) (App instalado no repositório; políticas em `.github/chainguard/`).

## Contribuindo e licença

Veja [CONTRIBUTING.md](CONTRIBUTING.md) e [SECURITY.md](SECURITY.md). Licença [Apache 2.0](LICENSE).

## Deploy

Cada release publica a imagem no GHCR e o job `cd` do `pipeline.yml` chama o hub de deploy do repositório `templates`, que entra no tailnet por OIDC e sincroniza a Application `fast-listeners` no ArgoCD. A versão da imagem não fica no Git: o ArgoCD Image Updater acompanha as tags `vX.Y.Z`. Os manifests estão em `infrastructure/`.
