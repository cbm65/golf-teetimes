# Adding a New Metro — Process

## Overview

Five-phase process to add all public courses for a new metro area:

1. **Phase 1: Course List** — Exhaustive list of every public course within an hour of the metro city
2. **Phase 2: Automated Discovery** — Run platform discovery scripts against the course list
3. **Phase 3: Manual Gap Fill** — HAR capture for courses not found by automated discovery
4. **Phase 4: GolfNow Discovery** — Area search for GolfNow-only courses not already added
5. **Phase 5: GolfNow Platform Swap** — Check each GolfNow course for direct platform availability

## Phase 1: Course List

Compile a **comprehensive, exhaustive** list of every public golf course within approximately one hour of the metro city. Include municipal, retirement community, lesser-known, and edge-of-metro courses. Save to `discovery/courses/{metro}.txt` — one course per line, `#` comments ignored. Format: `Course Name | City`.

## Phase 2: Automated Discovery

Run each platform's discovery tool against the course list. Each tool probes its platform's API, then validates with a 3-date tee time check.

```bash
# Run in this order (TeeItUp first — establishes baseline):
go run cmd/discover-teeitup/main.go {STATE} -f discovery/courses/{metro}.txt
go run cmd/discover-golfwithaccess/main.go {STATE} -f discovery/courses/{metro}.txt
go run cmd/discover-chronogolf/main.go {STATE} -f discovery/courses/{metro}.txt
go run cmd/discover-foreup/main.go --match {STATE} -f discovery/courses/{metro}.txt
go run cmd/discover-quick18/main.go {STATE} -f discovery/courses/{metro}.txt
go run cmd/discover-courseco/main.go {STATE} -f discovery/courses/{metro}.txt
go run cmd/discover-cpsgolf/main.go {STATE} -f discovery/courses/{metro}.txt
```

For metros spanning two states (e.g. Charlotte NC/SC), run each script for both states.

**Cross-platform dedup rule**: A course appears on ONE platform only. Other platforms take precedence over TeeItUp. GolfNow courses are NOT added in Phase 2.

**Ingestion checklist** for each confirmed course:
1. Already in any `platforms/data/*.json`? → SKIP
2. In the correct metro area? → SKIP if wrong metro
3. Name match actually correct? → SKIP if wrong course
4. Clean display name, derive key, add to appropriate JSON file

## Phase 3: Manual Gap Fill

For each course not found by automated discovery:
1. Visit the course's booking page in a browser (incognito)
2. Capture a HAR file of the tee time loading
3. Give Claude the HAR — Claude identifies the platform and extracts config
4. Claude appends the course to `platforms/data/*.json`

## Phase 4: GolfNow Discovery

```bash
go run discovery/golfnow-discover.go {metro}
```

Searches GolfNow area API, cross-references against all existing courses, outputs missing GolfNow-only courses. Exclude non-golf facilities (simulators, par-3s, entertainment venues, practice ranges).

## Phase 5: GolfNow Platform Swap

Check each GolfNow course's own website for direct platform booking. If it uses a supported platform, capture HAR, swap from golfnow.json to the direct platform.

---

## Adding a Metro Entry

Add to `metros.go` before running discovery:

```go
"charlotte": {
    Name:    "Charlotte",
    Slug:    "charlotte",
    State:   "NC",
    Tagline: "Public Courses Across the Queen City",
},
```

Course/city counts are computed automatically at startup.

---

## Key Derivation Rules

1. Start with display name
2. Strip: "Golf Course", "Golf Club", "Golf Resort", "Country Club", "Golf Complex", "Golf Links", "Golf Center", "GC", "CC"
3. Strip leading: "The ", "Golf Club of ", "Golf Club at "
4. Lowercase, replace non-alphanumeric with hyphens, collapse multiples, trim edges

Examples: `Aguila Golf Course → aguila`, `Red Mountain Ranch Country Club → red-mountain-ranch`, `Golf Club of Estrella → estrella`, `Trilogy Golf Club at Vistancia → trilogy-at-vistancia`

## Display Name Rules

- Multi-course clubs: `"ClubName - CourseName"` (frontend groups on `" - "`)
- Remove state/region suffixes, expand abbreviations, add missing "Golf Course"/"Golf Club" suffixes
- No trailing whitespace

---

## JSON Entry Formats by Platform

### TeeItUp

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

- `alias` — from HAR URL: `{alias}.book.teeitup.com` or `{alias}.book-v2.teeitup.golf`
- `facilityId` — string, from Kenna `/facilities` response or HAR `?fid=` param

### ForeUP

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

- Extract `courseId`, `bookingClass` (`booking_class=`), and `scheduleId` (`schedule_id=`) from HAR tee-times API URL
- `bookingUrl` — leave empty

### Chronogolf

```json
{
  "key": "arizona-biltmore",
  "metro": "phoenix",
  "courseIds": "uuid1,uuid2",
  "clubId": "",
  "numericCourseId": "",
  "affiliationTypeId": "82990",
  "bookingUrl": "https://www.chronogolf.com/club/{slug}",
  "names": { "Estates": "Arizona Biltmore - Estates" },
  "city": "Phoenix",
  "state": "AZ"
}
```

- Extract from `__NEXT_DATA__` in page HTML: `club.courses[].uuid`, `club.defaultAffiliationTypeId`, `club.slug`
- `names` maps API course name → display name

### GolfWithAccess (Troon)

```json
{
  "key": "quintero",
  "metro": "phoenix",
  "courseIds": ["uuid"],
  "slug": "quintero-golf-club",
  "bookingUrl": "https://golfwithaccess.com/course/{slug}/reserve-tee-time",
  "displayName": "Quintero Golf Club",
  "city": "Peoria",
  "state": "AZ"
}
```

### Quick18

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

### CPS Golf

```json
{
  "key": "indian-tree",
  "metro": "denver",
  "baseUrl": "https://indiantree.cps.golf",
  "apiKey": "",
  "websiteId": "e6d9cd59-...",
  "siteId": "1",
  "courseIds": "",
  "bookingUrl": "https://indiantree.cps.golf/onlineresweb/search-teetime",
  "names": { "Regulation 18": "Indian Tree Golf Club" },
  "city": "Arvada",
  "state": "CO",
  "timezone": "America/Denver"
}
```

- `apiKey` and `courseIds` can be empty — fetcher auto-detects at runtime
- Has two auth modes (apiKey or bearer token) — fetcher handles both
- **Legacy V3 interface** (`e.cps.golf/{Name}V3/`) uses server-rendered HTML — NOT supported, skip these

### CourseRev

```json
{
  "key": "highland-creek",
  "metro": "charlotte",
  "subDomain": "highlandcreekgolfclub",
  "courseId": 1,
  "bookingUrl": "https://highlandcreekgolfclub.bookings.courserev.ai/tee-times",
  "displayName": "Highland Creek Golf Club",
  "city": "Charlotte",
  "state": "NC"
}
```

- Extract from `course/mco/details` POST response: `golfCourses[].id`, `groupName`
- `subDomain` from URL: `{subDomain}.bookings.courserev.ai`

### ClubCaddie

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

- `courseId` from POST body of `/webapi/TeeTimes` (NOT the logo URL)

### RGuest

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

### TeeSnap

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

### GolfNow

```json
{
  "key": "apache-creek",
  "metro": "phoenix",
  "facilityId": 1440,
  "searchUrl": "https://www.golfnow.com/tee-times/facility/1440/search",
  "bookingUrl": "https://www.golfnow.com/tee-times/facility/1440/search",
  "displayName": "Apache Creek Golf Club",
  "city": "Apache Junction",
  "state": "AZ"
}
```

- `facilityId` is an integer (not string)
- Frontend shows "Book via GolfNow" for courses with `golfnow.com` in bookingUrl

### PurposeGolf

```json
{
  "key": "sherrill-park-1",
  "metro": "dallas",
  "courseId": 2,
  "slug": "SherrillParkCourse1",
  "displayName": "Sherrill Park Golf Course #1",
  "city": "Richardson",
  "state": "TX",
  "bookingUrl": "https://booking.purposegolf.com/courses/SherrillParkCourse1/2/teetimes"
}
```

### TeeQuest

```json
{
  "key": "indian-creek-creek",
  "metro": "dallas",
  "siteId": "57",
  "courseTag": "57-1",
  "displayName": "Indian Creek Golf Club - Creek",
  "city": "Carrollton",
  "state": "TX",
  "bookingUrl": "https://teetimes.teequest.com/57?paymentTab=pay-at-course"
}
```

### ResortSuite

```json
{
  "key": "fields-ranch-east",
  "metro": "dallas",
  "baseUrl": "https://omnipgafriscoexperiences.com",
  "courseId": "002",
  "displayName": "Fields Ranch East",
  "city": "Frisco",
  "state": "TX",
  "bookingUrl": "https://omnipgafriscoexperiences.com"
}
```

---

## Instructions for AI Assistants

### Efficient Append Pattern

As JSON data files grow, avoid reading entire files when adding entries:

```python
import json, os

entry = json.dumps(new_entry, indent=2)
with open('platforms/data/foreup.json', 'rb+') as f:
    f.seek(-2, os.SEEK_END)
    f.write(b',\n  ' + entry.encode() + b'\n]')
```

**When starting a new chat session, do NOT read full JSON data files.** The HAR tells you the platform and all config fields. If unsure about schema, read only the first entry (`f.read(500)`). Discovery scripts handle deduplication.

### Cumulative Zip After Every Change

After every course addition (or batch of additions), produce a fresh zip of the full project and present it to the user:

```bash
cd /home/claude/golf-teetimes && zip -r /mnt/user-data/outputs/golf-teetimes.zip . -x '.git/*' 'node_modules/*' '*.zip' > /dev/null 2>&1
```

This ensures the user always has a complete, up-to-date snapshot after each change.

### User's Update Command

The user applies updates by extracting the zip into their local repo:

```bash
unzip -o ~/Downloads/golf-teetimes.zip -d ~/golf-teetimes/
```

The `-d` path **must** point to the repo root (`~/golf-teetimes/`), not `~/`.

---

## Platform Quick Reference

| Platform | Discovery Script | HAR Identification |
|----------|-----------------|-------------------|
| TeeItUp | `discover-teeitup` | `kenna.io` or `teeitup.golf`/`teeitup.com` in URLs |
| ForeUP | `discover-foreup` | `foreupsoftware.com` in URLs |
| Chronogolf | `discover-chronogolf` | `chronogolf.com` in URLs |
| GolfWithAccess | `discover-golfwithaccess` | `golfwithaccess.com` in URLs |
| Quick18 | `discover-quick18` | `quick18.com` in URLs |
| CPS Golf | `discover-cpsgolf` | `cps.golf` in URLs (modern: `/onlineresweb/`, legacy V3: skip) |
| CourseRev | None | `courserev.ai` in URLs |
| ClubCaddie | None | `clubcaddie.com` in URLs |
| RGuest | None | `rguest.com` in URLs |
| TeeSnap | None | `teesnap.net` in URLs |
| GolfNow | `golfnow-discover` (Phase 4) | `golfnow.com` in URLs |
| PurposeGolf | None | `purposegolf.com` in URLs |
| TeeQuest | None | `teequest.com` in URLs |
| ResortSuite | None | SOAP/XML to `/wso2wsas/services/RSWS` |
| CourseCo | `discover-courseco` | `totaleintegrated.net` in URLs |
| Prophet | None (**DISABLED** — AWS WAF) | `prophetservices.com` in URLs |

## HAR Capture Tips

- Use **incognito** for clean sessions
- **Fetch/XHR filter** in Network tab before export to reduce HAR size
- Sanitized HARs are fine for platform identification; unsanitized needed only for auth debugging

---

## Historical Notes

<details>
<summary>Per-Metro Discovery Results (click to expand)</summary>

### Phoenix (Feb 2026)
92 input courses, 62 confirmed. TeeItUp dominated (41/62). Phase 4/5: 133 GolfNow facilities found, 72 added, 22 swapped to direct platforms.

### Las Vegas (Feb 2026)
42 input courses, 4 confirmed (mostly GolfNow-only market). CPS Golf bearer token auth discovered here.

### Atlanta (Feb 2026)
52 input courses, 39 confirmed across 6 platforms. TeeItUp found 21 auto + 7 HAR. Booking site HTML fallback added to TeeItUp discovery.

### Dallas (Feb 2026)
All manual HAR (Phase 3 only). 43 courses across 10 platforms. Three new platforms discovered: PurposeGolf, TeeQuest, ResortSuite. Prophet Services disabled due to AWS WAF.

### Charlotte (Feb 2026)
55 input courses, 39 confirmed across 8 platforms. CourseRev platform added (3 courses). CPS Golf legacy V3 identified as unsupported.

</details>

<details>
<summary>TeeItUp Discovery Script Improvements</summary>

- **Booking site HTML fallback** — tries `{alias}.book.teeitup.com` when Kenna API returns 404
- **`strip-city-suffix`** — strips city from core name end
- **`trim-trailing`** — removes last slug segment
- **Candidates not yet added**: state code suffix (`{exact}-{state}`), `golf-village` suffix, `.golf` domain fallback

</details>

<details>
<summary>Platform-Specific Technical Notes</summary>

**TeeItUp**: Kenna Commerce API. Some courses use UUID aliases on `book-v2.teeitup.golf` (undiscoverable). Some have alternate names (e.g. "Chastain Park" for "North Fulton GC").

**ForeUP**: Many courses use ForeUP as teesheet backend with another platform for consumer booking ("third_party_backend"). Always need `bookingClass` and `scheduleId` from HAR.

**Chronogolf**: Massive directory (50+ listed_only per metro) but rarely active for booking (~2 per metro).

**CPS Golf**: Two auth modes (apiKey or bearer token) — auto-detected. Subdomains are unpredictable. Legacy V3 interface (`e.cps.golf/`) not supported. Multi-course sites need separate entries with specific `courseIds`.

**ClubCaddie**: No central directory. `courseId` from POST body, not logo URL. Use `player=1`.

**Prophet**: DISABLED. AWS WAF JS challenge. Would need headless browser.

</details>
