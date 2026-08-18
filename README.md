# flywheel

`flywheel` is a local CLI that lets Claude build **MOWL / Intelligent
Cycling** spinning courses from Spotify playlists — LLM-driven and
playlist-first. The CLI holds no course-design intelligence: song selection,
phase structure, cadence/intensity, and time/TSS targeting are all Claude's
judgment calls, guided by an editable style glossary (`styles.yaml`) and a
declarative course spec (`course.yaml`). `flywheel` just handles the MOWL
side — importing the playlist and creating the program/segments/intervals —
so Claude can design the ride and `flywheel` creates it. The repo ships a
Claude **skill** (`skill/SKILL.md`) that teaches an agent the whole
ride-building workflow end to end.

Built in public.

## Install

```
go install github.com/minicodemonkey/flywheel/cmd/flywheel@latest
```

Requires a MOWL / Intelligent Cycling account.

## Quickstart

```
flywheel init            # writes a starter styles.yaml to ~/.config/flywheel/
flywheel auth login      # MOWL email/password, caches a session token
```

Then the core loop, usually driven by Claude via the shipped skill:

1. **Inspect a Spotify playlist** (already imported into MOWL, or imported on
   first inspect) to see what you're working with:

   ```
   flywheel playlist inspect <spotify-playlist-id> --json
   ```

   Returns each track's index, title, artist, BPM, and duration — the data
   you author the spec against.

2. **Author `course.yaml`** by hand or via Claude. A small worked example:

   ```yaml
   name: "Heavy Rock 55"
   category: "My Rides"              # personal MOWL category, created if missing
   activity: cycling
   targets: { duration_min: 55, tss: 75 }
   style: [road_cycling, punchy]      # advisory; resolved via styles.yaml
   playlist:
     spotify_id: "0478H01T6WxqFi0fevIPP1"
   segments:
     - name: "Warmup"
       type: warmup                   # warmup|intervals|climb|tabata|recovery|cooldown
       tracks: [1, 2]                 # song indices from `playlist inspect`
       intervals:
         - { duration: 180, cadence: [80,85], intensity: 45, position: seated }
     - name: "Main"
       type: intervals
       tracks: [3,4,5,6,7,8,9,10]
       style: [interval]
       intervals:
         - { duration: 120, cadence: [90,95],  intensity: 70, position: seated }
         - { duration: 90,  cadence: [95,105], intensity: 90, position: standing }
     - name: "Cooldown"
       type: cooldown
       tracks: [11]
       intervals:
         - { duration: 180, cadence: [70,75], intensity: 40, position: seated }
   ```

3. **Preview** — validates and renders the timeline, no writes:

   ```
   flywheel preview course.yaml
   ```

   Check total time against `targets.duration_min` and estimated TSS against
   `targets.tss`, see how each `style` tag resolved, then iterate on the
   spec until both land.

4. **Apply** — creates the course in MOWL (imports the playlist, creates the
   category/program/segments/intervals, links the playlist, attaches
   segments in order). Idempotent — re-applying the same file updates in
   place instead of duplicating:

   ```
   flywheel apply course.yaml
   ```

## Commands reference

Every subcommand also accepts the global `--json` flag for machine-readable
output.

| Command | Description |
|---|---|
| `flywheel init` | Write a starter `styles.yaml` into the config dir (`~/.config/flywheel/`), if one doesn't already exist. |
| `flywheel auth login` | Authenticate with MOWL (`--email`, or prompts; password via `MOWL_PASSWORD` env var or a prompt) and cache a session token. |
| `flywheel playlist inspect <spotify-id>` | Import/read a Spotify playlist via MOWL and print each track's index, title, artist, BPM, and duration. `--sections` also fetches audio-analysis section starts per track. |
| `flywheel preview <course.yaml>` | Validate and render a course's timeline, per-segment breakdown, and estimated TSS — no writes. |
| `flywheel apply <course.yaml>` | Validate and create (or idempotently update) a course in MOWL; reports the server-computed TSS. |
| `flywheel list` | List MOWL programs this account has created. |
| `flywheel delete <program-id>` | Delete a created program (and its private category if empty). |
| `flywheel lookups` | Dump valid MOWL segment categories and activity types, plus flywheel's segment-type/position alias maps, so nothing is guessed. |

## `course.yaml` reference

Structure mirrors MOWL: a **Program** with `targets`, a linked `playlist`,
and ~3 **segments** spanning multiple songs, each with **intervals**.

| Field | Type | Meaning |
|---|---|---|
| `name` | string | Program name. |
| `category` | string | Personal MOWL category; created if missing. |
| `activity` | string | Activity type, e.g. `cycling`. |
| `targets.duration_min` | number | Target ride length in minutes (fixed by the playlist; tune the playlist to hit it). |
| `targets.tss` | number | Target Training Stress Score. |
| `style` | []string | Advisory tags (program-level), resolved via `styles.yaml`; written into the description, no direct MOWL field. |
| `playlist.spotify_id` | string | Spotify playlist ID to import and link. |
| `segments[].name` | string | Segment name. |
| `segments[].type` | enum | One of `warmup`, `intervals`, `climb`, `tabata`, `recovery`, `cooldown` — maps to a MOWL segment category (and sets warmup/cooldown flags). |
| `segments[].tracks` | []int | Song indices (from `playlist inspect`) this segment spans. |
| `segments[].style` | []string | Advisory tags, segment-level. |
| `segments[].intervals[].duration` | number | Interval length in seconds. |
| `segments[].intervals[].cadence` | [number,number] | `[rpm_from, rpm_to]`. |
| `segments[].intervals[].intensity` | number or [number,number] | **% of FTP.** A scalar is steady state; `[from,to]` is a ramp. |
| `segments[].intervals[].position` | enum | `seated` / `standing` / … (see `flywheel lookups`). |

Validation (`preview` and `apply`): every playlist track is assigned to
exactly one segment; each segment's interval durations sum to the real
length of its assigned tracks (±5s tolerance); segment `type` and `position`
must resolve to valid MOWL IDs.

## `styles.yaml`

An editable glossary of style tags — compact, concrete interval recipes
(position, cadence range, minimum interval length, intensity swing, etc.)
that Claude resolves `style` tags against for a consistent starting point,
while still deviating when a specific song's energy or BPM calls for it.
`flywheel` doesn't "execute" styles; `preview` just surfaces how a tag
resolved so you can see it before applying.

Ships with sensible defaults (`road_cycling`, `punchy`, `climb`, `tabata`,
`recovery`) at [`styles.yaml`](styles.yaml) in this repo; `flywheel init`
copies them to `~/.config/flywheel/styles.yaml`, where they're yours to
edit.

## Optional Spotify MCP

`flywheel` never talks to Spotify itself — every command takes a Spotify
playlist ID, nothing more. Building or tuning the playlist (searching the
catalog, adding/reordering tracks) is a separate, optional step best done by
Claude through a **Spotify MCP server**, e.g.
[`marcelmarais/spotify-mcp-server`](https://github.com/marcelmarais/spotify-mcp-server).
That server needs its own one-time Spotify Developer app (client id/secret/
redirect + OAuth), configured outside of `flywheel`. If no Spotify MCP is
available, just point `flywheel playlist inspect` at an existing Spotify
playlist link/ID instead.

## The Claude skill

[`skill/SKILL.md`](skill/SKILL.md) is the shipped Claude skill. Installing
it into an agent's skills teaches it the entire workflow above — eliciting
ride parameters, using a Spotify MCP if available, inspecting the playlist,
authoring and iterating `course.yaml`, previewing, and applying — so you can
just ask for "a 55-minute heavy-rock ride" and let Claude drive the CLI.

## Provisional / live-verification note

A few implementation details were derived from API response shapes rather
than a full live round-trip and are expected to be confirmed (and, if
needed, adjusted) on a real account: the exact program↔playlist link
mechanism, the "standing" `PositionType` id, and the calibration of
`preview`'s local Coggan TSS estimate against MOWL's server-computed TSS.
None of this blocks normal use — `preview` and `apply` may report slightly
different TSS numbers until calibrated; see
[docs/superpowers/notes/2026-08-18-api-findings.md](docs/superpowers/notes/2026-08-18-api-findings.md)
for the full details and the smoke-test procedure.

## Links

- Design spec: [docs/superpowers/specs/2026-08-18-flywheel-design.md](docs/superpowers/specs/2026-08-18-flywheel-design.md)
- Implementation plan: [docs/superpowers/plans/2026-08-18-flywheel.md](docs/superpowers/plans/2026-08-18-flywheel.md)
- API findings: [docs/superpowers/notes/2026-08-18-api-findings.md](docs/superpowers/notes/2026-08-18-api-findings.md)

## Disclaimer

Built in public, as a personal tool. Not affiliated with, endorsed by, or
supported by MOWL or Intelligent Training Group. Uses the same REST API the
official MOWL app uses; use your own account and use it responsibly — this
project is not responsible for any account or API issues that result from
its use.
