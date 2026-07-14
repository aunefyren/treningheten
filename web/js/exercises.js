// /exercises — a searchable, sortable activity timeline. Backed by GET /auth/activities
// (see controllers/activity.go), which returns per-activity items with metrics aggregated
// from their sets. Two modes off one feed: BROWSE (sort by date → grouped by day + session)
// and FIND (a metric sort or a type filter → flat, optionally ranked). See docs/wip.md.

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
    items: [],
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
                <div class="text-body" style="text-align: center;">
                    Your activity timeline. Scroll recent sessions, or filter and sort to find a
                    specific one — your longest run, a certain padel match, your oldest ride.
                </div>
                <div class="button-collection">
                    <button onclick="window.location.href = '/gear';" class="regular-button" type="submit">Manage gear</button>
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
                    <button class="regular-button" onclick="loadFeed(false)">Load more</button>
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
        feedState.items = [];
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

            feedState.items = feedState.items.concat(result.activities || []);
            feedState.total = result.total || 0;
            feedState.hasMore = !!result.has_more;
            feedState.offset += (result.activities || []).length;

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
// activity, so we show a flat (optionally ranked) list rather than day/session groups.
function feedIsFindMode() {
    return feedState.sort !== "date" || !!feedState.actionID;
}

function renderFeed() {
    var resultsEl = document.getElementById("feed-results");
    var countEl = document.getElementById("feed-count");
    var moreEl = document.getElementById("feed-more");

    if (feedState.total === 0) {
        resultsEl.innerHTML = `<div class="feed-empty">No activities match. Try clearing the filters.</div>`;
        countEl.textContent = "";
        moreEl.style.display = "none";
        return;
    }

    countEl.textContent = "Showing " + feedState.items.length + " of " + feedState.total;

    if (feedState.sort !== "date") {
        resultsEl.innerHTML = feedState.items.map(function(item, i) {
            return feedActivityRow(item, { showDate: true, rank: i + 1 });
        }).join("");
    } else if (feedIsFindMode()) {
        // Type-filtered but chronological: flat list, no rank.
        resultsEl.innerHTML = feedState.items.map(function(item) {
            return feedActivityRow(item, { showDate: true });
        }).join("");
    } else {
        resultsEl.innerHTML = renderBrowseGroups(feedState.items);
    }

    moreEl.style.display = feedState.hasMore ? "block" : "none";
}

// renderBrowseGroups groups the (date-descending, session-adjacent) items into day headers
// and session sub-blocks.
function renderBrowseGroups(items) {
    var html = "";
    var currentDay = null;
    var currentSession = null;
    var openSession = false;
    var openDay = false;

    function closeSession() {
        if (openSession) { html += `</div>`; openSession = false; }
    }
    function closeDay() {
        closeSession();
        if (openDay) { html += `</div>`; openDay = false; }
    }

    items.forEach(function(item) {
        var dayKey = feedDayKey(item.date);
        if (dayKey !== currentDay) {
            closeDay();
            currentDay = dayKey;
            currentSession = null;
            html += `<div class="feed-day"><div class="feed-day-header">${feedDayLabel(item.date)}</div>`;
            openDay = true;
        }
        if (item.exercise_id !== currentSession) {
            closeSession();
            currentSession = item.exercise_id;
            var count = item.session_activity_count || 1;
            var countLabel = count + (count === 1 ? " activity" : " activities");
            html += `
                <div class="feed-session">
                    <div class="feed-session-header">
                        <span class="feed-session-time">${feedTimeOnly(item.time)}</span>
                        <span class="feed-session-count">${countLabel}</span>
                    </div>`;
            openSession = true;
        }
        html += feedActivityRow(item, { showDate: false });
    });

    closeDay();
    return html;
}

// feedActivityRow renders one activity. opts.showDate shows the full date on the right
// (find mode); opts.rank prepends a rank badge (metric sorts).
function feedActivityRow(item, opts) {
    opts = opts || {};
    var icon = feedActionIcon(item);
    var chips = feedMetricChips(item).join(" · ");
    var note = (item.note && item.note.trim())
        ? `<span class="feed-note" title="${escapeHTML(item.note)}">📝</span>`
        : "";
    var when = opts.showDate ? feedWhenLabel(item) : "";
    var rank = opts.rank ? `<div class="feed-rank">${opts.rank}</div>` : "";

    return `
        <div class="feed-row clickable" onclick="exerciseRedirect('${item.exercise_day_id}')">
            ${rank}
            <div class="feed-row-icon">${icon}</div>
            <div class="feed-row-body">
                <div class="feed-row-title">${escapeHTML(item.action_name || "Activity")}${note}</div>
                <div class="feed-row-metrics">${chips || "&nbsp;"}</div>
            </div>
            ${when ? `<div class="feed-row-when">${when}</div>` : ""}
        </div>
    `;
}

// feedActionIcon prefers the action's SVG logo, falling back to a type-based glyph.
function feedActionIcon(item) {
    if (item.action_has_logo && item.action_name) {
        return `<img src="/assets/actions/${encodeURIComponent(item.action_name)}.svg" class="feed-logo color-invert" onerror="this.outerHTML='${feedActionGlyph(item.action_type)}'">`;
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
