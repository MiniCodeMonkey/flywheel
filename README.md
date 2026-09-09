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

Or download a binary from the [releases
page](https://github.com/MiniCodeMonkey/flywheel/releases). Check what you are
running with `flywheel version`; see [RELEASING.md](RELEASING.md) for how
releases are cut.

Requires a MOWL / Intelligent Cycling account.

## Config location

`flywheel` stores its config, cached auth token, and your editable
`styles.yaml` in the OS config dir:

- macOS: `~/Library/Application Support/flywheel/`
- Linux: `$XDG_CONFIG_HOME/flywheel/` (usually `~/.config/flywheel/`)

The token file is written `0600`; your password is never stored.

## Quickstart

```
flywheel init            # writes a starter styles.yaml to the config dir
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
     spotify_id: "EXPLAYLIST0000000000001"
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
   segments in order). Re-applying the same file replaces rather than
   duplicates: a program with the same name is deleted first, so the program
   ID changes on every apply.

   ```
   flywheel apply course.yaml
   ```

## Commands reference

Every subcommand also accepts the global `--json` flag for machine-readable
output.

| Command | Description |
|---|---|
| `flywheel init` | Write a starter `styles.yaml` into the config dir, if one doesn't already exist. |
| `flywheel auth login` | Authenticate with MOWL (`--email`, or prompts; password via `MOWL_PASSWORD` env var or a prompt) and cache a session token. |
| `flywheel playlist inspect <spotify-id>` | Import/read a Spotify playlist via MOWL and print each track's index, title, artist, BPM, and duration. `--sections` also fetches each track's musical sections with start, duration and loudness — the data a ride needs to follow the music. `--wait` (default 3m) bounds how long to wait for a freshly imported playlist to finish indexing. |
| `flywheel preview <course.yaml>` | Validate and render a course's timeline, per-segment breakdown, and estimated TSS — no writes. |
| `flywheel scaffold <spotify-id> --segment "Name:type:1-3"` | Generate a `course.yaml` whose intervals follow the music: one interval per musical section, intensity driven by each section's loudness relative to its own track. Tunes toward `--tss`. |
| `flywheel show <program-id>` | Read a program back from MOWL with its segments and intervals; `--intervals` prints every one. |
| `flywheel apply <course.yaml>` | Validate and create a course in MOWL, replacing any same-named program; reports the server-computed TSS. |
| `flywheel version` | Print the version and VCS revision the binary was built from — check this when `preview` numbers look wrong. |
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
exactly one segment; the whole course's interval durations sum to the
playlist's real length (±5s tolerance); an individual segment may start or
end mid-track, drifting up to 120s from the tracks it claims, which is what
lets an active recovery run 45s inside a four-minute song; segment `type` and
`position` must resolve to valid MOWL IDs.

## `styles.yaml`

An editable glossary of style tags — compact, concrete interval recipes
(position, cadence range, minimum interval length, intensity swing, etc.)
that Claude resolves `style` tags against for a consistent starting point,
while still deviating when a specific song's energy or BPM calls for it.
`flywheel` doesn't "execute" styles; `preview` just surfaces how a tag
resolved so you can see it before applying.

Ships with sensible defaults (`road_cycling`, `punchy`, `climb`, `tabata`,
`recovery`) at [`styles.yaml`](styles.yaml) in this repo; `flywheel init`
copies them to the config dir (see below), where they're yours to
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
it teaches an agent the entire workflow above — eliciting ride parameters,
using a Spotify MCP if available, inspecting the playlist, authoring and
iterating `course.yaml`, previewing, and applying — so you can just ask for
*"a 55-minute heavy-rock ride"* and let Claude drive the CLI.

**Install it** (Claude Code discovers skills under `~/.claude/skills/<name>/`):

```bash
# from a clone of this repo
mkdir -p ~/.claude/skills/flywheel
cp skill/SKILL.md ~/.claude/skills/flywheel/SKILL.md
```

Then start a new Claude Code session and it will pick the skill up
automatically when you ask to build a spinning ride. (For project-scoped
use instead of global, copy it under `<your-project>/.claude/skills/flywheel/`.)

The skill assumes the `flywheel` binary is installed and on your `PATH`, and
that you've run `flywheel init` and `flywheel auth login` once (below).

### Example prompts

Start with the shape of the ride. Duration and TSS are the two numbers worth
naming; everything else the agent will ask about or decide.

> Use flywheel to generate a new 55 minute workout, aim for a TSS of about 88.

> Build me a 45-minute ride around this playlist, nothing above threshold --
> I'm riding easy today. `https://open.spotify.com/playlist/...`

Steer the music. With a Spotify MCP connected the agent can build the
playlist too, so describe the mix rather than picking tracks:

> Let's start with the playlist. I want a mix of rock (AC/DC, Queen, Guns'n'Roses) and maybe some electronic in the mix --
> see my Spotify liked songs. Split into 3-4 segments.

> Swap the two slowest tracks in the climb block for something above 170 BPM.

Steer the structure. These map onto the design rules in the skill:

> Add active recovery segments between the work blocks, and one at the end.

> Every work segment should end on red or fire, never blue or white.

> TSS can be lower, just keep it at least 68.

Iterate on a ride you already have. Applying replaces the program with the
same name, so refinement is just another prompt:

> Show me what's actually in program 324212 and check the recovery segments
> never go above blue.

> That climb block is too long. Cut it to three tracks and push the extra
> intensity into the finish instead.

The agent drives the CLI throughout -- scaffolding, previewing and applying --
so you can stay at the level of how the ride should *feel*.

## How TSS works

`flywheel` targets Training Stress Score (TSS) accurately. MOWL derives an
interval's intensity from its **Coggan power zone** (1-7), not the raw FTP
percentage, and computes program TSS with a Normalized-Power model:

```
IF_np = ( Σ durationᵢ · IF(zoneᵢ)⁴ / Σ durationᵢ )^¼
TSS   = totalHours · IF_np² · 100
```

`flywheel` sets each interval's zone from your `intensity` (% of FTP) and
`preview` uses the same model, so its estimated TSS matches the server
value `apply` reports (verified live to within rounding). Design against
`preview`'s TSS with confidence; `apply` prints the authoritative
server TSS as a final confirmation.

## Disclaimer

Built in public, as a personal tool. Not affiliated with, endorsed by, or
supported by MOWL or Intelligent Training Group. Uses the same REST API the
official MOWL app uses; use your own account and use it responsibly — this
project is not responsible for any account or API issues that result from
its use.
