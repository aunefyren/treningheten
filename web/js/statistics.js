// /statistics — five tabs over the user's own training: an Overview of a period, personal
// Records, Activity types (one type in depth, with its soundtrack and GPS heatmap), Seasons
// and Body (weight). The tab and the chosen activity type live in the URL (?tab=, ?action=)
// so a view can be reloaded or linked. Every number is computed server-side; this file only
// lays them out. See docs/statistics.md.

var statsTabs = [
    { key: "overview", label: "Overview" },
    { key: "records", label: "Records" },
    { key: "types", label: "Activity types" },
    { key: "seasons", label: "Seasons" },
    { key: "body", label: "Body" }
];

var statsPeriods = [
    { key: "4w", label: "4 weeks", previous: "the previous 4 weeks" },
    { key: "12w", label: "12 weeks", previous: "the previous 12 weeks" },
    { key: "year", label: "This year", previous: "the same time last year" },
    { key: "all", label: "All time", previous: "" }
];

// isAllTimePeriod is the one check for "no narrowing": the period key is "all" on both the
// Overview and Activity types tabs (the UI state, not the response). Custom is never all time.
function isAllTimePeriod(key) {
    return key === "all";
}

var statsTypePeriods = statsPeriods.concat([{ key: "custom", label: "Custom" }]);

var overviewMetrics = [
    { key: "workouts", label: "Workouts" },
    { key: "time", label: "Time" },
    { key: "distance", label: "Distance" }
];

var statsState = {
    tab: "overview",
    loaded: {},
    overviewPeriod: "4w",
    overviewMetric: "workouts",
    overview: null,
    actions: [],
    typesActionID: "",
    typesPeriod: "12w",
    typesStatistics: null,
    typesTrendMetric: "",
    seasons: [],
    charts: {}
};

function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
        user_id = login_data.data.id

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
    }

    document.getElementById('content').innerHTML = statisticsPageHTML();
    document.getElementById('card-header').innerHTML = 'Math and stuff.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        readStatisticsURL();
        selectStatisticsTab(statsState.tab);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

// ── Page shell and tabs ──────────────────────────────────────────────────────

function statisticsPageHTML() {
    var tabs = statsTabs.map(function(tab) {
        return `<button type="button" class="user-stat-tab stats-tab" role="tab" id="stats-tab-${tab.key}" aria-controls="stats-panel-${tab.key}" aria-selected="false" tabindex="-1" onclick="selectStatisticsTab('${tab.key}')" onkeydown="statisticsTabKey(event)">${tab.label}</button>`;
    }).join("");

    return `
        <div class="stats-page">
            <div class="user-stat-tabs stats-tabs" role="tablist" aria-label="Statistics">${tabs}</div>

            <section class="stats-panel" role="tabpanel" id="stats-panel-overview" aria-labelledby="stats-tab-overview" hidden>
                <div class="user-stat-tabs stats-segments" role="group" aria-label="Period" id="overview-periods"></div>
                <div class="stat-grid stats-headline" id="overview-totals"></div>
                <div class="stat-panel">
                    <div class="stats-panel-head">
                        <h2 class="stats-heading" id="overview-trend-title">Workouts</h2>
                        <div class="user-stat-tabs stats-segments stats-segments--small" role="group" aria-label="Chart" id="overview-metrics"></div>
                    </div>
                    <div class="stats-chart"><canvas id="overview-chart"></canvas></div>
                </div>
                <div class="stats-columns">
                    <div class="stat-panel">
                        <h2 class="stats-heading">Streaks</h2>
                        <div class="stat-grid stats-grid-2" id="overview-streaks"></div>
                    </div>
                    <div class="stat-panel">
                        <h2 class="stats-heading">Activity mix</h2>
                        <ol class="stats-mix" id="overview-mix"></ol>
                    </div>
                </div>
                <div class="stat-panel">
                    <div class="stats-panel-head">
                        <h2 class="stats-heading">Records set in this period</h2>
                        <button type="button" class="btn btn--ghost btn--sm" id="overview-records-link" onclick="selectStatisticsTab('records')">All records</button>
                    </div>
                    <div id="overview-records"></div>
                </div>
            </section>

            <section class="stats-panel" role="tabpanel" id="stats-panel-records" aria-labelledby="stats-tab-records" hidden>
                <div id="records-list"></div>
            </section>

            <section class="stats-panel" role="tabpanel" id="stats-panel-types" aria-labelledby="stats-tab-types" hidden>
                <div class="stats-controls">
                    <div class="field stats-type-field">
                        <label for="types-action" class="field-label">Activity type</label>
                        <select id="types-action" onchange="chooseStatisticsType(this.value)">
                            <option value="">Choose an activity type</option>
                        </select>
                    </div>
                    <div class="user-stat-tabs stats-segments" role="group" aria-label="Period" id="types-periods"></div>
                    <div class="field-row stats-custom" id="types-custom" hidden>
                        <div class="field">
                            <label for="types-start" class="field-label">From</label>
                            <input type="date" id="types-start" onchange="loadTypeStatistics()">
                        </div>
                        <div class="field">
                            <label for="types-end" class="field-label">To</label>
                            <input type="date" id="types-end" onchange="loadTypeStatistics()">
                        </div>
                    </div>
                </div>
                <div class="user-stats-empty" id="types-empty">
                    <div class="user-stats-empty-title">Choose an activity type</div>
                    <div class="user-stats-empty-hint">Pick one of the types you have logged to see its totals, trend and bests over a period.</div>
                </div>
                <div id="types-content" hidden>
                    <div class="stat-panel" id="types-records-panel" hidden>
                        <div class="stats-panel-head">
                            <h2 class="stats-heading">Bests in this period</h2>
                            <button type="button" class="btn btn--ghost btn--sm" id="types-records-link" onclick="selectStatisticsTab('records')">All-time records</button>
                        </div>
                        <div class="records-grid" id="types-records"></div>
                    </div>
                    <div id="types-stats"></div>
                    <div class="stat-panel" id="types-trend-panel" hidden>
                        <div class="stats-panel-head">
                            <h2 class="stats-heading" id="types-trend-title">Trend</h2>
                            <div class="user-stat-tabs stats-segments stats-segments--small" role="group" aria-label="Chart" id="types-trend-metrics"></div>
                        </div>
                        <div class="stats-chart"><canvas id="types-trend-chart"></canvas></div>
                    </div>
                    <div id="types-soundtrack"></div>
                    <div class="stat-panel" id="activity-heatmap-wrapper" hidden>
                        <h2 class="stats-heading">Where you went</h2>
                        <div id="activity-heatmap-canvas" class="heatmap-canvas"></div>
                    </div>
                    <p class="stats-links"><a id="types-workouts-link" href="/exercises">See these workouts</a></p>
                </div>
            </section>

            <section class="stats-panel" role="tabpanel" id="stats-panel-seasons" aria-labelledby="stats-tab-seasons" hidden>
                <div class="stats-controls">
                    <div class="field stats-type-field">
                        <label for="select_season" class="field-label">Season</label>
                        <select id="select_season" onchange="chooseSeason()"></select>
                    </div>
                </div>
                <div class="user-stats-empty" id="season-empty" hidden>
                    <div class="user-stats-empty-title">No seasons yet</div>
                    <div class="user-stats-empty-hint">Join a season from the front page and its weeks show up here once it starts.</div>
                </div>
                <div id="season-content" hidden>
                    <div class="stat-grid stats-headline" id="season-tiles"></div>
                    <div class="stat-panel">
                        <h2 class="stats-heading">Workouts per week</h2>
                        <div class="stats-chart"><canvas id="season-weeks-chart"></canvas></div>
                        <p class="stats-caption">The dashed line is your weekly goal. Weeks on sick leave are marked in grey.</p>
                    </div>
                    <div class="stat-panel">
                        <h2 class="stats-heading">Workouts per weekday</h2>
                        <div class="stats-chart"><canvas id="season-weekday-chart"></canvas></div>
                    </div>
                </div>
            </section>

            <section class="stats-panel" role="tabpanel" id="stats-panel-body" aria-labelledby="stats-tab-body" hidden>
                <div class="stats-controls">
                    <button type="button" class="btn btn--primary" onclick="getWeights(true)"><img src="/assets/plus.svg" class="color-invert" alt="">Log weight</button>
                </div>
                <div class="user-stats-empty" id="body-empty" hidden>
                    <div class="user-stats-empty-title">No weight logged yet</div>
                    <div class="user-stats-empty-hint">Log your first weight to start the chart.</div>
                </div>
                <div id="body-content" hidden>
                    <div class="stat-grid stats-headline" id="body-tiles"></div>
                    <div class="stat-panel">
                        <h2 class="stats-heading">Weight</h2>
                        <div class="stats-chart"><canvas id="body-weight-chart"></canvas></div>
                    </div>
                </div>
            </section>
        </div>
    `;
}

// readStatisticsURL restores the tab and the activity type from ?tab= and ?action=.
function readStatisticsURL() {
    var params = new URLSearchParams(window.location.search);
    var tab = params.get("tab");
    if (statsTabs.some(function(t) { return t.key === tab; })) {
        statsState.tab = tab;
    }
    statsState.typesActionID = params.get("action") || "";
}

function writeStatisticsURL() {
    var url = new URL(window.location.href);
    url.searchParams.set("tab", statsState.tab);
    if (statsState.tab === "types" && statsState.typesActionID) {
        url.searchParams.set("action", statsState.typesActionID);
    } else {
        url.searchParams.delete("action");
    }
    window.history.replaceState({}, "", url);
}

// selectStatisticsTab shows one tab and loads its data the first time it opens.
function selectStatisticsTab(key) {
    statsState.tab = key;
    statsTabs.forEach(function(tab) {
        var button = document.getElementById("stats-tab-" + tab.key);
        var active = tab.key === key;
        button.classList.toggle("user-stat-tab-active", active);
        button.setAttribute("aria-selected", active ? "true" : "false");
        button.tabIndex = active ? 0 : -1;
        document.getElementById("stats-panel-" + tab.key).hidden = !active;
    });
    writeStatisticsURL();

    if (statsState.loaded[key]) {
        return;
    }
    statsState.loaded[key] = true;
    if (key === "overview") {
        placeSegments("overview-periods", statsPeriods, statsState.overviewPeriod, "chooseOverviewPeriod");
        placeSegments("overview-metrics", overviewMetrics, statsState.overviewMetric, "chooseOverviewMetric");
        loadOverview();
    } else if (key === "records") {
        loadRecords();
    } else if (key === "types") {
        placeSegments("types-periods", statsTypePeriods, statsState.typesPeriod, "chooseTypePeriod");
        placeDefaultDates();
        loadTypeOptions();
    } else if (key === "seasons") {
        getSeasons();
    } else if (key === "body") {
        getWeights(false);
    }
}

// statisticsTabKey moves between tabs with the arrow keys, as a tablist should.
function statisticsTabKey(event) {
    var index = statsTabs.findIndex(function(t) { return t.key === statsState.tab; });
    if (event.key === "ArrowRight" || event.key === "ArrowLeft") {
        var step = event.key === "ArrowRight" ? 1 : -1;
        var next = statsTabs[(index + step + statsTabs.length) % statsTabs.length].key;
        selectStatisticsTab(next);
        document.getElementById("stats-tab-" + next).focus();
        event.preventDefault();
    }
}

// placeSegments renders a segmented control; the active segment is aria-pressed.
function placeSegments(containerID, options, activeKey, handler) {
    document.getElementById(containerID).innerHTML = options.map(function(option) {
        var active = option.key === activeKey;
        return `<button type="button" class="user-stat-tab${active ? " user-stat-tab-active" : ""}" aria-pressed="${active}" onclick="${handler}('${option.key}')">${option.label}</button>`;
    }).join("");
}

// statsGet is the page's one GET: it parses the JSON, reports errors, and hands the result on.
function statsGet(path, onSuccess) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
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
                onSuccess(result);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + path);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

// ── Formatting ───────────────────────────────────────────────────────────────

// escapeStat makes user-provided text safe for innerHTML (titles, artists and artwork URLs
// come from third-party providers).
function escapeStat(value) {
    return String(value == null ? "" : value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

function unitLabel(unit) {
    return unit === "miles" ? "mi" : unit;
}

function formatNumber(value, decimals) {
    return Number(value).toLocaleString("en-GB", { minimumFractionDigits: decimals, maximumFractionDigits: decimals });
}

// formatClock is a stopwatch time: 4:38, 22:41 or 1:02:03.
function formatClock(seconds) {
    seconds = Math.round(seconds);
    var hours = Math.floor(seconds / 3600);
    var minutes = Math.floor((seconds % 3600) / 60);
    var rest = seconds % 60;
    if (hours > 0) {
        return hours + ":" + padNumber(minutes, 2) + ":" + padNumber(rest, 2);
    }
    return minutes + ":" + padNumber(rest, 2);
}

// formatHours is a long total: 45 min, 3 h 20 min, 112 h.
function formatHours(seconds) {
    var minutes = Math.round(seconds / 60);
    if (minutes < 60) {
        return minutes + " min";
    }
    var hours = Math.floor(minutes / 60);
    var rest = minutes % 60;
    if (hours >= 100 || rest === 0) {
        return hours + " h";
    }
    return hours + " h " + rest + " min";
}

function formatDay(value) {
    return new Date(value).toLocaleDateString("en-GB", { day: "numeric", month: "short", year: "numeric" });
}

function plural(count, one, many) {
    return count === 1 ? one : many;
}

// cssToken reads a colour token so charts follow the theme.
function cssToken(name) {
    return getComputedStyle(document.documentElement).getPropertyValue(name).trim();
}

// statCard renders one metric tile: a quiet label above a display-font value. family sets
// the accent stripe so a tile's colour says what kind of metric it is. detail is an optional
// quiet line under the value.
function statCard(label, value, unit, family, detail) {
    var unitHtml = unit ? `<span class="stat-card-unit">${escapeStat(unit)}</span>` : "";
    var detailHtml = detail ? `<div class="stat-card-detail">${detail}</div>` : "";
    return `
        <div class="stat-card" data-family="${family}">
            <div class="stat-card-label">${escapeStat(label)}</div>
            <div class="stat-card-value">${escapeStat(value)}${unitHtml}</div>
            ${detailHtml}
        </div>`;
}

// statSection wraps a group of cards under a heading; empty groups render nothing, so an
// activity type without that kind of data simply has no such section.
function statSection(title, cards) {
    if (!cards.length) {
        return "";
    }
    return `
        <div class="stat-section">
            <h3 class="stat-section-title">${escapeStat(title)}</h3>
            <div class="stat-grid">${cards.join("")}</div>
        </div>`;
}

function dayLink(exerciseDayID, text) {
    return `<a href="/exercises/${exerciseDayID}">${escapeStat(text)}</a>`;
}

// replaceChart draws a Chart.js chart on a canvas, destroying the one drawn there before.
function replaceChart(canvasID, config) {
    if (statsState.charts[canvasID]) {
        statsState.charts[canvasID].destroy();
    }
    config.options = config.options || {};
    config.options.maintainAspectRatio = false;
    config.options.responsive = true;
    config.options.legend = config.options.legend || { display: false };
    statsState.charts[canvasID] = new Chart(document.getElementById(canvasID), config);
}

// chartAxes is the shared light axis styling: muted ticks, hairline grid.
function chartAxes(yTicks, xExtra) {
    var tick = cssToken("--mediumblue");
    var grid = cssToken("--grey");
    var x = Object.assign({ gridLines: { display: false }, ticks: { fontColor: tick, autoSkip: true, maxRotation: 0 } }, xExtra || {});
    return {
        xAxes: [x],
        yAxes: [{ gridLines: { color: grid, zeroLineColor: grid }, ticks: Object.assign({ fontColor: tick, beginAtZero: true }, yTicks || {}) }]
    };
}

// ── Overview ─────────────────────────────────────────────────────────────────

function chooseOverviewPeriod(key) {
    statsState.overviewPeriod = key;
    placeSegments("overview-periods", statsPeriods, key, "chooseOverviewPeriod");
    loadOverview();
}

function chooseOverviewMetric(key) {
    statsState.overviewMetric = key;
    placeSegments("overview-metrics", overviewMetrics, key, "chooseOverviewMetric");
    placeOverviewChart();
}

function loadOverview() {
    document.getElementById("overview-records-link").hidden = isAllTimePeriod(statsState.overviewPeriod);
    statsGet("auth/statistics/overview?period=" + statsState.overviewPeriod, function(result) {
        statsState.overview = result.overview;
        placeOverview();
    });
}

// deltaLine compares a period's figure with the previous period's, in words. Counts take
// "fewer", amounts (time, distance) take "less".
function deltaLine(current, previous, format, isAmount) {
    if (previous == null) {
        return "";
    }
    var period = statsPeriods.find(function(p) { return p.key === statsState.overviewPeriod; });
    var difference = current - previous;
    if (Math.abs(difference) < 0.05) {
        return "Same as " + period.previous;
    }
    var amount = format ? format(Math.abs(difference)) : Math.abs(difference);
    var direction = difference > 0 ? " more" : (isAmount ? " less" : " fewer");
    return amount + direction + " than " + period.previous;
}

function mainDistance(distances) {
    return distances && distances.length ? distances[0] : null;
}

function placeOverview() {
    var overview = statsState.overview;
    var totals = overview.totals;
    var previous = overview.previous;

    var tiles = [];
    tiles.push(statCard("Workouts", totals.workouts, "", "neutral", deltaLine(totals.workouts, previous ? previous.workouts : null)));
    tiles.push(statCard("Active time", formatHours(totals.duration_seconds), "", "time",
        deltaLine(totals.duration_seconds, previous ? previous.duration_seconds : null, formatHours, true)));
    tiles.push(statCard("Active days", totals.active_days, "", "neutral", deltaLine(totals.active_days, previous ? previous.active_days : null)));
    var distance = mainDistance(totals.distances);
    if (distance) {
        var previousDistance = previous ? (previous.distances.find(function(d) { return d.unit === distance.unit; }) || { amount: 0 }).amount : null;
        var other = totals.distances.slice(1).map(function(d) { return formatNumber(d.amount, 1) + " " + unitLabel(d.unit); }).join(", ");
        var detail = other ? "Also " + other : deltaLine(distance.amount, previousDistance, function(v) { return formatNumber(v, 1) + " " + unitLabel(distance.unit); }, true);
        tiles.push(statCard("Distance", formatNumber(distance.amount, 1), unitLabel(distance.unit), "movement", detail));
    }
    document.getElementById("overview-totals").innerHTML = tiles.join("");

    placeOverviewChart();

    var streaks = overview.streaks;
    document.getElementById("overview-streaks").innerHTML =
        statCard("Weeks in a row", streaks.week_current, "", "time", "Best ever: " + streaks.week_best)
        + statCard("Days in a row", streaks.day_current, "", "time", "Best ever: " + streaks.day_best);

    var mix = overview.mix;
    var longest = mix.length ? Math.max.apply(null, mix.map(function(m) { return m.duration_seconds; })) : 0;
    document.getElementById("overview-mix").innerHTML = mix.length ? mix.slice(0, 8).map(function(item) {
        var pct = longest > 0 ? Math.max(2, Math.round(item.duration_seconds / longest * 100)) : 0;
        return `
            <li class="stats-mix-row">
                <button type="button" class="stats-mix-name" onclick="openStatisticsType('${item.action_id}')">${escapeStat(item.name)}</button>
                <span class="stats-mix-bar" aria-hidden="true"><span class="stats-mix-fill" style="width: ${pct}%"></span></span>
                <span class="stats-mix-figure">${formatHours(item.duration_seconds)}<span class="stats-mix-count">${item.activities} ${plural(item.activities, "time", "times")}</span></span>
            </li>`;
    }).join("") : `<li class="stats-note">Nothing logged in this period.</li>`;

    var records = overview.recent_records;
    document.getElementById("overview-records").innerHTML = records.length
        ? `<div class="records-grid">${records.map(function(r) { return recordTileHTML(r.record, r.action_id, r.action_name); }).join("")}</div>`
        : `<p class="stats-note">No new records in this period. Keep going.</p>`;
}

function placeOverviewChart() {
    var overview = statsState.overview;
    if (!overview) {
        return;
    }
    var metric = statsState.overviewMetric;
    var monthly = overview.bucket === "month";
    var unit = mainDistance(overview.totals.distances);

    var labels = overview.buckets.map(function(bucket) {
        var date = new Date(bucket.start);
        return monthly
            ? date.toLocaleDateString("en-GB", { month: "short", year: "2-digit" })
            : date.toLocaleDateString("en-GB", { day: "numeric", month: "short" });
    });
    var values = overview.buckets.map(function(bucket) {
        if (metric === "time") {
            return Math.round(bucket.duration_seconds / 360) / 10;
        }
        if (metric === "distance") {
            var match = unit ? bucket.distances.find(function(d) { return d.unit === unit.unit; }) : null;
            return match ? match.amount : 0;
        }
        return bucket.workouts;
    });

    var per = monthly ? "month" : "week";
    var titles = { workouts: "Workouts per " + per, time: "Hours per " + per, distance: (unit ? unitLabel(unit.unit) : "Distance") + " per " + per };
    document.getElementById("overview-trend-title").textContent = titles[metric];

    replaceChart("overview-chart", {
        type: "bar",
        data: { labels: labels, datasets: [{ data: values, backgroundColor: cssToken("--module-accent"), hoverBackgroundColor: cssToken("--mediumblue"), maxBarThickness: 36 }] },
        options: { scales: chartAxes({ precision: metric === "workouts" ? 0 : undefined }) }
    });
}

// ── Records ──────────────────────────────────────────────────────────────────

var recordCache = {};

function loadRecords() {
    statsGet("auth/records", function(result) {
        placeRecords(result.records);
    });
}

function effortDistance(key) {
    return key.indexOf("effort_") === 0 ? parseInt(key.substring(7), 10) : 0;
}

// recordValueParts formats a record's value for its kind: the headline, its unit and a
// quiet detail (pace for a best effort, the set behind a weight).
function recordValueParts(record, entry) {
    var value = entry.value;
    switch (record.kind) {
        case "time":
            // A best effort: foot sports read as pace, everything else as speed.
            var meters = effortDistance(record.key);
            var detail = "";
            if (meters > 0) {
                detail = footSports.indexOf(record.sport) !== -1
                    ? formatClock(value / (meters / 1000)) + " /km"
                    : formatNumber(meters / value * 3.6, 1) + " km/h";
            }
            return { value: formatClock(value), unit: "", detail: detail };
        case "distance":
            return { value: formatNumber(value, value >= 100 ? 0 : 1), unit: unitLabel(entry.unit), detail: "" };
        case "duration":
            return { value: formatClock(value), unit: "", detail: "" };
        case "elevation":
            return { value: formatNumber(value, 0), unit: "m", detail: "" };
        case "weight":
            var setDetail = entry.reps ? formatNumber(entry.reps, 0) + " " + plural(entry.reps, "rep", "reps") : "";
            if (record.key === "estimated_one_rep_max") {
                setDetail = entry.reps ? "Epley, from a set of " + formatNumber(entry.reps, 0) : "";
            }
            if (record.key === "most_volume") {
                setDetail = "Weight × reps, all sets";
            }
            return { value: formatNumber(value, value % 1 === 0 ? 0 : 1), unit: entry.unit, detail: setDetail };
        case "reps":
            return { value: formatNumber(value, 0), unit: "reps", detail: "" };
    }
    return { value: String(value), unit: entry.unit || "", detail: "" };
}

function recordValueText(record, entry) {
    var parts = recordValueParts(record, entry);
    return parts.value + (parts.unit ? " " + parts.unit : "");
}

// recordTileHTML is one record: the label, the value, when it was set and what it beat. The
// whole tile opens the record's (all-time) history. actionName is shown when tiles of several
// types mix. inPeriod renders a period's best instead: tagged when it is also the all-time
// record, otherwise showing that record to compare with. With allTime every tile is the
// all-time record, so the badge and the comparison line are left out.
function recordTileHTML(record, actionID, actionName, inPeriod, allTime) {
    recordCache[actionID + ":" + record.key] = record;
    var parts = recordValueParts(record, record.current);
    var fresh, previous;
    if (inPeriod) {
        fresh = record.all_time_record && !allTime ? `<span class="meta-tag meta-tag--success">All-time record</span>` : "";
        previous = record.all_time_best && !allTime
            ? "Record " + escapeStat(recordValueText(record, record.all_time_best)) + " from " + formatDay(record.all_time_best.set_at)
            : "";
    } else {
        fresh = record.fresh ? `<span class="meta-tag meta-tag--success">New</span>` : "";
        previous = record.previous
            ? "Beat " + escapeStat(recordValueText(record, record.previous)) + " from " + formatDay(record.previous.set_at)
            : "First time logged";
    }
    var type = actionName ? `<span class="record-type">${escapeStat(actionName)}</span>` : "";
    var label = (actionName ? actionName + ", " : "") + record.label + ": " + recordValueText(record, record.current);

    return `
        <button type="button" class="record-tile" onclick="openRecordHistory('${actionID}', '${record.key}')" aria-label="${escapeStat(label)}. Show history">
            ${type}
            <span class="record-label">${escapeStat(record.label)}${fresh}</span>
            <span class="record-value">${escapeStat(parts.value)}${parts.unit ? `<span class="record-unit">${escapeStat(parts.unit)}</span>` : ""}</span>
            ${parts.detail ? `<span class="record-detail">${escapeStat(parts.detail)}</span>` : ""}
            <span class="record-when">${formatDay(record.current.set_at)}</span>
            ${previous ? `<span class="record-previous">${previous}</span>` : ""}
        </button>`;
}

function placeRecords(groups) {
    var list = document.getElementById("records-list");
    if (!groups.length) {
        list.innerHTML = `
            <div class="user-stats-empty">
                <div class="user-stats-empty-title">No records yet</div>
                <div class="user-stats-empty-hint">Records appear as you log workouts: your longest, heaviest and fastest, per activity type.</div>
            </div>`;
        return;
    }
    list.innerHTML = groups.map(function(group) {
        return `
            <section class="stat-panel records-group">
                <div class="stats-panel-head">
                    <h2 class="stats-heading">${escapeStat(group.action.name)}<span class="stats-heading-note">${group.activities} ${plural(group.activities, "time", "times")}</span></h2>
                    <div class="btn-group">
                        <button type="button" class="btn btn--ghost btn--sm" onclick="openStatisticsType('${group.action.id}')">Statistics</button>
                        <a class="btn btn--ghost btn--sm" href="/exercises?action=${group.action.id}">Workouts</a>
                    </div>
                </div>
                <div class="records-grid">${group.records.map(function(record) { return recordTileHTML(record, group.action.id); }).join("")}</div>
            </section>`;
    }).join("");
}

// openRecordHistory shows how a record improved: a step chart and the list of days it was
// beaten on, each linking to that day.
function openRecordHistory(actionID, key) {
    var record = recordCache[actionID + ":" + key];
    TRModal.open({ eyebrow: "Record history", title: record ? record.label : "Record", body: `<div class="gear-empty">Loading…</div>` });
    statsGet("auth/records/history?action_id=" + actionID + "&key=" + encodeURIComponent(key), function(result) {
        var history = result.history;
        var shape = { key: key, kind: result.kind, sport: record ? record.sport : "" };
        var rows = history.slice().reverse().map(function(entry, index) {
            var note = index === 0 ? "Current record" : "";
            return `
                <li class="record-history-row">
                    <span class="record-history-value">${escapeStat(recordValueText(shape, entry))}</span>
                    <span class="record-history-date">${dayLink(entry.exercise_day_id, formatDay(entry.set_at))}</span>
                    <span class="record-history-note">${note}</span>
                </li>`;
        }).join("");
        var chart = history.length > 1 ? `<div class="stats-chart stats-chart--modal"><canvas id="record-history-chart"></canvas></div>` : "";
        TRModal.setBody(`
            ${chart}
            <ol class="record-history">${rows}</ol>
        `);
        if (history.length > 1) {
            drawRecordHistory(shape, history);
        }
    });
}

function drawRecordHistory(shape, history) {
    var isTime = shape.kind === "time" || shape.kind === "duration";
    var points = history.map(function(entry) { return { t: new Date(entry.set_at), y: entry.value }; });
    // Carry the record forward to today so the last step has a length.
    points.push({ t: new Date(), y: history[history.length - 1].value });
    var accent = cssToken("--module-accent");

    replaceChart("record-history-chart", {
        type: "line",
        data: { datasets: [{ data: points, steppedLine: "before", fill: false, borderColor: accent, pointBackgroundColor: accent, pointRadius: points.map(function(_, i) { return i === points.length - 1 ? 0 : 4; }) }] },
        options: {
            scales: chartAxes(
                {
                    beginAtZero: false,
                    // Times read best at the top, like every other record.
                    reverse: shape.kind === "time",
                    callback: function(value) { return isTime ? formatClock(value) : formatNumber(value, value % 1 === 0 ? 0 : 1); }
                },
                { type: "time", time: { unit: "month", tooltipFormat: "D MMM YYYY", displayFormats: { month: "MMM YYYY" } } }
            ),
            tooltips: { callbacks: { label: function(item) { return recordValueText(shape, { value: item.yLabel, unit: history[0].unit }); } } }
        }
    });
}

// ── Activity types ───────────────────────────────────────────────────────────

// openStatisticsType jumps to the Activity types tab with one type chosen.
function openStatisticsType(actionID) {
    statsState.typesActionID = actionID;
    var select = document.getElementById("types-action");
    if (statsState.loaded.types && select) {
        select.value = actionID;
        selectStatisticsTab("types");
        loadTypeStatistics();
        return;
    }
    selectStatisticsTab("types");
}

function loadTypeOptions() {
    statsGet("auth/actions?experienced=true", function(result) {
        statsState.actions = result.actions || [];
        statsState.actions.sort(function(a, b) { return a.name.localeCompare(b.name); });
        var select = document.getElementById("types-action");
        statsState.actions.forEach(function(action) {
            var option = document.createElement("option");
            option.value = action.id;
            option.text = action.name;
            select.add(option);
        });
        if (statsState.typesActionID) {
            select.value = statsState.typesActionID;
            loadTypeStatistics();
        }
    });
}

function chooseStatisticsType(actionID) {
    statsState.typesActionID = actionID;
    writeStatisticsURL();
    loadTypeStatistics();
}

function chooseTypePeriod(key) {
    statsState.typesPeriod = key;
    placeSegments("types-periods", statsTypePeriods, key, "chooseTypePeriod");
    document.getElementById("types-custom").hidden = key !== "custom";
    loadTypeStatistics();
}

function placeDefaultDates() {
    var today = new Date();
    var start = new Date();
    start.setDate(today.getDate() - 30);
    document.getElementById("types-end").value = isoDate(today);
    document.getElementById("types-start").value = isoDate(start);
}

function isoDate(date) {
    return date.getFullYear() + "-" + padNumber(date.getMonth() + 1, 2) + "-" + padNumber(date.getDate(), 2);
}

// typePeriodRange turns the period choice into the start and end days to ask for.
function typePeriodRange() {
    var today = new Date();
    var start = new Date(today);
    switch (statsState.typesPeriod) {
        case "4w":
            start.setDate(today.getDate() - 27);
            break;
        case "12w":
            start.setDate(today.getDate() - 83);
            break;
        case "year":
            start = new Date(today.getFullYear(), 0, 1);
            break;
        case "all":
            start = new Date(2000, 0, 1);
            break;
        case "custom":
            return { start: document.getElementById("types-start").value, end: document.getElementById("types-end").value };
    }
    return { start: isoDate(start), end: isoDate(today) };
}

function loadTypeStatistics() {
    var actionID = statsState.typesActionID;
    var allTime = isAllTimePeriod(statsState.typesPeriod);
    document.getElementById("types-records-link").hidden = allTime;
    document.getElementById("types-empty").hidden = !!actionID;
    document.getElementById("types-content").hidden = !actionID;
    resetActivityHeatmap();
    if (!actionID) {
        return;
    }
    var range = typePeriodRange();
    if (!range.start || !range.end) {
        return;
    }
    document.getElementById("types-workouts-link").href = "/exercises?action=" + actionID;

    var query = "?start=" + range.start + "&end=" + range.end;
    statsGet("auth/actions/" + actionID + "/statistics" + query, function(result) {
        statsState.typesStatistics = result.statistics;
        placeTypeStatistics(result.statistics);
        if (result.statistics.totals.distances.length) {
            statsGet("auth/actions/" + actionID + "/heatmap" + query, function(heatmap) {
                renderActivityHeatmap(heatmap.tracks);
            });
        }
    });
    // The period's bests: the same records, limited to the chosen days.
    statsGet("auth/records?action_id=" + actionID + "&start=" + range.start + "&end=" + range.end, function(result) {
        var group = result.records[0];
        var panel = document.getElementById("types-records-panel");
        panel.hidden = !group || !group.records.length;
        if (group) {
            document.getElementById("types-records").innerHTML = group.records.map(function(record) { return recordTileHTML(record, actionID, null, true, allTime); }).join("");
        }
    });
}

// placeTypeStatistics renders one activity type and period: Totals, Averages and Sensors (only
// the measures this type has), the trend chart and the soundtrack. The period's bests are the
// record tiles above (loadTypeStatistics).
function placeTypeStatistics(stats) {
    var wrapper = document.getElementById("types-stats");
    if (stats.activities === 0) {
        wrapper.innerHTML = `<div class="stat-panel"><p class="stats-note">No ${escapeStat(stats.action.name)} in this period. Try a longer one.</p></div>`;
        document.getElementById("types-trend-panel").hidden = true;
        document.getElementById("types-soundtrack").innerHTML = "";
        return;
    }

    var totals = [];
    totals.push(statCard("Activities", stats.activities, "", "neutral", stats.workouts !== stats.activities ? "In " + stats.workouts + " workouts" : ""));
    stats.totals.distances.forEach(function(d) { totals.push(statCard("Distance", formatNumber(d.amount, 1), unitLabel(d.unit), "movement")); });
    if (stats.totals.duration_seconds) {
        var moving = stats.totals.moving_seconds && stats.totals.moving_seconds < stats.totals.duration_seconds ? formatHours(stats.totals.moving_seconds) + " moving" : "";
        totals.push(statCard("Time", formatHours(stats.totals.duration_seconds), "", "time", moving));
    }
    if (stats.totals.repetitions) { totals.push(statCard("Reps", formatNumber(stats.totals.repetitions, 0), "", "strength")); }
    stats.totals.volume.forEach(function(v) { totals.push(statCard("Volume", formatNumber(v.amount, 0), v.unit, "strength", "Weight × reps")); });

    var averages = [];
    var avg = stats.averages;
    if (avg.distance) { averages.push(statCard("Distance", formatNumber(avg.distance.amount, 2), unitLabel(avg.distance.unit), "movement")); }
    if (avg.duration_seconds) { averages.push(statCard("Time", formatClock(avg.duration_seconds), "", "time")); }
    if (avg.distance && avg.duration_seconds && stats.totals.distances.length === 1) {
        var paceSeconds = (stats.totals.moving_seconds || stats.totals.duration_seconds) / stats.totals.distances[0].amount;
        if (isFootType(stats.action)) {
            averages.push(statCard("Pace", formatClock(paceSeconds), "/" + unitLabel(avg.distance.unit), "movement"));
        } else {
            averages.push(statCard("Speed", formatNumber(3600 / paceSeconds, 1), unitLabel(avg.distance.unit) + "/h", "movement"));
        }
    }
    if (avg.repetitions) { averages.push(statCard("Reps", formatNumber(avg.repetitions, 0), "", "strength")); }
    if (avg.volume) { averages.push(statCard("Volume", formatNumber(avg.volume.amount, 0), avg.volume.unit, "strength")); }

    var sensors = [];
    var s = stats.sensors;
    if (s.avg_heartrate) { sensors.push(statCard("Avg heart rate", s.avg_heartrate, "bpm", "effort", s.max_heartrate ? "Max " + s.max_heartrate + " bpm" : "")); }
    if (s.avg_cadence) { sensors.push(statCard("Avg cadence", s.avg_cadence, s.cadence_unit, "movement")); }
    if (s.avg_watts) { sensors.push(statCard("Avg power", s.avg_watts, "W", "effort")); }
    if (s.elevation_gain_m) { sensors.push(statCard("Elevation gain", formatNumber(s.elevation_gain_m, 0), "m", "movement")); }

    wrapper.innerHTML = `<div class="stat-panel">`
        + statSection("Totals", totals)
        + statSection("Averages per activity", averages)
        + statSection("Sensors", sensors)
        + `</div>`;

    placeTypeTrendMetrics(stats);
    placeActivityMediaStatistics(stats.media);
}

var footSports = ["Run", "TrailRun", "VirtualRun", "Walk", "Hike"];

function isFootType(action) {
    return footSports.indexOf(action.strava_name) !== -1;
}

// trendMetricsFor lists the trend charts a type's data supports.
function trendMetricsFor(stats) {
    var trend = stats.trend;
    var has = function(field) { return trend.some(function(p) { return p[field] > 0; }); };
    var metrics = [];
    if (has("distance") && (has("moving_seconds") || has("duration_seconds"))) {
        metrics.push(isFootType(stats.action) ? { key: "pace", label: "Pace" } : { key: "speed", label: "Speed" });
    }
    if (has("distance")) { metrics.push({ key: "distance", label: "Distance" }); }
    if (has("estimated_one_rep_max")) { metrics.push({ key: "one_rep_max", label: "One-rep max" }); }
    if (has("top_weight")) { metrics.push({ key: "top_weight", label: "Top set" }); }
    if (has("duration_seconds")) { metrics.push({ key: "duration", label: "Duration" }); }
    if (trend.some(function(p) { return p.avg_heartrate; })) { metrics.push({ key: "heartrate", label: "Heart rate" }); }
    return metrics;
}

function placeTypeTrendMetrics(stats) {
    var metrics = trendMetricsFor(stats);
    var panel = document.getElementById("types-trend-panel");
    panel.hidden = stats.trend.length < 2 || metrics.length === 0;
    if (panel.hidden) {
        return;
    }
    if (!metrics.some(function(m) { return m.key === statsState.typesTrendMetric; })) {
        statsState.typesTrendMetric = metrics[0].key;
    }
    statsState.typesTrendMetrics = metrics;
    placeSegments("types-trend-metrics", metrics, statsState.typesTrendMetric, "chooseTypeTrendMetric");
    drawTypeTrend();
}

function chooseTypeTrendMetric(key) {
    statsState.typesTrendMetric = key;
    placeSegments("types-trend-metrics", statsState.typesTrendMetrics, key, "chooseTypeTrendMetric");
    drawTypeTrend();
}

// trendValue reads one activity's value for the chosen metric, or null when it has none.
function trendValue(point, metric, unit) {
    var seconds = point.moving_seconds || point.duration_seconds;
    switch (metric) {
        case "pace":
            return point.distance > 0 && seconds > 0 && point.distance_unit === unit ? seconds / point.distance : null;
        case "speed":
            return point.distance > 0 && seconds > 0 && point.distance_unit === unit ? Math.round(point.distance / seconds * 36000) / 10 : null;
        case "distance":
            return point.distance > 0 && point.distance_unit === unit ? point.distance : null;
        case "one_rep_max":
            return point.estimated_one_rep_max || null;
        case "top_weight":
            return point.top_weight || null;
        case "duration":
            return point.duration_seconds || null;
        case "heartrate":
            return point.avg_heartrate || null;
    }
    return null;
}

// rollingMean smooths a series over the last `window` points, skipping gaps.
function rollingMean(values, window) {
    return values.map(function(_, i) {
        var slice = values.slice(Math.max(0, i - window + 1), i + 1).filter(function(v) { return v != null; });
        return slice.length ? slice.reduce(function(a, b) { return a + b; }, 0) / slice.length : null;
    });
}

function drawTypeTrend() {
    var stats = statsState.typesStatistics;
    var metric = statsState.typesTrendMetric;
    var unit = stats.totals.distances.length ? stats.totals.distances[0].unit : "";
    var weightUnit = stats.trend.length ? stats.trend[0].weight_unit : "";
    var points = stats.trend.map(function(point) {
        return { t: new Date(point.date), y: trendValue(point, metric, unit) };
    }).filter(function(p) { return p.y != null; });
    var smooth = rollingMean(points.map(function(p) { return p.y; }), 5).map(function(y, i) { return { t: points[i].t, y: y }; });

    var isClock = metric === "pace" || metric === "duration";
    var labels = {
        pace: "Pace per " + unitLabel(unit),
        speed: "Speed, " + unitLabel(unit) + "/h",
        distance: "Distance, " + unitLabel(unit),
        one_rep_max: "Estimated one-rep max, " + weightUnit,
        top_weight: "Heaviest set, " + weightUnit,
        duration: "Duration",
        heartrate: "Average heart rate, bpm"
    };
    document.getElementById("types-trend-title").textContent = labels[metric];

    var accent = cssToken("--module-accent");
    var line = cssToken("--mediumblue");
    replaceChart("types-trend-chart", {
        type: "line",
        data: {
            datasets: [
                { data: points, showLine: false, fill: false, pointRadius: 3, pointBackgroundColor: accent, borderColor: accent },
                { data: smooth, fill: false, pointRadius: 0, borderWidth: 2, borderColor: line, lineTension: 0.3 }
            ]
        },
        options: {
            scales: chartAxes(
                {
                    beginAtZero: false,
                    // Faster pace is a lower number; flip it so better reads as up.
                    reverse: metric === "pace",
                    callback: function(value) { return isClock ? formatClock(value) : formatNumber(value, value % 1 === 0 ? 0 : 1); }
                },
                { type: "time", time: { tooltipFormat: "D MMM YYYY", minUnit: "day" } }
            ),
            tooltips: { filter: function(item) { return item.datasetIndex === 0; }, callbacks: { label: function(item) { return isClock ? formatClock(item.yLabel) : formatNumber(item.yLabel, 1); } } }
        }
    });
}

// ── Soundtrack ───────────────────────────────────────────────────────────────

// soundtrackSub joins an artist and a play count into a "Artist · N plays" subline.
function soundtrackSub(artist, count) {
    var parts = [];
    if (artist) { parts.push(artist); }
    if (count > 1) { parts.push(count + " plays"); }
    return parts.join(" · ");
}

// soundtrackTopRow renders a highlighted media item: cover art (or a typed glyph when the
// provider gave no artwork) beside a label / title / subline.
function soundtrackTopRow(label, title, sub, artwork, glyph) {
    var art = artwork
        ? `<div class="soundtrack-art" style="background-image:url('${escapeStat(artwork)}')"></div>`
        : `<div class="soundtrack-art soundtrack-art-glyph">${glyph}</div>`;
    var subHtml = sub ? `<div class="soundtrack-top-sub">${escapeStat(sub)}</div>` : "";
    return `
        <div class="soundtrack-top">
            ${art}
            <div class="soundtrack-top-body">
                <div class="soundtrack-top-label">${escapeStat(label)}</div>
                <div class="soundtrack-top-title">${escapeStat(title)}</div>
                ${subHtml}
            </div>
        </div>`;
}

// soundtrackColumn is one side of the soundtrack panel (Music or Spoken): its top rows
// and a row of pill-shaped totals. Returns "" when there is nothing to show.
function soundtrackColumn(title, rows, meta) {
    if (!rows && !meta.length) {
        return "";
    }
    var metaHtml = meta.length
        ? `<div class="soundtrack-meta">${meta.map(function(m) { return `<span>${escapeStat(m)}</span>`; }).join("")}</div>`
        : "";
    return `
        <div class="soundtrack-col">
            <div class="soundtrack-col-title">${escapeStat(title)}</div>
            ${rows}
            ${metaHtml}
        </div>`;
}

// placeActivityMediaStatistics renders the period's soundtrack as its own amber-marked panel,
// split into Music and Spoken columns so a run soundtracked by a podcast reads as richly as
// one soundtracked by songs. Empty when nothing was listened to (media is null).
function placeActivityMediaStatistics(media) {
    var wrapper = document.getElementById("types-soundtrack");
    wrapper.innerHTML = "";
    if (!media) {
        return;
    }

    var musicRows = "";
    if (media.top_track) {
        musicRows += soundtrackTopRow("Top song", media.top_track.title, soundtrackSub(media.top_track.artist, media.top_track.count), media.top_track.artwork, "♪");
    }
    if (media.top_artist) {
        musicRows += soundtrackTopRow("Top artist", media.top_artist.title, soundtrackSub("", media.top_artist.count), media.top_artist.artwork, "♫");
    }
    var musicMeta = [];
    if (media.songs > 0) { musicMeta.push(media.songs + (media.songs == 1 ? " song" : " songs")); }
    if (media.unique_artists > 0) { musicMeta.push(media.unique_artists + (media.unique_artists == 1 ? " artist" : " artists")); }
    if (media.listening_time > 0) { musicMeta.push(secondsToDurationString(media.listening_time)); }
    var musicCol = soundtrackColumn("Music", musicRows, musicMeta);

    var spokenRows = "";
    if (media.top_podcast) {
        var episodes = media.top_podcast.count;
        var episodesLabel = episodes > 0 ? (episodes + (episodes == 1 ? " episode" : " episodes")) : "";
        spokenRows += soundtrackTopRow("Top podcast", media.top_podcast.title, episodesLabel, media.top_podcast.artwork, "🎙");
    }
    if (media.top_audiobook) {
        spokenRows += soundtrackTopRow("Top audiobook", media.top_audiobook.title, media.top_audiobook.artist, media.top_audiobook.artwork, "📖");
    }
    var spokenMeta = [];
    if (media.podcast_episodes > 0) { spokenMeta.push(media.podcast_episodes + (media.podcast_episodes == 1 ? " episode" : " episodes")); }
    if (media.audiobooks > 0) { spokenMeta.push(media.audiobooks + (media.audiobooks == 1 ? " book" : " books")); }
    if (media.spoken_time > 0) { spokenMeta.push(secondsToDurationString(media.spoken_time)); }
    var spokenCol = soundtrackColumn("Spoken", spokenRows, spokenMeta);

    if (!musicCol && !spokenCol) {
        return;
    }

    wrapper.innerHTML = `
        <div class="soundtrack-panel">
            <div class="soundtrack-head"><span class="soundtrack-mark">♫</span>Soundtrack</div>
            <div class="soundtrack-cols">${musicCol}${spokenCol}</div>
        </div>`;
}

// ── GPS heatmap ──────────────────────────────────────────────────────────────

var activityHeatmapInstance = null;
var activityHeatLayer = null;

// resetActivityHeatmap hides the heatmap and drops the current heat layer. Called when
// the activity/date filter changes so stale points are not shown while reloading.
function resetActivityHeatmap() {
    var wrapper = document.getElementById("activity-heatmap-wrapper");
    if (wrapper) {
        wrapper.hidden = true;
    }
    if (activityHeatmapInstance && activityHeatLayer) {
        activityHeatmapInstance.removeLayer(activityHeatLayer);
        activityHeatLayer = null;
    }
}

// HEATMAP_GRID is the cell size (degrees, ~67 m) used to count how many distinct
// activities pass through an area. Coarse enough that two runs of the same route fall
// in the same cell, fine enough to keep routes distinct.
var HEATMAP_GRID = 0.0006;
// HEATMAP_BUCKETS quantizes visit frequency into colour bands, which also lets adjacent
// same-frequency segments merge into one polyline (fewer layers to render).
var HEATMAP_BUCKETS = 8;

// renderActivityHeatmap draws the period's GPS tracks (GET /actions/:id/heatmap, which
// inherits the activity-type and date filters) as route polylines tinted by visit frequency.
// Drawing lines (not a point-density blob) keeps routes continuous, and colouring by how many
// distinct activities pass through each cell means a single run stays cool while genuinely
// frequented routes warm up — overlapping translucent lines reinforce that.
function renderActivityHeatmap(tracks) {

    var wrapper = document.getElementById("activity-heatmap-wrapper");
    if (!wrapper) {
        return;
    }

    var model = buildHeatmapModel(tracks);

    // No GPS data for this activity/period (or the map library is unavailable): keep the
    // section hidden.
    if (!model.tracks.length || typeof L === "undefined") {
        if (typeof L === "undefined" && model.tracks.length) {
            console.log("Leaflet is not loaded.");
        }
        wrapper.hidden = true;
        return;
    }

    wrapper.hidden = false;

    if (!activityHeatmapInstance) {
        activityHeatmapInstance = L.map("activity-heatmap-canvas");
        L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            maxZoom: 19,
            attribution: "&copy; OpenStreetMap contributors"
        }).addTo(activityHeatmapInstance);
    }

    if (activityHeatLayer) {
        activityHeatmapInstance.removeLayer(activityHeatLayer);
    }

    // A shared canvas renderer keeps thousands of segments performant, and translucent
    // strokes composite where routes overlap, adding to the frequency tint. Each segment
    // gets a wide, soft glow underlay plus a crisp line on top; collecting them into two
    // groups (glow first, lines second) keeps every glow beneath every line so the halos
    // never paint over a neighbouring route's crisp stroke.
    var renderer = L.canvas({ padding: 0.5 });
    var glowLayers = [];
    var lineLayers = [];
    for (var t = 0; t < model.tracks.length; t++) {
        drawTrackSegments(model.tracks[t], model.counts, model.maxCount, renderer, glowLayers, lineLayers);
    }
    activityHeatLayer = L.layerGroup(glowLayers.concat(lineLayers)).addTo(activityHeatmapInstance);

    // Open on the densest cluster (usually the most-frequented area) rather than fitting
    // all points, which would zoom out to fit far-away one-off activities.
    var center = densestCenter(model.allPoints);
    activityHeatmapInstance.setView(center, 13);

    // The container starts hidden, so Leaflet may have measured a zero size; recompute
    // once it is visible.
    setTimeout(function() {
        activityHeatmapInstance.invalidateSize();
        activityHeatmapInstance.setView(center, 13);
    }, 200);

}

// heatmapCellKey rounds a coordinate to its grid cell so points from different
// activities that pass through the same place share a key.
function heatmapCellKey(lat, lng) {
    return Math.round(lat / HEATMAP_GRID) + ":" + Math.round(lng / HEATMAP_GRID);
}

// buildHeatmapModel turns the GPS tracks (one [lat, lng] series per activity) into thinned
// tracks plus a grid of visit frequencies. Tracks are thinned by an adaptive stride (chosen
// so the total sample count stays under a cap) to keep rendering light. Frequency counts
// DISTINCT activities per cell — one activity passing through a cell many times still counts
// once — so the tint reflects how often a place is revisited, not how densely the GPS sampled
// a single track. Returns { tracks: [[lat,lng]...], counts, maxCount, allPoints }.
function buildHeatmapModel(rawTracks) {

    var model = { tracks: [], counts: {}, maxCount: 0, allPoints: [] };
    if (!rawTracks) {
        return model;
    }

    var total = 0;
    for (var a = 0; a < rawTracks.length; a++) {
        total += rawTracks[a].length;
    }
    var cap = 60000;
    var stride = Math.max(3, Math.ceil(total / cap));

    for (var i = 0; i < rawTracks.length; i++) {
        // Cells this single activity touches, deduplicated so it contributes at most once
        // per cell to the frequency count.
        var cellsThisActivity = {};
        var data = rawTracks[i];
        var track = [];

        for (var k = 0; k < data.length; k += stride) {
            var coordinate = data[k];
            if (!coordinate || coordinate.length < 2) {
                continue;
            }
            var pt = [coordinate[0], coordinate[1]];
            track.push(pt);
            model.allPoints.push(pt);
            cellsThisActivity[heatmapCellKey(pt[0], pt[1])] = true;
        }

        // Keep the final sample so the drawn line reaches the real end of the track.
        var last = data[data.length - 1];
        if (last && last.length >= 2 && (data.length - 1) % stride !== 0) {
            var endPt = [last[0], last[1]];
            track.push(endPt);
            model.allPoints.push(endPt);
            cellsThisActivity[heatmapCellKey(endPt[0], endPt[1])] = true;
        }

        if (track.length >= 2) {
            model.tracks.push(track);
        }

        for (var key in cellsThisActivity) {
            model.counts[key] = (model.counts[key] || 0) + 1;
            if (model.counts[key] > model.maxCount) {
                model.maxCount = model.counts[key];
            }
        }
    }

    return model;
}

// heatmapColor maps a 0..1 frequency fraction to a colour, sweeping the hue from blue
// (rare) through green/yellow to red (most frequented). High saturation and a slightly
// deeper lightness give the crisp line punchy contrast.
function heatmapColor(fraction) {
    var hue = (1 - fraction) * 220;
    return "hsl(" + hue + ", 100%, 48%)";
}

// heatmapGlowColor is the same hue as the crisp line but lighter, used for the soft
// underlay that gives each route a coloured halo.
function heatmapGlowColor(fraction) {
    var hue = (1 - fraction) * 220;
    return "hsl(" + hue + ", 100%, 62%)";
}

// heatmapBucket converts a cell's visit count into a 0..(HEATMAP_BUCKETS-1) band. With a
// single activity (maxCount <= 1) everything lands in band 0, so a lone run stays cool.
function heatmapBucket(count, maxCount) {
    var fraction = maxCount > 1 ? (count - 1) / (maxCount - 1) : 0;
    return Math.round(fraction * (HEATMAP_BUCKETS - 1));
}

// drawTrackSegments splits one track into runs of consecutive points that share a
// frequency band and pushes each run as a single coloured polyline, so the line's colour
// follows how often each stretch is revisited while keeping the layer count down.
function drawTrackSegments(track, counts, maxCount, renderer, glowLayers, lineLayers) {
    if (track.length < 2) {
        return;
    }

    var bucketAt = function(pt) {
        return heatmapBucket(counts[heatmapCellKey(pt[0], pt[1])] || 1, maxCount);
    };

    var runStart = 0;
    var runBucket = bucketAt(track[0]);

    for (var i = 1; i < track.length; i++) {
        var bucket = bucketAt(track[i]);
        if (bucket !== runBucket) {
            // Include point i as the shared vertex so segments join without gaps.
            pushHeatmapSegment(track.slice(runStart, i + 1), runBucket, renderer, glowLayers, lineLayers);
            runStart = i;
            runBucket = bucket;
        }
    }
    pushHeatmapSegment(track.slice(runStart), runBucket, renderer, glowLayers, lineLayers);
}

function pushHeatmapSegment(points, bucket, renderer, glowLayers, lineLayers) {
    if (points.length < 2) {
        return;
    }
    var fraction = bucket / (HEATMAP_BUCKETS - 1);

    // Soft, wide underlay for the halo.
    glowLayers.push(L.polyline(points, {
        renderer: renderer,
        color: heatmapGlowColor(fraction),
        weight: 9,
        opacity: 0.2,
        lineCap: "round",
        lineJoin: "round"
    }));

    // Crisp line on top.
    lineLayers.push(L.polyline(points, {
        renderer: renderer,
        color: heatmapColor(fraction),
        weight: 4,
        opacity: 0.8,
        lineCap: "round",
        lineJoin: "round"
    }));
}

// densestCenter buckets points into coarse (~1 km) cells, finds the cell with the most
// samples, and returns the average coordinate of that cell — a cheap "biggest cluster".
function densestCenter(points) {

    var cells = {};
    var precision = 100; // round to 2 decimals -> ~1.1 km cells
    var bestKey = null;
    var bestCount = 0;

    for (var i = 0; i < points.length; i++) {
        var latRounded = Math.round(points[i][0] * precision) / precision;
        var lngRounded = Math.round(points[i][1] * precision) / precision;
        var key = latRounded + "," + lngRounded;

        var cell = cells[key];
        if (!cell) {
            cell = { count: 0, latSum: 0, lngSum: 0 };
            cells[key] = cell;
        }
        cell.count++;
        cell.latSum += points[i][0];
        cell.lngSum += points[i][1];

        if (cell.count > bestCount) {
            bestCount = cell.count;
            bestKey = key;
        }
    }

    if (!bestKey) {
        return [points[0][0], points[0][1]];
    }

    var best = cells[bestKey];
    return [best.latSum / best.count, best.lngSum / best.count];
}

// ── Seasons ──────────────────────────────────────────────────────────────────

function getSeasons() {
    statsGet("auth/seasons", function(result) {
        placeSeasonsInput(result.seasons);
    });
}

// placeSeasonsInput lists the seasons the user joined that have started, newest first, and
// opens the one running now (else the latest).
function placeSeasonsInput(seasonsArray) {
    var now = new Date();
    var seasons = seasonsArray.filter(function(season) {
        var joined = season.goals.some(function(goal) { return goal.user.id == user_id; });
        return joined && new Date(season.start) < now;
    });
    seasons.sort(function(a, b) { return new Date(b.start) - new Date(a.start); });
    statsState.seasons = seasons;

    if (!seasons.length) {
        document.getElementById("season-empty").hidden = false;
        document.getElementById("select_season").disabled = true;
        return;
    }

    var select = document.getElementById("select_season");
    seasons.forEach(function(season) {
        var option = document.createElement("option");
        option.text = season.name;
        option.value = season.id;
        select.add(option);
    });
    var running = seasons.find(function(season) { return new Date(season.start) <= now && now <= new Date(season.end); });
    select.value = (running || seasons[0]).id;
    chooseSeason();
}

function chooseSeason() {
    var seasonID = document.getElementById("select_season").value;
    if (!seasonID) {
        return;
    }
    statsGet("auth/seasons/" + seasonID + "/weeks-personal", function(result) {
        placeSeasonStatistics(result.leaderboard, result.weekdays, result.wheel_statistics);
    });
}

// placeSeasonStatistics renders one season: the figures as tiles, the weekly chart against
// the goal (sick-leave weeks in grey) and the weekday chart.
function placeSeasonStatistics(weeks, weekdays, wheelStatistics) {
    document.getElementById("season-content").hidden = false;
    weeks = weeks.slice().reverse();

    var labels = [], workouts = [], goals = [], pointColours = [];
    var longestStreak = 0, bestWeek = 0, total = 0, sickWeeks = 0, goal = 0, weeksAtGoal = 0;
    var accent = cssToken("--module-accent");
    var resting = cssToken("--grey");

    weeks.forEach(function(week) {
        var user = week.user;
        labels.push("Week " + week.week_number);
        workouts.push(user.week_completion_interval);
        goals.push(user.exercise_goal);
        pointColours.push(user.sick_leave ? resting : accent);
        goal = user.exercise_goal;
        total += user.week_completion_interval;
        longestStreak = Math.max(longestStreak, user.current_streak);
        bestWeek = Math.max(bestWeek, user.week_completion_interval);
        if (user.sick_leave) { sickWeeks++; }
        if (user.week_completion_interval >= user.exercise_goal) { weeksAtGoal++; }
    });

    var tiles = [];
    tiles.push(statCard("Weekly goal", goal, plural(goal, "workout", "workouts"), "neutral"));
    tiles.push(statCard("Weeks at goal", weeks.length ? Math.floor(weeksAtGoal / weeks.length * 100) : 0, "%", "neutral", weeksAtGoal + " of " + weeks.length + " " + plural(weeks.length, "week", "weeks")));
    tiles.push(statCard("Longest streak", longestStreak, plural(longestStreak, "week", "weeks"), "time"));
    tiles.push(statCard("Best week", bestWeek, plural(bestWeek, "workout", "workouts"), "neutral"));
    tiles.push(statCard("Workouts", total, "", "neutral"));
    if (sickWeeks) { tiles.push(statCard("Sick leave", sickWeeks, plural(sickWeeks, "week", "weeks"), "time")); }
    tiles.push(statCard("Wheels spun", wheelStatistics.wheel_spins, "", "neutral"));
    tiles.push(statCard("Wheels won", wheelStatistics.wheels_won, "", "neutral"));
    document.getElementById("season-tiles").innerHTML = tiles.join("");

    replaceChart("season-weeks-chart", {
        type: "line",
        data: {
            labels: labels,
            datasets: [
                { data: workouts, fill: false, lineTension: 0, borderColor: accent, pointBackgroundColor: pointColours, pointBorderColor: pointColours, pointRadius: 4 },
                { data: goals, fill: false, lineTension: 0, borderColor: cssToken("--mediumblue"), borderDash: [6, 4], borderWidth: 1.5, pointRadius: 0 }
            ]
        },
        options: { scales: chartAxes({ precision: 0 }) }
    });

    replaceChart("season-weekday-chart", {
        type: "bar",
        data: {
            labels: ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"],
            datasets: [{ data: [weekdays.monday, weekdays.tuesday, weekdays.wednesday, weekdays.thursday, weekdays.friday, weekdays.saturday, weekdays.sunday], backgroundColor: accent, maxBarThickness: 36 }]
        },
        options: { scales: chartAxes({ precision: 0 }) }
    });
}

// ── Body ─────────────────────────────────────────────────────────────────────

// getWeights loads the weight log: path=true opens the log modal, otherwise the Body tab
// is drawn.
function getWeights(path) {
    statsGet("auth/weights", function(result) {
        clearResponse();
        if (!path) {
            placeWeights(result.weights);
        } else {
            viewWeight(result.weights);
        }
    });
}

// weightChange compares the latest weight with the last one logged at least `days` before.
function weightChange(sorted, days) {
    var latest = sorted[sorted.length - 1];
    var cutoff = new Date(latest.date);
    cutoff.setDate(cutoff.getDate() - days);
    var earlier = null;
    for (var i = sorted.length - 1; i >= 0; i--) {
        if (new Date(sorted[i].date) <= cutoff) {
            earlier = sorted[i];
            break;
        }
    }
    if (!earlier) {
        return null;
    }
    return Math.round((latest.weight - earlier.weight) * 10) / 10;
}

function changeCard(label, change) {
    if (change == null) {
        return statCard(label, "–", "", "neutral", "Not enough history yet");
    }
    var sign = change > 0 ? "+" : change < 0 ? "−" : "±";
    return statCard(label, sign + formatNumber(Math.abs(change), 1), "kg", "neutral");
}

function placeWeights(weightsArray) {
    var empty = !weightsArray || weightsArray.length === 0;
    document.getElementById("body-empty").hidden = !empty;
    document.getElementById("body-content").hidden = empty;
    if (empty) {
        return;
    }

    var sorted = weightsArray.slice().sort(function(a, b) { return new Date(a.date) - new Date(b.date); });
    var latest = sorted[sorted.length - 1];
    document.getElementById("body-tiles").innerHTML =
        statCard("Latest", formatNumber(latest.weight, 1), "kg", "neutral", formatDay(latest.date))
        + changeCard("Last 30 days", weightChange(sorted, 30))
        + changeCard("Last 90 days", weightChange(sorted, 90));

    var accent = cssToken("--module-accent");
    var points = sorted.map(function(w) { return { t: new Date(w.date), y: w.weight }; });
    replaceChart("body-weight-chart", {
        type: "line",
        data: { datasets: [{ data: points, fill: false, lineTension: 0, borderColor: accent, pointBackgroundColor: accent, pointRadius: 3 }] },
        options: {
            scales: chartAxes({ beginAtZero: false, precision: 0 }, { type: "time", distribution: "linear", time: { tooltipFormat: "D MMM YYYY", minUnit: "day" } }),
            tooltips: { callbacks: { label: function(item) { return formatNumber(item.yLabel, 1) + " kg"; } } }
        }
    });
}

function viewWeight(weights) {
    var now = new Date();
    var weightsHTML = ``;

    for (let index = 0; index < weights.length; index++) {
        const weight = weights[index];
        const timeString = GetDateString(new Date(weight.date), false)
        weightsHTML += `
            <div class="weight-value">
                <div class="weight-date u-fs-sm">${timeString}</div>
                <div class="weight-amount">${weight.weight} kg</div>
                <div class="weight-actions">
                    <button type="button" class="btn btn--icon" title="Delete weight" aria-label="Delete weight" onclick="deleteWeight('${weight.id}');">
                        <img src="/assets/trash-2.svg" alt="">
                    </button>
                </div>
            </div>
        `;
    }

    var htmlContent = `
        <div class="trm-row">
            <div class="trm-field">
                <label class="trm-label" for="weightValue">Weight (kg)</label>
                <input type="number" name="weightValue" id="weightValue" autocomplete="off" min="0" max="500" step="0.1" value="" />
            </div>
            <div class="trm-field">
                <label class="trm-label" for="weightTime">Date</label>
                <input type="date" name="weightTime" id="weightTime" autocomplete="off" value="${isoDate(now)}" />
            </div>
        </div>
        <button class="btn btn--primary btn--block" id="register-button" type="submit" onclick="addWeight()">Save weight</button>
        <hr>
        <div class="weight-values-wrapper">
            ${weightsHTML}
        </div>
    `;

    TRModal.open({ eyebrow: "Body", title: "Log weight", body: htmlContent });
}

function deleteWeight(weightID) {
    if (!confirm("Delete this weight?")) {
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
                toggleModal();
                error(result.error);
            } else {
                getWeights(true);
                getWeights(false);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/weights/" + weightID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function addWeight() {
    var weightValueString = document.getElementById("weightValue").value;
    var weightTimeString = document.getElementById("weightTime").value;
    var weightValue = parseFloat(weightValueString);

    if (isNaN(weightValue) || weightValue <= 0) {
        alert("Enter a weight above zero.");
        return;
    }

    try {
        weightTimeString = new Date(weightTimeString).toISOString()
    } catch (error) {
        console.log(error)
    }

    var form_obj = {
        "weight" : weightValue,
        "date" : weightTimeString
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
                toggleModal();
                error(result.error);
            } else {
                getWeights(true);
                getWeights(false);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/weights");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}


// Page bootstrap (formerly an inline <script> in web/html/statistics.html).
var groups;
var jwt;
var userList = {};

document.addEventListener('DOMContentLoaded', function() {
    info('Loading...');
    showLoggedOutMenu();
    jwt = get_cookie('treningheten');
    get_login(jwt);
});
