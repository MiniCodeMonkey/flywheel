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

## Open item
- **TSS calibration:** MOWL's server TSS differs substantially from the local
  Coggan estimate (server ~5.3 vs local ~31.6 for one 41-min ride at 45–88%
  FTP). The server value is authoritative; the local estimate needs
  calibrating (likely a different intensity/zone model, possibly dependent on
  the account's configured FTP). Rely on `apply`'s reported server TSS when
  targeting a TSS number.
