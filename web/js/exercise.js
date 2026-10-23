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

    document.getElementById('content').innerHTML = dayPageHTML(exerciseDayID);
    document.getElementById('card-header').innerHTML = 'Write every detail.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        loadExerciseList()
        loadGearList()
        getExerciseDay(exerciseDayID);
        getExerciseDayWeek(exerciseDayID);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

// ── Day page shell ───────────────────────────────────────────────────────────
// Everything around the workouts: the day header (date, week strip, per-season results,
// day note, gear), the add row, Combine and the related links. The header reads the day
// from GET /exercise-days/:id and the week from GET /exercise-days/:id/week; see
// docs/exercises.md, "Day header".

var exerciseDayWeek = null;

function dayPageHTML(exerciseDayID) {
    return `
        <div class="day-page">
            <section class="day-head" aria-labelledby="day-title">
                <div class="day-head-top">
                    <div class="day-title-block">
                        <h1 class="day-title" id="day-title">…</h1>
                        <p class="day-date" id="day-date"></p>
                    </div>
                    <div class="day-week-count" id="day-week-count"></div>
                </div>
                <ol class="day-week" id="day-week" aria-label="Days of the week"></ol>
                <div class="day-seasons" id="day-seasons"></div>
                <p class="day-settled" id="day-settled" hidden>This week is settled. You can still edit workouts, but not add or delete them, or change what counts.</p>
                <div class="field day-note-field">
                    <div class="day-note-label-row">
                        <label for="exercise-day-note" class="field-label">Day note</label>
                        <span class="day-note-status" id="day-note-status" role="status"></span>
                    </div>
                    <textarea id="exercise-day-note" rows="3" oninput="clearDayNoteStatus()" onchange="updateExerciseDay('${exerciseDayID}')"></textarea>
                </div>
                <div class="day-gear" id="day-gear"></div>
            </section>

            <div class="day-workouts" id="exercisesWrapper"></div>
            <p class="day-empty" id="day-empty" hidden>Nothing logged on this day yet.</p>
            <div class="day-add" id="day-add"></div>

            <div class="btn-group day-combine" id="stravaCombineButtonWrapper" hidden>
                <button type="button" class="btn btn--ghost" id="day-combine-button" onclick="combineStravaExercises()">Combine workouts</button>
            </div>

            <nav class="day-links" id="day-links" aria-label="Related pages">
                <a href="/exercises">All exercises</a>
                <a href="/gear">Gear</a>
            </nav>
        </div>
    `;
}

function getExerciseDayWeek(exerciseDayID) {
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
            } else {
                exerciseDayWeek = result.week;
                placeDayWeek();
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/exercise-days/" + exerciseDayID + "/week");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// sameCalendarDay compares two dates by local calendar day.
function sameCalendarDay(a, b) {
    return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate();
}

// isFutureDay reports whether a date falls after today, by local calendar day.
function isFutureDay(date) {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    const day = new Date(date);
    day.setHours(0, 0, 0, 0);
    return day.getTime() > today.getTime();
}

function workoutCountLabel(count) {
    return count === 1 ? "workout" : "workouts";
}

// placeDayWeek renders everything that depends on the week: the count, the strip, the
// per-season rows, the settled note and the add row. Called on load and after any change
// that can move a count (add, delete, restore, counting toggle).
function placeDayWeek() {
    const week = exerciseDayWeek;
    if (!week) return;

    const thisWeek = week.days.some(day => sameCalendarDay(new Date(day.date), new Date()));
    document.getElementById('day-week-count').innerHTML = `
        <span class="day-week-count-value">${week.workouts}</span>
        <span class="day-week-count-label">${workoutCountLabel(week.workouts)} ${thisWeek ? "this" : "that"} week</span>
    `;

    document.getElementById('day-week').innerHTML = week.days.map(weekDayHTML).join("");
    document.getElementById('day-seasons').innerHTML = week.seasons.map(daySeasonHTML).join("");

    const viewedDay = week.days.find(day => day.id === exerciseDayID);
    const future = viewedDay ? isFutureDay(viewedDay.date) : false;
    document.getElementById('day-settled').hidden = week.loggable || future;

    const links = document.getElementById('day-links');
    if (thisWeek && !links.querySelector('.day-links-front')) {
        links.insertAdjacentHTML('beforeend', `<a class="day-links-front" href="/">This week on the front page</a>`);
    }

    placeDayAdd(viewedDay, future);
}

// weekDayHTML renders one tile of the week strip. A day with a row links straight to it;
// a loggable day without one goes through the get-or-create lookup, like the front page's
// edit button; anything else (a future day, or a past day nobody logged) can't be opened.
function weekDayHTML(day, index) {
    const date = new Date(day.date);
    const weekdayName = date.toLocaleDateString('en-GB', { weekday: 'short' });
    const fullName = GetDateString(date, true);
    const viewing = day.id && day.id === exerciseDayID;
    const today = sameCalendarDay(date, new Date());

    var classes = "day-week-tile";
    if (viewing) classes += " is-viewing";
    if (today) classes += " is-today";
    if (day.workouts > 0) classes += " has-workouts";

    const inner = `
        <span class="day-week-name">${weekdayName}</span>
        <span class="day-week-figure">${day.workouts}</span>
    `;
    const label = `${fullName}, ${day.workouts} ${workoutCountLabel(day.workouts)}${today ? ", today" : ""}`;

    var tile;
    if (viewing) {
        tile = `<span class="${classes}" aria-current="date" aria-label="${label}">${inner}</span>`;
    } else if (day.id) {
        tile = `<a class="${classes}" href="/exercises/${day.id}" aria-label="${label}">${inner}</a>`;
    } else if (day.loggable) {
        // The API numbers weekdays as Go's time.Weekday (Sunday=0); the strip starts on Monday.
        tile = `<button type="button" class="${classes}" onclick="openWeekDay(${(index + 1) % 7})" aria-label="${label}">${inner}</button>`;
    } else {
        const reason = isFutureDay(day.date) ? "You can't log workouts on future days" : "Nothing logged";
        tile = `<span class="${classes} is-unavailable" title="${reason}" aria-label="${label}. ${reason}">${inner}</span>`;
    }

    return `<li>${tile}</li>`;
}

// openWeekDay opens a weekday of the current week that has no day row yet; the lookup
// creates it (see APIGetExerciseDayInWeek).
function openWeekDay(weekDay) {
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
            } else {
                window.location = '/exercises/' + result.exercise.id;
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/exercise-days/week?weekDay=" + weekDay);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

// daySeasonHTML renders the week's result in one season, from the leaderboard's numbers.
// A sick-leave week shows the hero's resting wording instead of a bar.
function daySeasonHTML(season) {
    const name = `<span class="day-season-name">${escapeHTML(season.name)}</span>`;
    if (season.sick_leave) {
        return `<div class="day-season is-resting">${name}<span class="day-season-rest">Sick leave, streak kept</span></div>`;
    }

    const pct = Math.min(100, Math.round(season.completion * 100));
    const complete = season.completion >= 1;
    return `
        <div class="day-season${complete ? " is-complete" : ""}">
            ${name}
            <span class="day-season-bar" role="progressbar" aria-valuemin="0" aria-valuemax="${season.goal}" aria-valuenow="${Math.min(season.workouts, season.goal)}" aria-label="${escapeHTML(season.name)}, weekly goal">
                <span class="day-season-fill" style="width: ${pct}%"></span>
            </span>
            <span class="day-season-figure">${season.workouts} / ${season.goal}${complete ? `<img src="/assets/check.svg" class="day-season-check" alt="Weekly goal met">` : ""}</span>
        </div>
    `;
}

// placeDayAdd renders the add row, or the reason there is none. The rules mirror
// APICreateExercise: current week only, not in the future, and at most workout_limit
// counted workouts per day.
function placeDayAdd(viewedDay, future) {
    const add = document.getElementById('day-add');
    if (!viewedDay) {
        add.innerHTML = "";
        return;
    }

    if (future) {
        add.innerHTML = `<p class="day-add-note">You can log workouts on this day once it arrives.</p>`;
    } else if (!viewedDay.loggable) {
        add.innerHTML = "";
    } else if (viewedDay.workouts >= exerciseDayWeek.workout_limit) {
        add.innerHTML = `<p class="day-add-note">A day can hold ${exerciseDayWeek.workout_limit} workouts that count. Mark one as not counting to add another.</p>`;
    } else {
        add.innerHTML = `
            <div class="workout-view">
                <button type="button" class="we-add-exercise" onclick="addExercise('${exerciseDayID}')">
                    <img src="/assets/plus.svg" alt="">Add workout
                </button>
            </div>
        `;
    }
}

// placeDayGear lists the gear on this day's workouts (from the cached day) beside the
// always-available Manage gear button.
function placeDayGear() {
    const names = [];
    Object.values(exerciseCache).forEach(exercise => {
        if (!exercise.is_on) return;
        (exercise.operations || []).forEach(operation => {
            if (!operation || !operation.gear) return;
            // Prefer the live gear list, so a rename in the Manage gear modal shows at once.
            const gear = gearList.find(g => g.id === operation.gear.id) || operation.gear;
            const nick = gear.nickname ? " (" + gear.nickname + ")" : "";
            const name = gear.name + nick;
            if (!names.includes(name)) names.push(name);
        });
    });

    const summary = names.length > 0
        ? `<span class="day-gear-label">Gear used</span> <span class="day-gear-names">${escapeHTML(names.join(", "))}</span>`
        : `<span class="day-gear-label">No gear on this day</span>`;

    document.getElementById('day-gear').innerHTML = `
        <img src="/assets/sliders.svg" class="day-gear-icon" alt="">
        <span class="day-gear-text">${summary}</span>
        <button type="button" class="btn btn--ghost btn--sm" onclick="openGearModal()">Manage gear</button>
    `;
}

// placeDayEmpty shows the empty state when the day holds no visible workouts. A deleted
// workout with no activities renders nothing, so an empty wrapper doesn't count.
function placeDayEmpty() {
    const workouts = Array.from(document.getElementById('exercisesWrapper').children);
    document.getElementById('day-empty').hidden = workouts.some(el => el.innerHTML.trim() !== "");
}

// placeCombineButton shows Combine once two or more Strava workouts are ticked, and says
// how many it will merge.
function placeCombineButton() {
    const checked = document.querySelectorAll('.stravaCombineCheck:checked').length;
    const wrapper = document.getElementById('stravaCombineButtonWrapper');
    if (!wrapper) return;
    wrapper.hidden = checked < 2;
    document.getElementById('day-combine-button').textContent = `Combine ${checked} workouts`;
}

document.addEventListener('change', function(event) {
    if (event.target.classList && event.target.classList.contains('stravaCombineCheck')) {
        placeCombineButton();
    }
});

function clearDayNoteStatus() {
    document.getElementById('day-note-status').textContent = "";
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
            info("Loading day...");
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
    const date = new Date(Date.parse(exerciseDay.date));
    document.getElementById('day-title').textContent = date.toLocaleDateString('en-GB', { weekday: 'long' });
    document.getElementById('day-date').textContent = date.toLocaleDateString('en-GB', { day: 'numeric', month: 'long', year: 'numeric' });
    document.getElementById('exercise-day-note').value = HTMLDecode(exerciseDay.note || '');

    placeExercises(exerciseDay.exercises);
}

var exerciseCache = {};

function placeExercises(exercises) {
    exercisesHTML = "";
    counter = 1;
    exerciseCache = {};

    exercises.forEach(exercise => {
        exercise._count = counter;
        exerciseCache[exercise.id] = exercise;

        var exerciseGenerated = generateExerciseHTML(exercise, counter)
        var exerciseHTML = `
            <div class="day-workout" id="exercise-${exercise.id}">
                ${exerciseGenerated}
            </div>
        `;

        if(exerciseGenerated) {
            exercisesHTML += exerciseHTML;
            counter += 1;
        }
    });

    document.getElementById('exercisesWrapper').innerHTML = exercisesHTML;
    placeDayEmpty();
    placeDayGear();
    placeCombineButton();
}

// startOfWeek returns local midnight on the Monday of the given date's week. Used to decide
// whether a session still sits in the current (editable) week.
function startOfWeek(date) {
    const d = new Date(date);
    d.setHours(0, 0, 0, 0);
    const monday = (d.getDay() + 6) % 7; // Mon=0 … Sun=6
    d.setDate(d.getDate() - monday);
    return d;
}

// isExerciseWeekEditable mirrors the backend freeze: a session can only be deleted or have
// its goal-counting flag changed while its week is still the current week. Sessions with no
// time (date-only) are treated as editable — the backend has the final say either way.
function isExerciseWeekEditable(exercise) {
    if (!exercise || !exercise.time) return true;
    return startOfWeek(new Date(exercise.time)).getTime() === startOfWeek(new Date()).getTime();
}

// sessionDeleteButton renders the trash action, frozen (non-clickable, dimmed) once the
// session's week has ended so the UI matches the backend's "can't remove after the week"
// rule. Shared by the summary card and the editor so they can't drift.
function sessionDeleteButton(exercise, count) {
    if (!isExerciseWeekEditable(exercise)) {
        return `<img src="/assets/trash-2.svg" class="btn_logo wv-action-icon is-frozen" title="Can't delete a workout after its week has ended">`;
    }
    return `<img src="/assets/trash-2.svg" class="btn_logo clickable wv-action-icon" title="Delete workout" onclick="updateExercise('${exercise.id}', false, ${count}, '${exercise.time}', true)">`;
}

// sessionRestoreButton renders the "Restore" action for a deleted (is_on=false) session,
// frozen (disabled, dimmed) once its week has ended so it matches the backend's "can't
// restore after the week" rule.
function sessionRestoreButton(exercise, count) {
    if (!isExerciseWeekEditable(exercise)) {
        return `<button type="button" disabled title="Can't restore a workout after its week has ended" class="btn btn--sm is-frozen"><img src="/assets/refresh-cw.svg" class="color-invert">Restore</button>`;
    }
    return `<button type="submit" onclick="updateExercise('${exercise.id}', true, ${count}, '${exercise.time}', true)" id="restore-exercise-button-${exercise.id}" class="btn btn--sm"><img src="/assets/refresh-cw.svg" class="color-invert">Restore</button>`;
}

// generateExerciseHTML returns the session's markup, or "" for a switched-off session with
// no activities — there is nothing to restore, so it isn't shown at all.
function generateExerciseHTML(exercise, count, forceFullEditor = false) {
    var exerciseHTML = "";

    if(exercise.is_on) {

        if(!forceFullEditor) {
            return renderWorkoutSummary(exercise, count);
        }

        exerciseHTML = renderWorkoutEditor(exercise, count);
    } else if (exercise.operations.length > 0){
        const activityCount = exercise.operations.length;
        exerciseHTML = `
            <div class="day-deleted" id="exercise-sub-${exercise.id}">
                <span class="day-deleted-text">Deleted workout, ${activityCount} ${activityCount === 1 ? "activity" : "activities"}</span>
                ${sessionRestoreButton(exercise, count)}
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
    // The server's headline names the session (the source's own title, else one generated from
    // the activity categories); the joined activity titles are only a fallback.
    const summaryLine = (exercise.headline && exercise.headline.title)
        || operations.map(activityTitle).join("  +  ") || "Empty workout";
    const subCards = operations.map(op => renderActivitySubCard(op, exercise)).join("");
    const noCountBadge = exercise.counts_toward_goal ? ""
        : `<span class="wv-nocount-badge" title="This workout is logged but doesn't count toward your weekly goal">Doesn't count</span>`;
    const privateBadge = exercise.private
        ? `<span class="wv-nocount-badge" title="Only you can see this workout. It still counts toward your weekly goal">Hidden</span>`
        : "";

    return `
        <div class="workout-view">
            <div class="wv-session" id="exercise-sub-${exercise.id}">
                <div class="wv-session-head">
                    <div class="wv-session-meta">
                        <span class="wv-session-date">${dateLine}</span>
                        <span class="wv-session-summary">${escapeHTML(summaryLine)}${noCountBadge}${privateBadge}</span>
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
        combineHTML = `<input class="clickable stravaCombineCheck wv-combine" type="checkbox" id="${ids}" title="Select to combine with another workout">`;
        if (exercise.strava_id.length > 1) {
            divideHTML = `<img src="/assets/scissors.svg" class="btn_logo clickable wv-action-icon" title="Split into separate workouts" onclick="divideStravaExercises('${exercise.id}')">`;
        }
    }
    return `
        <div class="wv-actions">
            ${combineHTML}
            ${divideHTML}
            <img src="/assets/edit.svg" class="btn_logo clickable wv-action-icon" title="Edit workout" onclick="switchToFullEditor('${exercise.id}')">
            ${sessionDeleteButton(exercise, count)}
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
        // An item already playing when the session began has its start clamped to the
        // session start, so several can share 00:00. Mark it rather than let the
        // timeline claim they all started together.
        const carried = !!it.started_before;
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

        return { it, stamp, stat, spoken, carried };
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
            <span class="wv-track-time${t.carried ? ' wv-track-time--carried' : ''}"${t.carried ? ' title="Already playing when the workout started"' : ''}>${escapeHTML(t.stamp)}</span>
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
    // Pauses only earn a sub-line when they are long enough to matter.
    const movingDetail = movingSecs && totalSecs - movingSecs >= 30 ? secondsToDurationString(movingSecs) + " moving" : "";

    // The server attaches the processed stream summary (segments, route, HR zones and
    // header stats) whenever the activity has Strava streams. Prefer it over re-deriving
    // stats in JS; fall back to the raw stream only for older/uncomputed data.
    const summary = operation.stream_summary || null;
    const streams = streamSet ? streamSet.strava_streams : null;
    const setForStreams = streamSet || sets[0] || {};

    const gearHTML = gearLineHTML(operation);

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
                ${statTileHTML(durationHTML, "", "Duration", movingDetail)}
                ${statTileHTML(distanceHTML, "", "Distance")}
                ${speedTileHTML(operation, summary, hasDistance ? distance : 0, movingSecs || totalSecs)}
                ${cardioStatTiles(summary, streams)}
            </div>
            ${renderStreamSectionsHTML(operation, summary, setForStreams, true)}
        </div>
    `;
}

// renderStreamSectionsHTML renders the sensor sections below an activity's stat tiles —
// route and splits, elevation, the heart-rate/cadence/power trace, HR zones and effort
// analysis — and schedules the chart/map draws. Shared by the cardio and time cards; each
// section appears only when the server summary (already scoped to the activity type, so a
// padel session carries no splits or pace analysis) or the raw stream carries it.
// movingTimeline plots the trace over moving samples only, which suits distance sports;
// court and gym sports pass false and plot the recorded clock, since standing still is part
// of the session rather than a pause.
function renderStreamSectionsHTML(operation, summary, set, movingTimeline) {
    const streams = set.strava_streams || null;
    const latlngData = streams && (streams.latlng || streams.lat_lng);
    const hasRoute = latlngData && latlngData.data && latlngData.data.length > 0;

    const setKey = set.id || operation.id;
    const traceID = `trace-${setKey}`;
    const elevCanvasID = `elev-chart-${setKey}`;
    const mapDivID = `route-map-${setKey}`;
    const splitsID = `splits-${setKey}`;

    const elevation = summary && summary.elevation;
    const elevProfile = (summary && summary.elevation_profile) || null;
    const hasElevChart = elevProfile && elevProfile.length > 1;
    const elevHTML = hasElevChart ? `${climbCaptionHTML(elevation)}<div class="wv-hr-wrapper"><canvas id="${elevCanvasID}" class="wv-elev-chart"></canvas></div>` : "";
    const splitsHTML = renderSplitsHTML(summary, operation, splitsID, hasRoute);
    const routeHTML = renderRouteHTML(hasRoute ? mapDivID : null, summary, splitsHTML);
    const trace = buildTrace(traceID, summary, streams, set, movingTimeline);

    const segments = (summary && summary.segments) || [];
    if (trace || hasRoute || hasElevChart) {
        scheduleCardioStreamRender(trace ? traceID : null, mapDivID, hasRoute, latlngData, segments, splitsID, hasElevChart ? elevCanvasID : null, elevProfile);
    }

    return `
        ${routeHTML}
        ${elevHTML}
        ${trace ? trace.html : ""}
        ${renderHRZonesHTML(summary)}
        ${renderAnalysisHTML(summary)}
    `;
}

// gearLineHTML surfaces an activity's gear in the read view so it reads without editing
// (cardio and time cards — the activities that carry gear).
function gearLineHTML(operation) {
    if (!operation.gear) return "";
    const nick = operation.gear.nickname ? " (" + operation.gear.nickname + ")" : "";
    return `<div class="wv-gear"><img src="/assets/sliders.svg" class="wv-gear-icon" alt="">${escapeHTML(operation.gear.name + nick)}</div>`;
}

// statTileHTML renders one headline tile: the value, its label, and an optional quiet
// detail line underneath (a max, a descent, the moving time) for the number behind the
// headline. The label may carry an ⓘ.
function statTileHTML(value, unit, label, detail) {
    const unitHTML = unit ? `<span class="wv-stat-unit">${unit}</span>` : "";
    const detailHTML = detail ? `<span class="wv-stat-detail">${detail}</span>` : "";
    return `<div class="wv-stat"><span class="wv-stat-value">${value}${unitHTML}</span><span class="wv-stat-label">${label}</span>${detailHTML}</div>`;
}

// cardioStatTile renders one metric tile, or nothing when the value is absent.
function cardioStatTile(value, unit, label, detail) {
    if (value == null || value === "" || (typeof value === "number" && !isFinite(value))) return "";
    return statTileHTML(value, unit, label, detail);
}

// cardioStatTiles renders the sensor tiles after duration/distance/speed. Each appears only
// when its channel was recorded, and carries its peak (or the related total) as a detail.
function cardioStatTiles(summary, streams) {
    const s = summary || {};
    var html = "";

    const descent = s.elevation && s.elevation.loss_m > 0 ? Math.round(s.elevation.loss_m) + " m down" : "";
    html += cardioStatTile(elevationGainMetres(summary, streams), "m", "Elevation", descent);

    const hrMax = s.heartrate_bpm ? "max " + Math.round(s.heartrate_bpm.max) : "";
    html += cardioStatTile(avgHeartrate(summary, streams), "bpm", "Avg HR", hrMax);

    if (s.cadence_rpm) {
        html += cardioStatTile(Math.round(s.cadence_rpm.avg), s.cadence_unit || "rpm", "Avg cadence", "max " + Math.round(s.cadence_rpm.max));
    }
    if (s.power) {
        html += cardioStatTile(Math.round(s.power.avg_w), "W", "Avg power", "max " + Math.round(s.power.max_w) + " W");
        if (s.power.normalized_w != null) {
            const work = s.power.work_kj > 0 ? Math.round(s.power.work_kj) + " kJ work" : "";
            html += cardioStatTile(Math.round(s.power.normalized_w), "W", "Norm. power" + infoMarkHTML("power"), work);
        }
    }
    if (s.temperature_c) html += cardioStatTile(Math.round(s.temperature_c.avg), "°C", "Temp");
    return html;
}

// speedTileHTML is the third headline tile: average pace for foot sports (the number
// runners and walkers think in), average speed with its peak for everything else.
function speedTileHTML(operation, summary, distance, speedSecs) {
    const unit = operation.distance_unit || "km";
    if (isFootActivity(operation)) {
        const pace = distance > 0 && speedSecs ? formatPace(speedSecs / distance) : "—";
        return statTileHTML(pace, distance > 0 && speedSecs ? "/" + paceUnitLabel(unit) : "", "Avg pace");
    }

    if (!(distance > 0) || !speedSecs) return statTileHTML("—", "", "Avg speed");
    const speed = parseFloat(distance / (speedSecs / 3600)).toFixed(1);
    var peak = "";
    if (summary && summary.speed && summary.speed.max_kmh > 0) {
        // The summary is always km/h; follow the activity's own unit.
        const max = isMileUnitJS(unit) ? summary.speed.max_kmh / 1.609344 : summary.speed.max_kmh;
        peak = "max " + max.toFixed(1);
    }
    return statTileHTML(speed, unit + "/h", "Avg speed", peak);
}

// Strava sports whose activities are measured on foot: they read as pace, and their cadence
// is steps per minute (see controllers/cadence.go).
const WV_FOOT_SPORTS = ["Run", "TrailRun", "VirtualRun", "Walk", "Hike"];

// isFootActivity keys off the action's Strava sport, falling back to the title for
// activities without one.
function isFootActivity(operation) {
    const sport = operation.action && operation.action.strava_name;
    if (sport) return WV_FOOT_SPORTS.indexOf(sport) !== -1;
    return /\brun|jog|walk|hike|hiking/.test(activityTitle(operation).toLowerCase());
}

// formatPace renders seconds per distance unit as m:ss.
function formatPace(secsPerUnit) {
    var m = Math.floor(secsPerUnit / 60);
    var sec = Math.round(secsPerUnit % 60);
    if (sec === 60) { m += 1; sec = 0; }
    return m + ":" + (sec < 10 ? "0" + sec : "" + sec);
}

function paceUnitLabel(unit) {
    return isMileUnitJS(unit) ? "mi" : (unit || "km");
}

// renderRouteHTML pairs the map with the splits so tracing a split on the route never needs
// a scroll in between: side by side when the card is wide, the map pinned above the list
// when it is narrow (see .wv-route in workout.css). Either half renders alone when the
// other is missing.
function renderRouteHTML(mapDivID, summary, splitsHTML) {
    if (!mapDivID) return splitsHTML;
    const mapHTML = `<div class="wv-route-map"><div id="${mapDivID}" class="wv-map"></div>${routeCaptionHTML(summary)}</div>`;
    if (!splitsHTML) return `<div class="wv-route">${mapHTML}</div>`;
    return `<div class="wv-route wv-route--paired"><div class="wv-route-layout">${mapHTML}${splitsHTML}</div></div>`;
}

// ── Over-time chart ─────────────────────────────────────────────────────────
// One chart for the time-based channels — heart rate, cadence, power — with a segmented
// switch between them, instead of a stacked chart per channel. Only recorded channels get a
// tab; a single channel gets a plain title. The readout above the chart carries the
// numbers the line can't (average, peak, low), and a dashed line marks the average.

// wvTraces holds each rendered trace's channels and Chart.js instance, keyed by trace id,
// so the delegated tab handler can redraw without re-deriving the data.
var wvTraces = {};

// buildTrace prepares the over-time section for one activity, or returns null when no
// time-based channel was recorded. The chart itself is drawn once the canvas exists.
// movingTimeline is passed through to traceTimeline.
function buildTrace(traceID, summary, streams, set, movingTimeline) {
    if (!streams) return null;
    const timeline = traceTimeline(streams, set, movingTimeline);
    const s = summary || {};
    const candidates = [
        {
            key: "hr", label: "Heart rate", unit: "bpm", colorToken: "--error",
            stream: streams.heartrate, factor: 1, skipZero: true, smooth: 1,
            stats: s.heartrate_bpm ? [["Avg", s.heartrate_bpm.avg], ["Max", s.heartrate_bpm.max], ["Min", s.heartrate_bpm.min]] : null
        },
        {
            // The raw stream is per leg for foot sports; the summary says which unit it shows.
            key: "cadence", label: "Cadence", unit: s.cadence_unit || "rpm", colorToken: "--mediumblue",
            stream: streams.cadence, factor: s.cadence_unit === "spm" ? 2 : 1, skipZero: true, smooth: 1,
            stats: s.cadence_rpm ? [["Avg", s.cadence_rpm.avg], ["Max", s.cadence_rpm.max]] : null
        },
        {
            // Raw watts swing hard second to second, so the line is a 10-sample rolling mean.
            key: "power", label: "Power", unit: "W", colorToken: "--mediumblue",
            stream: streams.watts, factor: 1, skipZero: false, smooth: 10,
            stats: s.power ? [["Avg", s.power.avg_w], ["Max", s.power.max_w]]
                .concat(s.power.normalized_w != null ? [["Normalised", s.power.normalized_w]] : []) : null
        }
    ];

    const channels = candidates
        .map(c => Object.assign(c, { points: tracePoints(c, timeline) }))
        .filter(c => c.points.length > 1);
    if (channels.length === 0) return null;

    wvTraces[traceID] = { channels: channels, maxSecs: timeline.maxSecs, chart: null };

    const head = channels.length > 1
        ? `<div class="user-stat-tabs wv-trace-tabs" role="group" aria-label="Chart">${channels.map((c, i) =>
            `<button type="button" class="user-stat-tab${i === 0 ? " user-stat-tab-active" : ""}" data-trace-channel="${c.key}" aria-pressed="${i === 0}">${c.label}</button>`
          ).join("")}</div>`
        : `<span class="wv-trace-title">${channels[0].label}</span>`;

    return {
        html: `
            <div class="wv-trace" id="${traceID}">
                <div class="wv-trace-head">${head}</div>
                <div class="wv-trace-readout">${traceReadoutHTML(channels[0])}</div>
                <div class="wv-hr-wrapper"><canvas id="${traceID}-canvas" class="wv-hr-chart"></canvas></div>
            </div>`
    };
}

// traceTimeline maps stream samples onto the chart's x-axis. For distance sports
// (movingTimeline) streams are distance-sampled, not time-sampled, so it keeps only the
// moving samples (velocity > 0.5 m/s) and spaces them evenly across moving_time so the axis
// matches the activity duration. Otherwise — or without a usable velocity channel — it
// plots the recorded clock.
function traceTimeline(streams, set, movingTimeline) {
    const n = streams.time && streams.time.data ? streams.time.data.length : 0;
    const velocity = streams.velocity_smooth && streams.velocity_smooth.data;
    const movingTimeSecs = set.moving_time || set.time;
    if (movingTimeline && velocity && velocity.length === n && movingTimeSecs) {
        const indices = [];
        for (let i = 0; i < n; i++) {
            if (velocity[i] > 0.5) indices.push(i);
        }
        const last = Math.max(indices.length - 1, 1);
        return { indices: indices, times: indices.map((idx, j) => Math.round(j / last * movingTimeSecs)), maxSecs: movingTimeSecs };
    }
    const indices = [];
    for (let i = 0; i < n; i++) indices.push(i);
    return { indices: indices, times: n ? streams.time.data.slice() : [], maxSecs: movingTimeSecs || (n ? streams.time.data[n - 1] : 0) };
}

// traceGapSeconds is the longest stretch without a reading the trace line still bridges.
// Past it the line breaks (a null point; Chart.js leaves gaps with spanGaps off), so a padel
// player standing between points shows as missing cadence, not a ramp across two minutes.
// Strava's smart recording spaces samples up to ~20 s apart, which stays bridged.
const traceGapSeconds = 30;

// tracePoints turns one channel into chart points along the timeline: scaled into the
// presented unit, optionally smoothed, and without the zero readings heart rate and cadence
// record while paused. A gap longer than traceGapSeconds between kept readings becomes a
// null point, so the line breaks there instead of interpolating.
function tracePoints(channel, timeline) {
    const data = channel.stream && channel.stream.data;
    if (!data || data.length === 0) return [];
    const values = timeline.indices.map(i => (data[i] || 0) * channel.factor);
    const smoothed = channel.smooth > 1 ? rollingMean(values, channel.smooth) : values;
    const points = [];
    var lastX = null;
    for (let j = 0; j < smoothed.length; j++) {
        if (channel.skipZero && !(values[j] > 0)) continue;
        const x = timeline.times[j];
        if (lastX != null && x - lastX > traceGapSeconds) {
            points.push({ x: Math.round((lastX + x) / 2), y: null });
        }
        points.push({ x: x, y: Math.round(smoothed[j]) });
        lastX = x;
    }
    return points.some(p => p.y > 0) ? points : [];
}

// rollingMean averages each value with the window - 1 before it.
function rollingMean(values, window) {
    const out = new Array(values.length);
    var sum = 0;
    for (let i = 0; i < values.length; i++) {
        sum += values[i];
        if (i >= window) sum -= values[i - window];
        out[i] = sum / Math.min(i + 1, window);
    }
    return out;
}

// traceReadoutHTML lists a channel's summary numbers in its unit; the first is the average.
function traceReadoutHTML(channel) {
    if (!channel.stats) return "";
    return channel.stats.map(([label, value]) =>
        `<span class="wv-trace-stat">${label} <span class="wv-trace-stat-value">${Math.round(value)}<span class="wv-stat-unit">${channel.unit}</span></span></span>`
    ).join("");
}

// drawTrace (re)draws a trace's chart for one channel.
function drawTrace(traceID, key) {
    const trace = wvTraces[traceID];
    if (!trace) return;
    const channel = trace.channels.find(c => c.key === key) || trace.channels[0];
    if (trace.chart) { trace.chart.destroy(); trace.chart = null; }
    const average = channel.stats ? channel.stats[0][1] : null;
    trace.chart = renderTraceChart(`${traceID}-canvas`, channel, trace.maxSecs, average);
}

// wireTraceTabs binds one delegated handler for every chart switch on the page, like the ⓘ
// marks — the read view re-renders on edit, so per-element listeners would leak.
var wvTraceTabsWired = false;
function wireTraceTabs() {
    if (wvTraceTabsWired) return;
    wvTraceTabsWired = true;
    document.addEventListener("click", function(e) {
        const tab = e.target.closest && e.target.closest("[data-trace-channel]");
        if (!tab) return;
        const section = tab.closest(".wv-trace");
        const trace = section && wvTraces[section.id];
        if (!trace) return;
        const key = tab.getAttribute("data-trace-channel");
        section.querySelectorAll("[data-trace-channel]").forEach(t => {
            const active = t === tab;
            t.classList.toggle("user-stat-tab-active", active);
            t.setAttribute("aria-pressed", active ? "true" : "false");
        });
        const channel = trace.channels.find(c => c.key === key);
        section.querySelector(".wv-trace-readout").innerHTML = channel ? traceReadoutHTML(channel) : "";
        drawTrace(section.id, key);
    });
}

// ── Explainers ──────────────────────────────────────────────────────────────
// The read view is dense with terms that assume training knowledge ("aerobic
// decoupling", "negative split", heart-rate reserve). Rather than crowd the layout with
// permanent captions, each term carries a quiet ⓘ that opens the shared modal — one
// registry here, one delegated handler, reusable anywhere a term needs unpacking.
const WV_INFO_TOPICS = {
    "decoupling": {
        title: "Aerobic decoupling",
        body: `How much your heart rate drifted upward through the activity — comparing the
               second half against the first at the same pace.
               <br><br>
               A low number means your body held the effort comfortably. Under 5% is a
               well-controlled aerobic effort. Above 8% usually means fatigue, heat, or
               starting faster than you could sustain. A negative number means the second
               half was the more efficient one.`
    },
    "pace-consistency": {
        title: "Pace consistency",
        body: `How much your pace varied from one full split to the next: the big number is
               the typical spread in seconds per kilometre, and the line beneath it is the
               same spread as a share of your average pace.
               <br><br>
               The share is the fairer guide, because 15 seconds is a bigger swing on a fast
               run than on a slow one. For a steady effort on flat ground, under 3% is very
               even, 3–6% is moderate, and over 6% is uneven.
               <br><br>
               Intervals, hills, traffic and stops all push it up by design. Judge those
               against your other sessions of the same kind, not against these bands.`
    },
    "power": {
        title: "Power",
        body: `Average power is the mean of every reading, including the zeros while you
               coast. Max is your single highest reading.
               <br><br>
               Normalised power weighs hard surges more heavily than the plain average. It
               estimates what a perfectly steady effort with the same cost to your body would
               have averaged. On a steady ride it sits close to the average. The further above
               the average it is, the more stop-start the ride was.
               <br><br>
               Work is the total energy you put into the pedals, in kilojoules. For cycling it
               is roughly the number of kilocalories you burned.`
    },
    "stops": {
        title: "Stops",
        body: `Stretches where you were stationary long enough that they were left out of
               your moving time — traffic lights, gates, or a rest.
               <br><br>
               The count is how many times you stopped; the time beside it is how long you
               spent stopped in total.`
    },
    "hr-zones": {
        title: "Heart-rate zones",
        body: `Five intensity bands, showing how long you spent in each. The bpm range
               beside each band is the heart rate it covers.
               <br><br>
               The bands are anchored on whatever is named next to the title: the maximum
               heart rate you configured, an age-based estimate, the highest rate seen
               across your activities, or the peak in this one activity.
               <br><br>
               You choose the zone system on your account page: the common five zones as a
               share of your maximum; the same five on heart-rate reserve (the Karvonen
               method, which measures effort across the range between rest and maximum and
               needs a resting heart rate); or Olympiatoppen's intensity scale, I-1 to I-5.`
    },
    "gradient": {
        title: "Effort by gradient",
        body: `Your average heart rate grouped by how steep the ground was, with each bar
               showing how much of the activity you spent on that kind of terrain.
               <br><br>
               It separates the work the route asked of you from the work your pace asked
               of you — a slow climb at a high heart rate is still hard.`
    },
    "splits": {
        title: "Splits",
        body: `Each completed unit of distance and the pace or speed you held across it.
               Bar length is relative to your fastest split, so the shape of the effort —
               a strong finish, a mid-run fade — reads at a glance.
               <br><br>
               A "negative split" means you covered the second half of the activity faster
               than the first, usually the mark of a well-judged effort.`
    }
};

// infoMarkHTML renders the ⓘ for a term. Sits inline after a label or section title.
function infoMarkHTML(key) {
    const topic = WV_INFO_TOPICS[key];
    if (!topic) return "";
    return `<button type="button" class="wv-info" data-wv-info="${key}" aria-label="What is ${escapeHTML(topic.title.toLowerCase())}?">i</button>`;
}

// wireInfoMarks binds one delegated handler for every ⓘ on the page. Registered once —
// the read view re-renders on edit, so per-element listeners would leak.
var wvInfoWired = false;
function wireInfoMarks() {
    if (wvInfoWired) return;
    wvInfoWired = true;
    document.addEventListener("click", function(e) {
        const mark = e.target.closest && e.target.closest("[data-wv-info]");
        if (!mark) return;
        const topic = WV_INFO_TOPICS[mark.getAttribute("data-wv-info")];
        if (!topic) return;
        TRModal.open({ title: topic.title, body: `<p class="wv-info-body">${topic.body}</p>` });
    });
}

// renderAnalysisHTML surfaces the server's second-order "coach" metrics — aerobic
// decoupling, pace consistency, stops, and heart rate by terrain gradient — as one calm
// "Effort analysis" block. Every piece is optional (each needs specific channels), so the
// section renders only what the stream actually supported and is skipped when empty.
function renderAnalysisHTML(summary) {
    const a = summary && summary.analysis;
    if (!a) return "";

    var tiles = "";
    if (a.decoupling_pct != null) {
        const verdict = decouplingVerdict(a.decoupling_pct);
        tiles += `
            <div class="wv-insight">
                <span class="wv-insight-value">${wvNum(a.decoupling_pct)}<span class="wv-stat-unit">%</span></span>
                <span class="wv-insight-label">Aerobic decoupling${infoMarkHTML("decoupling")}</span>
                <span class="wv-verdict wv-verdict-${verdict.tone}">${verdict.text}</span>
            </div>`;
    }
    if (a.pace_std_dev_seconds != null) {
        tiles += `
            <div class="wv-insight">
                <span class="wv-insight-value">±${Math.round(a.pace_std_dev_seconds)}<span class="wv-stat-unit">s/km</span></span>
                <span class="wv-insight-label">Pace consistency${infoMarkHTML("pace-consistency")}</span>
                ${a.pace_variation_pct != null ? `<span class="wv-insight-detail">${wvNum(a.pace_variation_pct)}% of your average pace</span>` : ""}
            </div>`;
    }
    if (a.breaks && a.breaks.count > 0) {
        const label = a.breaks.count === 1 ? "Stop" : "Stops";
        tiles += `
            <div class="wv-insight">
                <span class="wv-insight-value">${a.breaks.count}</span>
                <span class="wv-insight-label">${label} · ${secondsToDurationString(a.breaks.total_duration_seconds)}${infoMarkHTML("stops")}</span>
            </div>`;
    }
    const insightsHTML = tiles ? `<div class="wv-insights">${tiles}</div>` : "";
    const gradesHTML = renderGradientHTML(a.hr_by_gradient);

    if (!insightsHTML && !gradesHTML) return "";
    return `
        <div class="wv-analysis">
            <div class="wv-analysis-head"><span class="wv-analysis-title">Effort analysis</span></div>
            ${insightsHTML}
            ${gradesHTML}
        </div>`;
}

// decouplingVerdict maps aerobic decoupling to a plain read: low drift is a well-controlled
// aerobic effort (green), rising drift signals fatigue/cardiac drift (amber → red), and a
// negative value means the second half was actually more efficient.
function decouplingVerdict(pct) {
    if (pct < 0) return { tone: "good", text: "Even effort" };
    if (pct < 5) return { tone: "good", text: "Aerobic" };
    if (pct < 8) return { tone: "warn", text: "Some drift" };
    return { tone: "bad", text: "High drift" };
}

// renderGradientHTML shows how the terrain drove effort: one row per gradient band the
// activity spent time in, with a bar sized by that band's share of moving time and the
// average heart rate there. Tint runs cool (downhill) → blue (flat) → hot (climbs), so a
// hilly effort reads as rising work. Skipped unless there are at least two bands to contrast.
function renderGradientHTML(buckets) {
    if (!Array.isArray(buckets) || buckets.length < 2) return "";
    const total = buckets.reduce((sum, b) => sum + (b.seconds || 0), 0);
    if (!(total > 0)) return "";

    const rows = buckets.map(b => {
        const pct = Math.max(6, Math.round((b.seconds / total) * 100));
        const hr = b.avg_heartrate_bpm != null
            ? `<span class="wv-grade-hr">${b.avg_heartrate_bpm}<span class="wv-split-side-unit">bpm</span></span>`
            : `<span class="wv-grade-hr">—</span>`;
        return `
            <div class="wv-grade">
                <span class="wv-grade-label">${escapeHTML(b.label)}</span>
                <span class="wv-grade-track"><span class="wv-grade-bar wv-grade-${gradeToneClass(b.label)}" style="width:${pct}%"></span></span>
                <span class="wv-grade-time">${secondsToDurationString(b.seconds)}</span>
                ${hr}
            </div>`;
    }).join("");

    return `
        <div class="wv-grades">
            <div class="wv-grades-head"><span class="wv-grades-title">Effort by gradient${infoMarkHTML("gradient")}</span></div>
            <div class="wv-grade-list">${rows}</div>
        </div>`;
}

// gradeToneClass ties a gradient band to a bar tint on the same cool→hot ramp as the HR
// zones, so effort rises with the terrain.
function gradeToneClass(label) {
    switch (label) {
        case "steep descent": return "d2";
        case "descent": return "d1";
        case "flat": return "flat";
        case "climb": return "c1";
        case "steep climb": return "c2";
        default: return "flat";
    }
}

// climbCaptionHTML surfaces the activity's single biggest sustained ascent.
function climbCaptionHTML(elevation) {
    if (!elevation || !elevation.biggest_climb) return "";
    const c = elevation.biggest_climb;
    if (!(c.gain_m > 0)) return "";
    var line = `Biggest climb ${Math.round(c.gain_m)} m`;
    if (c.distance_km > 0) line += ` over ${wvNum(c.distance_km)} km`;
    if (c.grade_pct > 0) line += ` (${wvNum(c.grade_pct)}%)`;
    return `<div class="wv-route-caption">${line}</div>`;
}

// elevationGainMetres / avgHeartrate prefer the server summary and fall back to deriving
// from the raw stream (kept so activities predating the summary still show the metric).
function elevationGainMetres(summary, streams) {
    if (summary && summary.elevation && summary.elevation.gain_m > 0) return Math.round(summary.elevation.gain_m);
    if (streams && streams.altitude && streams.altitude.data && streams.altitude.data.length > 1) {
        const alt = streams.altitude.data;
        let gain = 0;
        for (let i = 1; i < alt.length; i++) { const d = alt[i] - alt[i - 1]; if (d > 0) gain += d; }
        return gain > 0 ? Math.round(gain) : null;
    }
    return null;
}

function avgHeartrate(summary, streams) {
    if (summary && summary.heartrate_bpm) return Math.round(summary.heartrate_bpm.avg);
    if (streams && streams.heartrate && streams.heartrate.data) {
        const vals = streams.heartrate.data.filter(v => v > 0);
        if (vals.length > 0) return Math.round(vals.reduce((a, b) => a + b, 0) / vals.length);
    }
    return null;
}

// routeCaptionHTML is a quiet one-line summary under the map: GPS distance + point count.
function routeCaptionHTML(summary) {
    if (!summary || !summary.route) return "";
    const r = summary.route;
    const bits = [];
    if (r.distance_km > 0) bits.push(wvNum(r.distance_km) + " km GPS");
    if (r.point_count) bits.push(r.point_count + " points");
    if (bits.length === 0) return "";
    return `<div class="wv-route-caption">${bits.join(" · ")}</div>`;
}

// isCyclingActivity decides whether splits read as speed (bikes) or pace (runs/walks).
function isCyclingActivity(operation) {
    const name = ((operation.action && operation.action.name) || operation.note || "").toLowerCase();
    return /cycl|bike|biking|ride|riding|spin/.test(name);
}

// renderSplitsHTML draws the per-distance splits as a scoreboard: each split's bar length
// is its speed relative to the fastest split, so the shape of the effort (a fast finish, a
// mid-run fade) reads at a glance. The fastest split is marked. from/to point indices are
// carried on each row; when the activity has a route the rows are buttons that trace their
// split on the map (see wireSplitHighlight).
function renderSplitsHTML(summary, operation, splitsID, interactive) {
    const segments = (summary && summary.segments) || [];
    if (segments.length < 2) return ""; // one split is just the whole activity — skip

    const cycling = isCyclingActivity(operation);
    // Relative bar width is keyed off speed (distance per second) so faster = longer.
    var maxRate = 0, fastestIdx = -1;
    segments.forEach((s, i) => {
        const rate = s.elapsed_seconds > 0 ? s.distance / s.elapsed_seconds : 0;
        if (rate > maxRate) { maxRate = rate; fastestIdx = i; }
    });

    const unit = operation.distance_unit || "km";
    const rows = segments.map((s, i) => {
        const rate = s.elapsed_seconds > 0 ? s.distance / s.elapsed_seconds : 0;
        const width = maxRate > 0 ? Math.max(12, Math.round(rate / maxRate * 100)) : 12;
        const primary = cycling ? splitSpeed(s, unit) : splitPace(s, unit);
        const hr = s.avg_heartrate_bpm != null ? `<span class="wv-split-side">${s.avg_heartrate_bpm}<span class="wv-split-side-unit">bpm</span></span>` : "";
        const climb = s.elevation_gain_m > 1 ? `<span class="wv-split-side">+${Math.round(s.elevation_gain_m)}<span class="wv-split-side-unit">m</span></span>` : "";
        // A short final split is annotated inside the bar track, not in the index cell —
        // the fixed-width index column would wrap it onto a second line and knock the
        // whole row out of alignment with the others.
        const partial = s.distance < 0.95 ? `<span class="wv-split-partial">${wvNum(s.distance)} ${unit}</span>` : "";
        const fast = i === fastestIdx ? " wv-split-fast" : "";
        const tag = interactive ? "button" : "div";
        const buttonAttrs = interactive ? ` type="button" aria-pressed="false"` : "";
        return `
            <${tag} class="wv-split${fast}"${buttonAttrs} data-from="${s.from_point}" data-to="${s.to_point}">
                <span class="wv-split-idx">${s.index}</span>
                <span class="wv-split-track"><span class="wv-split-bar" style="width:${width}%"></span><span class="wv-split-primary">${primary}</span>${partial}</span>
                ${hr}${climb}
            </${tag}>`;
    }).join("");

    const negSplit = isNegativeSplit(segments) ? `<span class="wv-splits-flag">Negative split</span>` : "";

    return `
        <div class="wv-splits" id="${splitsID}">
            <div class="wv-splits-head">
                <span class="wv-splits-title">Splits${infoMarkHTML("splits")}</span>
                <span class="wv-splits-meta">${negSplit}<span class="wv-splits-unit">per ${isMileUnitJS(unit) ? "mile" : (unit || "km")}</span></span>
            </div>
            <div class="wv-split-list">${rows}</div>
        </div>`;
}

// isNegativeSplit is true when the second half of the activity (by distance) was run at a
// faster pace than the first — the mark of a well-judged effort.
function isNegativeSplit(segments) {
    if (segments.length < 2) return false;
    var total = segments.reduce((sum, s) => sum + s.distance, 0);
    if (!(total > 0)) return false;
    var half = total / 2, acc = 0;
    var firstDist = 0, firstTime = 0, secondDist = 0, secondTime = 0;
    segments.forEach(s => {
        if (acc + s.distance <= half || firstDist === 0) {
            firstDist += s.distance; firstTime += s.elapsed_seconds;
        } else {
            secondDist += s.distance; secondTime += s.elapsed_seconds;
        }
        acc += s.distance;
    });
    if (!(firstDist > 0) || !(secondDist > 0)) return false;
    var firstPace = firstTime / firstDist, secondPace = secondTime / secondDist;
    return secondPace < firstPace * 0.99; // >1% faster second half
}

function isMileUnitJS(unit) {
    return /^(mi|mile|miles)$/i.test(unit || "");
}

// splitPace formats a split as minutes:seconds per unit (running/walking).
function splitPace(segment, unit) {
    if (!(segment.distance > 0) || !(segment.elapsed_seconds > 0)) return "—";
    const secsPerUnit = segment.elapsed_seconds / segment.distance;
    return `${formatPace(secsPerUnit)}<span class="wv-split-primary-unit">/${paceUnitLabel(unit)}</span>`;
}

// splitSpeed formats a split as distance-unit per hour (cycling).
function splitSpeed(segment, unit) {
    if (!(segment.distance > 0) || !(segment.elapsed_seconds > 0)) return "—";
    const speed = segment.distance / (segment.elapsed_seconds / 3600);
    return `${wvNum(parseFloat(speed.toFixed(1)))}<span class="wv-split-primary-unit">${unit || "km"}/h</span>`;
}

// renderHRZonesHTML draws the time-in-zone distribution: a single stacked bar (one segment
// per zone, sized by share) plus a legend. Colour rises from calm to hot to read as
// intensity, not decoration — the zone name and time carry the meaning in the legend.
function renderHRZonesHTML(summary) {
    const zones = (summary && summary.hr_zones) || [];
    if (zones.length === 0) return "";
    const anyTime = zones.some(z => z.seconds > 0);
    if (!anyTime) return "";

    const code = z => escapeHTML(z.code || ("Z" + z.zone));
    const bar = zones.map(z =>
        `<span class="wv-zone-seg wv-zone-${z.zone}" style="width:${z.percent}%" title="${code(z)} ${escapeHTML(z.name)}"></span>`
    ).join("");

    // Each zone carries its own bpm bounds (server-side, already Karvonen-adjusted when
    // the zones are anchored on heart-rate reserve) — show them, or the zone names alone
    // give no sense of what heart rate they actually stand for.
    const legend = zones.filter(z => z.seconds > 0).map(z =>
        `<span class="wv-zone-key"><i class="wv-zone-dot wv-zone-${z.zone}"></i>${code(z)} ${escapeHTML(z.name)}<span class="wv-zone-range">${hrZoneRange(z)}</span> · ${secondsToDurationString(z.seconds)} · ${wvNum(z.percent)}%</span>`
    ).join("");

    var basis = "";
    if (summary.hr_max_bpm) {
        // Where the max came from, then how the zone system uses it.
        var source;
        switch (summary.hr_max_basis) {
            case "max": source = "max HR"; break;
            case "observed_max": source = "your observed max"; break;
            case "age": source = "age-based max"; break;
            default: source = "this activity's peak";
        }
        var how;
        switch (summary.hr_zone_system) {
            case "reserve": how = `HR reserve · rest ${summary.hr_rest_bpm || "?"} bpm · ${source}`; break;
            case "olympiatoppen": how = `Olympiatoppen · % of ${source}`; break;
            default: how = `% of ${source}`;
        }
        basis = `<span class="wv-zones-basis">${how} ${summary.hr_max_bpm} bpm</span>`;
    }

    return `
        <div class="wv-zones">
            <div class="wv-zones-head"><span class="wv-zones-title">Heart-rate zones${infoMarkHTML("hr-zones")}</span>${basis}</div>
            <div class="wv-zone-bar">${bar}</div>
            <div class="wv-zone-legend">${legend}</div>
        </div>`;
}

// hrZoneRange renders a zone's heart-rate bounds. The top zone is open-ended (max_bpm 0),
// and the bottom zone starts at 0 unless the zones are anchored on heart-rate reserve.
function hrZoneRange(z) {
    return " " + formatZoneRange(z.min_bpm, z.max_bpm, "bpm");
}

// Display labels for the server's action vocabularies (models/action_category.go). Keys are
// the stored values; the empty key is the "not set" option of the create-action form.
const actionCategoryLabels = {
    strength: "💪 Strength", cardio: "🏃 Cardio", sport: "🎾 Sport", mobility: "🧘 Mobility", general: "🏅 General"
};
const actionEquipmentLabels = {
    "": "None / not sure", barbell: "Barbell", dumbbell: "Dumbbell", machine: "Machine", cable: "Cable",
    smith_machine: "Smith machine", kettlebell: "Kettlebell", band: "Band", suspension: "Suspension",
    bodyweight: "Bodyweight", trap_bar: "Trap bar", sandbag: "Sandbag"
};

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

    // The equipment belongs to the exercise (its action), so it leads the summary line.
    const equipment = operation.action && operation.action.equipment ? actionEquipmentLabels[operation.action.equipment] : null;
    var rollup = `${equipment ? escapeHTML(equipment) + " · " : ""}${sets.length} ${sets.length === 1 ? 'set' : 'sets'}`;
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

// renderTimeSubCard renders a time-based activity (padel, yoga, …). A Strava import with
// sensor data also gets the sensor tiles and sections; the server has already left the
// distance-derived views (speed, splits, pace analysis) out of its summary.
function renderTimeSubCard(operation, exercise) {
    const sets = operation.operation_sets || [];
    var secs = 0, stravaID = null, stravaSetID = null, streamSet = null;
    sets.forEach(s => {
        if (s.time) secs += s.time;
        else if (s.moving_time) secs += s.moving_time;
        if (!stravaID && s.strava_id) { stravaID = s.strava_id; stravaSetID = s.id; }
        if (!streamSet && s.strava_streams) streamSet = s;
    });
    if (!secs && operation.duration) secs = operation.duration;
    const durationHTML = secs ? secondsToDurationString(secs) : "—";

    const summary = operation.stream_summary || null;
    const sensorTilesHTML = streamSet ? cardioStatTiles(summary, streamSet.strava_streams) : "";
    const sectionsHTML = streamSet ? renderStreamSectionsHTML(operation, summary, streamSet, false) : "";

    return `
        <div class="wv-activity wv-activity-time">
            <div class="wv-activity-head">
                <div class="wv-activity-title">${activityActionIcon(operation)}<span>${escapeHTML(activityTitle(operation))}</span></div>
                <div class="wv-activity-head-actions">${stravaSyncButtonHTML(stravaSetID, stravaID)}${sourceBadge(operation, stravaID, exercise)}</div>
            </div>
            ${generateTagChipsHTML(operation.tags)}
            ${gearLineHTML(operation)}
            ${activityDescriptionHTML(operation)}
            <div class="wv-stats">
                ${statTileHTML(durationHTML, "", "Duration")}
                ${sensorTilesHTML}
            </div>
            ${sectionsHTML}
        </div>
    `;
}

// Deferred chart/map render: summary HTML is injected via innerHTML, so poll
// until the canvas/map nodes exist, then draw. (Ported from the old simple card.)
function scheduleCardioStreamRender(traceID, mapDivID, hasRoute, latlngData, segments, splitsID, elevCanvasID, elevProfile) {
    var attempts = 0;
    var interval = setInterval(function() {
        attempts++;
        var traceReady = !traceID || document.getElementById(traceID + "-canvas");
        var mapReady = !hasRoute || document.getElementById(mapDivID);
        var elevReady = !elevCanvasID || document.getElementById(elevCanvasID);
        if (traceReady && mapReady && elevReady) {
            clearInterval(interval);
            if (elevCanvasID && elevProfile) {
                renderElevationChart(elevCanvasID, elevProfile);
            }
            if (traceID) {
                drawTrace(traceID);
            }
            if (hasRoute) {
                const map = renderRouteMap(mapDivID, latlngData.data);
                if (map && segments && segments.length && splitsID) {
                    wireSplitHighlight(map, latlngData.data, splitsID);
                }
            }
        } else if (attempts > 50) {
            clearInterval(interval);
        }
    }, 50);
}

// wireSplitHighlight traces a split on the map. Hovering or focusing a row previews it;
// clicking (or tapping, where there is no hover) pins it until it is clicked again or
// another split is pinned, and leaving a row falls back to the pinned split. A pinned split
// outside the current map view is panned into it. Split from/to indices align with the raw
// latlng samples the map was drawn from.
function wireSplitHighlight(map, latlngData, splitsID) {
    if (typeof L === 'undefined') return;
    const container = document.getElementById(splitsID);
    if (!container) return;

    const rootStyle = getComputedStyle(document.documentElement);
    const hiColor = rootStyle.getPropertyValue("--mediumblue").trim() || "#3a6ea5";
    var highlight = null;
    var pinned = null;

    function draw(row) {
        if (highlight) { map.removeLayer(highlight); highlight = null; }
        if (!row) return null;
        const from = parseInt(row.getAttribute("data-from"), 10);
        const to = parseInt(row.getAttribute("data-to"), 10);
        if (isNaN(from) || isNaN(to) || to <= from) return null;
        const slice = latlngData.slice(from, to + 1).map(p => [p[0], p[1]]);
        if (slice.length < 2) return null;
        highlight = L.polyline(slice, { color: hiColor, weight: 7, opacity: 0.95 }).addTo(map);
        return highlight;
    }
    function restore() { draw(pinned); }
    function pin(row) {
        if (pinned) {
            pinned.classList.remove("wv-split-active");
            pinned.setAttribute("aria-pressed", "false");
        }
        pinned = pinned === row ? null : row;
        if (pinned) {
            pinned.classList.add("wv-split-active");
            pinned.setAttribute("aria-pressed", "true");
        }
        const line = draw(pinned);
        if (line && !map.getBounds().contains(line.getBounds())) {
            map.fitBounds(line.getBounds(), { padding: [24, 24] });
        }
    }

    container.querySelectorAll(".wv-split").forEach(row => {
        row.addEventListener("mouseenter", () => draw(row));
        row.addEventListener("mouseleave", restore);
        row.addEventListener("focus", () => draw(row));
        row.addEventListener("blur", restore);
        row.addEventListener("click", () => pin(row));
    });
}

// hexToRGBA fades a token colour for a chart fill. Chart.js needs a concrete colour
// value, so a token can't be used directly for a translucent area beneath a line.
function hexToRGBA(hex, alpha) {
    var h = (hex || "").replace("#", "");
    if (h.length === 3) h = h[0] + h[0] + h[1] + h[1] + h[2] + h[2];
    if (h.length !== 6) return hex;
    var n = parseInt(h, 16);
    return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`;
}

// renderTraceChart draws one over-time channel as a filled line, with a dashed line at its
// average when one is known. Returns the Chart.js instance so a tab switch can destroy it.
function renderTraceChart(canvasID, channel, maxSecs, average) {
    var canvas = document.getElementById(canvasID);
    if (!canvas || channel.points.length === 0) return null;

    // Read the theme tokens so the chart's axis text/grid match the light surface (was white-on-dark).
    var rootStyle = getComputedStyle(document.documentElement);
    var tickColor = rootStyle.getPropertyValue("--lightblue").trim() || "#7fa8cf";
    var gridColor = rootStyle.getPropertyValue("--grey").trim() || "#cecece";
    var lineColor = rootStyle.getPropertyValue(channel.colorToken).trim() || "#3a6ea5";
    var unit = channel.unit;

    var datasets = [{
        label: channel.label,
        data: channel.points,
        fill: true,
        borderColor: lineColor,
        pointBackgroundColor: lineColor,
        backgroundColor: hexToRGBA(lineColor, 0.2),
        tension: 0.3,
        pointRadius: 0,
        borderWidth: 2,
        spanGaps: false // tracePoints marks missing data with null points; keep those as breaks
    }];
    if (average != null) {
        // Span the whole axis, not just this channel's readings, so every tab's frame matches.
        var last = maxSecs || channel.points[channel.points.length - 1].x;
        datasets.push({
            label: "Average",
            data: [{ x: 0, y: average }, { x: last, y: average }],
            fill: false,
            borderColor: tickColor,
            borderDash: [5, 4],
            borderWidth: 1.5,
            pointRadius: 0,
            pointHitRadius: 0
        });
    }

    return new Chart(canvas, {
        type: "line",
        data: { datasets: datasets },
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
                        callback: function(value) { return value + " " + unit; }
                    }
                }]
            },
            tooltips: {
                filter: function(item) { return item.datasetIndex === 0; },
                callbacks: {
                    title: function(items) {
                        return secondsToDurationString(items[0].xLabel);
                    },
                    label: function(item) {
                        return item.yLabel + " " + unit;
                    }
                }
            }
        }
    });
}

// renderElevationChart draws the altitude-over-distance profile as a filled area chart,
// paired below the HR chart. Data is the server's down-sampled [{distance_km, altitude_m}]
// so no distance is re-derived here. Axis/line colours read from the theme tokens (like
// the HR chart) so it stays on the light surface.
function renderElevationChart(canvasID, profile) {
    var canvas = document.getElementById(canvasID);
    if (!canvas || !profile || profile.length < 2) return;

    var points = profile.map(function(p) { return { x: p.distance_km, y: p.altitude_m }; });

    // Without an explicit max, Chart.js rounds the axis up to the next nice tick, leaving
    // the profile stopping short of the right edge (a 3.7 km ride drawn on a 4 km axis).
    var maxKm = points.reduce(function(m, pt) { return pt.x > m ? pt.x : m; }, 0);

    var rootStyle = getComputedStyle(document.documentElement);
    var tickColor = rootStyle.getPropertyValue("--lightblue").trim() || "#7fa8cf";
    var gridColor = rootStyle.getPropertyValue("--grey").trim() || "#cecece";
    var lineColor = rootStyle.getPropertyValue("--mediumblue").trim() || "#3a6ea5";

    new Chart(canvas, {
        type: "line",
        data: {
            datasets: [{
                label: "Elevation (m)",
                data: points,
                fill: true,
                borderColor: lineColor,
                backgroundColor: "rgba(58, 110, 165, 0.18)",
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
                        max: maxKm || undefined,
                        callback: function(value) { return value + " km"; }
                    }
                }],
                yAxes: [{
                    gridLines: { color: gridColor },
                    ticks: {
                        fontColor: tickColor,
                        beginAtZero: false,
                        precision: 0,
                        callback: function(value) { return value + " m"; }
                    }
                }]
            },
            tooltips: {
                callbacks: {
                    title: function(items) { return wvNum(items[0].xLabel) + " km"; },
                    label: function(item) { return Math.round(item.yLabel) + " m"; }
                }
            }
        }
    });
}

function renderRouteMap(divID, latlngData) {
    var mapDiv = document.getElementById(divID);
    if (!mapDiv) { console.warn('[map] div not found:', divID); return null; }
    if (!latlngData || latlngData.length === 0) { console.warn('[map] no latlng data'); return null; }

    // Leaflet must be available globally
    if (typeof L === 'undefined') {
        console.warn('[map] Leaflet not loaded');
        mapDiv.style.display = 'none';
        return null;
    }

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
    return map;
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

    // The goal-counting toggle freezes with the same rule as delete: only changeable while
    // the session's week is current (the backend rejects the change otherwise).
    const weekEditable = isExerciseWeekEditable(exercise);
    const countsAttrs = weekEditable
        ? `onchange="${onChange}"`
        : `disabled title="Can't change after the week has ended"`;

    // Privacy has no week freeze — hiding a session changes nothing about a settled week.
    // It is Strava that owns it on imported sessions: every sync mirrors the activity's own
    // privacy setting, so the toggle is read-only there rather than silently reverting.
    const fromStrava = exercise.strava_id && exercise.strava_id.length > 0;
    const privateAttrs = fromStrava
        ? `disabled title="Set on Strava"`
        : `onchange="${onChange}"`;

    // Strava source links + combine/divide (parity with the summary card).
    var stravaLinks = "", combineHTML = "", divideHTML = "";
    if (exercise.strava_id && exercise.strava_id.length > 0) {
        stravaLinks = `<div class="we-strava-links">` + exercise.strava_id.map(id =>
            `<a class="we-strava-link" href="https://www.strava.com/activities/${id}" target="_blank">Strava activity ${id}<img src="/assets/external-link.svg" alt=""></a>`
        ).join("") + `</div>`;
        const ids = exercise.strava_id.join(";");
        combineHTML = `<input class="clickable stravaCombineCheck wv-combine" type="checkbox" id="${ids}" title="Select to combine with another workout">`;
        if (exercise.strava_id.length > 1) {
            divideHTML = `<img src="/assets/scissors.svg" class="btn_logo clickable wv-action-icon" title="Split into separate workouts" onclick="divideStravaExercises('${exercise.id}')">`;
        }
    }

    return `
        <div class="workout-view">
            <div class="we-session" id="exercise-sub-${exercise.id}">
                <div class="we-session-head">
                    <span class="we-eyebrow">Editing workout</span>
                    <div class="we-session-actions">
                        ${combineHTML}
                        ${divideHTML}
                        ${sessionDeleteButton(exercise, count)}
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
                    <label class="we-field we-field-check">
                        <span class="we-field-label">Counts toward goal</span>
                        <input class="clickable" type="checkbox" id="exercise-counts-input-${exercise.id}" ${exercise.counts_toward_goal ? 'checked' : ''} ${countsAttrs}>
                    </label>
                    <label class="we-field we-field-check">
                        <span class="we-field-label">Hidden from others</span>
                        <input class="clickable" type="checkbox" id="exercise-private-input-${exercise.id}" ${exercise.private ? 'checked' : ''} ${privateAttrs}>
                    </label>
                </div>
                <textarea class="we-note" id="exercise-note-${exercise.id}" rows="2" placeholder="Workout note" onchange="${onChange}">${escapeHTML(exercise.note || '')}</textarea>
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
            <img src="/assets/plus.svg" alt=""><span>Add activity</span>
        </button>
    `;
}

function generateOperationHTML(operation) {
    const liftingSel = operation.type == 'lifting' ? 'selected' : '';
    const movingSel  = operation.type == 'moving'  ? 'selected' : '';
    const timingSel  = operation.type == 'timing'  ? 'selected' : '';
    const actionName = operation.action ? operation.action.name : "";

    // Gear is edited per operation, on activities that wear it: distance (shoes, a bike)
    // and time-based ones (court shoes for padel). Strength cards omit the selector so
    // updateOperation never touches their gear.
    var gearRowHTML = "";
    if (operationTakesGear(operation)) {
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
                <input class="we-input we-action-input" type="text" autocomplete="off" id="operation-action-text-${operation.id}" name="operation-action-text" placeholder="Activity type" value="${escapeHTML(actionName)}" onkeyup="filterFunction('${operation.id}')" onfocus="showSelectDropdown('${operation.id}', true)">
                <div id="operation-action-text-list-${operation.id}" class="dropdown-actions-wrapper" style="display: none;">
                    ${processExerciseList(operation.id)}
                </div>
            </div>
            <img src="/assets/plus.svg" class="btn_logo clickable wv-action-icon" id="addActionWrapper-${operation.id}" title="Create new activity type" onclick="addAction('${operation.id}')">
            <img src="/assets/trash-2.svg" class="btn_logo clickable wv-action-icon" title="Remove activity" onclick="deleteOperation('${operation.id}')">
        </div>

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
                document.getElementById('day-note-status').textContent = "Saved";
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
    placeDayGear();
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

    var tagEls = document.querySelectorAll('#tag-selector-' + operationID + ' .tag-chip-selected');
    var tags = Array.from(tagEls).map(e => e.getAttribute('data-tag'));

    var descriptionEl = document.getElementById('operation-description-' + operationID);
    var description = descriptionEl ? descriptionEl.value : "";

    var form_obj = {
        "type": type,
        "action": action,
        "weight_unit": weight_unit,
        "distance_unit": distance_unit,
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
    placeDayGear();
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
    if(!on && !confirm("Delete this workout?")) {
        return;
    }

    // The editor inputs only exist in edit mode. When called from the summary
    // (e.g. the delete button), fall back to the cached exercise so we don't read
    // null elements or accidentally blank the note/time/duration.
    const cached = exerciseCache[exerciseID] || {};
    const noteEl = document.getElementById('exercise-note-' + exerciseID);
    const timeEl = document.getElementById('exercise-time-input-' + exerciseID);
    const timeOfDayEl = document.getElementById('exercise-timeofday-input-' + exerciseID);
    const countsEl = document.getElementById('exercise-counts-input-' + exerciseID);
    const privateEl = document.getElementById('exercise-private-input-' + exerciseID);

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

    // Only send the goal-counting flag when the editor is on screen; other update paths
    // (e.g. the summary-card delete) must leave the stored value untouched. The backend
    // treats an omitted field as "no change" (see ExerciseUpdateRequest).
    if (countsEl) {
        form_obj["counts_toward_goal"] = countsEl.checked;
    }

    // Same rule for privacy, minus the Strava case: there the toggle is read-only because
    // the next sync mirrors Strava's own setting, so sending it would be a lie.
    if (privateEl && !privateEl.disabled) {
        form_obj["private"] = privateEl.checked;
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
                error(result.error);
            } else {
                if (fromEditor) {
                    placeExercise(result.exercise, count, true);
                } else {
                    placeExercise(result.exercise, count);
                }
                // Deleting, restoring or changing what counts moves the week's numbers.
                getExerciseDayWeek(exerciseDayID);
                placeDayGear();
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
    placeDayEmpty();
    placeCombineButton();
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
    var formData = JSON.stringify({
        "exercise_day_id": exerciseDayID,
        "is_on": true,
        "note": "",
        "duration": null
    });

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
    xhttp.send(formData);
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
        <div class="day-workout" id="exercise-${exercise.id}">
            ${generateExerciseHTML(exercise, count + 1, true)}
        </div>
    `;
    element.insertAdjacentHTML("beforeend", exerciseHTML)
    injectBackButton(exercise.id);
    placeDayEmpty();
    getExerciseDayWeek(exerciseDayID);
}

function deleteExercise() {

}

function deleteOperation(operationID) {
    if(!confirm("Remove this activity from the workout?")) {
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
                <span class="trm-label">Kind of training</span>
                <select class="trm-select" id="new-action-category-input-${operationID}" name="new-action-category-input">
                    <option value="">Pick from the type above</option>
                    ${Object.entries(actionCategoryLabels).map(([value, label]) => `<option value="${value}">${label}</option>`).join("")}
                </select>
            </div>
            <div class="trm-field">
                <span class="trm-label">Equipment</span>
                <select class="trm-select" id="new-action-equipment-input-${operationID}" name="new-action-equipment-input">
                    ${Object.entries(actionEquipmentLabels).map(([value, label]) => `<option value="${value}">${label}</option>`).join("")}
                </select>
            </div>
            <div class="trm-field">
                <span class="trm-label">Body part</span>
                <input class="trm-input" type="text" id="new-action-bodypart-input-${operationID}" name="new-action-bodypart-input" placeholder="Legs" value="">
            </div>

            <button type="submit" class="btn btn--primary" onclick="createAction('${operationID}');"><img src="/assets/done.svg">Add and use</button>
        </div>
    `;

    TRModal.open({ eyebrow: "New activity type", title: "Add activity type", body: body });
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
    const ops = (exercise.operations || []).filter(operationTakesGear);
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
// suggest the user's primary — matching the session default. The suggestion is saved by the
// next updateOperation, so time-based activities get none: the primary is usually running
// shoes, and editing a padel card must not quietly assign them.
function currentOperationGearID(operation) {
    if (operation.gear) return operation.gear.id;
    if (operation.type !== 'moving') return "";
    const primary = gearList.find(g => g.is_primary && !g.retired);
    return primary ? primary.id : "";
}

function gearOptionLabel(gear) {
    const nick = gear.nickname ? " (" + gear.nickname + ")" : "";
    const dist = gear.distance ? " · " + gear.distance.toFixed(1) + " km" : "";
    const time = gear.duration_seconds ? " · " + gearTimeLabel(gear.duration_seconds) : "";
    const retired = gear.retired ? " · retired" : "";
    return gear.name + nick + dist + time + retired;
}

// operationTakesGear mirrors the server's rule (controllers/gear.go): distance and
// time-based activities carry gear, strength work does not.
function operationTakesGear(operation) {
    return !!operation && (operation.type === 'moving' || operation.type === 'timing');
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
// control is only a convenience to set every gear-carrying activity at once (distance
// and time-based; the server skips strength work too), so it is shown only when a
// session holds 2+ of them.
function generateGearFieldHTML(exercise) {
    const gearOps = (exercise.operations || []).filter(operationTakesGear);
    if (gearOps.length < 2) {
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
    onGearChanged = function() { renderGearModalBody(); refreshGearSelectors(); placeDayGear(); };
    TRModal.open({ eyebrow: "Gear", title: "Manage gear", body: `<div class="gear-empty">Loading gear…</div>` });
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
    const category = document.getElementById('new-action-category-input-' + operationID).value;
    const equipment = document.getElementById('new-action-equipment-input-' + operationID).value;

    var form_obj = {
        "name": name,
        "norwegian_name" : norwegian_name,
        "description": description,
        "type": type,
        "body_part": body_part,
        "category": category,
        "equipment": equipment
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
        alert("Choose two or more workouts to combine.");
        return;
    }

    if(!confirm("Combine these Strava workouts into one?")) {
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
    if(!confirm("Split this workout back into its Strava activities?")) {
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

// Bind the ⓘ explainer handler for the page. Delegated on `document`, so it survives every
// re-render of the read view and needs no re-binding.
wireInfoMarks();
wireTraceTabs();
