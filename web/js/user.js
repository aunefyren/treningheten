function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            var sunday_alert = login_data.data.sunday_alert
            user_id = login_data.data.id
            admin = login_data.data.admin
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            var sunday_alert = false
            user_id = 0
            admin = false
        }

        showAdminMenu(admin)

    } else {
        var email = ""
        var first_name = ""
        var last_name = ""
        var admin = false;
        var sunday_reminder = false
        user_id = 0
    }

    try {
        string_index = document.URL.lastIndexOf('/');
        requested_user_id = document.URL.substring(string_index+1);
    }
    catch {
        requested_user_id = 0
    }

    console.log("Wanted user: " + requested_user_id)

    var html = `

        <div class="module">
            <div class="user-profile-row">
            <div class="account-section account-section--profile" id="account-section-details">
                <div class="user-active-profile-photo">
                    <img class="user-active-profile-photo-img u-fill" id="user-active-profile-photo-img" src="/assets/images/barbell.gif">
                </div>

                <p id="user_name" class="user-name"></p>
                <p id="join_date"></p>
                <p id="user_admin"></p>

                <div class="user-links" id="user-links" style="display:none;">
                </div>
            </div>

            <div id="account-section-stats" class="account-section account-section--stats">
                <div id="user-stats-skeleton" class="user-stats-skeleton">
                    <!-- Streaks label + two cards -->
                    <div class="skeleton-block skel-label"></div>
                    <div class="skel-row">
                        <div class="skeleton-block skel-streak"></div>
                        <div class="skeleton-block skel-streak"></div>
                    </div>
                    <!-- Sessions label + three cards -->
                    <div class="skeleton-block skel-label skel-mt"></div>
                    <div class="skel-row">
                        <div class="skeleton-block skel-tile"></div>
                        <div class="skeleton-block skel-tile"></div>
                        <div class="skeleton-block skel-tile"></div>
                    </div>
                    <!-- Activity label + tab bar + stat cards -->
                    <div class="skeleton-block skel-label skel-mt"></div>
                    <div class="skeleton-block skel-bar"></div>
                    <div class="skel-row">
                        <div class="skeleton-block skel-tile"></div>
                        <div class="skeleton-block skel-tile"></div>
                        <div class="skeleton-block skel-tile"></div>
                    </div>
                </div>
                <div id="user-stats-content" class="user-stats-content" style="display:none;"></div>
            </div>
            </div><!-- /.user-profile-row -->
        </div>

        <div class="module" id="achievements-module" style="display: none;">

            <div id="achievements-title" class="title u-mb-1">
                Achievements:
            </div>

            <div id="achievements-box" class="achievements-box">
            </div>

        </div>

    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'A real person.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        GetUserData(requested_user_id);
        GetProfileImage(requested_user_id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function GetProfileImage(userID) {

    var img = document.getElementById("user-active-profile-photo-img");
    if (!img) {
        return;
    }
    img.onerror = function() { this.onerror = null; this.src = '/assets/images/barbell.gif'; };
    // No cache-buster needed: the API serves the default placeholder with `no-store`, so a
    // "no photo yet" response can't stick and mask a later upload, while real photos still cache.
    img.src = profileImageURL(userID, false);

}

function GetUserData(userID) {

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

                PlaceUserData(result.user)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceUserData(user_object) {

    document.getElementById("user_name").innerHTML = user_object.first_name + " " + user_object.last_name

    // parse date object
    try {
        var date = new Date(Date.parse(user_object.created_at));
        var date_string = GetDateString(date)
    } catch {
        var date_string = "Error"
    }

    document.getElementById("join_date").innerHTML = "Joined: " + date_string

    if(user_object.admin) {
        var admin_string = "Yes"
    } else {
        var admin_string = "No"
    }

    if(user_object.strava_id && user_object.strava_public) {
        userLinks = document.getElementById("user-links")

        userLinks.style.display = "flex"

        userLinks.innerHTML += `
            <div onclick="window.open('https://www.strava.com/athletes/${user_object.strava_id}', '_blank');" class="user-link clickable" title="Strava profile">
                <img src="/assets/strava-logo.svg">
            </div>
        `;
    }

    if(user_object.hevy_profile_url && user_object.hevy_public) {
        userLinks = document.getElementById("user-links")

        userLinks.style.display = "flex"

        userLinks.innerHTML += `
            <div onclick="window.open('${user_object.hevy_profile_url}', '_blank');" class="user-link clickable" title="Hevy profile">
                <img src="/assets/hevy.png">
            </div>
        `;
    }

    document.getElementById("user_admin").innerHTML = "Administrator: " + admin_string

    GetUserAchievements(user_object.id);

    if(user_object.share_statistics) {
        GetUserStats(user_object.id);
    } else {
        document.getElementById("account-section-stats").outerHTML = "";
        document.getElementById("account-section-details").style.border = "none";
    }
}

function GetUserAchievements(userID) {

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

                PlaceUserAchievements(result.achievements)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/achievements?user=" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceUserAchievements(achievementArray) {

    if(achievementArray.length > 0) {
        document.getElementById("achievements-module").style.display = "flex"
    } else {
        return;
    }

    for(var i = 0; i < achievementArray.length; i++) {

        // parse date object
        try {
            var date = new Date(Date.parse(achievementArray[i].last_given_at));
            var date_string = GetDateString(date, false)
        } catch {
            var date_string = "Error"
        }

        var classString = ""
        for(var j = 0; j < achievementArray[i].achievement_delegations.length; j++) {
            if(!achievementArray[i].achievement_delegations[j].seen && achievementArray[i].achievement_delegations[j].user_id == user_id) {
                classString += " new-achievement"
            }
        }

        var categoryColor = `var(--${achievementArray[i].category_color})`;
        var categoryText = ""
        if(achievementArray[i].category !== "Default") {
            categoryText = `<div class="meta-tag u-mb-1">${achievementArray[i].category}</div>`;
        }

        var stackableHTML = ``
        if(achievementArray[i].multiple_delegations) {
            stackableHTML = `<div class="meta-tag u-mt-1">Stackable</div>`;
        }

        var delegationSum = achievementArray[i].achievement_delegations.length
        var delegationSumHTML = ``;

        if(delegationSum > 1) {
            delegationSumHTML = `
                <div class="achievement-delegation-sum">
                    <b>${delegationSum}</b>
                </div>
            `;
        }

        var html = `

        <div class="achievement unselectable" title="${achievementArray[i].description}" tabindex="1">

            <div class="achievement-base ${classString}">

                ${delegationSumHTML}

                <div class="achievement-image" style="--cat-color: ${categoryColor};">
                    <img class="achievement-img u-fill" src="${achievementImageURL(achievementArray[i].id, true)}" onerror="${IMAGE_FALLBACK_ONERROR}">
                </div>

                <div class="achievement-title">
                    ${achievementArray[i].name}
                </div>

                <div class="achievement-date">
                    ${date_string}
                </div>     

            </div>     

            <div class="overlay">
                <div class="text-achievement"> 
                    ${categoryText}
                    <div class="u-mb-2"> 
                        ${achievementArray[i].name}
                    </div>
                    <div class="achievement-description"> 
                        ${achievementArray[i].description}
                    </div>
                    ${stackableHTML}
                </div>
            </div>

        </div>
        `;

        document.getElementById("achievements-box").innerHTML += html

    }

}

function GetUserStats(userID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                document.getElementById("user-stats-skeleton").style.display = "none";
                return;
            }
            if (result.error) {
                document.getElementById("user-stats-skeleton").style.display = "none";
            } else {
                PlaceUserStats(result.data);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID + "/statistics");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

function PlaceUserStats(data) {
    var skeleton = document.getElementById("user-stats-skeleton");
    var content  = document.getElementById("user-stats-content");
    if (!skeleton || !content) return;

    // ── helpers ──────────────────────────────────────────────────────────────
    function fmtDistance(km) {
        if (!km) return "0 km";
        return parseFloat(km).toFixed(1) + " km";
    }
    function fmtDuration(secs) {
        if (!secs) return "0 min";
        var h = Math.floor(secs / 3600);
        var m = Math.floor((secs % 3600) / 60);
        if (h > 0) return h + "h " + m + "m";
        return m + " min";
    }
    function statCard(label, value) {
        return `
            <div class="user-stat-card">
                <div class="user-stat-value">${value}</div>
                <div class="user-stat-label">${label}</div>
            </div>`;
    }
    function flameIcon(count) {
        var tier = count <= 0 ? 0 : count < 4 ? 1 : count < 12 ? 2 : 3;
        return `<span class="flame-icon flame-${tier}">🔥</span>`;
    }

    // ── streak section ───────────────────────────────────────────────────────
    var weekFlame  = flameIcon(data.streak_weeks);
    var dayFlame   = flameIcon(data.streak_days);

    var streakHTML = `
        <div class="user-stats-section">
            <div class="user-stats-section-title">Streaks</div>
            <div class="user-stats-row">
                <div class="user-streak-card unselectable" tabindex="1">
                    <div class="user-streak-card-base">
                        ${weekFlame}
                        <div class="user-streak-number">${data.streak_weeks}<span class="user-streak-unit">wk</span></div>
                        <div class="user-stat-label">Week streak</div>
                    </div>
                    <div class="streak-overlay">
                        <div class="streak-overlay-text">
                            <div class="streak-overlay-title">🔥 Week streak</div>
                            <div>Current: <b>${data.streak_weeks} wk</b></div>
                            <div class="u-dim">Best: <b>${data.streak_weeks_top} wk</b></div>
                        </div>
                    </div>
                </div>
                <div class="user-streak-card unselectable" tabindex="1">
                    <div class="user-streak-card-base">
                        ${dayFlame}
                        <div class="user-streak-number">${data.streak_days}<span class="user-streak-unit">d</span></div>
                        <div class="user-stat-label">Day streak</div>
                    </div>
                    <div class="streak-overlay">
                        <div class="streak-overlay-text">
                            <div class="streak-overlay-title">🔥 Day streak</div>
                            <div>Current: <b>${data.streak_days} d</b></div>
                            <div class="u-dim">Best: <b>${data.streak_days_top} d</b></div>
                        </div>
                    </div>
                </div>
            </div>
        </div>`;

    // ── exercise count section ───────────────────────────────────────────────
    var countsHTML = `
        <div class="user-stats-section">
            <div class="user-stats-section-title">Sessions</div>
            <div class="user-stats-row">
                ${statCard("This month", data.exercises_past_month)}
                ${statCard("This year",  data.exercises_past_year)}
                ${statCard("All time",   data.exercises_all_time)}
            </div>
        </div>`;

    // ── top activity section (tabbed: month / year / all time) ───────────────

    // The API withholds a figure it cannot report without describing individual sessions
    // (see its userStatisticsMinSampleSize). Both places that happens — a whole missing
    // breakdown, and one window of an otherwise-reported breakdown — say the same thing.
    // The threshold is interpolated from the response so this copy cannot drift from the
    // constant that enforces it.
    var minimumSample = parseInt(data.minimum_sample_size, 10);
    var sampleHint = isNaN(minimumSample)
        ? "It needs a few more logged activities."
        : `It needs at least ${minimumSample} logged activities.`;

    function emptyNoteHTML(title, hint) {
        return `
            <div class="user-stats-empty">
                <span class="user-stats-empty-title">${title}</span>
                <span class="user-stats-empty-hint">${hint}</span>
            </div>`;
    }

    function windowEmptyHTML() {
        return emptyNoteHTML("Not enough activity in this period", sampleHint);
    }

    var actHTML = "";
    var act = data.activity_statistics;
    if (act && act.action) {
        var actionName  = act.action.name || "";
        var actionLogo  = act.action.has_logo
            ? `<img src="/assets/actions/${act.action.name}.svg" class="stat-title-logo" onerror="this.style.display='none'">`
            : "";

        // An average is null when the window holds too few sessions to average without
        // republishing them individually (see the API's userStatisticsMinSampleSize), so
        // the card is left out entirely rather than rendered as a misleading "0 km".
        function optionalStatCard(label, value, format) {
            if (value === null || value === undefined) return "";
            return statCard(label, format(value));
        }

        function actPeriodHTML(period, label) {
            if (!period) return "";
            var rows = "";
            if (period.sums.distance > 0) {
                rows += statCard("Distance",     fmtDistance(period.sums.distance));
                rows += optionalStatCard("Avg distance", period.averages.distance, fmtDistance);
                rows += statCard("Best",         fmtDistance(period.tops.distance));
            }
            if (period.sums.time > 0) {
                rows += statCard("Total time",   fmtDuration(period.sums.time));
                rows += optionalStatCard("Avg time", period.averages.time, fmtDuration);
            }
            rows += statCard("Sessions", period.sums.operations);
            return rows;
        }

        var tabs = [
            { key: "month", label: "Month", period: act.past_month },
            { key: "year",  label: "Year",  period: act.past_year  },
            { key: "all",   label: "All",   period: act.all_time   },
        ];

        // A window is null when it holds too few sessions to report without republishing
        // them one by one. Its tab stays — dropping it would hide the fact silently — but
        // the panel explains itself instead of showing figures. The all-time window is
        // always present when there is a headline activity, so at least one tab has data.
        var firstWithData = tabs.findIndex(function(t) { return t.period; });
        var activeIndex = firstWithData === -1 ? 0 : firstWithData;

        var tabBtns = tabs.map(function(t, i) {
            var active = i === activeIndex ? " user-stat-tab-active" : "";
            return `<button class="user-stat-tab${active}" onclick="switchStatTab('${t.key}')">${t.label}</button>`;
        }).join("");

        var tabPanels = tabs.map(function(t, i) {
            var display = i === activeIndex ? "" : " style=\"display:none\"";
            var body = t.period ? actPeriodHTML(t.period, t.label) : windowEmptyHTML();
            return `<div class="user-stats-row user-stat-panel" id="stat-panel-${t.key}"${display}>${body}</div>`;
        }).join("");

        actHTML = `
            <div class="user-stats-section">
                <div class="user-stats-section-title">
                    ${actionLogo}${actionName}
                </div>
                <div class="user-stat-tabs">${tabBtns}</div>
                ${tabPanels}
            </div>`;
    } else {
        // No headline activity means the API found too few logged activities to name one
        // without describing a single workout instead of a person (see the API's
        // chooseHeadlineAction). Say so, rather than leaving a gap where a section was —
        // the sessions counts above are still on screen, so a silent hole reads as broken.
        actHTML = `
            <div class="user-stats-section">
                <div class="user-stats-section-title">Activity breakdown</div>
                ${emptyNoteHTML("Not enough activity yet", sampleHint)}
            </div>`;
    }

    content.innerHTML = streakHTML + countsHTML + actHTML;
    skeleton.style.display = "none";
    content.style.display  = "flex";   // reveal (visibility is state); layout is in .user-stats-content
}

function switchStatTab(key) {
    var panels = document.querySelectorAll(".user-stat-panel");
    panels.forEach(function(p) { p.style.display = "none"; });
    var tabs = document.querySelectorAll(".user-stat-tab");
    tabs.forEach(function(t) { t.classList.remove("user-stat-tab-active"); });

    var active = document.getElementById("stat-panel-" + key);
    if (active) active.style.display = "";

    var activeBtn = Array.from(tabs).find(function(t) {
        return t.getAttribute("onclick") === "switchStatTab('" + key + "')";
    });
    if (activeBtn) activeBtn.classList.add("user-stat-tab-active");
}
