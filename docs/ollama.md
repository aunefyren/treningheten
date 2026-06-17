# Ollama front-page message

A short, AI-generated greeting shown on the front page, gated by the `Ollama.Enabled`
config boolean. It is produced by `controllers/ollama.go` against an OpenAI-compatible
chat endpoint (`{URL}/v1/chat/completions`, optional `APIKey`, configured `Model`).

## Design principle: everything pre-computed

The model is treated as a *phraser*, not a *reasoner*. `buildOllamaPayload` flattens
one user's state into a JSON object of final facts — week count, season progress,
streaks, wheel stakes — and the system prompt forbids the model from recomputing,
inferring, or inventing anything not in the payload. This keeps small/local models
reliable and avoids them getting the wheel mechanics backwards.

## Payload

`ollamaPromptPayload` (per user, for a point in time):

- `today`, `days_left_in_week_including_today`, `user_first_name`
- `workouts_logged_this_week` — the authoritative weekly count (valid exercises,
  Monday–Sunday), the same unit goals are scored in.
- `recent_workouts` — up to 5 most recent worked-out days, each just `date`,
  `day_of_week`, `exercise_count`, `note`. Flavour only; bare day counts.
- `latest_workout` (optional) — a richer, summary-level view of the **single most
  recent** worked-out day, included only when it is within `latestWorkoutMaxAgeDays`
  (3) so the message never congratulates a stale session. See below.
- `in_any_season` + `seasons[]` — per-season flattened progress and stakes
  (`weekly_goal_met`, `workouts_remaining_to_meet_goal`, `current_streak_weeks`,
  `competing`, `prize_entries_to_win_if_a_rival_fails`, sick-leave fields).

### `latest_workout`

Built by `buildLatestWorkout`. It picks the most recent day with an enabled, active
exercise, drops it if older than 3 days, then rolls the day up via the MCP flattening
helpers (`flattenActivities` → `distanceToKm`, `durationToSeconds`) so units stay
consistent with the rest of the app:

- `date`, `day_of_week`, `days_ago`
- `activities` — distinct action names for the session (with the Hevy custom-exercise
  name fallback, so never "Unknown").
- `total_distance_km` — sum of set distances, normalised to km.
- `duration_minutes` — activity duration when present, else summed set times (counted
  once per activity to avoid Strava's duplicated values double-counting). Remember
  durations are stored as a **seconds** count, not nanoseconds.
- `note`

The system prompt says the model **may** weave a brief, natural remark about it into
the greeting when it adds something, but it is optional and must be skipped when the
block is absent — it is deliberately not "always comment", to keep messages varied.

This is distinct from the WIP idea of **per-exercise** AI feedback in its own space;
this block only lets the front-page greeting occasionally acknowledge the latest
session.

## Caching & scheduling

The rendered message is cached per user keyed by a SHA-256 hash of the payload
(`ollamaPayloadHash`), so identical state never re-hits the model. `OllamaAsyncRefreshCacheForUser`
regenerates in the background (cancelling any in-flight request for that user), and
`OllamaPreCacheForAllUsers` warms every user's cache (daily + on startup cron when
enabled).
