# DM CLI — Roadmap miglioramenti

Analisi completa delle capability esistenti e proposte di miglioramento.
Generata il 22 Feb 2026. Ultimo aggiornamento: 22 Feb 2026.

---

## Livello 1 — Impatto alto, sforzo moderato

### 1. Conversazione persistente su disco

La sessione interattiva (`dm ask`) perde tutto alla chiusura.
Salvare la cronologia (es. `~/.config/dm/history/`) permetterebbe:

- Riprendere una conversazione: `dm ask --resume`
- Rivedere la storia: `dm history`
- L'agente avrebbe contesto a lungo termine

### ~~2. Native Function Calling (OpenAI)~~ — SCARTATO

Valutato e scartato. Il sistema prompt-based JSON attuale e piu flessibile:
i toolkit vengono auto-scoperti dal catalogo senza modificare codice Go.
Con native function calling bisognerebbe rigenerare lo schema `tools[]`
ad ogni nuovo toolkit, perdendo il vantaggio plug-and-play.

### ~~3. Tool `read` — l'agente non puo leggere file~~ — COMPLETATO

Implementato in `tools/read.go`. Legge file (max 256KB, 500 righe) o
lista directory. L'agente lo usa per analizzare codice, config, log.

### ~~4. Tool `grep` — ricerca nel contenuto~~ — COMPLETATO

Implementato in `tools/grep.go`. Cerca pattern nei file con filtro
per estensione, case sensitivity, e limite risultati (max 50).

### 5. Pipe/stdin support

`echo "fix this error" | dm ask` o `git diff | dm ask "review this"`.
Integrare stdin come contesto rende il CLI componibile con altri strumenti.

### ~~6. Agent precision — temperature, JSON mode, system/user roles~~ — COMPLETATO

Implementato: `temperature: 0.2`, `response_format: json_object` (OpenAI) /
`format: json` (Ollama), separazione ruoli system/user, `max_tokens: 1024`,
catalogo compatto con function signature, processo decisionale strutturato
in 7 passi, cattura output tool nella history per multi-step reasoning.

---

## Livello 2 — Impatto medio, sforzo medio

### 7. `dm config` — gestione configurazione

Oggi bisogna editare `dm.agent.json` a mano.
Un comando `dm config set openai.model gpt-4o` o `dm config show` sarebbe piu pratico.

### 8. System prompt personalizzabile

Il system prompt e hardcoded. Permettere di impostarlo in `dm.agent.json`
(`"system_prompt": "Sei un esperto DevOps..."`) rende l'agente adattabile
al contesto di lavoro.

### 9. Rendering tabelle nel markdown

`ui/markdown.go` non gestisce tabelle. Quando l'agente risponde con una
tabella markdown, viene renderizzata male. Aggiungere supporto base per
tabelle allineate.

### 10. Max steps configurabile

Oggi il limite e 4 step hardcoded (`askMaxSteps`). Alcune domande complesse
richiedono piu passaggi. Renderlo configurabile via flag (`--max-steps 8`)
o config.

### ~~11. `dm ask --scope` — focus su un toolkit~~ — COMPLETATO

Implementato: `dm ask --scope stibs "stato del backend"` filtra il catalogo
per prefisso funzione o nome gruppo toolkit. Flag `-s` / `--scope`.

---

## Livello 3 — Idee strategiche

### 12. Plugin `fetch` / HTTP tool

L'agente non puo fare richieste HTTP. Un tool `fetch` (GET con output testo)
aprirebbe scenari come "controlla se l'API e online", "scarica questo JSON".

### 13. Alias/macro system

Comandi custom tipo `dm deploy` che mappa a una sequenza di plugin.
Definibili in config o in un file `aliases.json`.

### 14. `dm init` — setup wizard

Creazione guidata di `dm.agent.json` al primo avvio: scelta provider,
test connessione, selezione modello.

### 15. Multi-provider profiles

Definire profili nominati (`work`, `personal`, `local`) con provider/modello
diversi e switchare con `--profile work`.

### 16. Undo per operazioni distruttive

`rename` e `clean` potrebbero salvare un manifest di rollback prima di agire,
permettendo `dm undo`.

---

## Top 3 raccomandati (aggiornati)

| Priorita | Feature | Perche |
|-----------|-------------------------|------------------------------------------------------|
| 1 | Stdin/pipe support | Rende il CLI componibile: `git log \| dm ask "riassumi"` |
| 2 | Conversazione persistente | Trasforma l'agente da stateless a un vero assistente con memoria |
| 3 | Stdin/pipe support | Rende il CLI componibile con altri strumenti |

---

## Stato attuale (inventario)

### Comandi esistenti

| Comando | Descrizione |
|---------|-------------|
| `dm ask` | Agente AI (interattivo + one-shot, streaming, multi-step fino a 4 passi) |
| `dm plugins` | Gestione plugin (list, info, run, menu) |
| `dm tools` | Tool built-in (search, rename, recent, clean, system, read, grep, diff) |
| `dm doctor` | Diagnostica (config, provider, plugin, path) |
| `dm ps_profile` | Mostra simboli $PROFILE PowerShell |
| `dm completion` | Genera/installa completions shell |

### Tool built-in (Go)

| Tool | Rischio | Cosa fa |
|------|---------|---------|
| search | low | Cerca file per nome/estensione |
| rename | medium | Rinomina batch con preview |
| recent | low | File modificati di recente |
| clean | low/high | Rimuove cartelle vuote |
| system | low | Snapshot sistema/rete |
| read | low | Legge file o lista directory |
| grep | low | Cerca pattern nel contenuto dei file |
| diff | low | Mostra git changes o confronta due file |

### Toolkit PowerShell (22 file)

| Toolkit | Prefisso | Area |
|---------|----------|------|
| FileSystem Path | `fs_path_*` | Percorsi di sistema Windows |
| System | `sys_*` | Sistema operativo e rete locale |
| Docker | `dc_*` | Docker Compose generico |
| Browser | `browser_*` | Gestione browser |
| Excel | `xls_*` | Operazioni su file Excel |
| Text | `txt_*` | Encoding, hashing, conversione testo |
| Help | `help_*` | Introspezione, ricerca intento, quickref, env vars, prerequisiti |
| Toolkit Manager | `tk_*` | Gestione lifecycle toolkit |
| Start Dev | `start_*` | Launch strumenti sviluppo |
| Network | `net_*` | HTTP, download, diagnostica rete |
| Winget | `pkg_*` | Gestione pacchetti Windows |
| Archive | `arc_*` | Compressione e estrazione archivi |
| Scheduler | `sched_*` | Windows Task Scheduler |
| M365 Auth | `m365_*` | Autenticazione Microsoft 365 |
| SharePoint | `spo_*` | SharePoint Online generico |
| Power Automate | `flow_*` | Gestione flussi Power Automate |
| Power Apps | `pa_*` | Gestione Power Apps |
| KVP Star Site | `kvpstar_*` | SharePoint site specifico |
| Star IBS Applications | `star_ibs_*` | SharePoint site specifico |
| STIBS App | `stibs_app_*` | App inspection e monitoring |
| STIBS DB | `stibs_db_*` | Database MariaDB analytics |
| STIBS Docker | `stibs_docker_*` | Docker stack STIBS |

### Agent internals

| Feature | Stato |
|---------|-------|
| Temperature 0.2 | Attivo |
| JSON mode (response_format) | Attivo |
| System/user role separation | Attivo |
| Max tokens 1024 | Attivo |
| Compact catalog (function signature) | Attivo |
| Structured decision process (7 steps) | Attivo |
| Tool output capture in history | Attivo |
| JSON repair fallback (with slog.Warn) | Attivo |
| Decision cache (3 min TTL) | Attivo |
| Config cache (sync.Once per session) | Attivo |
| Catalog token budget (6000 warning) | Attivo |
| `--scope` catalog filtering | Attivo |
| sanitizeAnyMap (json.Marshal) | Attivo |
| Ollama /api/chat (uniform format) | Attivo |
| Loop detection | Attivo |
| Risk assessment (low/medium/high) | Attivo |
| Streaming (OpenAI + Ollama) | Attivo |
| Self-evolving (create_function) | Attivo |

### Quality gates

| Check | Quando |
|-------|--------|
| `go test ./...` | Pre-push hook, CI |
| `golangci-lint run` | Pre-push hook, CI |
| `go vet ./...` | CI |
| `go test -cover -race ./...` | CI (ubuntu + windows) |
| `scripts/pre-push.ps1` | Manuale o hook |
