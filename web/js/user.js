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
                    <img style="width: 100%; height: 100%;" class="user-active-profile-photo-img" id="user-active-profile-photo-img" src="/assets/images/barbell.gif">
                </div>

                <b><p id="user_name" style="margin-top: 1em; font-size: 1.25em;"></p></b>
                <p id="join_date" style=""></p>
                <p id="user_admin" style=""></p>

                <div class="user-links" id="user-links" style="display:none; margin-top: 1em;">
                </div>
            </div>

            <div id="account-section-stats" class="account-section account-section--stats">
                <div id="user-stats-skeleton" class="user-stats-skeleton">
                    <!-- Streaks label + two cards -->
                    <div class="skeleton-block" style="width: 5em; height: 0.8em; border-radius: 0.3em;"></div>
                    <div style="display:flex; gap: 0.5em; margin-top: 0.5em;">
                        <div class="skeleton-block" style="width: 6em; height: 5em; border-radius: 0.6em;"></div>
                        <div class="skeleton-block" style="width: 6em; height: 5em; border-radius: 0.6em;"></div>
                    </div>
                    <!-- Sessions label + three cards -->
                    <div class="skeleton-block" style="width: 5em; height: 0.8em; border-radius: 0.3em; margin-top: 1.2em;"></div>
                    <div style="display:flex; gap: 0.5em; margin-top: 0.5em;">
                        <div class="skeleton-block" style="width: 5em; height: 3.5em; border-radius: 0.6em;"></div>
                        <div class="skeleton-block" style="width: 5em; height: 3.5em; border-radius: 0.6em;"></div>
                        <div class="skeleton-block" style="width: 5em; height: 3.5em; border-radius: 0.6em;"></div>
                    </div>
                    <!-- Activity label + tab bar + stat cards -->
                    <div class="skeleton-block" style="width: 7em; height: 0.8em; border-radius: 0.3em; margin-top: 1.2em;"></div>
                    <div class="skeleton-block" style="width: 18em; max-width: 100%; height: 1.8em; border-radius: 0.5em; margin-top: 0.5em;"></div>
                    <div style="display:flex; gap: 0.5em; margin-top: 0.5em; flex-wrap: wrap;">
                        <div class="skeleton-block" style="width: 5.5em; height: 3.5em; border-radius: 0.6em;"></div>
                        <div class="skeleton-block" style="width: 5.5em; height: 3.5em; border-radius: 0.6em;"></div>
                        <div class="skeleton-block" style="width: 5.5em; height: 3.5em; border-radius: 0.6em;"></div>
                    </div>
                </div>
                <div id="user-stats-content" style="display:none; width: 100%;"></div>
            </div>
            </div><!-- /.user-profile-row -->
        </div>

        <div class="module" id="achievements-hr" style="display: none;">
            <hr>
        </div>

        <div class="module" id="achievements-module" style="display: none;">

            <div id="achievements-title" class="title" style="margin-bottom: 1em;">
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

                PlaceProfileImage(result.image)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID + "/image");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImage(imageBase64) {

    document.getElementById("user-active-profile-photo-img").src = imageBase64

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
            <div onclick="window.open('https://www.strava.com/athletes/${user_object.strava_id}', '_blank');" class="clickable" style="width: 2em; height: 2em;" title="Strava profile">
                <img src="/assets/strava-logo.svg" style="" class="">
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
        document.getElementById("achievements-hr").style.display = "flex"
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
            categoryText = `
            <div style="font-size: 0.70em; margin-bottom: 1em; border: solid 0.12rem; border-radius: 0.5em; width: auto; padding: 0.12rem 0.25rem;"> 
                ${achievementArray[i].category}
            </div>
            `;
        }

        var stackableHTML = ``
        if(achievementArray[i].multiple_delegations) {
            stackableHTML = `
                <div style="font-size: 0.70em; margin-top: 1em; border: solid 0.12rem; border-radius: 0.5em; width: auto; padding: 0.12rem 0.25rem;"> 
                    Stackable
                </div>
            `;
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

                <div class="achievement-image" style="border: solid 0.2em ${categoryColor};">
                    <img style="width: 100%; height: 100%;" class="achievement-img" id="achievement-img-${achievementArray[i].id}" src="/assets/images/barbell.gif">
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
                    <div style="margin-bottom: 0.5em;"> 
                        ${achievementArray[i].name}
                    </div>
                    <div style="" class="achievement-description"> 
                        ${achievementArray[i].description}
                    </div>
                    ${stackableHTML}
                </div>
            </div>

        </div>
        `;

        document.getElementById("achievements-box").innerHTML += html

        GetAchievementImage(achievementArray[i].id)

    }

}

function GetAchievementImage(achievementID) {

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

                PlaceAchievementImage(result.image, achievementID)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/achievements/" + achievementID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceAchievementImage(imageBase64, achievementID) {

    document.getElementById("achievement-img-" + achievementID).src = imageBase64

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
        var color = count <= 0  ? "var(--grey, #888)"
                  : count < 4   ? "#f5a623"
                  : count < 12  ? "#f07800"
                  :               "#e63000";
        return `<span style="color:${color}; font-size:1.2em;">🔥</span>`;
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
                            <div style="font-size: 1.1em;">🔥 Week streak</div>
                            <div>Current: <b>${data.streak_weeks} wk</b></div>
                            <div style="opacity: 0.7;">Best: <b>${data.streak_weeks_top} wk</b></div>
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
                            <div style="font-size: 1.1em;">🔥 Day streak</div>
                            <div>Current: <b>${data.streak_days} d</b></div>
                            <div style="opacity: 0.7;">Best: <b>${data.streak_days_top} d</b></div>
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
    var actHTML = "";
    var act = data.activity_statistics;
    if (act && act.action) {
        var actionName  = act.action.name || "";
        var actionLogo  = act.action.has_logo
            ? `<img src="/assets/actions/${act.action.name}.svg" class="color-invert" style="height:1.4em; width:1.4em; vertical-align:middle; margin-right:0.4em;" onerror="this.style.display='none'">`
            : "";

        function actPeriodHTML(period, label) {
            if (!period) return "";
            var rows = "";
            if (period.sums.distance > 0) {
                rows += statCard("Distance",     fmtDistance(period.sums.distance));
                rows += statCard("Avg distance", fmtDistance(period.averages.distance));
                rows += statCard("Best",         fmtDistance(period.tops.distance));
            }
            if (period.sums.time > 0) {
                rows += statCard("Total time",   fmtDuration(period.sums.time));
                rows += statCard("Avg time",     fmtDuration(period.averages.time));
            }
            rows += statCard("Sessions", period.sums.operations);
            return rows;
        }

        var tabs = [
            { key: "month", label: "Month", period: act.past_month },
            { key: "year",  label: "Year",  period: act.past_year  },
            { key: "all",   label: "All",   period: act.all_time   },
        ];

        var tabBtns = tabs.map(function(t) {
            var active = t.key === "month" ? " user-stat-tab-active" : "";
            return `<button class="user-stat-tab${active}" onclick="switchStatTab('${t.key}')">${t.label}</button>`;
        }).join("");

        var tabPanels = tabs.map(function(t) {
            var display = t.key === "month" ? "" : " style=\"display:none\"";
            return `<div class="user-stats-row user-stat-panel" id="stat-panel-${t.key}"${display}>${actPeriodHTML(t.period, t.label)}</div>`;
        }).join("");

        actHTML = `
            <div class="user-stats-section">
                <div class="user-stats-section-title">
                    ${actionLogo}${actionName}
                </div>
                <div class="user-stat-tabs">${tabBtns}</div>
                ${tabPanels}
            </div>`;
    }

    content.innerHTML = streakHTML + countsHTML + actHTML;
    skeleton.style.display = "none";
    content.style.display  = "flex";
    content.style.flexDirection = "column";
    content.style.gap = "0.5em";
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
