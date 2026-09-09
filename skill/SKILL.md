---
name: flywheel
description: Use when the user wants to build a spinning ride/course, design a MOWL (Intelligent Cycling) workout, turn a Spotify playlist into a workout or spin class, or asks for something like "make me a 55-minute ride" or "build a heavy-rock spin class" — drives the flywheel CLI end to end from a Spotify playlist to a created MOWL course.
---

# flywheel

Drives the `flywheel` CLI to build MOWL (Intelligent Cycling) spinning courses
from Spotify playlists. The CLI holds no design intelligence — song selection,
phase structure, cadence/intensity, and time/TSS targeting are all your
judgment calls, guided by an editable style glossary (`styles.yaml`) and a
declarative `course.yaml` spec.

## Prerequisites

- `flywheel` binary installed and on `PATH`.
- One-time setup, if not already done: `flywheel init` (writes `styles.yaml`)
  and `flywheel auth login` (caches a MOWL session token).
- A Spotify MCP server is optional — see "Optional Spotify MCP" below.

Run `flywheel version` first and compare the revision to the checked-out
source. An installed binary that lags the repo silently uses an older TSS
model and produces numbers that do not match `preview`. Rebuild with
`go install ./cmd/flywheel` when they differ.

Every subcommand accepts `--json`; prefer it when parsing programmatically.

## Workflow

1. **Elicit ride parameters.** Target duration, target TSS, style/vibe, and
   any artist/genre constraints. Ask only for what's missing.
2. **Get a Spotify playlist.** Build one with a Spotify MCP if available,
   otherwise ask for a playlist link or ID. The playlist fixes the ride's
   length — don't fight it with padding intervals.
3. **Inspect it:** `flywheel playlist inspect <spotify-id> --sections --json`.
   Always pass `--sections`: it returns each track's musical sections with
   `start`, `duration` and `loudness`, which is the data that makes a ride
   follow the music instead of a stopwatch.
4. **Scaffold `course.yaml`**, then edit it. `flywheel scaffold` applies the
   whole design method below in one command:

   ```
   flywheel scaffold <spotify-id> --tss 75 \
     --segment "Roll Out:warmup:1-3" \
     --segment "Hammer Climbs:climb:4-7" \
     --segment "Active Recovery:recovery:8" \
     --segment "Bass Intervals:intervals:9-12" \
     --segment "Active Recovery 2:recovery:13"
   ```

   Segments are `Name:type:tracks`, repeatable, in ride order; tracks accept
   ranges and lists (`4-7`, `1,3,5`). It writes `course.yaml` and reports the
   estimated TSS. Treat the output as a starting point and hand-edit it --
   the scaffolder cannot hear the song, so move blocks where the music
   argues for it. Read the method below before editing.
5. **Preview and iterate:** `flywheel preview course.yaml`. Never writes.
6. **Apply:** `flywheel apply course.yaml`. Report the program ID and the
   server-computed TSS.

Other commands: `flywheel list`, `flywheel show <program-id> [--intervals]`
(read a program back from MOWL to confirm what was actually stored),
`flywheel delete <program-id>`, `flywheel lookups` (valid segment/position/
activity types — don't guess).

## Design method

This is the part that decides whether a ride feels designed or generated.

### Align every interval to a musical section

Interval boundaries must land on section boundaries from `--sections`, never
on a uniform grid. A ride built on round 30/45/60-second blocks reads as
monotone no matter how the intensities move, because the changes fight the
song. Real rides have irregular intervals — 0:57, 0:14, 0:28, 0:43, 1:19 —
because songs do.

Fold any section shorter than ~13s into its neighbour, carrying loudness as
the duration-weighted mean. Sub-13s blocks are unrideable and MOWL's own
rides don't use them.

Long blocks are fine. A 2-minute, 2.5-minute or even 3-minute effort is a
normal thing to ask for, and around 3 minutes is the usual upper limit before
a block stops feeling like an interval. This is a style choice, not a rule --
some instructors ride long, some ride short. `scaffold --max-section` splits
anything longer (default 180s); pass `0` to never split.

### Drive intensity from loudness, not position in the ride

Rank each section by `loudness` **within its own track**. The loud sections
are the choruses and drops; the quiet ones are intros, verses and bridges.
Map that rank onto a zone so the chorus is the hard part. This is what makes
a ride track the music.

Apply a curve to the rank (rank^gamma, gamma > 1) rather than a linear map.
That keeps most sections low with sharp peaks on the loudest sections, which
is how you get a low average TSS while keeping real dynamics.

Two approaches that look reasonable and are wrong:

- **Demoting the longest intervals** to hit a TSS target destroys the
  mapping, because the longest sections are usually the loudest. You end up
  with a 14-second burst at fire and the big outro at blue.
- **Compressing the zone range** flattens the whole ride into two zones.

Tune the gamma until TSS lands. It preserves the ordering. `flywheel
scaffold` does this search for you and prints the gamma it settled on.

### Cadence is per track, not per interval

Set one cadence for a whole track, derived from its BPM: use half-time or
two-thirds time to land in a rideable 60-105 rpm. A 186 BPM metal track rides
93 rpm; a 124 BPM rock track becomes a 62 rpm standing climb. Varying cadence
per interval reads as noise. MOWL's own rides show a fixed narrow range like
`88-89` across a run of intervals.

Get texture from **position** instead: alternate seated and standing within
the same zone, standing on the loudest sections.

### Write intensity as MOWL's zone bands

Emit `intensity` as the band, not a single number, so the app shows the same
labels as a MOWL-authored ride:

| Band | %FTP | Coggan zone |
|---|---|---|
| white | `[0, 55]` | 1 |
| blue | `[56, 75]` | 2 |
| green | `[76, 90]` | 3 |
| yellow | `[91, 105]` | 4 |
| red | `[106, 120]` | 5 |
| fire | `[121, 150]` | 6 |
| owl | `[151, 200]` | 7 |

### TSS comes from zone buckets

MOWL derives TSS from each interval's Coggan zone, not its raw %FTP. Moving
an interval from 95% to 105% changes nothing (both zone 4); moving it to 106%
jumps it to zone 5 and costs far more. Zone 6 is very expensive — reserve it
for segment endings rather than sprinkling it through a block, or TSS
overshoots badly.

### Structure

- Work segments **end on red or above**, never blue or white. Build the last
  segment's ending as a ramp — red, then fire, then owl.
- Put an **Active Recovery segment** (`type: recovery`) after the warmup and
  between every pair of work segments, so no two work blocks touch.
- A segment may start or end **mid-track**. `preview` validates the course
  total against the playlist length (±5s) and allows each segment to drift up
  to 120s from the tracks it claims, so an active recovery can be 45s carved
  out of a four-minute song's outro rather than a whole track.
- **Active recovery is short**: 30-90s, never longer. Carve it from the tail
  of the preceding track, where the song is already backing off.
- The **program ends hot**, on red or above — never on a recovery or cooldown
  segment. The cooldown happens after the program, not inside it.
- Give the playlist a **cooldown tail**: leave the last track (or two)
  uncovered by any segment. The program ends on its hot interval and the
  music plays on while the rider spins down. Validation allows uncovered
  tracks only at the end; a gap earlier shifts everything after it.

## Things that will bite you

- **`tracks:` never reaches MOWL.** It is local validation only. MOWL lines
  segments up with the playlist purely by elapsed time, so alignment holds
  because the course's intervals sum to the playlist's real length. That is
  also why a segment boundary may fall mid-track.
- **`apply` replaces, it does not update.** It deletes the same-named program
  and creates a new one, so **the program ID changes on every apply**.
- **A freshly imported playlist indexes asynchronously.** Durations and BPM
  come back as 0 for minutes on a playlist MOWL has not seen. `inspect`
  polls (`--wait`, default 3m) and warns. Building against zero durations
  fails validation with "tracks are 0s".
- **`preview` TSS overestimates the server** on rides made of many short
  section-aligned intervals — measured at 5-6% high across several applies
  (69.9→65.7, 76.4→72.6, 76.2→72.0). On rides with few long intervals the two
  agree within ~1%. Target ~5% above the number you want and confirm against
  `apply`'s server TSS, which is authoritative.
- **Playlist name comes back empty** from the hydrate endpoint; cosmetic.

Use `flywheel show <program-id> --intervals` to settle any question about
what MOWL actually stored, rather than reasoning from the spec alone.

## `course.yaml` reference

```yaml
name: "Hammer & Bass 55"
category: "My Rides"              # personal MOWL category, created if missing
activity: cycling
targets: { duration_min: 56, tss: 72 }
style: [punchy]                    # advisory; resolved via styles.yaml
playlist:
  spotify_id: "EXPLAYLIST0000000000001"
segments:
  - name: "Roll Out"
    type: warmup                   # warmup|intervals|climb|tabata|recovery|cooldown
    tracks: [1, 2, 3]              # song indices from `playlist inspect`
    intervals:                     # durations come from section boundaries
      - { duration: 11, cadence: [94, 95], intensity: [0, 55], position: seated }
      - { duration: 36, cadence: [94, 95], intensity: [56, 75], position: seated }
  - name: "Active Recovery"
    type: recovery
    tracks: [4]
    intervals:
      - { duration: 138, cadence: [74, 75], intensity: [56, 75], position: seated }
```

`intensity` is a scalar (%FTP) or `[from,to]`; `cadence` is `[rpm_from,
rpm_to]`; `position` is `seated`/`standing` (see `lookups`); `duration` is
seconds. Full field semantics and the `styles.yaml` format: see the repo
README and `docs/superpowers/specs/2026-08-18-flywheel-design.md` (§4–5).

## Optional Spotify MCP

`flywheel` never talks to Spotify itself — it only takes a playlist ID. A
Spotify MCP server (e.g. `marcelmarais/spotify-mcp-server`) is optional
tooling for the playlist-authoring half, and needs its own Spotify Developer
app configured outside `flywheel`. If none is available, ask the user for an
existing playlist and continue from step 3.
