# Chute

Chute is a backend service that scrapes professional rodeo data from prorodeo.com,
stores it in a structured database, and serves it to a purpose-built announcer UI
that generates printable cheat sheets for rodeo announcers.

---

## How It Works

```
                        prorodeo.com
                             │
                    (every 6 hours)
                             │
                             ▼
              ┌─────── ferdinand service ────────┐
              │                                  │
              │  1. Fetch rodeo schedule          │
              │  2. Fetch results per rodeo       │
              │  3. Fetch athlete profiles        │
              │  4. Save to data/results/         │
              │                                  │
              └──────────────────────────────────┘
                             │
                     data/results/rodeo/
                      {rodeoID}/results.json
                             │
                    ┌────────┴────────┐
                    │                 │
              (online path)     (offline path)
                    │                 │
                    ▼                 ▼
             sheet service      local copy
             (HTTP API)         (synced from cloud)
                    │
                    ▼
              Announcer UI
                    │
                    ▼
            Generated PDF Sheet
```

The system is designed to work both **online** (hitting the cloud sheet service)
and **offline** (using a local copy of the data synced when internet was last available).

---

## Project Structure

```
chute/
├── api/                            # Entry points — binaries only, no reusable logic
│   ├── services/
│   │   ├── ferdinand/              # Scraper daemon: fetches from prorodeo.com on a ticker
│   │   │   └── main.go
│   │   └── sheet/                  # HTTP API: serves rodeo data and generates PDFs
│   │       └── main.go
│   └── frontends/
│       └── announcer/              # (planned) Announcer UI frontend
│
├── app/                            # Orchestration — coordinates between layers
│   └── domain/
│       ├── rodeoapp/               # Sync logic: fetch → parse → store
│       │   ├── rodeo.go
│       │   └── model.go
│       └── sheetapp/               # Sheet logic: load data → render PDF → respond
│           ├── announcer.go
│           └── model.go
│
├── business/                       # Core domain — no knowledge of HTTP or I/O format
│   └── data/
│       └── store/
│           └── rodeodb/            # File-tree storage (data/results/rodeo/{id}/results.json)
│               └── rodeodb.go      # Will be replaced with SQLite implementation
│
├── foundation/                     # Generic infrastructure — no rodeo knowledge
│   ├── logger/                     # Structured JSON logger (wraps slog)
│   │   └── logger.go
│   ├── web/                        # HTTP mux, Respond/RespondError helpers
│   │   └── web.go
│   └── pdf/                        # PDF rendering primitives (stub — library TBD)
│       └── pdf.go
│
├── zarf/                           # Deployment — no Go code
│   └── docker/
│       ├── Dockerfile.ferdinand
│       └── Dockerfile.sheet
│
├── data/                           # Scraped data (gitignored)
│   └── results/
│       └── rodeo/
│           └── {rodeoID}/
│               └── results.json
│
├── go.mod                          # Module: github.com/jto05/chute
├── Makefile
└── .gitignore
```

### Layer rules

Dependencies only flow **downward** — a layer may never import from a layer above it:

```
api  →  app  →  business  →  foundation
         ↘
       ferdinand (external module at ../ferdinand)
```

This means storage can be swapped (file tree → SQLite) without touching the HTTP
layer, and the PDF library can be changed without touching anything that calls it.

---

## ferdinand (external module)

The scraping logic lives in a separate Go module at `../ferdinand`
(`github.com/am29/ferdinand`). Chute imports it as a dependency via a `replace`
directive in `go.mod` for local development:

```
replace github.com/am29/ferdinand => ../ferdinand
```

Ferdinand exposes three HTTP calls against prorodeo.com:

| Function | Description |
|---|---|
| `FetchRodeosInDateRange` | Fetches the rodeo schedule for a date range |
| `FetchResults` | Fetches full event results for a single rodeo ID |
| `FetchAthlete` | Fetches a contestant profile by athlete ID |

Ferdinand is developed independently and versioned separately. To pick up changes
during local development, edit both repos freely — the `replace` directive means
Go always uses the local copy. Remove the directive and pin a tagged version before
deploying.

---

## Database Schema (planned — SQLite)

Three core tables, with junction tables for the many-to-many relationships:

```
rodeos ──────────────────── results ──── result_contestants ──── contestants
                               │                                      │
                               └──── result_livestock ──── livestock  │
                                                               │       │
                                                   stock_contractors   │
                                                                       │
rodeos ──── contestant_rodeos ────────────────────────────────────────┘
```

| Table | Purpose |
|---|---|
| `rodeos` | One row per rodeo event (keyed on prorodeo.com RodeoId) |
| `contestants` | One row per athlete (keyed on prorodeo.com athlete ID) |
| `contestant_rodeos` | Tracks which rodeos each contestant participated in |
| `results` | One row per place per event per round |
| `result_contestants` | Links results to contestants (handles team roping pairs) |
| `livestock` | Bulls, horses, steers, calves (keyed on prorodeo.com StockId) |
| `stock_contractors` | Stock contractor companies identified by brand |
| `result_livestock` | Links results to the animal used |

---

## Services

### ferdinand (`api/services/ferdinand`)

A long-running daemon. On startup and every 6 hours it:

1. Fetches the rodeo schedule for the configured date range
2. Skips rodeos already stored
3. Fetches full results for each new rodeo
4. Fetches athlete profiles for any contestants not yet in the store
5. Saves everything under `data/results/rodeo/`

Configured via environment variables (TODO):

| Variable | Default | Description |
|---|---|---|
| `SCRAPE_INTERVAL` | `6h` | How often to run a sync |
| `DATA_DIR` | `data/results/rodeo` | Root directory for stored results |
| `START_DATE` | `1/1/2026` | Start of date range to scrape |
| `END_DATE` | `12/31/2026` | End of date range to scrape |

### sheet (`api/services/sheet`)

An HTTP API that reads from the store and generates announcer PDFs on demand.

| Endpoint | Description |
|---|---|
| `GET /rodeos` | List all stored rodeos |
| `GET /rodeos/{id}` | Full structured data for a rodeo |
| `GET /rodeos/{id}/pdf` | Generate and download an announcer sheet PDF |

---

## Running Locally

```bash
# Fetch dependencies (requires ../ferdinand to exist)
make tidy

# Run the scraper
make run-ferdinand

# Run the sheet API
make run-sheet
```

---

## Milestones

### Milestone 1 — Data Foundation
- [ ] Inspect the `/athlete` endpoint response and update the `Contestant` model in ferdinand
- [ ] Add `FetchAthlete` and `ParseAthlete` to ferdinand
- [ ] Define the canonical domain types in `business/domain/rodeobus/model.go`
- [ ] Write a conversion layer mapping ferdinand's raw types to domain types
- [ ] Validate the file store end-to-end: run the scraper, inspect saved JSON

### Milestone 2 — SQLite Migration
- [ ] Implement the SQLite schema (rodeos, contestants, livestock, results tables)
- [ ] Replace `business/data/store/rodeodb/` with `business/domain/rodeobus/stores/sqlitedb/`
- [ ] Define the `Storer` interface in `business/domain/rodeobus/rodeobus.go`
- [ ] Update the sync flow to write through the SQLite store
- [ ] Verify aggregate fields (`total_earnings`, `total_wins`) update correctly on each sync

### Milestone 3 — Scraper Hardening
- [ ] Load all config from environment variables
- [ ] Handle `InProgress` rodeos: re-scrape until results are final
- [ ] Add rate limiting between `FetchAthlete` calls
- [ ] Add structured error logging and sync summary reporting

### Milestone 4 — Sheet API
- [ ] Flesh out REST endpoints with proper error handling
- [ ] Add CORS headers for browser clients
- [ ] Add a `/sync` endpoint for local copies to pull new records by timestamp
- [ ] Add a `/health` endpoint for deployment health checks

### Milestone 5 — PDF Generation
- [ ] Choose and integrate a PDF library into `foundation/pdf/`
- [ ] Design the announcer sheet layout with the end user
- [ ] Implement `RenderAnnouncerSheet` with real layout (tables, fonts, event sections)
- [ ] Test generated sheets against real rodeo data

### Milestone 6 — Announcer UI
- [ ] Decide on frontend stack (HTMX or React)
- [ ] Build rodeo picker: list rodeos filtered by date and location
- [ ] Wire up PDF preview and download
- [ ] Deploy frontend (static hosting or served from the sheet binary)

### Milestone 7 — Cloud Deployment
- [ ] Deploy ferdinand to AWS (EC2 + EBS volume, or ECS + EFS)
- [ ] Set up Litestream to replicate SQLite to S3
- [ ] Deploy sheet service with public HTTPS endpoint
- [ ] Configure offline sync: local app seeds from S3, stays in sync via `/sync` endpoint

### Milestone 8 — Production Hardening
- [ ] Add authentication to the sheet API (API key minimum)
- [ ] Set up alerting for consecutive sync failures
- [ ] Load test PDF generation under realistic usage
