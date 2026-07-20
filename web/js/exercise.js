// Activity tag vocabulary (mirrors models/tag.go). Strava auto-fills the first
// four (commute + workout_type); the rest are user-only because Strava's public
// API does not expose them.
const ACTIVITY_TAGS = [
    { slug: "race", label: "Race" },
    { slug: "long-run", label: "Long Run" },
    { slug: "workout", label: "Workout" },
    { slug: "commute", label: "Commute" },
    { slug: "for-a-cause", label: "For a Cause" },
    { slug: "recovery", label: "Recovery" },
    { slug: "with-pet", label: "With Pet" },
    { slug: "with-kid", label: "With Kid" },
];

function tagLabel(slug) {
    const tag = ACTIVITY_TAGS.find(t => t.slug === slug);
    return tag ? tag.label : slug;
}

// The user's gear (`gearList`), the render helpers and the CRUD (createGear/updateGearField/
// deleteGear) are shared with the /gear page — see web/js/gear-shared.js. This file keeps only
// the exercise-specific session gear *selector* and the modal open/render glue.

function escapeHTML(value) {
    return String(value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}

// Read-only chips for the summary card.
function generateTagChipsHTML(tags) {
    if (!tags || tags.length === 0) {
        return "";
    }
    const chips = tags
        .map(slug => `<span class="tag-chip tag-chip-readonly">${escapeHTML(tagLabel(slug))}</span>`)
        .join("");
    return `<div class="tag-list">${chips}</div>`;
}

// Toggleable chips for the full editor.
function generateTagSelectorHTML(operation) {
    const selected = operation.tags || [];
    const chips = ACTIVITY_TAGS
        .map(t => {
            const isSel = selected.includes(t.slug) ? " tag-chip-selected" : "";
            return `<span class="tag-chip clickable${isSel}" data-tag="${t.slug}" onclick="toggleTag('${operation.id}', this)">${escapeHTML(t.label)}</span>`;
        })
        .join("");
    return `<div class="tag-selector" id="tag-selector-${operation.id}">${chips}</div>`;
}

function toggleTag(operationID, el) {
    el.classList.toggle("tag-chip-selected");
    updateOperation(operationID);
}

function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
        user_id = login_data.data.id
        media_enabled = login_data.media_enabled === true;

        try {
            admin = login_data.data.admin
        } catch {
            admin = false
        }

        showAdminMenu(admin)

    } else {
        var login_data = false;
        user_id = 0
        admin = false;
        media_enabled = false;
    }

    try {
        string_index = document.URL.lastIndexOf('/');
        exerciseDayID = document.URL.substring(string_index+1);
        console.log(exerciseDayID);
    } catch {
        exerciseDayID = 0;
    }

    var html = `
        <div class="" id="front-page">

            <div class="module">
            
                <div class="text-body u-text-center">
                    <div class="exerciseDayWrapper" id="exerciseDayWrapper">
                        <p id="exercise-day-date" class="u-text-center">...</p>
                        <p id="exercise-day-exercise-goal" class="u-text-center">...</p>

                        <textarea onchange="updateExerciseDay('${exerciseDayID}')" class="day-note-area u-mt-1" id="exercise-day-note" name="exercise-day-exercise-note" rows="3" cols="33" placeholder="Notes"></textarea>
                    </div>
                </div>

                <hr class="u-my-1">

                <div class="exercisesWrapper" id="exercisesWrapper"></div>

                <div class="addExerciseWrapper clickable hover" id="addExerciseWrapper" title="Add session" onclick="addExercise('${exerciseDayID}');">
                    <img src="/assets/plus.svg" class="button-icon">
                </div>

                <div class="u-mt-3" style="display: none;" id="stravaCombineButtonWrapper">
                    <button type="submit" class="btn" id="" onclick="combineStravaExercises(); return false;">
                        Combine Strava exercises
                    </button>
                </div>

            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Write every detail.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        loadExerciseList()
        loadGearList()
        getExerciseDay(exerciseDayID);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function getExerciseDay(exerciseDayID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {

                error(result.error);

            } else {

                clearResponse();
                placeExerciseDay(result.exercise_day);
            }

        } else {
            info("Loading exercise day...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/exercise-days/" + exerciseDayID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeExerciseDay(exerciseDay) {
    var dateString = "Error"
    try {
        var resultDate = new Date(Date.parse(exerciseDay.date));
        dateString = date_start_string = GetDateString(resultDate, true)
    } catch(e) {
        console.log("Error: " + e)
    }

    document.getElementById('exercise-day-date').innerHTML = "<b>Date: " + dateString + "</b>";
    //document.getElementById('exercise-day-exercise-goal').innerHTML = "Exercise goal for week: " + exerciseDay.goal.exercise_interval;
    document.getElementById('exercise-day-note').innerHTML = exerciseDay.note;

    placeExercises(exerciseDay.exercises);
}

var exerciseCache = {};

function placeExercises(exercises) {
    exercisesHTML = "";
    counter = 1;
    exerciseCache = {};

    try {
        document.getElementById('stravaCombineButtonWrapper').style.display = "none";
    } catch(e) {
        console.log("Failed to remove Strava button. Error: " + e)
    }

    exercises.forEach(exercise => {
        exercise._count = counter;
        exerciseCache[exercise.id] = exercise;

        var exerciseGenerated = generateExerciseHTML(exercise, counter)
        var exerciseHTML = `
            <div class="exerciseWrapper" id="exercise-${exercise.id}">
                ${exerciseGenerated}
            </div>
        `;

        if(exerciseGenerated != null) {
            exercisesHTML += exerciseHTML;
            counter += 1;
        }
    });

    document.getElementById('exercisesWrapper').innerHTML = exercisesHTML;
}

function generateExerciseHTML(exercise, count, forceFullEditor = false) {
    var exerciseHTML = null;

    if(exercise.is_on) {

        if(!forceFullEditor) {
            return renderWorkoutSummary(exercise, count);
        }

        exerciseHTML = renderWorkoutEditor(exercise, count);
    } else if (exercise.operations.length > 0){
        exerciseHTML = `
            <div class="exerciseSubWrapper" id="exercise-sub-${exercise.id}">
                <h2>Deleted session</h2>
                
                <p>
                    Contains ${exercise.operations.length} exercise(s).
                </p>

                <input type="hidden" id="exercise-time-input-${exercise.id}" name="exercise-time-input" pattern="[0-9:]{0,}" placeholder="hh:mm:ss" value="${secondsToDurationString(exercise.duration)}">
                <textarea class="day-note-area u-mt-1" id="exercise-note-${exercise.id}" name="exercise-exercise-note" rows="3" cols="33" placeholder="Notes" style="display: none;">${exercise.note}</textarea>

                <button type="submit" onclick="updateExercise('${exercise.id}', true, ${count}, '${exercise.time});" id="restore-exercise-button-${exercise.id}" class="btn u-w-8"><img src="/assets/refresh-cw.svg" class="color-invert">Restore</button>

                <hr class="u-my-1">
            </div>
        `;
    }

    return exerciseHTML;
}

// ── Universal workout summary view ──────────────────────────────────────────
// Every session renders a read-only overview by default; the editor is an
// explicit mode (the edit action). The overview composes one sub-card per
// activity (cardio / strength / time) so it scales from a lone run to a
// run + bench + core session, regardless of source (Strava, manual, future HEVY).

function renderWorkoutSummary(exercise, count) {
    const operations = (exercise.operations || []).filter(op => op);

    var dateLine = "";
    if (exercise.time) {
        const date = new Date(exercise.time);
        const day = date.toLocaleDateString('en-GB', { weekday: 'short', day: 'numeric', month: 'short' });
        const clock = date.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false });
        dateLine = day + " · " + clock;
    }

    const totalDuration = exercise.duration ? secondsToDurationString(exercise.duration) : "—";
    const summaryLine = operations.map(activityTitle).join("  +  ") || "Empty session";
    const subCards = operations.map(op => renderActivitySubCard(op, exercise)).join("");

    return `
        <div class="workout-view">
            <div class="wv-session" id="exercise-sub-${exercise.id}">
                <div class="wv-session-head">
                    <div class="wv-session-meta">
                        <span class="wv-session-date">${dateLine}</span>
                        <span class="wv-session-summary">${escapeHTML(summaryLine)}</span>
                    </div>
                    <div class="wv-session-total">
                        <span class="wv-total-value">${totalDuration}</span>
                        <span class="wv-total-label">Total</span>
                    </div>
                </div>
                ${renderSessionActionRow(exercise, count)}
                <div class="wv-activities">
                    ${subCards}
                </div>
                ${mediaTimelineHTML(exercise)}
            </div>
        </div>
    `;
}

function renderSessionActionRow(exercise, count) {
    var combineHTML = "";
    var divideHTML = "";
    if (exercise.strava_id && exercise.strava_id.length > 0) {
        const ids = exercise.strava_id.join(";");
        combineHTML = `<input class="clickable stravaCombineCheck wv-combine" type="checkbox" id="${ids}" title="Select to combine with another session">`;
        if (exercise.strava_id.length > 1) {
            divideHTML = `<img src="/assets/scissors.svg" class="btn_logo clickable wv-action-icon" title="Split combined activities" onclick="divideStravaExercises('${exercise.id}')">`;
        }
        try { document.getElementById('stravaCombineButtonWrapper').style.display = "flex"; } catch(e) {}
    }
    return `
        <div class="wv-actions">
            ${combineHTML}
            ${divideHTML}
            <img src="/assets/edit.svg" class="btn_logo clickable wv-action-icon" title="Edit session" onclick="switchToFullEditor('${exercise.id}')">
            <img src="/assets/trash-2.svg" class="btn_logo clickable wv-action-icon" title="Delete session" onclick="updateExercise('${exercise.id}', false, ${count}, '${exercise.time}', true)">
        </div>
    `;
}

// Dispatch one activity to the right sub-card by its type.
// Shared Strava re-sync button used by every activity sub-card (cardio /
// strength / time), so the three renderers can't drift on when it appears.
// Needs the owning operation-set id — that's what stravaSyncOperationSet posts.
function stravaSyncButtonHTML(setID, stravaID) {
    if (!stravaID || !setID) return "";
    return `<img src="/assets/refresh-cw.svg" class="btn_logo clickable wv-action-icon" title="Re-sync from Strava" onclick="stravaSyncOperationSet('${setID}')">`;
}

function renderActivitySubCard(operation, exercise) {
    if (operation.type === 'lifting') {
        return renderStrengthSubCard(operation, exercise);
    } else if (operation.type === 'timing') {
        return renderTimeSubCard(operation, exercise);
    }
    return renderCardioSubCard(operation, exercise);
}

function activityActionIcon(operation) {
    const action = operation.action;
    if (action && action.has_logo) {
        return `<img src="/assets/actions/${action.name}.svg" class="wv-activity-icon" alt="">`;
    }
    const emoji = operation.type === 'lifting' ? '🏋️' : operation.type === 'timing' ? '⏱️' : '🏃';
    return `<span class="wv-activity-emoji">${emoji}</span>`;
}

function activityTitle(operation) {
    if (operation.note && operation.note.trim() !== '') return operation.note.trim();
    if (operation.action && operation.action.name) return operation.action.name;
    return operation.type === 'lifting' ? 'Strength' : operation.type === 'timing' ? 'Activity' : 'Cardio';
}

// Generic provider badge — derived client-side from the available ids (Strava if a strava
// id is present, Hevy if the parent exercise was imported from Hevy, else manual). This is
// the single place to swap when the backend grows a real `source` object.
function sourceBadge(operation, stravaID, exercise) {
    if (stravaID) {
        return `<a class="wv-source wv-source-strava" href="https://www.strava.com/activities/${stravaID}" target="_blank" title="View on Strava">
            <img src="/assets/strava-logo.svg" alt=""><span>Strava</span>
        </a>`;
    }
    // Hevy has no public per-workout URL, so this is a non-link source marker.
    if (exercise && exercise.hevy_workout_id) {
        return `<span class="wv-source wv-source-hevy" title="Imported from Hevy">
            <img src="/assets/hevy.png" alt=""><span>Hevy</span>
        </span>`;
    }
    return `<span class="wv-source wv-source-manual">Manual</span>`;
}

function activityDescriptionHTML(operation) {
    if (operation.description && operation.description.trim() !== '') {
        return `<p class="wv-description">${escapeHTML(operation.description.trim())}</p>`;
    }
    return "";
}

// The "soundtrack" overlay: the session's listening history plotted against the
// session clock (see docs/media.md). Only shown when the media feature is enabled;
// the re-pull action re-matches the session window against the connected provider.
// It is a session-level fact — the match window is one session window, so a
// multi-operation session shares one soundtrack rather than duplicating per activity.
function mediaTimelineHTML(exercise) {
    if (!media_enabled) return "";

    const items = exercise.media_playback || [];
    const repull = `<button class="wv-media-repull" title="Match listening history" onclick="mediaSyncExercise('${exercise.id}')"><img src="/assets/refresh-cw.svg" alt="Re-match"></button>`;
    const eq = `<span class="wv-eq" aria-hidden="true"><i></i><i></i><i></i></span>`;

    if (items.length === 0) {
        return `<div class="wv-media wv-media-empty">
            <span class="wv-media-eyebrow"><span class="wv-eq is-muted" aria-hidden="true"><i></i><i></i><i></i></span> Soundtrack</span>
            <span class="wv-media-empty-text">No matched listening</span>
            ${repull}
        </div>`;
    }

    // Source is plumbing, not content — show it only when a session genuinely spans
    // more than one provider, where a per-row tag disambiguates interleaved plays.
    const crossSource = new Set(items.map(it => it.provider).filter(Boolean)).size > 1;

    // Session start anchors elapsed time; the effort per track comes from a Strava
    // activity's streams when the session has one. A session can hold several
    // activities but only one carries streams in practice, so use the first found —
    // effort stats are best-effort for multi-activity sessions (see docs/media.md).
    const sessionStartMs = (exercise && exercise.time) ? Date.parse(exercise.time) : NaN;
    const streamOp = (exercise.operations || []).find(op => op && (op.operation_sets || []).some(s => s && s.strava_streams));
    const streamSet = streamOp ? (streamOp.operation_sets || []).find(s => s && s.strava_streams) : null;
    const streams = streamSet ? streamSet.strava_streams : null;

    // First pass: each track's elapsed stamp + effort over its play window. A track
    // scrobbles when it finishes, so its window is [start − length, start]; when the
    // length is unknown, fall back to the gap since the previous track.
    let prevEndSec = null;
    const tracks = items.map(it => {
        const startMs = Date.parse(it.started_at);
        let stamp = "";
        let endSec = null;
        if (!isNaN(sessionStartMs) && !isNaN(startMs)) {
            endSec = Math.round((startMs - sessionStartMs) / 1000);
            const shown = endSec < 0 ? 0 : endSec;
            if (shown <= 86400) stamp = secondsToDurationString(shown);
        }
        if (stamp === "" && !isNaN(startMs)) {
            const d = new Date(startMs);
            stamp = padNumber(d.getHours(), 2) + ":" + padNumber(d.getMinutes(), 2);
        }

        // Spoken media (podcast/audiobook) get minutes listened, not a per-beat effort
        // that would be meaningless over a long talk — so skip the stream stat for them.
        const spoken = it.media_type === 'podcast' || it.media_type === 'audiobook';

        let stat = null;
        if (!spoken && streams && endSec != null) {
            let startSec = it.track_length ? endSec - it.track_length
                         : (prevEndSec != null ? prevEndSec : endSec - 210);
            if (startSec < 0) startSec = 0;
            stat = streamWindowStats(streams, startSec, Math.max(startSec, endSec));
        }
        if (endSec != null) prevEndSec = endSec;

        return { it, stamp, stat, spoken };
    });

    // The peak-effort track (highest average heart rate) gets a quiet highlight.
    let peakHr = -1;
    tracks.forEach(t => { if (t.stat && t.stat.hr != null && t.stat.hr > peakHr) peakHr = t.stat.hr; });

    const rows = tracks.map(t => {
        const it = t.it;
        // Artist line carries the secondary label plus, on cross-source sessions only,
        // a quiet source tag. It renders when either piece is present.
        const parts = [];
        if (it.artist) parts.push(`<span class="name">${escapeHTML(it.artist)}</span>`);
        if (crossSource) parts.push(`<span class="wv-src">${escapeHTML(mediaSourceLabel(it.provider))}</span>`);
        const artist = parts.length ? `<span class="wv-track-artist">${parts.join(" ")}</span>` : "";
        const isPeak = t.stat && t.stat.hr != null && t.stat.hr === peakHr && peakHr > 0;
        const stat = t.spoken ? mediaSpanStatHTML(it) : trackStatHTML(t.stat, isPeak);
        return `<li class="wv-track${isPeak ? ' is-peak' : ''}">
            ${mediaNodeHTML(it.media_type)}
            <span class="wv-track-time">${escapeHTML(t.stamp)}</span>
            <span class="wv-track-main">
                <span class="wv-track-title">${escapeHTML(it.title)}</span>
                ${artist}
            </span>
            ${stat}
        </li>`;
    }).join("");

    return `<div class="wv-media">
        <div class="wv-media-head">
            <span class="wv-media-eyebrow">${eq} Soundtrack <span class="wv-media-count">${items.length}</span></span>
            ${repull}
        </div>
        <ul class="wv-tracklist">${rows}</ul>
    </div>`;
}

// streamWindowStats averages a Strava stream channel over an elapsed-time window
// [startSec, endSec]. streams.time.data is seconds-from-start and the channel arrays
// are parallel to it; when no time channel exists the sample index is used as the
// second. Returns null when no usable channel is present.
function streamWindowStats(streams, startSec, endSec) {
    if (!streams) return null;
    const timeData = (streams.time && streams.time.data) ? streams.time.data : null;
    const hrData = (streams.heartrate && streams.heartrate.data) ? streams.heartrate.data : null;
    const altData = (streams.altitude && streams.altitude.data) ? streams.altitude.data : null;
    const velData = (streams.velocity_smooth && streams.velocity_smooth.data) ? streams.velocity_smooth.data : null;

    const len = (hrData || altData || velData || []).length;
    if (!len) return null;

    let hrSum = 0, hrN = 0, velSum = 0, velN = 0, elevGain = 0, prevAlt = null;
    for (let i = 0; i < len; i++) {
        const t = timeData ? timeData[i] : i;
        if (t > endSec) break;
        const a = altData ? altData[i] : null;
        if (t < startSec) { prevAlt = a; continue; }
        if (hrData && hrData[i] > 0) { hrSum += hrData[i]; hrN++; }
        if (velData && velData[i] != null) { velSum += velData[i]; velN++; }
        if (a != null) {
            if (prevAlt != null && a > prevAlt) elevGain += a - prevAlt;
            prevAlt = a;
        }
    }

    return {
        hr: hrN ? Math.round(hrSum / hrN) : null,
        speedKmh: velN ? (velSum / velN) * 3.6 : null,
        elevGain: elevGain >= 1 ? Math.round(elevGain) : null
    };
}

// trackStatHTML renders the one most telling per-track metric: heart rate when it's
// recorded (intensity during the song), else pace, else climb. Empty when no streams.
function trackStatHTML(stat, isPeak) {
    if (!stat) return `<span class="wv-track-stat"></span>`;
    if (stat.hr != null) {
        const title = isPeak ? ' title="Hardest effort"' : '';
        return `<span class="wv-track-stat"${title}><span class="wv-track-stat-value">${stat.hr}</span><span class="wv-track-stat-unit">bpm</span></span>`;
    }
    if (stat.speedKmh != null) {
        return `<span class="wv-track-stat"><span class="wv-track-stat-value">${stat.speedKmh.toFixed(1)}</span><span class="wv-track-stat-unit">km/h</span></span>`;
    }
    if (stat.elevGain != null) {
        return `<span class="wv-track-stat"><span class="wv-track-stat-value">${stat.elevGain}</span><span class="wv-track-stat-unit">m&uarr;</span></span>`;
    }
    return `<span class="wv-track-stat"></span>`;
}

// mediaNodeHTML renders the rail node. A song keeps the plain amber dot; podcasts and
// audiobooks get a typed amber glyph so a mixed session reads at a glance. Type comes
// from the provider's classification (Audiobookshelf natively, Plex by library
// agent/name); Spotify is music-only, so its rows are always dots.
function mediaNodeHTML(mediaType) {
    if (mediaType === 'podcast') {
        return `<span class="wv-node wv-node--icon" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="2" width="6" height="11" rx="3" fill="currentColor" stroke="none"></rect><path d="M5 10.5a7 7 0 0 0 14 0"></path><line x1="12" y1="17.5" x2="12" y2="21"></line><line x1="8.5" y1="21" x2="15.5" y2="21"></line></svg></span>`;
    }
    if (mediaType === 'audiobook') {
        return `<span class="wv-node wv-node--icon" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round"><path d="M12 5.5C10 4.3 7 4 4 4.4v13.4c3-.4 6 0 8 1.2 2-1.2 5-1.6 8-1.2V4.4c-3-.4-6-.1-8 1.1z"></path><line x1="12" y1="5.5" x2="12" y2="19"></line></svg></span>`;
    }
    return `<span class="wv-node wv-node--song" aria-hidden="true"></span>`;
}

// mediaSpanStatHTML renders minutes listened for spoken media, from the item's track
// length (else its played span) — the row's right-hand metric adapts to content type.
function mediaSpanStatHTML(it) {
    let sec = it.track_length || 0;
    if (!sec && it.ended_at && it.started_at) {
        sec = Math.round((Date.parse(it.ended_at) - Date.parse(it.started_at)) / 1000);
    }
    if (!sec || sec < 0) return `<span class="wv-track-stat"></span>`;
    const mins = Math.max(1, Math.round(sec / 60));
    return `<span class="wv-track-stat"><span class="wv-track-stat-value">${mins}</span><span class="wv-track-stat-unit">min</span></span>`;
}

// mediaSourceLabel is the provider display name used by the cross-source tag.
function mediaSourceLabel(provider) {
    switch (provider) {
        case 'plex': return 'Plex';
        case 'spotify': return 'Spotify';
        case 'audiobookshelf': return 'Audiobookshelf';
        default: return provider ? provider.charAt(0).toUpperCase() + provider.slice(1) : '';
    }
}

function mediaSyncExercise(exerciseID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            if (result.error) {
                error(result.error);
            } else if (result.warning) {
                // One provider succeeded (or none matched) but another reported an
                // issue — surface it without blocking the result that did land.
                error(result.warning);
                location.reload();
            } else {
                success("Listening history updated.");
                location.reload();
            }
        } else {
            info("Matching listening history...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises/" + exerciseID + "/media-sync");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// Trim trailing zeros from a metric for display (80.0 -> "80", 2.5 -> "2.5").
function wvNum(n) {
    if (n == null) return "";
    return (Math.round(n * 100) / 100).toString();
}

function renderCardioSubCard(operation, exercise) {
    const sets = operation.operation_sets || [];

    // Aggregate across the activity's sets; pick the set carrying streams for the
    // chart/map (a Strava activity has exactly one).
    var distance = 0, hasDistance = false, movingSecs = 0, totalSecs = 0;
    var streamSet = null, stravaID = null;
    sets.forEach(s => {
        if (s.distance != null) { distance += s.distance; hasDistance = true; }
        if (s.moving_time) movingSecs += s.moving_time;
        if (s.time) totalSecs += s.time;
        if (!streamSet && s.strava_streams) streamSet = s;
        if (!stravaID && s.strava_id) stravaID = s.strava_id;
    });

    const durationSecs = totalSecs || movingSecs || operation.duration || 0;
    const durationHTML = durationSecs ? secondsToDurationString(durationSecs) : "—";
    const distanceHTML = hasDistance ? wvNum(parseFloat(distance.toFixed(2))) + " " + operation.distance_unit : "—";

    var avgHTML = "—";
    const speedSecs = movingSecs || totalSecs;
    if (hasDistance && speedSecs) {
        avgHTML = parseFloat(distance / (speedSecs / 3600)).toFixed(2) + " " + operation.distance_unit + "/h";
    }

    const streams = streamSet ? streamSet.strava_streams : null;
    const setForStreams = streamSet || sets[0] || {};
    const hasHeartrate = streams && streams.heartrate && streams.heartrate.data && streams.heartrate.data.length > 0;
    const latlngData = streams && (streams.latlng || streams.lat_lng);
    const hasRoute = latlngData && latlngData.data && latlngData.data.length > 0;

    var elevationHTML = "";
    if (streams && streams.altitude && streams.altitude.data && streams.altitude.data.length > 1) {
        const altData = streams.altitude.data;
        let gain = 0;
        for (let i = 1; i < altData.length; i++) {
            const delta = altData[i] - altData[i - 1];
            if (delta > 0) gain += delta;
        }
        if (gain > 0) {
            elevationHTML = `<div class="wv-stat"><span class="wv-stat-value">${Math.round(gain)}<span class="wv-stat-unit">m</span></span><span class="wv-stat-label">Elevation</span></div>`;
        }
    }

    var hrStatsHTML = "";
    if (hasHeartrate) {
        const hrVals = streams.heartrate.data.filter(v => v > 0);
        if (hrVals.length > 0) {
            const hrAvg = Math.round(hrVals.reduce((a, b) => a + b, 0) / hrVals.length);
            hrStatsHTML = `<div class="wv-stat"><span class="wv-stat-value">${hrAvg}<span class="wv-stat-unit">bpm</span></span><span class="wv-stat-label">Avg HR</span></div>`;
        }
    }

    const setKey = setForStreams.id || operation.id;
    const hrCanvasID = `hr-chart-${setKey}`;
    const mapDivID = `route-map-${setKey}`;
    const hrHTML = hasHeartrate ? `<div class="wv-hr-wrapper"><canvas id="${hrCanvasID}" class="wv-hr-chart"></canvas></div>` : "";
    const mapHTML = hasRoute ? `<div id="${mapDivID}" class="wv-map"></div>` : "";

    if (hasHeartrate || hasRoute) {
        scheduleCardioStreamRender(setForStreams, streams, hrCanvasID, mapDivID, hasHeartrate, hasRoute, latlngData);
    }

    // Surface the gear used (moving activities only) so it reads without editing.
    var gearHTML = "";
    if (operation.gear) {
        const nick = operation.gear.nickname ? " (" + operation.gear.nickname + ")" : "";
        gearHTML = `<div class="wv-gear"><img src="/assets/sliders.svg" class="wv-gear-icon" alt="">${escapeHTML(operation.gear.name + nick)}</div>`;
    }

    return `
        <div class="wv-activity wv-activity-cardio">
            <div class="wv-activity-head">
                <div class="wv-activity-title">${activityActionIcon(operation)}<span>${escapeHTML(activityTitle(operation))}</span></div>
                <div class="wv-activity-head-actions">${stravaSyncButtonHTML(setForStreams.id, stravaID)}${sourceBadge(operation, stravaID, exercise)}</div>
            </div>
            ${generateTagChipsHTML(operation.tags)}
            ${gearHTML}
            ${activityDescriptionHTML(operation)}
            <div class="wv-stats">
                <div class="wv-stat"><span class="wv-stat-value">${durationHTML}</span><span class="wv-stat-label">Duration</span></div>
                <div class="wv-stat"><span class="wv-stat-value">${distanceHTML}</span><span class="wv-stat-label">Distance</span></div>
                <div class="wv-stat"><span class="wv-stat-value">${avgHTML}</span><span class="wv-stat-label">Avg speed</span></div>
                ${elevationHTML}
                ${hrStatsHTML}
            </div>
            ${mapHTML}
            ${hrHTML}
        </div>
    `;
}

function renderStrengthSubCard(operation, exercise) {
    const sets = operation.operation_sets || [];
    var totalReps = 0, volume = 0, hasVolume = false, stravaID = null, stravaSetID = null;
    const setChips = sets.map(s => {
        if (!stravaID && s.strava_id) { stravaID = s.strava_id; stravaSetID = s.id; }
        const reps = (s.repetitions != null) ? s.repetitions : null;
        const weight = (s.weight != null) ? s.weight : null;
        if (reps != null) totalReps += reps;
        if (reps != null && weight != null) { volume += reps * weight; hasVolume = true; }
        var label;
        if (weight != null && reps != null) label = `${wvNum(weight)}<span class="wv-set-mult">×</span>${wvNum(reps)}`;
        else if (reps != null) label = `${wvNum(reps)} reps`;
        else if (weight != null) label = `${wvNum(weight)} ${operation.weight_unit}`;
        else label = "—";
        return `<span class="wv-set">${label}</span>`;
    }).join("");

    var rollup = `${sets.length} ${sets.length === 1 ? 'set' : 'sets'}`;
    if (totalReps > 0) rollup += ` · ${wvNum(totalReps)} reps`;
    if (hasVolume) rollup += ` · ${wvNum(Math.round(volume))} ${operation.weight_unit} vol`;

    return `
        <div class="wv-activity wv-activity-strength">
            <div class="wv-activity-head">
                <div class="wv-activity-title">${activityActionIcon(operation)}<span>${escapeHTML(activityTitle(operation))}</span></div>
                <div class="wv-activity-head-actions">${stravaSyncButtonHTML(stravaSetID, stravaID)}${sourceBadge(operation, stravaID, exercise)}</div>
            </div>
            ${generateTagChipsHTML(operation.tags)}
            ${activityDescriptionHTML(operation)}
            <div class="wv-set-grid">${setChips || '<span class="wv-set wv-set-empty">No sets</span>'}</div>
            <div class="wv-rollup">${rollup}</div>
        </div>
    `;
}

function renderTimeSubCard(operation, exercise) {
    const sets = operation.operation_sets || [];
    var secs = 0, stravaID = null, stravaSetID = null;
    sets.forEach(s => {
        if (s.time) secs += s.time;
        else if (s.moving_time) secs += s.moving_time;
        if (!stravaID && s.strava_id) { stravaID = s.strava_id; stravaSetID = s.id; }
    });
    if (!secs && operation.duration) secs = operation.duration;
    const durationHTML = secs ? secondsToDurationString(secs) : "—";

    return `
        <div class="wv-activity wv-activity-time">
            <div class="wv-activity-head">
                <div class="wv-activity-title">${activityActionIcon(operation)}<span>${escapeHTML(activityTitle(operation))}</span></div>
                <div class="wv-activity-head-actions">${stravaSyncButtonHTML(stravaSetID, stravaID)}${sourceBadge(operation, stravaID, exercise)}</div>
            </div>
            ${generateTagChipsHTML(operation.tags)}
            ${activityDescriptionHTML(operation)}
            <div class="wv-stats">
                <div class="wv-stat"><span class="wv-stat-value">${durationHTML}</span><span class="wv-stat-label">Duration</span></div>
            </div>
        </div>
    `;
}

// Deferred chart/map render: summary HTML is injected via innerHTML, so poll
// until the canvas/map nodes exist, then draw. (Ported from the old simple card.)
function scheduleCardioStreamRender(set, streams, hrCanvasID, mapDivID, hasHeartrate, hasRoute, latlngData) {
    var attempts = 0;
    var interval = setInterval(function() {
        attempts++;
        var hrReady = !hasHeartrate || document.getElementById(hrCanvasID);
        var mapReady = !hasRoute || document.getElementById(mapDivID);
        if (hrReady && mapReady) {
            clearInterval(interval);
            if (hasHeartrate) {
                // Streams are distance-sampled, not time-sampled, so keep only the
                // moving samples (velocity > 0.5 m/s) and space them evenly across
                // moving_time so the x-axis matches the activity duration.
                const hrRaw = streams.heartrate.data;
                const velocity = streams.velocity_smooth && streams.velocity_smooth.data;
                const movingTimeSecs = set.moving_time || set.time;
                let chartHrData, chartTimeData;
                if (velocity && velocity.length === hrRaw.length && movingTimeSecs) {
                    const movingIndices = [];
                    for (let i = 0; i < hrRaw.length; i++) {
                        if (velocity[i] > 0.5) movingIndices.push(i);
                    }
                    const n = movingIndices.length;
                    chartTimeData = movingIndices.map((idx, j) => Math.round(j / (n - 1) * movingTimeSecs));
                    chartHrData = movingIndices.map(idx => hrRaw[idx]);
                } else {
                    chartTimeData = streams.time.data;
                    chartHrData = hrRaw;
                }
                renderHeartrateChart(hrCanvasID, chartTimeData, chartHrData, movingTimeSecs);
            }
            if (hasRoute) {
                renderRouteMap(mapDivID, latlngData.data);
            }
        } else if (attempts > 50) {
            clearInterval(interval);
        }
    }, 50);
}

function renderHeartrateChart(canvasID, timeData, hrData, maxSecs) {
    var canvas = document.getElementById(canvasID);
    if (!canvas) return;

    // Build {x, y} points — both arrays are already aligned (same length)
    var points = [];
    for (var i = 0; i < hrData.length; i++) {
        if (hrData[i] > 0) {
            points.push({
                x: timeData[i],
                y: hrData[i]
            });
        }
    }

    if (points.length === 0) return;

    // Read the theme tokens so the chart's axis text/grid match the light surface (was white-on-dark).
    var rootStyle = getComputedStyle(document.documentElement);
    var tickColor = rootStyle.getPropertyValue("--lightblue").trim() || "#7fa8cf";
    var gridColor = rootStyle.getPropertyValue("--grey").trim() || "#cecece";

    new Chart(canvas, {
        type: "line",
        data: {
            datasets: [{
                label: "Heart Rate (bpm)",
                data: points,
                fill: true,
                borderColor: "rgba(220, 80, 80, 1)",
                pointBackgroundColor: "rgba(220, 80, 80, 1)",
                backgroundColor: "rgba(220, 80, 80, 0.2)",
                tension: 0.3,
                pointRadius: 0,
                borderWidth: 2
            }]
        },
        options: {
            legend: { display: false },
            scales: {
                xAxes: [{
                    type: "linear",
                    gridLines: { color: gridColor },
                    ticks: {
                        fontColor: tickColor,
                        autoSkip: true,
                        maxTicksLimit: 6,
                        min: 0,
                        max: maxSecs || undefined,
                        callback: function(value) {
                            return secondsToDurationString(value);
                        }
                    }
                }],
                yAxes: [{
                    gridLines: { color: gridColor },
                    ticks: {
                        fontColor: tickColor,
                        beginAtZero: false,
                        precision: 0,
                        callback: function(value) { return value + " bpm"; }
                    }
                }]
            },
            tooltips: {
                callbacks: {
                    title: function(items) {
                        return secondsToDurationString(items[0].xLabel);
                    },
                    label: function(item) {
                        return item.yLabel + " bpm";
                    }
                }
            }
        }
    });
}

function renderRouteMap(divID, latlngData) {
    var mapDiv = document.getElementById(divID);
    if (!mapDiv) { console.warn('[map] div not found:', divID); return; }
    if (!latlngData || latlngData.length === 0) { console.warn('[map] no latlng data'); return; }

    // Leaflet must be available globally
    if (typeof L === 'undefined') {
        console.warn('[map] Leaflet not loaded');
        mapDiv.style.display = 'none';
        return;
    }
    console.log('[map] rendering', latlngData.length, 'points, first point:', latlngData[0]);

    var map = L.map(divID, { zoomControl: true, scrollWheelZoom: false });

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        maxZoom: 18
    }).addTo(map);

    var latLngs = latlngData.map(function(point) {
        return [point[0], point[1]];
    });

    var polyline = L.polyline(latLngs, {
        color: 'rgba(220, 80, 80, 1)',
        weight: 3,
        opacity: 0.85
    }).addTo(map);

    // Start marker
    L.circleMarker(latLngs[0], {
        radius: 6, color: '#4caf50', fillColor: '#4caf50', fillOpacity: 1
    }).addTo(map);

    // End marker
    L.circleMarker(latLngs[latLngs.length - 1], {
        radius: 6, color: '#f44336', fillColor: '#f44336', fillOpacity: 1
    }).addTo(map);

    map.fitBounds(polyline.getBounds(), { padding: [16, 16] });
}

function switchToFullEditor(exerciseID) {
    const exercise = exerciseCache[exerciseID];
    if (!exercise) return;
    document.getElementById('exercise-' + exerciseID).innerHTML = generateExerciseHTML(exercise, exercise._count, true);
    injectBackButton(exerciseID);
}

// Re-render the read-only summary from the (cache-synced) exercise.
function switchToSummary(exerciseID) {
    const exercise = exerciseCache[exerciseID];
    if (!exercise) return;
    document.getElementById('exercise-' + exerciseID).innerHTML = generateExerciseHTML(exercise, exercise._count, false);
}

// Every active session has a summary view, so the editor always offers a way back.
// (Deleted sessions have no summary, so skip the button there.)
function injectBackButton(exerciseID) {
    const exercise = exerciseCache[exerciseID];
    if (exercise && exercise.is_on === false) return;
    const subWrapper = document.getElementById('exercise-sub-' + exerciseID);
    if (!subWrapper || subWrapper.querySelector('.back-to-summary')) return;
    const backBtn = document.createElement('button');
    backBtn.textContent = '← Back to summary';
    backBtn.className = 'btn back-to-summary';
    backBtn.onclick = function() { switchToSummary(exerciseID); };
    subWrapper.insertBefore(backBtn, subWrapper.firstChild);
}

// ── Workout editor (HEVY/Strong-style builder) ───────────────────────────────
// Renders entirely from the cached exercise (the single source of truth). Field
// changes / adds / deletes persist via the existing REST endpoints and reconcile
// the response back into exerciseCache, so the summary and editor never drift.

function renderWorkoutEditor(exercise, count) {
    var timeOfDay = "";
    if (exercise.time) {
        const date = new Date(exercise.time);
        timeOfDay = date.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false });
    }
    const durationStr = exercise.duration ? secondsToDurationString(exercise.duration) : "";
    const onChange = `updateExercise('${exercise.id}', true, ${count}, '${exercise.time}', true)`;

    // Strava source links + combine/divide (parity with the summary card).
    var stravaLinks = "", combineHTML = "", divideHTML = "";
    if (exercise.strava_id && exercise.strava_id.length > 0) {
        stravaLinks = `<div class="we-strava-links">` + exercise.strava_id.map(id =>
            `<a class="we-strava-link" href="https://www.strava.com/activities/${id}" target="_blank">Strava activity ${id}<img src="/assets/external-link.svg" alt=""></a>`
        ).join("") + `</div>`;
        const ids = exercise.strava_id.join(";");
        combineHTML = `<input class="clickable stravaCombineCheck wv-combine" type="checkbox" id="${ids}" title="Select to combine with another session">`;
        if (exercise.strava_id.length > 1) {
            divideHTML = `<img src="/assets/scissors.svg" class="btn_logo clickable wv-action-icon" title="Split combined activities" onclick="divideStravaExercises('${exercise.id}')">`;
        }
        try { document.getElementById('stravaCombineButtonWrapper').style.display = "flex"; } catch(e) {}
    }

    return `
        <div class="workout-view">
            <div class="we-session" id="exercise-sub-${exercise.id}">
                <div class="we-session-head">
                    <span class="we-eyebrow">Editing session</span>
                    <div class="we-session-actions">
                        ${combineHTML}
                        ${divideHTML}
                        <img src="/assets/trash-2.svg" class="btn_logo clickable wv-action-icon" title="Delete session" onclick="updateExercise('${exercise.id}', false, ${count}, '${exercise.time}', true)">
                    </div>
                </div>
                <div class="we-session-fields">
                    <label class="we-field">
                        <span class="we-field-label">Time</span>
                        <input class="we-input" type="time" id="exercise-timeofday-input-${exercise.id}" value="${timeOfDay}" onchange="${onChange}">
                    </label>
                    <label class="we-field">
                        <span class="we-field-label">Duration</span>
                        <input class="we-input" type="text" pattern="[0-9:]{0,}" id="exercise-time-input-${exercise.id}" placeholder="hh:mm:ss" value="${durationStr}" onchange="${onChange}">
                    </label>
                    ${generateGearFieldHTML(exercise)}
                </div>
                <textarea class="we-note" id="exercise-note-${exercise.id}" rows="2" placeholder="Session note" onchange="${onChange}">${escapeHTML(exercise.note || '')}</textarea>
                ${stravaLinks}
                <div class="we-exercises" id="operationsWrapper-${exercise.id}">
                    ${generateOperationsHTML(exercise.operations, exercise.id)}
                </div>
            </div>
        </div>
    `;
}

function generateOperationsHTML(operations, exerciseID) {
    var html = `<div class="we-exercise-list" id="operationsWrapper-sub-${exerciseID}">`;
    operations.forEach(operation => {
        html += `<div class="we-exercise" id="operation-${operation.id}">${generateOperationHTML(operation, exerciseID)}</div>`;
    });
    html += `</div>`;
    html += generateOperationAddButtonHTML(operations, exerciseID);
    return html;
}

function generateOperationAddButtonHTML(operations, exerciseID) {
    return `
        <button class="we-add-exercise clickable" onclick="addOperation('${exerciseID}')">
            <img src="/assets/plus.svg" alt=""><span>Add exercise</span>
        </button>
    `;
}

function generateOperationHTML(operation) {
    const liftingSel = operation.type == 'lifting' ? 'selected' : '';
    const movingSel  = operation.type == 'moving'  ? 'selected' : '';
    const timingSel  = operation.type == 'timing'  ? 'selected' : '';
    const actionName = operation.action ? operation.action.name : "";

    const equip = operation.equipment || "";
    const equipOptions = [
        ["", "Equipment"], ["barbells", "Barbells"], ["dumbbells", "Dumbbells"],
        ["bands", "Bands"], ["rope", "Rope"], ["bench", "Bench"],
        ["treadmill", "Treadmill"], ["machine", "Machine"]
    ];
    const equipHTML = equipOptions.map(([v, label]) =>
        `<option value="${v}" ${equip === v ? 'selected' : ''}>${label}</option>`
    ).join("");

    // Gear is edited per operation, but only makes sense for moving activities
    // (shoes / bike), mirroring Strava. Non-moving cards omit the selector so
    // updateOperation never touches their gear.
    var gearRowHTML = "";
    if (operation.type === 'moving') {
        gearRowHTML = `
            <div class="we-gear-row">
                <select class="we-input we-op-gear-select" id="operation-gear-${operation.id}" onchange="updateOperation('${operation.id}')">
                    ${gearOptionsHTML(currentOperationGearID(operation))}
                </select>
                <img src="/assets/sliders.svg" class="btn_logo clickable wv-action-icon" title="Manage gear" onclick="openGearModal()">
            </div>
        `;
    }

    return `
        <div class="we-exercise-head">
            <select class="we-type-select" id="operation-type-text-${operation.id}" title="Type" onchange="updateOperation('${operation.id}')">
                <option value="lifting" ${liftingSel}>💪</option>
                <option value="moving" ${movingSel}>🏃</option>
                <option value="timing" ${timingSel}>⏱️</option>
            </select>
            <div class="we-action" id="operation-action-${operation.id}">
                <input class="we-input we-action-input" type="text" autocomplete="off" id="operation-action-text-${operation.id}" name="operation-action-text" placeholder="Exercise name" value="${escapeHTML(actionName)}" onkeyup="filterFunction('${operation.id}')" onfocus="showSelectDropdown('${operation.id}', true)">
                <div id="operation-action-text-list-${operation.id}" class="dropdown-actions-wrapper" style="display: none;">
                    ${processExerciseList(operation.id)}
                </div>
            </div>
            <img src="/assets/plus.svg" class="btn_logo clickable wv-action-icon" id="addActionWrapper-${operation.id}" title="Create new exercise type" onclick="addAction('${operation.id}')">
            <img src="/assets/trash-2.svg" class="btn_logo clickable wv-action-icon" title="Remove exercise" onclick="deleteOperation('${operation.id}')">
        </div>

        <select class="we-equipment-select" id="operation-equipment-text-${operation.id}" name="operation-equipment-text" onchange="updateOperation('${operation.id}')">
            ${equipHTML}
        </select>

        ${gearRowHTML}

        ${generateTagSelectorHTML(operation)}

        <textarea class="we-note we-op-note" id="operation-description-${operation.id}" name="operation-description" rows="2" placeholder="Description" onchange="updateOperation('${operation.id}')">${operation.description ? escapeHTML(operation.description) : ''}</textarea>

        <input type="hidden" id="operation-distance-unit-${operation.id}" value="${operation.distance_unit}">
        <input type="hidden" id="operation-weight-unit-${operation.id}" value="${operation.weight_unit}">

        ${generateOperationSetsHTML(operation.operation_sets, operation)}
    `;
}

function generateOperationSetsHTML(operationSets, operation) {
    // Columns are declared once; which show is driven purely by the type modifier
    // class (we-type-*) in CSS, so the header and rows can never desync. Every set
    // keeps all inputs in the DOM (hidden ones included) so updateOperationSet can
    // read them regardless of type.
    var rows = "";
    var setCounter = 1;
    operationSets.forEach(operationSet => {
        rows += `<div class="we-set-row" id="operation-set-${operationSet.id}">${generateOperationSetHTML(operationSet, operation, setCounter)}</div>`;
        setCounter += 1;
    });

    return `
        <div class="we-sets we-type-${operation.type}">
            <div class="we-set-head">
                <span class="we-col we-col-set">Set</span>
                <span class="we-col we-col-weight">${operation.weight_unit}</span>
                <span class="we-col we-col-reps">Reps</span>
                <span class="we-col we-col-time">Time</span>
                <span class="we-col we-col-dist">${operation.distance_unit}</span>
                <span class="we-col we-col-avg">${operation.distance_unit}/h</span>
            </div>
            <div class="we-set-rows" id="operation-set-wrapper-sub-${operation.id}">
                ${rows}
            </div>
        </div>
        <button class="we-add-set clickable" id="addOperationSetWrapper-${operation.id}" onclick="addOperationSet('${operation.id}')">
            <img src="/assets/plus.svg" alt=""><span>Add set</span>
        </button>
    `;
}

function generateOperationSetHTML(operationSet, operation, setCounter) {
    const reps = (operationSet.repetitions != null) ? operationSet.repetitions : "";
    const weight = (operationSet.weight != null) ? operationSet.weight : "";
    const time = operationSet.moving_time ? secondsToDurationString(operationSet.moving_time) : "";
    const distance = (operationSet.distance != null) ? operationSet.distance : "";
    var average = "–";
    if (operationSet.distance != null && operationSet.time != null && operationSet.time > 0) {
        average = parseFloat(operationSet.distance / (operationSet.time / 3600)).toFixed(2);
    }
    const onChange = `updateOperationSet('${operationSet.id}', '${setCounter}')`;

    return `
        <span class="we-col we-col-set we-set-num clickable" id="operation-set-counter-${operationSet.id}" title="Delete set" onclick="deleteOperationSet('${operationSet.id}')">${setCounter}</span>
        <span class="we-col we-col-weight"><input class="we-set-input" type="number" min="0" inputmode="decimal" id="operation-set-weight-input-${operationSet.id}" placeholder="–" value="${weight}" onchange="${onChange}"></span>
        <span class="we-col we-col-reps"><input class="we-set-input" type="number" min="0" inputmode="numeric" id="operation-set-rep-input-${operationSet.id}" placeholder="–" value="${reps}" onchange="${onChange}"></span>
        <span class="we-col we-col-time"><input class="we-set-input" type="text" pattern="[0-9:]{0,}" id="operation-set-time-input-${operationSet.id}" placeholder="0:00" value="${time}" onchange="${onChange}"></span>
        <span class="we-col we-col-dist"><input class="we-set-input" type="number" min="0" inputmode="decimal" id="operation-set-distance-input-${operationSet.id}" placeholder="–" value="${distance}" onchange="${onChange}"></span>
        <span class="we-col we-col-avg we-set-avg">${average}</span>
    `;
}

function updateExerciseDay(exerciseDayID) {
    var exerciseDayNote = document.getElementById("exercise-day-note").value;

    var form_obj = {
        "note": exerciseDayNote
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                document.getElementById('exercise-day-note').innerHTML = result.exercise_day.note;
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercise-days/" + exerciseDayID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function getOperation(operationID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                console.log(result)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/operations/" + operationID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function getOperationSets(operationID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                console.log(result)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/operation-sets?operation_id=" + encodeURI(operationID));
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function addOperation(exerciseID) {
    var form_obj = {
        "exercise_id": exerciseID,
        "type": "lifting",
        "weight_unit": "kg",
        "distance_unit": "km"
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                placeNewOperation(result.operation, exerciseID)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/operations");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeNewOperation(operation) {
    const exercise = exerciseCache[operation.exercise];
    if (exercise) {
        if (!exercise.operations) exercise.operations = [];
        exercise.operations.push(operation);
    }
    rerenderExerciseEditor(operation.exercise);
}

// Re-render an exercise's editor from the cache (single source of truth) and
// re-attach the back-to-summary button.
function rerenderExerciseEditor(exerciseID) {
    const exercise = exerciseCache[exerciseID];
    if (!exercise) return;
    document.getElementById('exercise-' + exerciseID).innerHTML = generateExerciseHTML(exercise, exercise._count, true);
    injectBackButton(exerciseID);
}

// Remove an operation from whichever cached exercise holds it; returns that id.
function removeOperationFromCache(operationID) {
    for (const exID in exerciseCache) {
        const ex = exerciseCache[exID];
        if (ex && ex.operations) {
            const idx = ex.operations.findIndex(o => o.id === operationID);
            if (idx !== -1) {
                ex.operations.splice(idx, 1);
                return exID;
            }
        }
    }
    return null;
}

function addOperationSet(operationID) {
    var form_obj = {
        "operation_id": operationID,
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                placeNewOperationSet(result.operation_set, operationID, result.operation)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/operation-sets");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeNewOperationSet(operationSet, operationID, operation) {
    // The response operation carries its full set list; reconcile + re-render the
    // single operation card (keeps the cache and the new set row in sync).
    placeOperation(operation);
}

function updateOperation(operationID) {
    console.log('Updating operation...')
    toggleActionBorder(operationID, 'none');

    var type = document.getElementById('operation-type-text-' + operationID).value
    var action = document.getElementById('operation-action-text-' + operationID).value
    var weight_unit = document.getElementById('operation-weight-unit-' + operationID).value
    var distance_unit = document.getElementById('operation-distance-unit-' + operationID).value
    var equipment = document.getElementById('operation-equipment-text-' + operationID).value

    var tagEls = document.querySelectorAll('#tag-selector-' + operationID + ' .tag-chip-selected');
    var tags = Array.from(tagEls).map(e => e.getAttribute('data-tag'));

    var descriptionEl = document.getElementById('operation-description-' + operationID);
    var description = descriptionEl ? descriptionEl.value : "";

    var form_obj = {
        "type": type,
        "action": action,
        "weight_unit": weight_unit,
        "distance_unit": distance_unit,
        "equipment": equipment,
        "tags": tags,
        "description": description,
    };

    // Only moving ops render a gear selector; when present, send its value so
    // an empty selection clears gear and any other selection assigns it.
    var gearEl = document.getElementById('operation-gear-' + operationID);
    if (gearEl) {
        form_obj["gear_id"] = gearEl.value;
    }

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                console.log(result.error);
                toggleActionBorder(operationID, 'salmon');
            } else {
                placeOperation(result.operation)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/operations/" + operationID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeOperation(operation) {
    operationHTML = generateOperationHTML(operation)
    document.getElementById('operation-' + operation.id).innerHTML = operationHTML
    syncOperationToCache(operation)
}

// Keep exerciseCache in step with edits so switching back to the summary view
// (which re-renders from the cache) reflects updated tags/description/etc.
function syncOperationToCache(operation) {
    const exercise = exerciseCache[operation.exercise];
    if (!exercise || !exercise.operations) return;
    const idx = exercise.operations.findIndex(o => o.id === operation.id);
    if (idx !== -1) {
        exercise.operations[idx] = operation;
    }
}

function updateOperationSet(operationSetID, setCount) {
    var repetitions = parseFloat(document.getElementById('operation-set-rep-input-' + operationSetID).value)
    var weight = parseFloat(document.getElementById('operation-set-weight-input-' + operationSetID).value)
    var time = document.getElementById('operation-set-time-input-' + operationSetID).value
    var distance = parseFloat(document.getElementById('operation-set-distance-input-' + operationSetID).value)

    var timeFinal = parseDurationStringToSeconds(time);
    
    var form_obj = {
        "repetitions": repetitions,
        "weight": weight,
        "moving_time": timeFinal,
        "distance": distance,
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                placeOperationSet(result.operation_set, result.operation, setCount)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/operation-sets/" + operationSetID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeOperationSet(operationSet, operation, setCount) {
    operationSetHTML = generateOperationSetHTML(operationSet, operation, setCount)
    document.getElementById('operation-set-' + operationSet.id).innerHTML = operationSetHTML
    if (operation) {
        syncOperationToCache(operation)
    }
}

function updateExercise(exerciseID, on, count, originalTimeString, fromEditor = false) {
    if(!on && !confirm("Are you sure you want to delete this session?")) {
        return;
    }

    // The editor inputs only exist in edit mode. When called from the summary
    // (e.g. the delete button), fall back to the cached exercise so we don't read
    // null elements or accidentally blank the note/time/duration.
    const cached = exerciseCache[exerciseID] || {};
    const noteEl = document.getElementById('exercise-note-' + exerciseID);
    const timeEl = document.getElementById('exercise-time-input-' + exerciseID);
    const timeOfDayEl = document.getElementById('exercise-timeofday-input-' + exerciseID);

    var note = noteEl ? noteEl.value : (cached.note || "")
    var time = timeEl ? timeEl.value : (cached.duration ? secondsToDurationString(cached.duration) : "")
    var newIso = ""

    try {
        if (timeOfDayEl) {
            const localDate = new Date(originalTimeString);
            const [hours, minutes] = timeOfDayEl.value.split(':').map(Number);
            localDate.setHours(hours, minutes, 0, 0);
            newIso = toLocalISOString(localDate);
        } else {
            // No editor inputs — keep the existing time as-is.
            newIso = originalTimeString ? toLocalISOString(new Date(originalTimeString)) : "";
        }
    } catch(e) {
        console.log("failed to parse original time ISO. error: " + e)
        return
    }

    var form_obj = {
        "note": note,
        "is_on": on,
        "duration": parseDurationStringToSeconds(time),
        "time": newIso
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                if (fromEditor) {
                    placeExercise(result.exercise, count, true);
                } else {
                    placeExercise(result.exercise, count);
                }
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/exercises/" + exerciseID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeExercise(exercise, count, forceFullEditor = false) {
    // Keep the cache authoritative (the update endpoint returns the full exercise
    // tree) so switching back to the summary reflects session-level edits too.
    if (exercise && exercise.id) {
        exercise._count = exerciseCache[exercise.id] ? exerciseCache[exercise.id]._count : count;
        exerciseCache[exercise.id] = exercise;
    }
    document.getElementById('exercise-' + exercise.id).innerHTML = generateExerciseHTML(exercise, count, forceFullEditor)
    if (forceFullEditor) {
        injectBackButton(exercise.id);
    }
}

function loadExerciseList() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                exerciseList = result.actions
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/actions");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function processExerciseList(operationID) {
    exerciseListHTML = "";
    actions = exerciseList;

    actions.forEach(function(action){
        if(action.name && action.name != "") {
            exerciseListHTML += `
                <a value="" onclick="selectActionForOperation('${operationID}', '${action.name}')" class="dropdown-action clickable">
                    ${action.name}
                </a>
            `
        }
        if(action.norwegian_name && action.norwegian_name != "" && action.norwegian_name != action.name) {
            exerciseListHTML += `
                <a value="" onclick="selectActionForOperation('${operationID}', '${action.norwegian_name}')" class="dropdown-action clickable">
                    ${action.norwegian_name}
                </a>
            `
        }
    });

    return exerciseListHTML
}

function addExercise(exerciseDayID) {
    var form_obj = {
        "exercise_day_id": exerciseDayID,
        "on" : true,
        "note": "",
        "duration": null
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                placeNewExercise(result.exercise)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeNewExercise(exercise) {
    element = document.getElementById('exercisesWrapper')
    count = element.children.length
    // Cache it so the summary⇄editor toggle works without a page refresh, and open
    // a brand-new (empty) session straight in the editor — there's nothing to
    // summarise yet, and it's where the user needs to be to add exercises.
    exercise._count = count + 1;
    exerciseCache[exercise.id] = exercise;
    var exerciseHTML = `
        <div class="exerciseWrapper" id="exercise-${exercise.id}">
            ${generateExerciseHTML(exercise, count + 1, true)}
        </div>
    `;
    element.insertAdjacentHTML("beforeend", exerciseHTML)
    injectBackButton(exercise.id);
}

function deleteExercise() {

}

function deleteOperation(operationID) {
    if(!confirm("Are you sure you want to delete this exercise?")) {
        return;
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                const exID = removeOperationFromCache(operationID);
                if (exID) {
                    rerenderExerciseEditor(exID);
                } else {
                    document.getElementById('operation-' + operationID).remove();
                }
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/operations/" + operationID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function deleteOperationSet(operationSetID) {
    if(!confirm("Are you sure you want to delete this set?")) {
        return;
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                removeOperationSet(result.operation)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/operation-sets/" + operationSetID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function removeOperationSet(operation) {
    // Reconcile the cache and re-render the operation card (set numbers re-flow).
    placeOperation(operation);
}

function selectActionForOperation(operationID, actionName) {
    console.log('Selected: ' + actionName)
    document.getElementById('operation-action-text-' + operationID).value = actionName 
    showSelectDropdown(operationID, false)
    updateOperation(operationID)
}

function toggleActionBorder(operationID, color) {
    element = document.getElementById('operation-action-text-' + operationID)
    element.setAttribute('style', `border:0.2em solid ${color} !important`);
}

function addAction(operationID) {
    closeAllLists();

    const body = `
        <div id="add-action-wrapper-${operationID}">
            <div class="trm-field">
                <span class="trm-label">English name</span>
                <input class="trm-input" type="text" id="new-action-name-english-input-${operationID}" name="new-action-name-english-input" placeholder="Running" value="">
            </div>
            <p class="trm-or">or / and</p>
            <div class="trm-field">
                <span class="trm-label">Norwegian name</span>
                <input class="trm-input" type="text" id="new-action-name-norwegian-input-${operationID}" name="new-action-name-norwegian-input" placeholder="Løping" value="">
            </div>

            <div class="trm-field">
                <span class="trm-label">Sets, distance or time-based?</span>
                <select class="trm-select" id="new-action-type-input-${operationID}" name="new-action-type-input">
                    <option value="lifting">💪 Sets &amp; reps</option>
                    <option value="moving">🏃 Distance</option>
                    <option value="timing">⏱️ Time</option>
                </select>
            </div>

            <hr class="trm-divider">
            <p class="trm-section-label">Optional</p>

            <div class="trm-field">
                <span class="trm-label">Description</span>
                <textarea class="trm-textarea" id="new-action-description-input-${operationID}" name="new-action-description-input" rows="3" placeholder="Fast paced moving which can be..."></textarea>
            </div>
            <div class="trm-field">
                <span class="trm-label">Body part / category</span>
                <input class="trm-input" type="text" id="new-action-bodypart-input-${operationID}" name="new-action-bodypart-input" placeholder="Cardio" value="">
            </div>

            <button type="submit" class="btn btn--primary" onclick="createAction('${operationID}');"><img src="/assets/done.svg">Add and use</button>
        </div>
    `;

    TRModal.open({ eyebrow: "New exercise type", title: "Add exercise", body: body });
}

// ── Gear ──────────────────────────────────────────────────────────────────────
// Gear is selected once per session (the selector writes the chosen gear to all
// of the session's operations) and managed in a modal. See docs/gear.md.

function loadGearList() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                return;
            }
            if (!result.error) {
                gearList = result.gear || [];
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/gear");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// The gear assigned to a session is the gear shared by its operations. When the
// operations disagree (a combined session with mixed gear) nothing is shown; for
// a session with no gear yet, the user's primary gear is suggested as the default.
function currentSessionGearID(exercise) {
    const ops = exercise.operations || [];
    if (ops.length > 0) {
        const ids = ops.map(o => (o.gear ? o.gear.id : ""));
        const first = ids[0];
        if (ids.every(id => id === first)) {
            return first;
        }
        return "";
    }
    const primary = gearList.find(g => g.is_primary && !g.retired);
    return primary ? primary.id : "";
}

// Gear for a single operation: its stored gear, else (for a moving op with none)
// suggest the user's primary — matching the session default, unpersisted until changed.
function currentOperationGearID(operation) {
    if (operation.gear) return operation.gear.id;
    const primary = gearList.find(g => g.is_primary && !g.retired);
    return primary ? primary.id : "";
}

function gearOptionLabel(gear) {
    const nick = gear.nickname ? " (" + gear.nickname + ")" : "";
    const dist = gear.distance ? " · " + gear.distance.toFixed(1) + " km" : "";
    const retired = gear.retired ? " · retired" : "";
    return gear.name + nick + dist + retired;
}

function gearOptionsHTML(currentID) {
    let html = `<option value="">No gear</option>`;
    gearList.forEach(gear => {
        // Hide retired gear unless it is the one currently selected.
        if (gear.retired && gear.id !== currentID) {
            return;
        }
        const sel = gear.id === currentID ? "selected" : "";
        html += `<option value="${gear.id}" ${sel}>${escapeHTML(gearOptionLabel(gear))}</option>`;
    });
    return html;
}

// Gear is edited per operation (see generateOperationHTML). This session-level
// control is only a convenience to set every moving activity at once, so it is
// shown only when a session actually mixes 2+ moving activities.
function generateGearFieldHTML(exercise) {
    const movingOps = (exercise.operations || []).filter(o => o && o.type === 'moving');
    if (movingOps.length < 2) {
        return "";
    }
    const currentID = currentSessionGearID(exercise);
    return `
        <label class="we-field">
            <span class="we-field-label">Set gear for all</span>
            <div class="we-gear-row">
                <select class="we-input we-gear-select" id="exercise-gear-${exercise.id}" onchange="setExerciseGear('${exercise.id}', this.value)">
                    ${gearOptionsHTML(currentID)}
                </select>
                <img src="/assets/sliders.svg" class="btn_logo clickable wv-action-icon" title="Manage gear" onclick="openGearModal()">
            </div>
        </label>
    `;
}

// Rebuild every on-page gear selector from the current gearList, preserving each
// session's selection — used after the manage-gear modal changes the list.
function refreshGearSelectors() {
    document.querySelectorAll('.we-gear-select, .we-op-gear-select').forEach(sel => {
        const current = sel.value;
        sel.innerHTML = gearOptionsHTML(current);
        sel.value = current;
    });
}

function setExerciseGear(exerciseID, gearID) {
    var form_obj = { "gear_id": gearID ? gearID : null };
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                error("Could not reach API.");
                return;
            }
            if (result.error) {
                error(result.error);
            } else {
                success("Gear updated.");
                // Keep the cache in sync so the summary/editor reflect the change,
                // then re-render the editor so the per-operation selectors follow.
                const exercise = exerciseCache[exerciseID];
                if (exercise && exercise.operations) {
                    const gear = gearList.find(g => g.id === gearID) || null;
                    exercise.operations.forEach(o => { o.gear = gear; });
                    rerenderExerciseEditor(exerciseID);
                }
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/exercises/" + exerciseID + "/gear");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(JSON.stringify(form_obj));
}

function openGearModal() {
    closeAllLists();
    // While the modal is open, any gear change re-renders its body and refreshes the on-page selectors.
    onGearChanged = function() { renderGearModalBody(); refreshGearSelectors(); };
    TRModal.open({ eyebrow: "Equipment", title: "Manage gear", body: `<div class="gear-empty">Loading gear…</div>` });
    getGear();
}

// Renders the shared gear list + add form into the modal body (same `.gear-*` markup as the /gear
// page, via gear-shared.js), plus a link out to the full page.
function renderGearModalBody() {
    TRModal.setBody(`
        <div class="gear-list">${gearListHTML()}</div>
        ${gearAddFormHTML()}
        <p class="gear-empty u-mt-sm"><a href="/gear">Open the full gear page →</a></p>
    `);
}

// createGear / updateGearField / deleteGear now live in web/js/gear-shared.js (shared with /gear).

function createAction(operationID) {
    var name = document.getElementById('new-action-name-english-input-' + operationID).value
    var norwegian_name = document.getElementById('new-action-name-norwegian-input-' + operationID).value
    var description = document.getElementById('new-action-description-input-' + operationID).value
    var type = document.getElementById('new-action-type-input-' + operationID).value
    var body_part = document.getElementById('new-action-bodypart-input-' + operationID).value

    var form_obj = {
        "name": name,
        "norwegian_name" : norwegian_name,
        "description": description,
        "type": type,
        "body_part": body_part
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                alert("Could not reach API.");
                return;
            }
            
            if(result.error) {
                alert(result.error);
            } else {
                selectActionForOperation(operationID, result.action.name)
                closeModal();
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/actions");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

window.addEventListener('click', function(e){
    // A click inside an exercise-name field or its dropdown should leave the list
    // open; anything else closes (and persists) any open action dropdowns.
    if (e.target.closest('.we-action, .operationAction, .dropdown-actions-wrapper')) {
        return;
    }
    closeAllLists();
});

function closeAllLists() {
    lists = document.getElementsByClassName('dropdown-actions-wrapper');
    for(var i = 0; i < lists.length; i++) {
        if(lists[i].style.display == 'flex') {
            operationID = lists[i].id.replace('operation-action-text-list-', '')
            lists[i].style.display = 'none';
            updateOperation(operationID)
        }
    }
}

function combineStravaExercises() {
    checkButtons = document.getElementsByClassName('stravaCombineCheck')

    var stravaIDArray = []

    for(var i = 0; i < checkButtons.length; i++) {
        if(checkButtons[i].checked) {
            stravaIDArray.push(checkButtons[i].id)
        }
    }

    if(stravaIDArray.length < 2) {
        alert("Choose two or more exercises to combine.");
        return;
    }

    if(!confirm("Are you sure you want to combine these Strava exercises?")) {
        return;
    }

    combineStravaExercisesAPI(stravaIDArray);
}

function combineStravaExercisesAPI(stravaIDArray) {
    var form_data = JSON.stringify(stravaIDArray);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                alert(result.message)
                location.reload(); 
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises/strava-combine");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function divideStravaExercises(exerciseID) {
    if(!confirm("Are you sure you want to divide all these Strava exercises?")) {
        return;
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                alert(result.message)
                location.reload(); 
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises/" + exerciseID + "/strava-divide");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function stravaSyncOperationSet(operationSetID) {
    if(!confirm("Are you sure you want to update this with data from Strava?")) {
        return
    }

    var form_obj = {};
    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                location.reload(); 
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/operation-sets/" + operationSetID + "/strava-sync");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}