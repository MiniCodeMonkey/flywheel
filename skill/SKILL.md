---
name: flywheel
description: Use when the user wants to build a spinning ride/course, design a MOWL (Intelligent Cycling) workout, turn a Spotify playlist into a workout or spin class, or asks for something like "make me a 55-minute ride" or "build a heavy-rock spin class" — drives the flywheel CLI end to end from a Spotify playlist to a created MOWL course.
---

# flywheel

Drives the `flywheel` CLI to build MOWL (Intelligent Cycling) spinning courses
from Spotify playlists. The CLI holds no design intelligence — song
selection, phase structure, cadence/intensity, and time/TSS targeting are all
your judgment calls, guided by an editable style glossary
(`styles.yaml`) and a declarative `course.yaml` spec.

## Prerequisites

- `flywheel` binary installed and on `PATH`.
- One-time setup, if not already done: `flywheel init` (writes `styles.yaml`)
  and `flywheel auth login` (caches a MOWL session token).
- A Spotify MCP server is optional — see "Spotify playlist" below.

Every `flywheel` subcommand accepts `--json` for machine-readable output;
prefer it when parsing results programmatically.

## Workflow

1. **Elicit ride parameters.** Target duration (minutes), target TSS,
   style/vibe (free text plus any `styles.yaml` tags), and any artist/track/
   genre constraints. Ask only for what's missing.
2. **Get a Spotify playlist.** If a Spotify MCP is available, use its tools
   to build or adjust a playlist that roughly hits the target duration. If
   not, ask the user for an existing Spotify playlist link or ID. Either way
   you end up with a `spotify_id`. The playlist fixes the ride's length —
   don't fight it with padding intervals.
3. **Inspect it:** `flywheel playlist inspect <spotify-id> --json`. Read
   each track's index, title, artist, duration, BPM, and energy sections —
   this is the data you author the spec against.
4. **Author `course.yaml`** (see schema below): roughly 3 segments spanning
   multiple songs — warmup, work/main, cooldown. Each segment's interval
   durations must sum to the real length of its assigned tracks (±5s
   tolerance). Set cadence near each track's BPM. Resolve any `style` tags
   through `styles.yaml` into concrete defaults, but deviate when the music
   calls for it.
5. **Preview and iterate:** `flywheel preview course.yaml`. Check total time
   against target duration and estimated TSS against target TSS, and see how
   each `style` tag resolved. Adjust intensities/cadence, or add/drop songs
   via the Spotify MCP and re-inspect, until both land. `preview` never
   writes anything — iterate freely.
6. **Apply:** `flywheel apply course.yaml`. This imports the playlist,
   creates the category/program/segments/intervals, links the playlist, and
   attaches segments in order. Idempotent — re-applying the same file updates
   in place rather than duplicating. Report the created program and the
   server-computed TSS to the user (compare it to `preview`'s estimate).

Other commands: `flywheel list` (courses this account created), `flywheel
delete <program-id>` (remove a course and its private category if empty, for
cleanup/iteration), `flywheel lookups` (valid segment types, position types,
activity types, bike types — don't guess these).

## Design heuristics

- **Playlist-first.** The playlist fixes ride length; don't invent silence or
  padding to hit a duration — swap songs instead.
- **Phase shape.** Ease in (warmup), build through the work/main segment(s),
  ease out (cooldown). Avoid abrupt intensity jumps between segments.
- **Stay aligned to the music.** Interval boundaries should land on song or
  section edges from the `inspect` data, not arbitrary timestamps.
- **`style` is advisory, not gospel.** Resolve tags via `styles.yaml` for a
  consistent starting point, then deviate when a specific song's energy or
  BPM argues for something different.
- **`intensity` is % of FTP**, not absolute watts — a scalar is steady state,
  `[from,to]` is a ramp.

## `course.yaml` reference

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

`intensity` is a scalar (%FTP) or `[from,to]` for a ramp; `cadence` is
`[rpm_from, rpm_to]`; `position` is `seated`/`standing`/etc (see `lookups`);
`duration` is seconds. Full field semantics, validation rules, and the
`styles.yaml` format: see the repo README and
`docs/superpowers/specs/2026-08-18-flywheel-design.md` (§4–5) rather than
duplicating them here.

## Optional Spotify MCP

`flywheel` never talks to Spotify itself — it only ever takes a Spotify
playlist ID. A Spotify MCP server (e.g. `marcelmarais/spotify-mcp-server` or
`varunneal/spotify-mcp`) is optional tooling for the playlist-authoring half
of the workflow, and needs its own one-time Spotify Developer app (client
id/secret/redirect + OAuth) configured outside of `flywheel`. Check whether
playlist-authoring tools are available in the current session; if so, use
them to build/tune the playlist. If not, degrade gracefully — ask the user
for an existing Spotify playlist link or ID and continue from step 3.

## Note on provisional details

Some CLI behavior (exact playlist-link format, the standing position ID,
TSS-formula calibration against MOWL's server-side number) is confirmed by
running the workflow live rather than fixed in advance. This is expected and
non-blocking — `preview`'s estimated TSS and `apply`'s server-reported TSS may
differ slightly; report both if they diverge.
