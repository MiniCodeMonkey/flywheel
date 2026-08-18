# MOWL API findings — live-verified

Status after a live end-to-end run against a real account (create + read +
delete, cleaned up).

## Confirmed working
- **Auth:** two-step ticket flow; envelope `Error` is a JSON string (`""` on
  success), not a `{Message}` object.
- **Playlist hydration:** `PUT /v1/Spotify/Playlist` imports (idempotent — a
  re-import of the same Spotify playlist returns the existing ITC PlaylistID,
  no duplicate). Track metadata comes from `GET /v1/Spotify/Playlists/{spotifyId}`,
  whose track fields are `{ID, Name, Uri, DurationMs, Artist, Tempo, IsPlayable}`
  — `Tempo` is the server-detected BPM (0 until imported/indexed), `DurationMs`
  and `Name` are available immediately.
- **Program↔playlist link:** set `PlaylistID` (the ITC playlist id from the
  import response) in the `PUT /v1/programs` create body. `PUT /v1/programs`
  ALWAYS creates a new program (it ignores `ProgramID` in the body), so there
  is no in-place update and no separate link call — link on create.
- **Positions:** `PositionTypeID` 1 = Seated, 2 = Standing (both accepted live).
- **Segment categories:** warmup(11)/intervals(10)/cooldown(15) etc. are all
  writable by a normal RegisteredUser.
- **Categories:** `PUT /v1/programcategories` always creates a new category, so
  `apply` reuses an existing same-named category (via `POST /v1/Users/{id}/camps`)
  instead of duplicating. `delete` removes the program and then best-effort
  deletes the now-empty private category.
- **TSS endpoint shape:** `GET /v1/calculations/program/{id}/TSS` returns
  `{"ProgramID": N, "TSS": <float>}` (an object, not a bare number).

## TSS model (reverse-engineered live)
- MOWL's TSS is driven ENTIRELY by each interval's `ScaleCoggan` power zone
  (1-7), NOT by `FTPFrom/FTPTo` or `Intensity` (those had zero effect in
  controlled probes). We never set it, so every interval scored as zone 1 —
  hence the ~6x under-read before the fix.
- Per-zone intensity factors (measured, 3600s single intervals):
  z1=0.2775, z2=0.6575, z3=0.8300, z4=0.9800, z5=1.1300, z6=1.3575, z7=1.5025.
- FTP does not modulate IF within a zone (z2 @ 60% == z2 @ 75% == 43.23 TSS/hr).
- The whole-program TSS is Normalized-Power weighted:
  `IF_np = (Σ durᵢ·IF(zoneᵢ)⁴ / Σ durᵢ)^¼`, `TSS = totalHours·IF_np²·100`
  (a z2/z3 mix probe gave server 57.57 vs formula 57.55).
- Fix: `flywheel` maps each interval's %FTP to a Coggan zone, sets
  `ScaleCoggan`, and `preview` uses the same NP model — preview now matches
  the server TSS to within rounding (31.9 vs 31.74 on a 41-min ride).
