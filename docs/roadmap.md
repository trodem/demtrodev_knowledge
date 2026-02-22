# DM CLI — Roadmap miglioramenti

Analisi completa delle capability esistenti e proposte di miglioramento.
Generata il 22 Feb 2026.

---

## Livello 1 — Impatto alto, sforzo moderato

### 1. Conversazione persistente su disco

La sessione interattiva (`dm ask`) perde tutto alla chiusura.
Salvare la cronologia (es. `~/.config/dm/history/`) permetterebbe:

- Riprendere una conversazione: `dm ask --resume`
- Rivedere la storia: `dm history`
- L'agente avrebbe contesto a lungo termine

### ~~2. Native Function Calling (OpenAI)~~ — SCARTATO

Valutato e scartato. Il sistema prompt-based JSON attuale è più flessibile:
i toolkit vengono auto-scoperti dal catalogo senza modificare codice Go.
Con native function calling bisognerebbe rigenerare lo schema `tools[]`
ad ogni nuovo toolkit, perdendo il vantaggio plug-and-play.
Il repair prompt + fallback a `action=answer` coprono i casi limite.

### 3. Tool `read` — l'agente non può leggere file

`search` trova file, ma l'agente non può leggerne il contenuto.
Un tool `read` (con limite di righe) lo renderebbe molto più utile per
debugging, analisi codice, review.

### 4. Tool `grep` — ricerca nel contenuto

`search` cerca solo per nome file. Un tool `grep` (wrapper di
`filepath.Walk` + `strings.Contains`) permetterebbe all'agente di cercare
dentro i file.

### 5. Pipe/stdin support

`echo "fix this error" | dm ask` o `git diff | dm ask "review this"`.
Integrare stdin come contesto rende il CLI componibile con altri strumenti.

---

## Livello 2 — Impatto medio, sforzo medio

### 6. `dm config` — gestione configurazione

Oggi bisogna editare `dm.agent.json` a mano.
Un comando `dm config set openai.model gpt-4o` o `dm config show` sarebbe più pratico.

### 7. System prompt personalizzabile

Il system prompt è hardcoded. Permettere di impostarlo in `dm.agent.json`
(`"system_prompt": "Sei un esperto DevOps..."`) rende l'agente adattabile
al contesto di lavoro.

### 8. Progress bar per operazioni lunghe

`backup` e `search` su directory grandi non danno feedback.
Una barra di progresso (`[=====>    ] 45%`) migliorerebbe la UX.

### 9. Rendering tabelle nel markdown

`ui/markdown.go` non gestisce tabelle. Quando l'agente risponde con una
tabella markdown, viene renderizzata male. Aggiungere supporto base per
tabelle allineate.

### 10. Max steps configurabile

Oggi il limite è 4 step hardcoded (`askMaxSteps`). Alcune domande complesse
richiedono più passaggi. Renderlo configurabile via flag (`--max-steps 8`)
o config.

---

## Livello 3 — Idee strategiche

### 11. Plugin `fetch` / HTTP tool

L'agente non può fare richieste HTTP. Un tool `fetch` (GET con output testo)
aprirebbe scenari come "controlla se l'API è online", "scarica questo JSON".

### 12. Alias/macro system

Comandi custom tipo `dm deploy` che mappa a una sequenza di plugin.
Definibili in config o in un file `aliases.json`.

### 13. `dm init` — setup wizard

Creazione guidata di `dm.agent.json` al primo avvio: scelta provider,
test connessione, selezione modello.

### 14. Multi-provider profiles

Definire profili nominati (`work`, `personal`, `local`) con provider/modello
diversi e switchare con `--profile work`.

### 15. Undo per operazioni distruttive

`rename` e `clean` potrebbero salvare un manifest di rollback prima di agire,
permettendo `dm undo`.

---

## Top 3 raccomandati

| Priorità | Feature | Perché |
|-----------|-------------------------|------------------------------------------------------|
| 1 | Tool `read` | Senza questo l'agente è "cieco" — trova file ma non li legge |
| 2 | Stdin/pipe support | Rende il CLI componibile: `git log \| dm ask "riassumi"` |
| 3 | Conversazione persistente | Trasforma l'agente da stateless a un vero assistente con memoria |

---

## Stato attuale (inventario)

### Comandi esistenti

| Comando | Descrizione |
|---------|-------------|
| `dm ask` | Agente AI (interattivo + one-shot) |
| `dm plugins` | Gestione plugin (list, info, run, menu) |
| `dm tools` | Tool built-in (search, rename, recent, backup, clean, system) |
| `dm doctor` | Diagnostica (config, provider, plugin, path) |
| `dm ps_profile` | Mostra simboli $PROFILE PowerShell |
| `dm completion` | Genera/installa completions shell |

### Tool built-in (Go)

| Tool | Rischio | Cosa fa |
|------|---------|---------|
| search | low | Cerca file per nome/estensione |
| rename | medium | Rinomina batch con preview |
| recent | low | File modificati di recente |
| backup | medium | Backup zip di directory |
| clean | low/high | Rimuove cartelle vuote |
| system | low | Snapshot sistema/rete |

### Toolkit PowerShell (14 file)

| Toolkit | Prefisso | Area |
|---------|----------|------|
| FileSystem | fs_* | Operazioni su file |
| System | sys_* | Sistema operativo |
| Git | g_* | Operazioni Git |
| Docker | dc_* | Docker Compose |
| Browser | browser_* | Automazione browser |
| Excel | excel_* | Operazioni Excel |
| Help | help_* | Introspezione toolkit |
| Toolkit Manager | tk_* | Gestione toolkit |
| Start Dev | dev_* | Ambiente sviluppo |
| Text | txt_* | Encoding, hashing, testo |
| KVP Star Site | - | SharePoint M365 |
| Star IBS Applications | - | Applicazioni M365 |
| STIBS App | stibs_* | App specifiche |
| STIBS DB | stibs_* | Database |
| STIBS Docker | stibs_* | Docker specifico |

### Cosa manca (gap principali)

- Nessuna persistenza conversazione su disco
- Nessun tool per leggere/scrivere file (l'agente è "cieco")
- Nessuna ricerca nel contenuto dei file (solo per nome)
- No stdin/pipe support
- No function calling nativo (usa prompt-based JSON)
- System prompt hardcoded
- Max steps hardcoded (4)
- Nessun rendering tabelle nel terminale
- Nessun `dm config` / `dm init` / `dm version` dedicato
- Nessun undo per operazioni distruttive
