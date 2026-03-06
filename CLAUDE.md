# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Golf tee time aggregator ("Birdie Alerts") that searches 26+ golf booking platforms in parallel and displays available tee times for municipal/public courses across 20+ US metros. Includes SMS alert system via Twilio.

## Tech Stack

- **Backend:** Go 1.25, standard library only (no external dependencies)
- **Frontend:** Vanilla JS, HTML templates, CSS (no frameworks)
- **SMS:** Twilio API
- **Deployment:** Railway
- **Data:** Course configs embedded via Go `embed` package from `platforms/data/*.json`

## Build & Run

```bash
go build -o golf-teetimes .     # Build binary
go run .                         # Run dev server on :8080
```

No test suite exists. No Makefile. No linter configured.

## Environment Variables

- `TWILIO_ACCOUNT_SID` / `TWILIO_AUTH_TOKEN` / `TWILIO_FROM_NUMBER` — required for SMS alerts

## Architecture

### Request Flow
`main.go` (routes) -> `handlers.go` (handlers + caching) -> `platforms/*.go` (platform fetchers)

### Key Files
- **main.go** — HTTP server, route registration, template loading
- **handlers.go** — Request handlers, in-memory cache (5min TTL), singleflight dedup
- **metros.go** — Metro definitions (slug, display name, state, coordinates, tagline)
- **alerts.go** — Alert CRUD, Twilio SMS integration, background alert checker (1min interval)
- **platforms/registry.go** — Global course registry
- **platforms/data.go** — Loads embedded JSON configs into registry at init
- **platforms/types.go** — Shared types (TeeTime, Course, etc.)
- **platforms/{platform}.go** — Per-platform fetch implementations (26+ platforms)
- **static/app.js** — Client-side tee time filtering, sorting, display
- **static/alerts.js** — Alert CRUD UI

### Platform System
Each platform (TeeItUp, ForeUp, Chronogolf, GolfNow, etc.) has its own `.go` file in `platforms/` with a fetch function and a corresponding JSON config in `platforms/data/`. Courses are registered in a global registry keyed by metro slug. Each course exists on exactly one platform (cross-platform dedup).

### Alert System
Alerts stored in `alerts.json` on disk. Background goroutine checks every minute, matches tee times against criteria (course, date, time window, players, holes), sends SMS via Twilio, then deactivates.

## Adding a New Metro

Follow the 5-phase discovery process documented in `DISCOVERY.md`:
1. Compile course list in `discovery/courses/{metro}.txt`
2. Run discovery tools in `cmd/discover-*` against each platform
3. Gap-fill with website scraping
4. Search GolfNow area API
5. Check GolfNow courses for direct platform availability

Discovery tools: `go run cmd/discover-{platform}/main.go`

## Adding a New Platform

1. Create `platforms/{platform}.go` with fetch function
2. Create `platforms/data/{platform}.json` with course configs
3. Register in `platforms/data.go` init function
4. Each course config needs: key, metro, displayName, city, state, plus platform-specific IDs

## Conventions

- **Course keys:** lowercase, hyphenated, no suffixes like "-golf-course" (e.g., `aguila` not `aguila-golf-course`)
- **Multi-course clubs:** Display name format is `"ClubName - CourseName"`
- **Platform precedence:** Direct platform APIs preferred over TeeItUp; GolfNow used as fallback
- **Routes:** `/{metro}` for pages, `/{metro}/teetimes?date=YYYY-MM-DD` for JSON API
- **Templates:** Go `html/template` in `templates/` directory
