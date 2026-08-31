// /exercises — a searchable, sortable workout timeline. Backed by GET /auth/activities
// (see controllers/activity.go), which returns SESSIONS: each one carries the metrics rolled
// up across the whole workout plus the activities inside it. The session is the unit here
// because a workout logged with the front page's "+" button has no activities at all, and it
// still belongs in the timeline. Two modes off one feed: BROWSE (sort by date → grouped by
// day) and FIND (a metric sort or a type filter → flat, optionally ranked). A session with
// exactly one activity collapses into a single card — the common case shouldn't read as a
// box wrapped around one row. See docs/exercises.md.

var feedState = {
    actionID: "",
    sort: "date",
    order: "desc",
    q: "",
    hasDistance: false,
    start: "",
    end: "",
    offset: 0,
    limit: 20,
    total: 0,
    hasMore: false,
    loading: false,
    sessions: [],
    actions: []
};

var feedSearchTimer = null;

// Sort options: label → {sort, order}. The value stored on the <select> is the key.
var feedSortOptions = [
    { key: "newest",   label: "Newest first",     sort: "date",     order: "desc" },
    { key: "oldest",   label: "Oldest first",     sort: "date",     order: "asc"  },
    { key: "distance", label: "Longest distance", sort: "distance", order: "desc" },
    { key: "duration", label: "Longest time",     sort: "duration", order: "desc" },
    { key: "weight",   label: "Heaviest",         sort: "weight",   order: "desc" },
    { key: "reps",     label: "Most reps",        sort: "reps",     order: "desc" }
];

function load_page(result) {

    if (result !== false) {
        var login_data = JSON.parse(result);
        user_id = login_data.data.id;

        try {
            admin = login_data.data.admin;
        } catch {
            admin = false;
        }

        showAdminMenu(admin);
    } else {
        user_id = 0;
        admin = false;
    }

    var sortOptionsHTML = feedSortOptions.map(function(o) {
        return `<option value="${o.key}">${o.label}</option>`;
    }).join("");

    var html = `
        <div id="front-page">

            <div class="module">
                <div class="text-body u-text-center">
                    Your workout timeline. Scroll recent sessions, or filter and sort to find a
                    specific one — your longest run, a certain padel match, your oldest ride.
                </div>
                <div class="btn-group">
                    <button onclick="window.location.href = '/gear';" class="btn" type="submit">Manage gear</button>
                </div>
            </div>

            <div class="feed-panel">

                <div class="feed-controls">
                    <select id="feed-type" class="feed-input" onchange="feedApplyControls()">
                        <option value="">All activity types</option>
                    </select>
                    <select id="feed-sort" class="feed-input" onchange="feedApplyControls()">
                        ${sortOptionsHTML}
                    </select>
                    <input id="feed-q" class="feed-input" type="text" placeholder="Search notes & types" oninput="feedDebouncedSearch()">
                </div>
                <div class="feed-controls feed-controls-secondary">
                    <label class="feed-daterange">From <input id="feed-start" class="feed-input" type="date" onchange="feedApplyControls()"></label>
                    <label class="feed-daterange">To <input id="feed-end" class="feed-input" type="date" onchange="feedApplyControls()"></label>
                    <label class="feed-check"><input type="checkbox" id="feed-hasdist" onchange="feedApplyControls()"> With distance only</label>
                </div>

                <div class="feed-count" id="feed-count"></div>
                <div class="feed-results" id="feed-results"></div>

                <div class="feed-loading" id="feed-loading" style="display:none;"><div class="trh-spinner"></div></div>
                <div class="feed-more" id="feed-more" style="display:none;">
                    <button class="btn" onclick="loadFeed(false)">Load more</button>
                </div>

            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Everything you have logged, in one line.';
    clearResponse();

    if (result !== false) {
        showLoggedInMenu();
        loadActions();
        loadFeed(true);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

// escapeHTML makes user/provider text safe in HTML text and attributes.
function escapeHTML(value) {
    return String(value == null ? "" : value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

// loadActions fills the type filter with the user's activity types.
function loadActions() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                return;
            }
            if (result.error || !result.actions) {
                return;
            }
            feedState.actions = result.actions;
            var select = document.getElementById("feed-type");
            if (!select) {
                return;
            }
            result.actions.forEach(function(action) {
                var option = document.createElement("option");
                option.value = action.id;
                option.text = action.name;
                select.add(option);
            });
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/actions");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

// feedApplyControls reads the control values into state and reloads from the top.
function feedApplyControls() {
    var sortKey = document.getElementById("feed-sort").value;
    var sortOption = feedSortOptions.find(function(o) { return o.key === sortKey; }) || feedSortOptions[0];

    feedState.actionID = document.getElementById("feed-type").value || "";
    feedState.sort = sortOption.sort;
    feedState.order = sortOption.order;
    feedState.q = document.getElementById("feed-q").value.trim();
    feedState.start = document.getElementById("feed-start").value || "";
    feedState.end = document.getElementById("feed-end").value || "";
    feedState.hasDistance = document.getElementById("feed-hasdist").checked;

    loadFeed(true);
}

function feedDebouncedSearch() {
    if (feedSearchTimer) {
        clearTimeout(feedSearchTimer);
    }
    feedSearchTimer = setTimeout(feedApplyControls, 350);
}

// loadFeed fetches a page. reset=true clears the accumulated items and starts at offset 0
// (a filter/sort change); reset=false appends the next page (Load more).
function loadFeed(reset) {
    if (feedState.loading) {
        return;
    }
    if (reset) {
        feedState.offset = 0;
        feedState.sessions = [];
    }
    feedState.loading = true;
    document.getElementById("feed-loading").style.display = "flex";
    document.getElementById("feed-more").style.display = "none";

    var params = [];
    params.push("sort=" + encodeURIComponent(feedState.sort));
    params.push("order=" + encodeURIComponent(feedState.order));
    params.push("limit=" + feedState.limit);
    params.push("offset=" + feedState.offset);
    if (feedState.actionID) { params.push("action_id=" + encodeURIComponent(feedState.actionID)); }
    if (feedState.q) { params.push("q=" + encodeURIComponent(feedState.q)); }
    if (feedState.start) { params.push("start=" + encodeURIComponent(feedState.start)); }
    if (feedState.end) { params.push("end=" + encodeURIComponent(feedState.end)); }
    if (feedState.hasDistance) { params.push("has_distance=true"); }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            feedState.loading = false;
            document.getElementById("feed-loading").style.display = "none";

            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            if (result.error) {
                error(result.error);
                return;
            }
            clearResponse();

            feedState.sessions = feedState.sessions.concat(result.sessions || []);
            feedState.total = result.total || 0;
            feedState.hasMore = !!result.has_more;
            feedState.offset += (result.sessions || []).length;

            renderFeed();
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/activities?" + params.join("&"));
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

// findMode: a metric sort or an activity-type filter means the user is hunting a specific
// workout, so we show a flat (optionally ranked) list rather than day groups.
function feedIsFindMode() {
    return feedState.sort !== "date" || !!feedState.actionID;
}

// feedHasFilters reports whether anything is narrowing the feed, so an empty result can tell
// the difference between "nothing matched" and "nothing logged yet".
function feedHasFilters() {
    return !!(feedState.actionID || feedState.q || feedState.start || feedState.end || feedState.hasDistance);
}

function renderFeed() {
    var resultsEl = document.getElementById("feed-results");
    var countEl = document.getElementById("feed-count");
    var moreEl = document.getElementById("feed-more");

    if (feedState.total === 0) {
        resultsEl.innerHTML = feedHasFilters()
            ? `<div class="feed-empty">No workouts match. Try clearing the filters.</div>`
            : `<div class="feed-empty">Nothing logged yet. Tick a day on the front page and it shows up here.</div>`;
        countEl.textContent = "";
        moreEl.style.display = "none";
        return;
    }

    countEl.textContent = "Showing " + feedState.sessions.length + " of " + feedState.total
        + (feedState.total === 1 ? " workout" : " workouts");

    if (feedState.sort !== "date") {
        resultsEl.innerHTML = feedState.sessions.map(function(session, i) {
            return feedSessionCard(session, { showDate: true, rank: i + 1 });
        }).join("");
    } else if (feedIsFindMode()) {
        // Type-filtered but chronological: flat list, no rank.
        resultsEl.innerHTML = feedState.sessions.map(function(session) {
            return feedSessionCard(session, { showDate: true });
        }).join("");
    } else {
        resultsEl.innerHTML = renderBrowseGroups(feedState.sessions);
    }

    moreEl.style.display = feedState.hasMore ? "block" : "none";
}

// renderBrowseGroups groups the (date-descending) sessions under day headers.
function renderBrowseGroups(sessions) {
    var html = "";
    var currentDay = null;
    var openDay = false;

    sessions.forEach(function(session) {
        var dayKey = feedDayKey(session.date);
        if (dayKey !== currentDay) {
            if (openDay) { html += `</div>`; }
            currentDay = dayKey;
            html += `<div class="feed-day"><div class="feed-day-header">${feedDayLabel(session.date)}</div>`;
            openDay = true;
        }
        html += feedSessionCard(session, { showDate: false });
    });

    if (openDay) { html += `</div>`; }
    return html;
}

// feedSessionCard renders one workout. A session with a single activity collapses — the card
// takes that activity's icon and name and draws no breakdown, so the common case (one
// imported run) reads as one line. Two or more activities get the breakdown underneath.
// opts.showDate shows the full date on the right (find mode); opts.rank prepends a rank badge.
function feedSessionCard(session, opts) {
    opts = opts || {};
    var activities = session.activities || [];
    var collapsed = activities.length === 1;

    // Mixed or empty work has no one type, so it keeps the generic glyph rather than
    // picking a winner from its parts.
    var icon = collapsed ? feedActionIcon(activities[0]) : feedActionGlyph("");
    var chips = feedSessionChips(session, collapsed ? activities[0] : null).join(" · ");

    var note = (session.note && session.note.trim())
        ? `<span class="feed-note" title="${escapeHTML(session.note)}">📝</span>`
        : "";
    var count = activities.length > 1
        ? `<span class="feed-session-count">${activities.length} activities</span>`
        : "";
    var noCount = session.counts_toward_goal ? ""
        : `<span class="feed-nocount" title="Logged but doesn't count toward your weekly goal">Doesn't count</span>`;
    var hidden = session.private
        ? `<span class="feed-nocount" title="Only you can see this session. It still counts toward your weekly goal">Hidden</span>`
        : "";

    var when = opts.showDate ? feedWhenLabel(session) : feedTimeOnly(session.time);
    var rank = opts.rank ? `<div class="feed-rank">${opts.rank}</div>` : "";

    var breakdown = collapsed || activities.length === 0 ? "" : `
            <div class="feed-activities">
                ${activities.map(feedActivityRow).join("")}
            </div>`;

    return `
        <div class="feed-session clickable" onclick="exerciseRedirect('${session.exercise_day_id}')">
            <div class="feed-session-head">
                ${rank}
                <div class="feed-row-icon">${icon}</div>
                <div class="feed-row-body">
                    <div class="feed-row-title"><span class="feed-title-name">${escapeHTML(feedSessionTitle(session))}</span>${note}${count}${noCount}${hidden}</div>
                    <div class="feed-row-metrics">${chips || "&nbsp;"}</div>
                </div>
                ${when ? `<div class="feed-row-when">${when}</div>` : ""}
            </div>${breakdown}
        </div>
    `;
}

// feedSessionTitle names the workout by what is in it: the single activity when collapsed,
// otherwise the distinct activity types (two, then "+N more"). A session logged without any
// activity is simply a workout, which is exactly what the user said it was.
function feedSessionTitle(session) {
    var activities = session.activities || [];
    if (activities.length === 0) {
        return "Workout";
    }

    var names = [];
    activities.forEach(function(activity) {
        var name = activity.action_name || "Activity";
        if (names.indexOf(name) === -1) {
            names.push(name);
        }
    });

    if (names.length <= 2) {
        return names.join(" + ");
    }
    return names.slice(0, 2).join(" + ") + " + " + (names.length - 2) + " more";
}

// feedActivityRow renders one activity inside a session's breakdown. It is deliberately
// unboxed — the session card is the frame, and a second border system inside it would fight
// the panel (see docs/styleguide.md).
function feedActivityRow(item) {
    var chips = feedMetricChips(item).join(" · ");
    var note = (item.note && item.note.trim())
        ? `<span class="feed-note" title="${escapeHTML(item.note)}">📝</span>`
        : "";

    return `
        <div class="feed-activity">
            <div class="feed-activity-icon">${feedActionIcon(item)}</div>
            <div class="feed-activity-name">${escapeHTML(item.action_name || "Activity")}${note}</div>
            <div class="feed-activity-metrics">${chips}</div>
        </div>
    `;
}

// feedActionIcon prefers the action's SVG logo, falling back to a type-based glyph.
function feedActionIcon(item) {
    if (item.action_has_logo && item.action_name) {
        return `<img src="/assets/actions/${encodeURIComponent(item.action_name)}.svg" class="feed-logo" onerror="this.outerHTML='${feedActionGlyph(item.action_type)}'">`;
    }
    return feedActionGlyph(item.action_type);
}

function feedActionGlyph(type) {
    switch ((type || "").toLowerCase()) {
        case "cardio":   return "🏃";
        case "sport":    return "🎾";
        case "strength":
        case "lifting":  return "🏋";
        case "swimming": return "🏊";
        case "cycling":  return "🚴";
        default:         return "🏅";
    }
}

// feedSessionChips builds the card's metric list from the session totals. When the card is
// collapsed onto its single activity, that activity's stream scalars (heart rate, climb) come
// along — they are per-activity readings with no meaningful session-wide sum.
function feedSessionChips(session, activity) {
    var chips = [];
    if (session.distance > 0) {
        chips.push(session.distance.toFixed(2) + " " + (session.distance_unit || "km"));
    }
    if (session.duration_seconds > 0) {
        chips.push(secondsToDurationString(session.duration_seconds));
    }
    if (session.repetitions > 0) {
        chips.push(Math.round(session.repetitions) + " reps");
    }
    if (session.top_weight > 0) {
        chips.push("top " + session.top_weight + " " + (session.weight_unit || "kg"));
    }
    if (activity) {
        if (activity.avg_heartrate > 0) {
            chips.push(activity.avg_heartrate + " bpm");
        }
        if (activity.elevation_gain_m > 1) {
            chips.push("+" + Math.round(activity.elevation_gain_m) + " m");
        }
    }
    if (chips.length === 0 && session.set_count > 0) {
        chips.push(session.set_count + (session.set_count === 1 ? " set" : " sets"));
    }
    return chips;
}

// feedMetricChips builds the floated metric list from whichever aggregates are present.
function feedMetricChips(item) {
    var chips = [];
    if (item.distance > 0) {
        chips.push(item.distance.toFixed(2) + " " + (item.distance_unit || "km"));
    }
    if (item.duration_seconds > 0) {
        chips.push(secondsToDurationString(item.duration_seconds));
    }
    if (item.repetitions > 0) {
        chips.push(Math.round(item.repetitions) + " reps");
    }
    if (item.top_weight > 0) {
        chips.push("top " + item.top_weight + " " + (item.weight_unit || "kg"));
    }
    // Stream scalars (present only for Strava-backed activities), so a card reads its effort
    // and climb without opening it. Kept unit-safe — pace/speed lives on the detail view.
    if (item.avg_heartrate > 0) {
        chips.push(item.avg_heartrate + " bpm");
    }
    if (item.elevation_gain_m > 1) {
        chips.push("+" + Math.round(item.elevation_gain_m) + " m");
    }
    if (chips.length === 0 && item.set_count > 0) {
        chips.push(item.set_count + (item.set_count === 1 ? " set" : " sets"));
    }
    return chips;
}

function feedDayKey(dateString) {
    var d = new Date(dateString);
    return d.getFullYear() + "-" + d.getMonth() + "-" + d.getDate();
}

function feedDayLabel(dateString) {
    try {
        var d = new Date(dateString);
        return GetDayOfTheWeek(d) + " · " + GetDateString(d, false);
    } catch {
        return "Unknown date";
    }
}

function feedWhenLabel(item) {
    try {
        var d = new Date(item.date);
        var label = GetDayOfTheWeek(d) + "<br>" + GetDateString(d, false);
        var t = feedTimeOnly(item.time);
        return t ? label + " · " + t : label;
    } catch {
        return "";
    }
}

// feedTimeOnly formats a session time (HH:MM) or returns "" when the session has no clock
// time (a date-only entry).
function feedTimeOnly(timeString) {
    if (!timeString) {
        return "";
    }
    try {
        var d = new Date(timeString);
        if (isNaN(d.getTime())) {
            return "";
        }
        // Midnight almost always means "no real time set" rather than an actual 00:00 session.
        if (d.getHours() === 0 && d.getMinutes() === 0) {
            return "";
        }
        return ("0" + d.getHours()).slice(-2) + ":" + ("0" + d.getMinutes()).slice(-2);
    } catch {
        return "";
    }
}

function exerciseRedirect(exerciseDayID) {
    window.location = '/exercises/' + exerciseDayID;
}
