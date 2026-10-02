# fast-listeners

Serviço **listeners** da plataforma de corridas [fast-platform](https://github.com/guilhermelinosp/fast-platform): lê o outbox do PostgreSQL (acordado por `LISTEN/NOTIFY`, com reconciliação periódica) e publica os eventos no Kafka.

> **Estado atual:** este repositório foi criado a partir do [hellnet-worker-template](https://github.com/guilhermelinosp/hellnet-worker-template) e contém o esqueleto do worker. A implementação em uso está em `cmd/listeners` do fast-platform.

[![pipeline](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/pipeline.yml/badge.svg)](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/pipeline.yml)
[![pr-check](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/pr-check.yml/badge.svg)](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/pr-check.yml)
[![CodeQL](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/codeql.yml/badge.svg)](https://github.com/guilhermelinosp/fast-listeners/actions/workflows/codeql.yml)

## Início rápido

```bash
go run ./cmd/listeners
```

Sem `HELLNET_TELEMETRY_ENDPOINT`, o worker roda e imprime um `tick` a cada 5 s.

## Configuração

| Variável | Descrição | Padrão |
|---|---|---|
| `HELLNET_TELEMETRY_ENDPOINT` | URL do collector OTLP/HTTP (sem ela, a telemetria não exporta) | *vazio* |
| `HELLNET_TELEMETRY_SERVICE` | nome do serviço reportado pela telemetria | nome do módulo |

Um `.env` ao lado do binário é carregado quando existe (`internal/env`); variáveis já definidas no ambiente têm prioridade.

A implementação do fast-platform lê também `DATABASE_*`, `KAFKA_*` e `HELLNET_CACHE_*`: veja a tabela de configuração do fast-platform.

## Arquitetura

```text
PostgreSQL ── NOTIFY outbox_events ──► fast-listeners ──► Kafka ──► fast-sockets
```

Na plataforma, o `fast-listeners` é acordado por `LISTEN/NOTIFY` e publica no Kafka os eventos do outbox (`order.requested.v1` e `order.accepted.v1`); como rede de segurança, reconcilia a cada 5 s os eventos da última hora.

```text
cmd/listeners/main.go   telemetria -> contexto com sinais -> loop do worker -> encerramento gracioso
internal/env         leitura opcional de .env e helpers de variáveis de ambiente
```

O `main.go` inicia a telemetria (`telemetry.New`), roda o `workerLoop` em uma goroutine (um ticker de 5 s que chama o `runJob`, instrumentado como span de worker `tick`) e, em `SIGINT`/`SIGTERM`, cancela o contexto e espera até 10 s o loop parar. Hoje `doWork` só imprime o horário: o trabalho real (outbox -> Kafka) é implementado em [fast-platform](https://github.com/guilhermelinosp/fast-platform) (`cmd/listeners`).

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

Os workflows chamam workflows reutilizáveis de [templates](https://github.com/guilhermelinosp/templates), fixados por SHA de commit. O release precisa do secret `HELLNET_ACTIONS_PRIVATE_KEY` e da variável `HELLNET_ACTIONS_CLIENT_ID`.

## Contribuindo e licença

Veja [CONTRIBUTING.md](CONTRIBUTING.md) e [SECURITY.md](SECURITY.md). Licença [Apache 2.0](LICENSE).
