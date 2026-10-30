# uJobs

Interface web para o [ai-job-search](https://github.com/ai-job-search-dev/ai-job-search).

O ai-job-search é um framework de busca de emprego que roda no Claude Code. Ele encontra vagas, avalia o quanto cada uma combina com você, escreve CV e carta sob medida e prepara para entrevistas. O uJobs só dá uma tela para esse fluxo: **todo o trabalho continua sendo feito pelo ai-job-search**. O uJobs lê os arquivos que ele grava e roda os comandos dele. Sem um checkout do ai-job-search, o uJobs não faz nada.

> Projeto independente da comunidade. Não é afiliado ao ai-job-search nem à Anthropic. Se o ai-job-search te ajudou, apoie o autor pelo link no README do projeto.

## O que a tela faz

| Página | Fonte no ai-job-search |
|---|---|
| **Candidaturas**: suas candidaturas, com links para o CV, a carta e a vaga, e atalhos para `/interview` e `/outcome` | `job_search_tracker.csv`, `cv/`, `cover_letters/` |
| **Vagas encontradas**: o histórico de `/scrape` e `/rank`, com o fit e um botão `/apply` | `job_scraper/seen_jobs.json` |
| **Buscar nos portais**: busca direta em cada portal | os CLIs em `.agents/skills/*/cli` |
| **Comandos**: roda qualquer comando (`/setup`, `/scrape`, `/apply <url>`…) e permite responder às perguntas do Claude | `claude -p` na pasta do ai-job-search |

## Requisitos

- Um checkout do ai-job-search **já configurado**, seguindo o [SETUP.md](https://github.com/ai-job-search-dev/ai-job-search/blob/main/SETUP.md) dele: Claude Code, Bun, Python e LaTeX
- Go 1.24+ e Node 20+ (só para compilar)

## Compilar e rodar

```bash
cd web && npm install && npm run build && cd ..
go build -o ujobs .
./ujobs --root /caminho/para/ai-job-search            # ou AI_JOB_SEARCH_DIR=...
```

Depois abra http://127.0.0.1:8080.

### Escolhendo o framework HTTP

O mesmo binário traz três implementações do backend. Todas expõem a mesma API e passam pelos mesmos testes:

```bash
./ujobs --root ... --router chi    # padrão (go-chi.io)
./ujobs --root ... --router gin    # gin-gonic.com
./ujobs --root ... --router echo   # echo.labstack.com
```

| Flag | Padrão | |
|---|---|---|
| `--root` | `$AI_JOB_SEARCH_DIR` | pasta do ai-job-search |
| `--router` | `chi` | `gin`, `echo` ou `chi` |
| `--addr` | `127.0.0.1:8080` | endereço de escuta |
| `--claude` | `claude` | executável do Claude Code |
| `--bun` | `bun` | executável do Bun |

## Desenvolvimento

```bash
go run . --root ../ai-job-search     # API em :8080
cd web && npm run dev                # Quasar com hot reload, faz proxy de /api para :8080
go test ./...                        # roda a mesma bateria de testes nos três routers
```

Estrutura:

```
main.go                 flags e inicialização
internal/core           lê os arquivos e roda os CLIs do ai-job-search (sem framework)
internal/api            api.go: rotas, erros e segurança compartilhados
                        gin.go, echo.go, chi.go: um adaptador por framework
web/                    app Quasar (Vue 3 + Vite), embutido no binário via embed.go
```

## Segurança

O uJobs dispara o Claude Code com acesso aos seus dados pessoais. Por isso:

- Por padrão ele escuta só em `127.0.0.1`. **Não exponha na rede.**
- Requisições com `Host` que não seja loopback são recusadas, o que bloqueia DNS rebinding. Escritas exigem `application/json`, então outro site não consegue disparar comandos pelo seu navegador.
- Só roda os comandos que existem em `.claude/commands` e `.claude/skills` do checkout.
- O Claude roda com `--permission-mode acceptEdits`. Comandos de shell continuam limitados à lista de permissões do próprio `.claude/settings.json` do ai-job-search, então as proteções dele valem igual.
- `/files` só serve `cv/`, `cover_letters/` e `documents/applications/`.

## Licença

MIT. O ai-job-search também é MIT, © dos autores dele.
