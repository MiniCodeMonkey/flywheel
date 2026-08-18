# flywheel — design spec

**Status:** approved design, pre-implementation
**Date:** 2026-08-18
**Repo:** github.com/minicodemonkey/flywheel (public, built in public)
**Binary:** `flywheel`

## 1. Purpose

`flywheel` is a local CLI that lets Claude build **MOWL / Intelligent Cycling**
spinning courses from Spotify playlists, end to end, as an **LLM-driven**
process rather than a deterministic generator.

MOWL (app bundle `com.IntelligentTrainingGroup.IntelligentCycling`, an Unreal
Engine app) is backed by a documented REST API at `https://api.mowl.com`. A
"ride" is authored playlist-first: the playlist fixes the track list and total
ride length, and a phased workout (segments of intervals) is laid over that
timeline. Doing this by hand in the app is tedious; `flywheel` gives Claude the
tooling to do the MOWL side programmatically while Claude supplies all the
design judgment.

Non-goal: the CLI holds **no** course-design intelligence. Song selection,
phase structure, cadence/intensity choices, and time/TSS targeting all live in
Claude, guided by an editable style glossary and the declarative course spec.
The repo also ships a Claude **skill** (see §6) that teaches an agent the
end-to-end ride-building workflow, so the design knowledge travels with the
tool.

## 2. System architecture

Two halves, orchestrated by Claude:

```
  YOU: "Build a 55-min, TSS-75 road_cycling ride, heavy rock"
        │
        ├──────────────► Spotify MCP server (existing, not built here)
        │                 • search catalog by artist/track/genre
        │                 • create / modify a Spotify playlist
        │                 • returns a Spotify playlist ID
        │
        └──────────────► flywheel CLI (this project, Go)
                          • flywheel playlist inspect <spotify-id>
                              → tracks: index, title, artist, duration, BPM, sections
                          • Claude authors course.yaml against that data
                          • flywheel preview course.yaml  → timeline + est. TSS (no writes)
                          • flywheel apply   course.yaml  → creates the course in MOWL
```

- **Spotify half** — an existing third-party **Spotify MCP server** (e.g.
  `marcelmarais/spotify-mcp-server`). Claude uses its tools directly to design
  and create the playlist. Requires a one-time Spotify Developer app
  (client id/secret/redirect + OAuth); those credentials live with the MCP
  server, **not** in `flywheel`. We do not build or maintain the Spotify half.
- **MOWL half** — the `flywheel` CLI (this repo). No MOWL tooling exists
  anywhere, so this is the real deliverable.

Claude is the orchestrator: assembles the playlist (Spotify MCP), inspects it
(flywheel), authors and iterates the spec, then applies it (flywheel).

## 3. The MOWL data & API model (confirmed against the live API)

A MOWL course is: **Program → Segments → Intervals**, with a linked Spotify
**Playlist**. All of the following was verified end-to-end against the live API
with a real account (create + delete round-trips, then cleaned up).

- **Auth** — two-step ticket flow, then a session token in a raw
  `Authorization` header:
  1. `POST /v1/Authentication/Authenticate/{appPublicKey}` body
     `{Email, Password}` → ticket GUID.
  2. `GET /v1/Authentication/Ticket/{ticket}/{appPrivateKey}` → session token.
  - `appPublicKey = C00869F2-4457-486D-ADAD-47409605B187`,
    `appPrivateKey = 8285D895-29F6-425B-AE60-1725A68FF42E` (constant, from the
    app; may change with app versions — treat as config).
  - Required on every request: headers `itc-client-os-family: MacOS` and
    `itc-client-version: <appversion>`.
- **Import playlist (one call, auto-BPM):**
  `PUT /v1/Spotify/Playlist` body `{"SpotifyPlaylistID": "<id>"}` → creates an
  ITC Playlist with all tracks and **server-detected BPM per track**. Returns
  the ITC `PlaylistID`.
- **Read music:** `GET /v1/Playlists/{spotifyId}` (tracks: SpotifyTrackID,
  Artist, Title, BPM, Duration ms), `GET /v1/Spotify/Tracks/{id}/AudioAnalysis`
  (`sections`: `{start, duration, loudness}` — section-level energy).
- **Authored workout (all writable on a non-instructor RegisteredUser):**
  1. `PUT /v1/programcategories` `{Name, IsPublic:false, ...}` → category
     (personal). Delete: `DELETE /v1/programcategories/{id}`.
  2. `PUT /v1/programs` `{Name, ProgramCategoryID, IsPublic:false,
     ActivityTypeID:0, BikeTypeID:1}` → ProgramID.
     Delete: `DELETE /v1/programs/{id}`.
  3. `PUT /v1/segments` `{Name, ProgramID, IsWarmup|IsCooldown|...,
     SegmentCategoryID?, ActivityTypeID:0, BikeTypeID:1, IsPublic:false}`
     → SegmentID.
  4. `PUT /v1/segments/{id}/intervals/multiple`
     `{ReplaceExisting:true, Intervals:[{Duration, RPMFrom, RPMTo, Intensity,
     FTPFrom, FTPTo, PositionTypeID, ...}]}`.
  5. `PUT /v1/programs/{id}/segments` `{SegmentIDs:[...]}` — attach in order.
- **Program ↔ playlist link:** `GET /v1/Playlists/program/{programID}` returns a
  program's playlist. **OPEN ITEM (resolve in implementation):** exact binding
  mechanism — set `PlaylistID`/`SpotifyUrl` in the `PUT /v1/programs` body vs. a
  dedicated call. Resolve by setting it and reading back.
- **Server TSS:** `GET /v1/calculations/program/{programID}/TSS` returns the
  authoritative TSS after `apply` (used to confirm/validate our estimate).
- **Lookups (valid IDs):** `GET /v1/ActivityTypes` (Cycling 0, Running 1),
  `GET /v1/segmentcategories` (functional ones: Warm up 11, Standard Intervals
  10, Climbs 12/13/26, Tabata 41, Active Recovery 14, Cool Down & Stretch 15),
  PositionType (1 = Seated; enumerate the rest during build), BikeType
  (default 1), intensity zone sets `GET /v1/intensityzonesets`.
- Account unit is `ftp` (`DefaultIntensityUnit`), so intensity is authored as
  **% of FTP**.

## 4. Interface between Claude and the CLI: declarative spec

Claude authors a single `course.yaml`; `preview` dry-runs it; `apply` creates it
atomically. This plays to LLM strengths (write/revise a structured doc), avoids
Claude juggling server IDs, and gives a git-trackable artifact.

### 4.1 `course.yaml` schema

Structure mirrors MOWL exactly: **Program → ~3 Segments → Intervals**. A segment
spans multiple songs; its intervals are laid across that whole block.

```yaml
name: "Heavy Rock 55"
category: "My Rides"                 # personal MOWL category (created if missing)
activity: cycling
targets:
  duration_min: 55                   # fixed by the playlist; Claude tunes the playlist to hit it
  tss: 75
style: [road_cycling, punchy]    # advisory tags; resolved via styles.yaml + written to description

playlist:
  spotify_id: "EXPLAYLIST0000000000001"

segments:
  - name: "Warmup"
    type: warmup                     # → SegmentCategory "Warm up" (11); sets IsWarmup
    tracks: [1, 2]                   # song indices from `playlist inspect`
    intervals:
      - { duration: 180, cadence: [80,85], intensity: 45, position: seated }
      - { duration: 240, cadence: [85,90], intensity: 55, position: seated }
  - name: "Main"
    type: intervals                  # → "Standard Intervals" (10); or climb / tabata
    tracks: [3,4,5,6,7,8,9,10]
    style: [interval]
    intervals:
      - { duration: 120, cadence: [90,95],  intensity: 70, position: seated }
      - { duration: 90,  cadence: [95,105], intensity: 90, position: standing }
  - name: "Cooldown"
    type: cooldown                   # → "Cool Down & Stretch" (15); sets IsCooldown
    tracks: [11]
    intervals:
      - { duration: 180, cadence: [70,75], intensity: 40, position: seated }
```

**Field semantics:**
- `intensity` — **% of FTP**. Scalar = steady (`FTPFrom==FTPTo`); `[65,75]` =
  ramp. `Intensity` (legacy scalar) set to the midpoint.
- `cadence` — `[rpm_from, rpm_to]` → `RPMFrom/RPMTo`.
- `position` — `seated` / `standing` / … → `PositionTypeID` (from lookups).
- `duration` — seconds.
- `type` (segment) — maps to a MOWL **SegmentCategory** + warmup/cooldown flags:
  `warmup`→11, `intervals`→10, `climb`→12, `tabata`→41, `recovery`→14,
  `cooldown`→15.
- `style` (program or segment) — advisory intent with **no** MOWL field.
  Resolved via `styles.yaml` (below) into concrete interval defaults Claude
  applies, and written into the program/segment description.

### 4.2 `styles.yaml` — editable style glossary

Compact, concrete recipes the user carries in their head, captured so Claude
applies them consistently. Ships with sensible starters; user-editable. Example:

```yaml
road_cycling:      # road cycling: higher RPM, seated, longer intervals
  position: seated
  cadence: [90, 100]
  min_interval_sec: 120
  intensity_swing: small        # gentle ramps, not hard on/off
punchy:
  intensity_swing: large
  min_interval_sec: 30
climb:
  position: standing
  cadence: [60, 75]
  intensity_bias: high
tabata:
  pattern: 20s_on_10s_off
```

`flywheel` does not "execute" styles; it surfaces the resolved defaults in
`preview` so Claude and the user see how a tag was interpreted. Claude remains
free to deviate when the music calls for it.

## 5. CLI command surface

Thin wrapper over the MOWL API; `--json` on every command for machine-readable
output. All design lives in Claude + `course.yaml` + `styles.yaml`.

- `flywheel auth login` — one-time; runs the ticket→token flow, caches the
  token (chmod 600). Prompts for password or reads `MOWL_PASSWORD`; never stores
  the password by default. Auto-refreshes the token on 401 and retries.
- `flywheel playlist inspect <spotify-id|url>` — imports/reads the playlist,
  prints each track's index, title, artist, duration, BPM, and energy sections
  (JSON). The data Claude authors the spec against.
- `flywheel preview <course.yaml>` — validates + renders the full timeline,
  per-segment breakdown, total time vs `targets.duration_min`, and **estimated
  TSS** vs `targets.tss`. Shows how each `style` tag resolved. **No writes.**
- `flywheel apply <course.yaml>` — imports playlist → creates category /
  program / segments / intervals → links the playlist → attaches segments in
  order. Atomic and idempotent (re-applying updates in place, does not
  duplicate). Prints the resulting program and the **server-computed TSS**.
- `flywheel list` — list courses (programs) this account has created.
- `flywheel delete <program-id>` — delete a created course (and its private
  category if empty). For iterating/cleanup.
- `flywheel lookups` — dump valid segment types, position types, activity
  types, bike types, etc., so nothing is guessed.

### 5.1 TSS estimation (preview)

Coggan structured-workout estimate, computed locally so Claude can iterate
without writing:

```
IF_i = intensity_i / 100                         # fraction of FTP
TSS   = Σ ( duration_i_hours × IF_i² ) × 100
```

`preview` prints total estimated TSS and a per-segment breakdown; `apply`
reports the authoritative server TSS for comparison. If the two diverge
materially, the local formula is tuned during implementation to match MOWL.

### 5.2 Validation rules (preview + apply)

- Every playlist track is assigned to exactly one segment; `tracks` indices
  resolve; declared titles (if present) match the inspected playlist.
- Each segment's interval durations sum to the sum of its tracks' real lengths,
  within a small tolerance (e.g. ±5 s) — keeps the workout aligned to the music.
- Total time = Σ track durations, reported against the target.
- Estimated TSS reported against the target.
- Segment `type` and `position` resolve to valid MOWL IDs.
- Violations are errors in `apply` (block the write) and warnings/errors in
  `preview` (Claude fixes the spec and re-previews).

## 6. Shipped Claude skill (`flywheel`)

`flywheel` ships with a Claude **skill** so an agent knows how to drive the CLI
to build good rides without the user re-explaining the workflow each time. It
lives in the repo and is installed into the user's skills directory (or shipped
as a plugin); it is part of the deliverable, not an afterthought.

### 6.1 What the skill contains

- **Trigger / description** — fires when the user asks to build/design a MOWL or
  spinning ride/course from a playlist (e.g. "make me a 55-minute ride",
  "build a heavy-rock spin class", "turn this Spotify playlist into a course").
- **End-to-end workflow** the agent follows:
  1. Elicit ride parameters: target duration, target TSS, style/vibe, and any
     artist / track / genre constraints.
  2. **Playlist:** if a Spotify MCP is available, create or adjust a playlist to
     roughly hit the target duration; otherwise ask the user to point at an
     existing Spotify playlist. (See 6.2.)
  3. `flywheel playlist inspect <id>` → read tracks, durations, BPM, sections.
  4. Author `course.yaml`: ~3 segments spanning multiple songs, warmup +
     work/main + cooldown; resolve `style` tags via `styles.yaml`; align
     interval boundaries to song/section edges; set cadence near track BPM.
  5. `flywheel preview` → check total time vs target and estimated TSS vs
     target; iterate (adjust intensities/cadence, or add/drop songs via the
     Spotify MCP and re-inspect) until both land.
  6. `flywheel apply` → create the course; confirm the server TSS; report the
     program to the user.
- **Design heuristics** — playlist-first (the playlist fixes the length);
  phase shape (ease in, build, ease out); keep each segment's interval time
  aligned to its songs; honor `style` recipes but deviate when the music calls
  for it; respect the account's `ftp` intensity unit.
- **Reference** — the `course.yaml` and `styles.yaml` schemas, the command
  surface, and the validation rules (pointing at this spec / README rather than
  duplicating them).

### 6.2 Optional Spotify MCP

- The skill treats a Spotify MCP as **optional**. It documents a known-good
  option (e.g. `marcelmarais/spotify-mcp-server`) and its one-time Spotify
  Developer app setup, but does not hard-depend on any specific server.
- **Detection & fallback:** if Spotify playlist-authoring tools are present, the
  agent uses them to build/tune the playlist; if not, it degrades gracefully to
  working with a user-provided existing Spotify playlist. `flywheel` itself
  never needs the Spotify MCP — it only takes a Spotify playlist id.

### 6.3 Authoring

The skill is written using the `superpowers:writing-skills` skill during
implementation, and kept concise (workflow + heuristics + schema reference),
delegating exhaustive detail to the README and this spec.

## 7. Auth & configuration

- Config dir `~/.config/flywheel/` (outside the repo): `config.yaml`
  (email, `app_public_key`, `app_private_key`, client-version header,
  api base URL) and `token` (cached session token, chmod 600).
- `flywheel auth login` prompts for the password or reads `MOWL_PASSWORD`;
  the password is not stored. Token auto-refreshes on 401.
- Spotify Developer credentials live with the Spotify MCP server, not here.
- `.gitignore` covers local artifacts; `~/.config/flywheel/` is outside the
  repo. Safe to build in public.

## 8. Project structure (Go)

```
flywheel/
  cmd/flywheel/           main.go — Cobra root + subcommands
  internal/
    mowl/                 API client: auth, playlists, programs, segments, intervals, lookups
    spec/                 course.yaml + styles.yaml parsing, validation, resolution
    plan/                 preview rendering + TSS estimation
    apply/                orchestration: import → create → link → attach (atomic/idempotent)
    config/               ~/.config/flywheel load/save, token cache
  docs/superpowers/specs/ this spec
  styles.yaml             default style glossary (shipped, user-editable)
  skill/                  the shipped Claude skill (SKILL.md) — see §6
  README.md               build-in-public intro + setup (incl. Spotify MCP + Developer app)
  go.mod
```

Each `internal/*` package has one clear responsibility and a small interface so
units are testable in isolation.

## 9. Testing strategy

- **`mowl` client** — unit tests against recorded HTTP fixtures (httptest
  server) for auth, import, create/attach, TSS, and 401-refresh. No live calls
  in CI.
- **`spec`** — table-driven tests for parsing, style resolution, and every
  validation rule (track coverage, duration-sum tolerance, ID resolution).
- **`plan`** — TSS formula tests with known-answer cases; golden-file tests for
  `preview` rendering.
- **`apply`** — orchestration tested against a mock `mowl` client asserting the
  call sequence and idempotency (re-apply updates, no duplicates).
- **Manual live smoke** — a documented, opt-in end-to-end run against the real
  API that creates and then deletes a throwaway course (not in CI).

## 10. Open items to resolve during implementation

1. **Program↔playlist binding** — confirm whether `PlaylistID`/`SpotifyUrl` on
   the `PUT /v1/programs` body links the playlist, or a dedicated call is
   needed. Verify via read-back on `GET /v1/Playlists/program/{programID}`.
2. **Local vs. server TSS** — calibrate the local Coggan estimate against the
   server value; adjust if MOWL uses a different model.
3. **PositionType / BikeType enumerations** — pull full lists via lookups and
   map friendly names.
4. **`Intensity` scalar vs FTPFrom/To semantics** — confirm what MOWL stores /
   displays for an `ftp`-unit account and drive consistently.
5. **Idempotency key** — how a re-`apply` finds the existing program (e.g. by
   name within the personal category) to update in place.

## 11. Explicit non-goals (YAGNI)

- No Spotify API code (delegated to the MCP server).
- No granular per-object subcommands beyond what's listed; the spec file is the
  path (add later only if a real need appears).
- No GUI, no server, no multi-user/account management.
- No instructor/public-course publishing flow; personal private courses only.
