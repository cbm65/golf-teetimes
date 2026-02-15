# Adding a New Metro — Process

## Overview

Three-phase process to add all public courses for a new metro area. Designed to automate ~90% of course discovery and minimize manual work.

## Phase 1: Course List

Compile a master list of every public golf course in the metro area and save it to
`discovery/courses/{metro}.txt` — one course per line, `#` comments ignored.
Format: `Course Name | City` (city is needed for slug-based discovery tools).

**Input:** Metro name (e.g., "Phoenix, AZ")
**Output:** `discovery/courses/phoenix.txt`

### Course List Files

```
discovery/courses/
├── denver.txt          # TODO
├── phoenix.txt         # 92 courses
├── lasvegas.txt        # TODO
└── ...
```

## Phase 2: Automated Discovery

Run each platform's discovery tool against the course list file. Each tool probes
its platform's API, then validates confirmed matches with a 3-date tee time check
(next Wednesday, next Saturday, Saturday after that).

### Step 1: Run TeeItUp FIRST

TeeItUp is the dominant platform (78% of Phoenix courses) and should always run first.
Its results establish the **baseline** — when a course appears on both TeeItUp and
another platform, the other platform wins (direct booking links preferred over TeeItUp).
All subsequent scripts should check for TeeItUp overlaps and replace them.

```bash
# 1. TeeItUp — ALWAYS RUN FIRST — alias probe with multi-facility sibling matching
go run cmd/discover-teeitup/main.go AZ -f discovery/courses/phoenix.txt

# 2. GolfWithAccess (Troon) — slug probe, high value for resort/sunbelt courses
go run cmd/discover-golfwithaccess/main.go AZ -f discovery/courses/phoenix.txt

# 3. Chronogolf — slug probe against club pages
go run cmd/discover-chronogolf/main.go AZ -f discovery/courses/phoenix.txt

# 4. ForeUP — two-phase: one-time index build, then match against index
go run cmd/discover-foreup/main.go --index 1 30000
go run cmd/discover-foreup/main.go --match AZ -f discovery/courses/phoenix.txt

# 5. ClubCaddie — NO discovery script (HAR capture only, see Phase 3)

# 6. Quick18 — subdomain probe
go run cmd/discover-quick18/main.go AZ -f discovery/courses/phoenix.txt

# 7. CourseCo — subdomain probe (only 1 known course so far, no discovery script yet)
go run cmd/discover-courseco/main.go AZ -f discovery/courses/phoenix.txt

# 8. CPS Golf — subdomain probe against {slug}.cps.golf
go run cmd/discover-cpsgolf/main.go AZ -f discovery/courses/phoenix.txt
```

Each tool outputs a JSON results file to `discovery/results/` with statuses:
- **confirmed** — course found on platform with live tee times
- **listed_only** — course page exists but 0 tee times (directory listing, not actively booking)
- **third_party_backend** — (ForeUP only) ForeUP is the teesheet backend but another platform handles consumer booking
- **wrong_state** — slug matched a course in another state (rejected)
- **wrong_city** — slug matched the right state but wrong city (warning, not rejected)
- **miss** — no match found

### Step 2: Ingest results — CRITICAL: cross-platform dedup

When adding confirmed courses to `platforms/data/*.json`, you MUST check whether
each course is already present on another platform. A course should only appear on
ONE platform. **Other platforms take precedence over TeeItUp** — if a course exists on
both TeeItUp AND another platform (ForeUP, Chronogolf, GolfWithAccess, etc.), remove
it from TeeItUp and use the other platform. Direct booking links through smaller
platforms are preferred. For collisions between two non-TeeItUp platforms, whichever
platform the course actually books through is the one we use.

**GolfNow exclusion:** We intentionally skip GolfNow-only courses. Our value
proposition is direct booking links (no middleman markup). Most GolfNow courses are
already discoverable through TeeItUp since many courses use GolfNow as a backend
while TeeItUp handles consumer-facing booking.

Ingestion checklist for each confirmed course:
1. Is this course already in any `platforms/data/*.json`? → **SKIP**
2. Is this course in the correct metro area? (e.g. Tucson courses are NOT Phoenix) → **SKIP**
3. Is the name match actually correct? (e.g. "Wickenburg Ranch" ≠ "Wickenburg Country Club") → **SKIP if wrong**
4. Clean the display name per rules below
5. Derive the key per rules below
6. Add to the appropriate JSON file

### Step 3: Verify and test

```bash
go run .
```

Visit `http://localhost:8080/phoenix` and verify courses appear in the dropdown.

## Key Derivation Rules

The `key` field in every JSON entry must follow these rules:

1. Start with the display name
2. Strip golf suffixes: "Golf Course", "Golf Club", "Golf Resort", "Country Club", "Golf Complex", "Golf Links", "Golf Center", "GC", "CC"
3. Strip leading "The ", "Golf Club of ", "Golf Club at "
4. Lowercase
5. Replace non-alphanumeric with hyphens, collapse multiples, trim edges

Examples:
```
Aguila Golf Course              → aguila
Stonecreek Golf Club            → stonecreek
Red Mountain Ranch Country Club → red-mountain-ranch
Golf Club of Estrella           → estrella
The Phoenician Golf Club        → phoenician
Trilogy Golf Club at Vistancia  → trilogy-at-vistancia
Painted Mountain Golf Resort    → painted-mountain
```

Keys must be unique across all platforms within a metro. Use hyphens, not underscores.

## Display Name Normalization Rules

All course names must follow these conventions so the frontend dropdown groups
multi-course clubs correctly and displays clean names.

### The " - " convention

Multi-course clubs use `"ClubName - CourseName"` format. The frontend
`getBaseCourse()` splits on `" - "` to group sub-courses under one dropdown entry.

Examples:
```
Arizona Biltmore - Estates      → dropdown shows "Arizona Biltmore"
Arizona Biltmore - Links        → grouped under "Arizona Biltmore"
Wigwam - Gold                   → dropdown shows "Wigwam"
Wigwam - Blue                   → grouped under "Wigwam"
```

Single-course clubs use their full name with no separator:
```
Aguila Golf Course
Coldwater Golf Club
Starfire Golf Club
```

### Cleanup rules

When adding courses from discovery results, clean display names:

1. **Remove state/region suffixes**: "Longbow Golf Club, AZ" → "Longbow Golf Club"
2. **Remove "(AZ)" or "(state)" suffixes**: "Rolling Hills Golf Course (AZ)" → "Rolling Hills Golf Course"
3. **Remove internal sub-course names for single-facility entries**: "Talking Stick Golf Club - Piipaash (South)" → "Talking Stick Golf Club"
4. **Expand abbreviations**: "Lookout Mountain G.C." → "Lookout Mountain Golf Club"
5. **Add missing suffixes**: "Palo Verde" → "Palo Verde Golf Course", "Paradise Valley Golf" → "Paradise Valley Golf Course"
6. **No trailing whitespace** in names or map keys
7. **Chronogolf `names` map**: Keys are the API course name (e.g. `"Estates"`), values use the " - " convention (e.g. `"Arizona Biltmore - Estates"`)

### Common pitfalls caught during Phoenix ingestion

These are real mistakes we made and corrected:
- `Longbow Golf Club, AZ` — had state suffix left over from API
- `Lookout Mountain G.C.` — abbreviation not expanded
- `Talking Stick Golf Club - Piipaash (South)` — sub-course name not stripped
- `Palo Verde` — missing "Golf Course" suffix entirely
- `The Phoenician` — missing "Golf Club" suffix
- `Paradise Valley Golf` — incomplete suffix ("Golf" without "Course")
- Keys were using underscores and including golf suffixes (e.g. `aguila_golf_course` instead of `aguila`)

## Discovery Script Architecture

All 7 scripts share common patterns that emerged through trial and error:

### Common features across all scripts

- **3-date tee time validation**: probes next Wednesday, next Saturday, Saturday+7 to confirm active booking
- **State validation**: every script validates that matched courses are in the target state (critical — without this, generic names like "Stonecreek" match courses in Oregon, "Riverview" matches California, "Foothills" matches Colorado)
- **City validation**: warns on mismatch but does NOT reject (facility cities often differ from input — e.g. "Laveen" vs "Phoenix" for Aguila)
- **Multi-alias/slug generation**: tries 8-20+ naming variants per course to maximize hit rate
- **Dead slug/alias caching**: tracks 404'd slugs to skip them in subsequent iterations
- **Deduplication**: tracks discovered facility/club/course IDs to prevent the same course being recorded twice
- **Alias/slug source tracking**: records which naming pattern matched (e.g. "swap-golf-club", "core+state+city") for debugging
- **JSON results output**: saves full results to `discovery/results/` for ingestion
- **Timestamped log output**: prefixes every line with `[HH:MM:SS.mmm]` for debugging timing issues

### Alias/slug generation patterns

Each platform has different URL conventions, but the name-munging patterns are similar.
Starting from an input like "Stonecreek Golf Club" in Phoenix, AZ:

| Pattern | Example | Used by |
|---------|---------|---------|
| Exact name slug | `stonecreek-golf-club` | TeeItUp, Chronogolf |
| Name + state + city | `stonecreek-golf-club-arizona-phoenix` | Chronogolf |
| Name + state | `stonecreek-golf-club-arizona` | Chronogolf |
| Suffix swap (course↔club↔resort) | `stonecreek-golf-course` | TeeItUp, Chronogolf |
| Core name only | `stonecreek` | TeeItUp, Quick18 |
| Core + "golf" | `stonecreekgolf` | Quick18 |
| Core + city | `stonecreek-phoenix` | TeeItUp |
| "The" prefix removal | `phoenician-golf-club` | All |
| "The" prefix addition | `the-legacy-golf-club` | TeeItUp |
| City prefix stripping | `silverado-golf-club` (from "Scottsdale Silverado") | TeeItUp |
| Hyphenated full | `stonecreek-golf-club` | Quick18 |
| Joined alpha | `stonecreekgolfclub` | Quick18 |
| "gc" abbreviation | `stonecreek-gc` | TeeItUp |

### Fuzzy matching guidelines

For scripts that match input names against an index (ForeUP) or API-returned names (TeeItUp sibling matching):

- **Normalize both sides**: strip golf suffixes, "the", punctuation, lowercase, collapse whitespace
- **Minimum length guard**: require 5+ chars for containment matches to prevent false positives ("mesa" should NOT match "mesa verde")
- **City-aware matching**: when input includes city (e.g. "Raven Golf Club Phoenix"), strip city before comparing since API names don't include it
- **Skip prepositions**: ignore "at", "of", "the", "in", "and" when doing word-overlap matching
- **Watch for same-city same-prefix collisions**: "Wickenburg Ranch Golf Club" vs "Wickenburg Country Club" are DIFFERENT courses — the fuzzy matcher must not conflate them

### Platform-specific notes

**TeeItUp** — Uses Kenna Commerce API. The discovery script generates 15-22 alias candidates per course. Alias probing hits `https://phx-api-be-east-1b.kenna.io/facilities` with `x-be-alias` header. Multi-facility aliases (e.g. `city-of-phoenix-golf-courses`) return all facilities — the script cross-matches all siblings against the full input list. Single-course aliases that work during discovery may fail in production if the course is actually booked under a shared alias — always prefer the shared alias when one exists. During Las Vegas ingestion, added `exactStatus` field to miss results for diagnosing why specific courses weren't found (e.g. Kenna API returned no facility for the slug despite it being a valid TeeItUp course).

**Booking site HTML fallback** (added during Atlanta ingestion): The Kenna `/facilities` API has coverage gaps — some valid aliases return 404 from the API but work fine at `{alias}.book.teeitup.com`. Stone Mountain Golf Club was discovered this way (alias `stone-mountain` returned 404 from Kenna but the booking site HTML contained all facility data). The fallback probes `{alias}.book.teeitup.com` directly, extracts the confirmed alias from `<input id="alias" value="...">`, and parses facility IDs/names from embedded RSC JSON (`gCPlayFacilityId`, `gFName`). Only tried for the top 2 alias candidates (core name, exact slug) to limit extra HTTP requests.

**Alternate name limitation**: Courses known by a different name than their official name (e.g. "North Fulton Golf Course" booking as "Chastain Park") cannot be discovered automatically. These require manual HAR capture in Phase 3.

**UUID alias variant** (discovered during Atlanta ingestion): Some TeeItUp courses use `book-v2.teeitup.golf` with a UUID subdomain instead of a human-readable alias (e.g. College Park Golf Course uses `c70350ae-...`). The Kenna API works identically — the UUID is passed as `x-be-alias`. These are completely undiscoverable by alias probing; HAR capture is the only path.

**Chronogolf** — Owned by Lightspeed. Has a massive directory (57 "listed_only" out of 92 for Phoenix) but almost no courses actually book through it (only 2 confirmed for Phoenix). Most Chronogolf listings are SEO/marketplace directory entries — the courses actually book through TeeItUp, GolfNow, etc. Discovery probes `https://www.chronogolf.com/club/{slug}` and extracts `__NEXT_DATA__` JSON. The `names` map translates API course names to display names. Multi-course clubs return tee times tagged with the course name from the API (e.g. "Estates"), which gets mapped to the display name (e.g. "Arizona Biltmore - Estates"). The lookup is case-insensitive with whitespace trimming.

**ForeUP** — Two-phase discovery: (1) build a master index by sweeping course_id 1–30000 (one-time, ~30min, checkpoint every 500), (2) match input courses against the index by state + fuzzy name. Many courses use ForeUP as a teesheet backend while a different platform (TeeItUp, GolfNow, etc.) handles consumer-facing booking. The discovery script detects this by checking for a booking class named "Online Third Party". If present, the course is marked `third_party_backend` and should NOT be added as a ForeUP course. The script deduplicates by `course_id` and uses a 5-char minimum guard for fuzzy matching. ForeUP's booking URL does not support date pre-fill (Angular SPA).

**ClubCaddie** — Has no central directory, API, or enumerable ID scheme. Each course lives at `apimanager-cc{server}.clubcaddie.com/webapi/view/{apiKey}` where both `{server}` and `{apiKey}` are non-sequential. Discovery is **manual only** via HAR capture from the course's booking page. When capturing a HAR, the key fields to extract are: the server number (from the hostname), the apiKey (from the URL path), and the courseId (from the TeeTimes POST body). The booking URL supports date pre-fill: `https://apimanager-cc{server}.clubcaddie.com/webapi/view/{apiKey}/slots?date={MM/DD/YYYY}`.

**CPS Golf** — Subdomain-based: `{slug}.cps.golf`. Discovery probes the unauthenticated `/onlineresweb/Home/Configuration` endpoint — if it returns valid JSON with `siteName`, the site exists. Then `/OnlineCourses` returns course details including `websiteId`, `courseId`, `courseName`, `timezoneId`, and `holes`. State validation uses the timezone (e.g. `America/Denver` → CO). A single CPS Golf site can host multiple courses (e.g. Cascata + Serket on `caspublic.cps.golf`).

**CPS Golf has two auth modes** discovered during Las Vegas ingestion:
- **Older sites** (e.g. Indian Tree, Green Valley Ranch): Configuration returns an `apiKey` field. The fetcher sends it as `x-apikey` header. No bearer token needed.
- **Newer sites** (e.g. Cascata, Serket, Fossil Trace): Configuration has NO `apiKey`. These require a short-lived Bearer token obtained from `POST /identityapi/myconnect/token/short` with form body `client_id=onlinereswebshortlived`. Returns a JWT valid for 10 minutes. The fetcher gets a fresh token each cycle.

The fetcher dynamically detects which mode to use: it fetches Configuration first, and if no `apiKey` is present, it obtains a bearer token. It also dynamically fetches `courseIds` from OnlineCourses when not specified in config. This means `apiKey` and `courseIds` can both be empty in the JSON config — the fetcher will figure it out at runtime.

**CPS Golf discovery limitation**: subdomain patterns are unpredictable. Cascata lives at `caspublic.cps.golf`, not `cascata.cps.golf`. The discovery script's slug probe won't find these — they require manual HAR capture (Phase 3).

The booking URL is `https://{slug}.cps.golf/onlineresweb/search-teetime` and supports date pre-fill via `?Date={YYYY-MM-DD}`.

**Quick18** — Subdomain-based: `{subdomain}.quick18.com/teetimes/searchmatrix?teedate={YYYYMMDD}`. Generates 8-12 subdomain candidates per course (joined alpha, hyphenated, with/without suffixes). State validation parses the page HTML for address patterns like `, AZ 85XXX` since there's no structured state field in the response. This is critical — without it, generic subdomains like `stonecreek` match courses in Oregon. Dead subdomain caching avoids re-probing known-bad subdomains. During Las Vegas ingestion, added `" - "` variant slug pattern that uses just the base facility name (e.g. "Angel Park" from "Angel Park - Palm Course") — important for multi-course facilities where the subdomain is the club name, not the course name.

**GolfWithAccess (Troon)** — Troon's booking platform (largest golf management company, 900+ courses worldwide). Centralized at `golfwithaccess.com/course/{slug}/reserve-tee-time`. Page HTML contains React Server Components data with courseId UUID, city, and state (full name like "Arizona" → converted to "AZ"). Tee times at `/api/v1/tee-times?courseIds={uuid}&players=1&startAt=00:00:00&endAt=23:59:59&day={date}`. High value for resort/sunbelt markets where Troon is heavily concentrated. Generates slug candidates with suffix swaps and "the-" prefix.

**CourseCo** — Uses `{subdomain}.totaleintegrated.net`. Subdomains are the course name joined without separators (e.g. `kenmcdonald`). CourseID is the uppercase of the subdomain. API at `courseco-gateway.totaleintegrated.net/Booking/Teetimes?CourseID={ID}&TeeTimeDate={date}`. Only 1 known course so far (Ken McDonald) — **need more examples before the discovery script is reliable**.

**Denver (MemberSports)** — The API returns raw names like "Kennedy Links". The `normalizeDenverName()` function in `denver.go` converts known multi-course prefixes to the " - " convention (e.g. "Kennedy Links" → "Kennedy - Links").

## Phoenix Discovery Results (Feb 2026)

First full metro discovery run. 92 input courses, 62 confirmed across 9 platforms:

| Platform | Confirmed | Listed Only | Wrong State | Misses | New Courses Added |
|----------|-----------|-------------|-------------|--------|-------------------|
| TeeItUp | 41 | 24 | 0 | 29 | 41 |
| GolfWithAccess | 12 | 1 | 0 | 79 | 3 (10 overlapped TeeItUp) |
| Chronogolf | 2 | 57 | 11 | 22 | 2 |
| ForeUP | 5 | 16 | 0 | 70 | 1 (4 overlapped TeeItUp) |
| Quick18 | 9 | 5 | 2 | 76 | 8 (1 overlapped TeeItUp) |
| CourseCo | 1 | — | — | — | 1 (manual HAR) |
| RGuest | 4 | — | — | — | 4 (manual HAR) |
| TeeSnap | 1 | — | — | — | 1 (manual HAR) |
| ClubCaddie | pending | — | — | — | — |

Key takeaways:
- **TeeItUp dominates Phoenix** — 41 of 62 courses (66%), should always run first
- GolfWithAccess (Troon) found 12 but 10 overlapped TeeItUp — still added 3 new courses
- Chronogolf has massive directory coverage but almost no active booking (2 of 57 listed)
- ForeUP confirmed 5 but 3 were already on TeeItUp and 1 was wrong city — only 1 genuinely new
- Quick18 found 8 new courses that no other platform had
- Manual HAR capture filled 6 more courses across CourseCo, RGuest, and TeeSnap
- State validation prevented 13 wrong-state false positives across all scripts
- Cross-platform dedup prevented 15+ duplicate entries
- **GolfNow intentionally skipped** — most GolfNow courses already found via TeeItUp
- TeeItUp discovery script improved during Phoenix run: added "the-" prefix generation (caught The Legacy) and city-stripping (caught Scottsdale Silverado)

## Las Vegas Discovery Results (Feb 2026)

42 input courses, 4 confirmed across 3 platforms (most Las Vegas courses are GolfNow-only):

| Platform | Confirmed | Notes |
|----------|-----------|-------|
| TeeItUp | 1 | Boulder City Golf Course (manual HAR — discovery script missed it due to Kenna API probe gap) |
| Quick18 | 1 | Angel Park (manual HAR — multi-course with Palm/Mountain, uses `" - "` variant slug) |
| CPS Golf | 2 | Cascata + Serket on caspublic.cps.golf (manual HAR — subdomain not guessable) |

Key takeaways:
- **Las Vegas is heavily GolfNow** — most courses only book through GolfNow, which we intentionally skip
- **CPS Golf auth fix was the big win** — the bearer token discovery (`/myconnect/token/short`) fixed all 5 CPS Golf courses across both Denver and Las Vegas
- **CPS Golf subdomains are unpredictable** — Cascata/Serket live at `caspublic.cps.golf`, not `cascata.cps.golf`. Discovery scripts can't find these; manual HAR is required.
- **Multi-course CPS Golf sites** need separate JSON entries per course with specific `courseIds` values (e.g. Cascata=4, Serket=5) but shared `baseUrl`, `websiteId`, and `siteId`
- **TeeItUp discovery misses** can be diagnosed with the `exactStatus` field in miss results (added during this metro). Boulder City was missed because the Kenna API probe returned no facility for its slug.
- **Quick18 discovery** improved with `" - "` variant slug pattern (base facility name without course suffix)

## HAR Capture Tips

When doing Phase 3 manual gap-fill, proper HAR capture is critical:

### Chrome sanitized vs unsanitized HARs

Chrome's **Export HAR (sanitized)** button strips `Authorization`, `Cookie`, and other sensitive headers. This is usually fine for identifying platforms and URL patterns, but **will hide auth tokens** needed to understand authentication flows.

**To get unsanitized HARs:**
- **Firefox** (easiest): DevTools → Network → gear icon → "Save All As HAR" — always unsanitized
- **Chrome workaround**: Right-click a specific request → Copy → Copy as cURL — includes all headers including Authorization

**When you need unsanitized:** Only when debugging auth failures (401s) where you need to see the actual Bearer token or cookie values. For normal platform identification, sanitized HARs work fine.

### Incognito captures

Always use **incognito/private browsing** for HAR captures to ensure a clean session with no cached tokens. This shows the complete auth flow from scratch.

### Fetch/XHR filter

If the full HAR is too large (common on marketing-heavy golf sites), use the **Fetch/XHR** filter in the Network tab before exporting. This strips out images, CSS, JS, and fonts, keeping only API calls.

## Adding Courses to JSON

### TeeItUp entry format

```json
{
  "key": "aguila",
  "metro": "phoenix",
  "alias": "aguila-golf-course",
  "facilityId": "287",
  "displayName": "Aguila Golf Course",
  "city": "Laveen",
  "state": "AZ"
}
```

Fields from discovery results:
- `key` — derived from display name per key rules above
- `metro` — target metro slug
- `alias` — from discovery results `alias` field (use shared alias when one exists)
- `facilityId` — from discovery results `facility.id` (**must be a string**, not a number — e.g. `"287"` not `287`)
- `displayName` — from discovery results `facility.name`, cleaned per rules above
- `city` — from discovery results `facility.locality`
- `state` — target state code

### Chronogolf entry format

```json
{
  "key": "arizona-biltmore",
  "metro": "phoenix",
  "courseIds": "8e62fd78-03bf-4665-9a3a-cc0da2826ac7,768df054-3367-45b1-906c-24af39e410ad",
  "clubId": "",
  "numericCourseId": "",
  "affiliationTypeId": "82990",
  "bookingUrl": "https://www.chronogolf.com/club/arizona-biltmore-golf-club-arizona-phoenix",
  "names": {
    "Estates": "Arizona Biltmore - Estates",
    "Links": "Arizona Biltmore - Links"
  },
  "city": "Phoenix",
  "state": "AZ"
}
```

Fields from discovery results:
- `courseIds` — comma-separated UUIDs from `club.courses[].uuid`
- `affiliationTypeId` — from `club.defaultAffiliationTypeId`
- `bookingUrl` — `https://www.chronogolf.com/club/{slug}`
- `names` — map of API course name → display name (use " - " convention for multi-course)

### ForeUP entry format

```json
{
  "key": "painted-mountain",
  "metro": "phoenix",
  "courseId": "21954",
  "bookingClass": "12668",
  "scheduleId": "9443",
  "bookingUrl": "",
  "displayName": "Painted Mountain Golf Resort",
  "city": "Mesa",
  "state": "AZ"
}
```

Fields from discovery results:
- `courseId` — from discovery results `courseId` field
- `bookingClass` — from discovery results `bookingClassId` field (the active, non-third-party class)
- `scheduleId` — from discovery results `scheduleId` field
- `bookingUrl` — leave empty (URL is constructed at runtime)
- `displayName` — from discovery results `name` field, cleaned per rules above

### Quick18 entry format

```json
{
  "key": "papago",
  "metro": "phoenix",
  "subdomain": "papago",
  "bookingUrl": "https://papago.quick18.com",
  "displayName": "Papago Golf Course",
  "city": "Phoenix",
  "state": "AZ"
}
```

Fields from discovery results:
- `subdomain` — the confirmed Quick18 subdomain
- `bookingUrl` — `https://{subdomain}.quick18.com`
- `displayName` — input course name (already clean if course list was curated)
- Optional fields: `domain` (if non-standard), `namePrefix` (for multi-course), `holes` (if fixed)

### ClubCaddie entry format

```json
{
  "key": "the-links",
  "metro": "denver",
  "baseUrl": "https://apimanager-cc37.clubcaddie.com",
  "apiKey": "ajfdabab",
  "courseId": "103491",
  "bookingUrl": "https://apimanager-cc37.clubcaddie.com/webapi/view/ajfdabab",
  "displayName": "The Links Golf Course",
  "city": "Highlands Ranch",
  "state": "CO"
}
```

Fields from discovery results:
- `baseUrl` — `https://apimanager-cc{server}.clubcaddie.com`
- `apiKey` — from discovery results `apiKey` field
- `courseId` — from discovery results `courseId` field (extracted from slots page, as string)
- `bookingUrl` — `https://apimanager-cc{server}.clubcaddie.com/webapi/view/{apiKey}`

### CPS Golf entry format

```json
{
  "key": "indian-tree",
  "metro": "denver",
  "baseUrl": "https://indiantree.cps.golf",
  "apiKey": "8ea2914e-cac2-48a7-a3e5-e0f41350bf3a",
  "websiteId": "e6d9cd59-8d46-4334-8601-08dad3012d25",
  "siteId": "1",
  "courseIds": "",
  "bookingUrl": "https://indiantree.cps.golf/onlineresweb/search-teetime",
  "names": {
    "Regulation 18": "Indian Tree Golf Club"
  },
  "city": "Arvada",
  "state": "CO",
  "timezone": "America/Denver"
}
```

Fields from discovery results or HAR:
- `baseUrl` — `https://{slug}.cps.golf` (NOTE: slug may not match course name — e.g. Cascata is at `caspublic.cps.golf`)
- `apiKey` — from Configuration response `.apiKey`. **Can be empty** — newer sites have no apiKey and use bearer token auth instead (fetcher handles this automatically)
- `websiteId` — from OnlineCourses response `[0].websiteId`
- `siteId` — always `"1"` (site-level ID, NOT the course-level siteId from OnlineCourses)
- `courseIds` — specific courseId int to fetch, or **empty string** to auto-detect from OnlineCourses at runtime. Use specific IDs when a site hosts multiple courses and you want one entry per course (e.g. Cascata=4, Serket=5 on caspublic.cps.golf)
- `bookingUrl` — `https://{slug}.cps.golf/onlineresweb/search-teetime` (date appended automatically as `?Date=YYYY-MM-DD`)
- `names` — map CPS course name → display name (e.g. "Regulation 18" → "Indian Tree Golf Club")
- `timezone` — from OnlineCourses response `[0].timezoneId`. Used as `x-timezoneid` header (should match user's timezone, not course location — but America/Denver works for all current sites)

### GolfWithAccess (Troon) entry format

```json
{
  "key": "quintero",
  "metro": "phoenix",
  "courseIds": ["416b2e7c-83c1-498c-8958-2422033218c2"],
  "slug": "quintero-golf-club",
  "bookingUrl": "https://golfwithaccess.com/course/quintero-golf-club/reserve-tee-time",
  "displayName": "Quintero Golf Club",
  "city": "Peoria",
  "state": "AZ"
}
```

Fields from discovery results or HAR:
- `courseIds` — array of UUID(s) from page HTML RSC data
- `slug` — the confirmed URL slug (may differ from input name, e.g. `the-westin-kierland-golf-club`)
- `bookingUrl` — `https://golfwithaccess.com/course/{slug}/reserve-tee-time`

### CourseCo entry format

```json
{
  "key": "ken-mcdonald",
  "metro": "phoenix",
  "subdomain": "kenmcdonald",
  "courseId": "KENMCDONALD",
  "bookingUrl": "https://kenmcdonald.totaleintegrated.net",
  "displayName": "Ken McDonald Golf Course",
  "city": "Tempe",
  "state": "AZ"
}
```

Fields from HAR:
- `subdomain` — the subdomain on `totaleintegrated.net`
- `courseId` — uppercase of subdomain (confirmed pattern from 1 example)
- `bookingUrl` — `https://{subdomain}.totaleintegrated.net`

### RGuest entry format

```json
{
  "key": "camelback-ambiente",
  "metro": "phoenix",
  "tenantId": "2281",
  "propertyId": "camelback-golf-club",
  "courseId": "410",
  "playerTypeId": "1560",
  "bookingUrl": "https://book.rguest.com/onecart/golf/courses/2281/camelback-golf-club",
  "displayName": "Camelback Golf Club - Ambiente",
  "city": "Scottsdale",
  "state": "AZ"
}
```

Fields from HAR (no discovery script — manual only):
- `tenantId` — numeric tenant ID from RGuest URL
- `propertyId` — property slug from RGuest URL
- `courseId` — from `getAvailableCourses` API response
- `playerTypeId` — from `getAvailableTeeSlots` response (use public/online rate type)
- `bookingUrl` — `https://book.rguest.com/onecart/golf/courses/{tenantId}/{propertyId}`

### TeeSnap entry format

```json
{
  "key": "sundance",
  "metro": "phoenix",
  "subdomain": "sundancegolfclub",
  "courseId": "1801",
  "bookingUrl": "https://sundancegolfclub.teesnap.net",
  "displayName": "Sundance Golf Club",
  "city": "Buckeye",
  "state": "AZ"
}
```

Fields from HAR (no discovery script yet — need more examples):
- `subdomain` — the subdomain on `teesnap.net`
- `courseId` — from `customer-api/teetimes-day?course={id}` URL
- `bookingUrl` — `https://{subdomain}.teesnap.net`

## Phase 3: Manual Gap Fill

After automated discovery, compare the master list against what was found. For each missing course:

1. Visit the course's booking page in a browser
2. Capture a HAR file of the tee time loading request
3. Give Claude the HAR — Claude identifies the platform and extracts the config
4. Claude adds the course to the appropriate `platforms/data/*.json`

This catches:
- Courses with non-obvious aliases/slugs/subdomains
- Courses on platforms without discovery scripts (ClubCaddie, RGuest, etc.)
- Private/semi-private courses with unusual booking configurations

**Input:** HAR files for each missing course
**Output:** Remaining courses added to `platforms/data/*.json`

### Platforms without discovery scripts (need HAR)

These platforms require manual HAR capture — either because they have no guessable URL pattern, require authentication/session tokens to discover, or we don't have enough examples yet:

| Platform | Why no discovery script |
|----------|----------------------|
| ClubCaddie | No central directory, API, or enumerable ID scheme. Server numbers and API keys are random. Only discoverable via HAR capture from the course's booking page. |
| RGuest | Numeric tenant IDs not derivable from course names. Hotel/resort platform (Agilysys) — only a handful of golf courses per metro. Manual HAR is faster. |
| TeeSnap | Subdomain-probeable (`{slug}.teesnap.net`) but only 1 known course so far. **Need more examples to validate patterns before building script.** |
| CourseCo | Has a discovery script (`discover-courseco`) but only 1 known course. **Need more examples to validate patterns before relying on it.** |
| MemberSports | Denver-specific so far, API known but no discovery script built |
| EZLinks | No known discovery pattern |
| CourseRev | No known discovery pattern |
| GolfNow | **Intentionally skipped.** Most GolfNow courses already found via TeeItUp. Value proposition is direct booking links (no middleman markup). |

## Adding a Metro Entry

Before running discovery, add the metro to `metros.go`:

```go
"phoenix": {
    Name:    "Phoenix",
    Slug:    "phoenix",
    State:   "AZ",
    Tagline: "Public Courses Across Metro Phoenix",
},
```

Course count and city count are computed automatically from the JSON data at startup.

## Discovery Tools Status

| Platform | Tool | Status |
|----------|------|--------|
| TeeItUp | discover-teeitup | ✅ Built — multi-alias probe (15-22 candidates), sibling matching, fuzzyMatchWithCity, "the-" prefix, city-stripping. **Run first — establishes baseline.** Other platforms take precedence when overlaps found. |
| GolfWithAccess | discover-golfwithaccess | ✅ Built — slug probe, RSC page extraction for courseId/city/state, suffix swaps, "the-" prefix |
| Chronogolf | discover-chronogolf | ✅ Built — slug probe with name+state+city patterns, suffix swaps, __NEXT_DATA__ extraction |
| ForeUP | discover-foreup | ✅ Built — two-phase index+match, third-party detection, 5-char fuzzy guard, course_id dedup |
| ClubCaddie | — | No script — manual HAR capture only (no central directory or enumerable IDs) |
| Quick18 | discover-quick18 | ✅ Built — subdomain probe (8-12 candidates), HTML state validation, dead subdomain caching |
| CourseCo | discover-courseco | ⚠️ Built but unvalidated — only 1 known course. Need more examples to confirm patterns. |
| TeeSnap | — | ⏳ Pending — subdomain-probeable but only 1 known course. Need more examples. |
| RGuest | — | ❌ Not feasible — numeric tenant IDs, not guessable |
| GolfNow | — | ❌ Intentionally skipped — direct booking value prop, most courses found via TeeItUp |
| CPS Golf | discover-cpsgolf | ✅ Built — subdomain probe ({slug}.cps.golf/onlineresweb/Home/Configuration), OnlineCourses extraction, timezone state validation. **Limitation**: subdomains are unpredictable (e.g. `caspublic` for Cascata) — manual HAR needed for non-obvious slugs. Fetcher handles both apiKey and bearer token auth automatically. |
| MemberSports | — | ❌ API known from Denver, script not built |

## File Structure

```
cmd/
├── discover-teeitup/
│   └── main.go              # TeeItUp discovery — RUN FIRST (alias probe + sibling matching)
├── discover-golfwithaccess/
│   └── main.go              # GolfWithAccess/Troon discovery (slug probe + RSC extraction)
├── discover-chronogolf/
│   └── main.go              # Chronogolf discovery (slug probe + __NEXT_DATA__)
├── discover-foreup/
│   └── main.go              # ForeUP discovery (index build + fuzzy match)
├── discover-quick18/
│   └── main.go              # Quick18 discovery (subdomain probe)
├── discover-courseco/
│   └── main.go              # CourseCo discovery (subdomain probe — needs more examples)
├── discover-cpsgolf/
│   └── main.go              # CPS Golf discovery (subdomain probe + Configuration/OnlineCourses API)
└── ...

discovery/
├── courses/
│   ├── phoenix.txt          # Phase 1 course list (92 courses)
│   ├── lasvegas.txt         # Phase 1 course list (42 courses)
│   ├── denver.txt           # (TODO)
│   └── ...
├── foreup-index.json        # ForeUP master index (4188 courses, IDs 1-30000)
└── results/                 # Auto-generated discovery results (gitignored)

platforms/data/
├── teeitup.json        # 41 courses (Phoenix) + 1 (Las Vegas)
├── golfwithaccess.json # 3 courses (Phoenix)
├── chronogolf.json     # 2 courses (Phoenix)
├── foreup.json         # 1 course (Phoenix)
├── quick18.json        # 8 courses (Phoenix) + 1 (Las Vegas)
├── courseco.json       # 1 course (Phoenix)
├── rguest.json         # 4 courses (Phoenix)
├── teesnap.json        # 1 course (Phoenix)
├── clubcaddie.json     # Denver courses
├── cpsgolf.json        # 3 courses (Denver) + 2 (Las Vegas)
├── golfnow.json        # (intentionally empty — skipping GolfNow-only courses)
└── ...
```

Each JSON file is an array of course configs. Every entry has `key`, `metro`, `city`, and `state` plus platform-specific fields. Adding a course is a single JSON entry — no other files need editing.
