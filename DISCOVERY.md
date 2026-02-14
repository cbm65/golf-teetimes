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
├── phoenix.txt         # 91 courses
├── lasvegas.txt        # TODO
└── ...
```

## Phase 2: Automated Discovery

Run each platform's discovery tool against the course list file. Each tool probes
its platform's API, then validates confirmed matches with a 3-date tee time check
(next Wednesday, next Saturday, Saturday after that).

### Step 1: Run discovery scripts

```bash
# TeeItUp
go run cmd/discover-teeitup/main.go AZ -f discovery/courses/phoenix.txt

# Chronogolf
go run cmd/discover-chronogolf/main.go AZ -f discovery/courses/phoenix.txt

# ForeUP (one-time index build, then match)
go run cmd/discover-foreup/main.go --index 1 30000
go run cmd/discover-foreup/main.go --match AZ -f discovery/courses/phoenix.txt
```

Each tool outputs a JSON results file to `discovery/results/` with statuses:
- **confirmed** — course found on platform with live tee times
- **listed_only** — course page exists but 0 tee times (directory listing)
- **third_party_backend** — (ForeUP only) course has a "third party" booking class, meaning ForeUP is just the teesheet backend and another platform (TeeItUp, GolfNow, etc.) handles consumer booking. These should NOT be added as ForeUP courses.
- **wrong_state** — slug matched a course in another state (rejected)
- **miss** — no match found

### Step 2: Upload results to Claude

Upload the JSON results file to Claude and ask Claude to add the confirmed courses
to the project. Claude will:

1. Parse the results JSON
2. Extract confirmed courses with their platform-specific IDs
3. Add entries to the appropriate `platforms/data/*.json` file
4. Apply display name normalization (see below)

Example prompt:
> "Here are the TeeItUp discovery results for Phoenix. Add all confirmed courses to the project."

### Step 3: Verify and test

```bash
go run .
```

Visit `http://localhost:8080/phoenix` and verify courses appear in the dropdown.

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
Kennedy - Links                 → dropdown shows "Kennedy"
Kennedy - Creek                 → grouped under "Kennedy"
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
2. **Remove city disambiguators from name**: "Raven Golf Club - Phoenix" → "Raven Golf Club Phoenix" (use `city` field for city display instead)
3. **Remove internal sub-course names for single-course TeeItUp entries**: "Talking Stick Golf Club - Piipaash (South)" → "Talking Stick Golf Club"
4. **No trailing whitespace** in names or map keys
5. **Chronogolf `names` map**: Keys are the API course name (e.g. `"Estates"`), values use the " - " convention (e.g. `"Arizona Biltmore - Estates"`)

### Platform-specific notes

**TeeItUp** — `displayName` in teeitup.json is what appears in the UI. The API returns tee times tagged with this name. Single-course facilities use the club name directly. Multi-course facilities are usually separate TeeItUp entries (separate facilityIds) so each gets its own displayName.

**Chronogolf** — The `names` map translates API course names to display names. Multi-course clubs return tee times tagged with the course name from the API (e.g. "Estates"), which gets mapped to the display name (e.g. "Arizona Biltmore - Estates"). The lookup is case-insensitive with whitespace trimming, so minor API variations (e.g. "GOLD" vs "Gold") will still match.

**ForeUP** — Many courses use ForeUP as a teesheet backend while a different platform (TeeItUp, GolfNow, etc.) handles consumer-facing booking. The discovery script detects this by checking for a booking class named "Online Third Party" or similar. If present, the course is marked `third_party_backend` and should NOT be added as a ForeUP course — it will be discovered by the platform that actually handles booking. The script also deduplicates by `course_id` so the same ForeUP course can't match multiple input names. ForeUP's booking URL does not support date pre-fill via URL parameters (it's an Angular SPA), so the "Book Now" link lands on the course's booking page with today's date.

**Denver (MemberSports)** — The API returns raw names like "Kennedy Links". The `normalizeDenverName()` function in `denver.go` converts known multi-course prefixes to the " - " convention (e.g. "Kennedy Links" → "Kennedy - Links").

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
- `key` — derived from display name (lowercase, strip "Golf Course"/"Golf Club", hyphenate)
- `metro` — target metro slug
- `alias` — from discovery results `alias` field
- `facilityId` — from discovery results `facility.id`
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
  "affiliationTypeId": "",
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
- `bookingUrl` — `https://www.chronogolf.com/club/{slug}`
- `names` — map of API course name → display name (use " - " convention)

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
- `key` — derived from display name (lowercase, strip "Golf Course"/"Golf Club"/"Golf Resort", hyphenate)
- `metro` — target metro slug
- `courseId` — from discovery results `courseId` field
- `bookingClass` — from discovery results `bookingClassId` field
- `scheduleId` — from discovery results `scheduleId` field
- `bookingUrl` — leave empty (URL is constructed at runtime from courseId/scheduleId)
- `displayName` — from discovery results `name` field, cleaned per rules above (remove state suffixes like "(AZ)")
- `city` — from discovery results `city` field
- `state` — target state code

## Phase 3: Manual Gap Fill

After automated discovery, compare the master list against what was found. For each missing course:

1. Visit the course's booking page in a browser
2. Capture a HAR file of the tee time loading request
3. Give Claude the HAR — Claude identifies the platform and extracts the config
4. Claude adds the course to the appropriate `platforms/data/*.json`

This catches courses with non-obvious aliases, unusual API configs, or platforms we don't have discovery tools for yet.

**Input:** HAR files for each missing course
**Output:** Remaining courses added to `platforms/data/*.json`

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

| Platform       | Tool         | Status  |
|----------------|--------------|---------|
| TeeItUp        | discover-teeitup | Built |
| ForeUP         | discover-foreup  | Built (index + match, third-party detection, course_id dedup) |
| Chronogolf     | discover-chronogolf | Built (slug probe + tee time validation) |
| GolfNow        | discover-golfnow | TODO |
| CPS Golf       | discover-cpsgolf | TODO |
| MemberSports   | discover-membersports | TODO |
| ClubCaddie     | discover-clubcaddie | Built (index + match) |
| Quick18        | discover-quick18 | TODO |

## File Structure

```
cmd/
├── discover-teeitup/
│   └── main.go              # TeeItUp discovery tool
├── discover-foreup/
│   └── main.go              # ForeUP discovery tool (index + match)
├── discover-chronogolf/
│   └── main.go              # Chronogolf discovery tool (slug probe)
└── ...

discovery/
├── courses/
│   ├── phoenix.txt          # Phase 1 course list
│   ├── denver.txt           # (TODO)
│   └── ...
└── results/                 # Auto-generated discovery results
    ├── teeitup-az-2026-02-13-203817.json
    └── chronogolf-az-2026-02-13-221107.json

platforms/data/
├── teeitup.json        # TeeItUp courses (all metros)
├── chronogolf.json     # Chronogolf courses (all metros)
├── foreup.json         # ForeUP courses
├── golfnow.json        # GolfNow courses
└── ...
```

Each JSON file is an array of course configs. Every entry has `key`, `metro`, `city`, and `state` plus platform-specific fields. Adding a course is a single JSON entry — no other files need editing.
