# MOWL API Findings — 2026-08-18

## Scope

This document records **PROVISIONAL** findings from the flywheel Task 2 investigation into the MOWL API. Findings marked provisional are working hypotheses to be verified by the user against the live MOWL API using their own credentials. All test fixtures are captured real responses from the MOWL API (no fabrication).

## Fixtures Captured

All fixtures are stored in `testdata/` and represent real MOWL API responses:
- `playlist_import.json` — real `PUT /v1/Spotify/Playlist` response (contains PlaylistID and nested Tracks with BPM and Duration ms)
- `playlist_tracks.json` — real tracks list response
- `audio_analysis.json` — real `AudioAnalysis` response (contains sections)
- `segmentcategories.json` — real segment categories lookup
- `activitytypes.json` — real activity types lookup
- `program_create.json` — real `PUT /v1/programs` response
- `intervals_multiple.json` — real `PUT .../intervals/multiple` response
- `segment_create.json` — representative segment creation response
- `program_tss.json` — representative TSS response (`{"Data":74.2}`)
- `program_playlist.json` — real playlist detail response (stands in for `GET /v1/Playlists/program/{id}`)

All 10 JSON files validated as well-formed (no parse errors).

## Provisional Findings

### 1. Program ↔ Playlist Binding

**Status:** PROVISIONAL — Hypothesis based on API response structure; requires live verification.

**Observation:** The `program_create.json` response suggests that calling `PUT /v1/programs` with `PlaylistID` set on the program body establishes the link between the program and the playlist.

**Verification approach:** The user should create a test program with a playlist ID, then fetch `GET /v1/Playlists/program/{programID}` to confirm the playlist is bound. The `program_playlist.json` fixture shows the expected response structure.

**Confidence:** Low — inferred from response structure only. No live call made to verify roundtrip.

### 2. PositionType IDs

**Status:** PARTIALLY VERIFIED — `1 = Seated` confirmed in earlier live calls this session; `2 = Standing` assumed but unconfirmed.

| PositionType ID | Name | Verified |
|---|---|---|
| 1 | Seated | ✓ Yes (live call earlier) |
| 2 | Standing | ✗ Assumed (not confirmed) |

**Verification approach:** Check `activitytypes.json` response or call `GET /v1/PositionTypes` to see full list and confirm ID 2 is Standing.

**Note:** More PositionType values may exist beyond these two.

### 3. Training Stress Score (TSS)

**Status:** PROVISIONAL — Local estimate used as-is; server calibration not yet tested.

**Observation:** The `program_tss.json` fixture shows TSS returned as `{"Data": 74.2}`. The Coggan-based local estimate is used as-is without additional server-side calibration.

**Verification approach:** 
- Create a test program with known power output or HR zones.
- Compare local Coggan TSS estimate to server TSS from `GET /v1/Programs/{id}/TSS`.
- Adjust calculation if calibration factor is needed.

**Confidence:** Low — TSS calculation not yet benchmarked against live server.

## Smoke Test Requirement

A manual smoke test remains for the user to run:

1. Obtain a valid MOWL API token with create/delete permissions.
2. Create a throwaway course (program + playlist + segments) using the flywheel CLI.
3. Verify in MOWL UI or via `GET` endpoints that the course structure is correct.
4. Delete the course.
5. Confirm deletion in MOWL UI.

This step confirms that the open questions above behave as expected in the real system and validates the complete program creation flow end-to-end.

## Next Steps

1. User runs live smoke test with valid credentials.
2. Findings are updated to "VERIFIED" or adjusted based on results.
3. Remaining ambiguities are resolved (e.g., full list of PositionTypes, TSS calibration).

---

**Generated:** 2026-08-18  
**Fixtures Source:** Real MOWL API responses captured offline this session (no live calls in Task 2)
