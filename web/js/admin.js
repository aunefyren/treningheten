// The signed-in admin's user ID, so the user list doesn't offer to disable yourself.
var currentAdminID = null;

// Every user, disabled ones included, as served by GET admin/users.
var adminUsers = [];

function load_page(result) {

    var admin = false

    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            admin = login_data.data.admin
            currentAdminID = login_data.data.id
        } catch {
            admin = false
        }

        showAdminMenu(admin)
    }

    var html = `
                <div class="modules" id="admin-page">

                    <div class="server-info-module" id="server-info-module">

                        <div class="server-info" id="server-info">
                            <h3 id="server-info-title">Server info</h3>
                            <p class="server-info-loading">Loading...</p>
                        </div>

                    </div>

                    <div class="server-info-module" id="stats-module">

                        <div class="server-info" id="admin-stats">
                            <h3 id="admin-stats-title">Statistics</h3>
                            <p class="server-info-loading">Loading...</p>
                        </div>

                    </div>

                    <div class="admin-module admin-module--wide" id="users-module">

                        <h3 id="users-module-title">Users:</h3>

                        <div class="admin-user-list" id="admin-user-list">
                            <p class="server-info-loading">Loading...</p>
                        </div>

                    </div>

                    <div class="invitation-module" id="invitation-module">

                        <div class="invites" id="invites">

                            <h3 id="invitation-module-title">Invites:</h3>

                            <div class="invite-list" id="invite-list">
                            </div>

                            <button type="submit" onclick="generateInvite();" id="generate_invite_button" class="btn"><img src="assets/plus.svg" class="color-invert">Generate</button>

                        </div>

                    </div>

                    <div class="debt-module" id="debt-module">

                        <div class="debt-form" id="debt-form">

                            <h3 id="debt-module-title">Debt:</h3>

                            <form action="" onsubmit="event.preventDefault(); generateDebt();">

                                <div class="field">
                                    <label for="debt-week" class="field-label">Week with debt</label>
                                    <input type="date" id="debt-week" name="debt-week" required>
                                </div>

                                <div class="field">
                                    <label for="debt-user" class="field-label">User <span class="field-hint">(optional)</span></label>
                                    <select id="debt-user" class="admin-user-select" data-placeholder="Choose optional user"></select>
                                </div>

                                <button type="submit" id="generate-debt-button" class="btn btn--block"><img src="assets/plus.svg" class="color-invert">Generate debt</button>

                            </form>

                        </div>

                    </div>

                    <div class="prize-module" id="prize-module">

                        <div class="prize-form" id="prize-form">

                            <h3 id="prize-module-title">Prize:</h3>

                            <form action="" onsubmit="event.preventDefault(); addPrize();">

                                <div class="field">
                                    <label for="prize-name" class="field-label">Name of prize</label>
                                    <input type="text" id="prize-name" name="prize-name" autocomplete="off" required>
                                </div>

                                <div class="field">
                                    <label for="prize-quantity" class="field-label">Quantity of prize</label>
                                    <input type="number" id="prize-quantity" name="prize-quantity" value="1" min="1" required>
                                </div>

                                <button type="submit" id="add-prize-button" class="btn btn--block"><img src="assets/done.svg" class="color-invert">Add prize</button>

                            </form>

                        </div>

                    </div>

                    <div class="add-season-module" id="add-season-module">

                        <div class="season-form" id="season-form">

                            <h3 id="season-module-title">Season:</h3>

                            <form action="" onsubmit="event.preventDefault(); addSeason();">

                                <div class="field-row">
                                    <div class="field">
                                        <label for="season-start" class="field-label">Start of season (monday)</label>
                                        <input type="date" id="season-start" name="season-start" required>
                                    </div>
                                    <div class="field">
                                        <label for="season-end" class="field-label">End of season (sunday)</label>
                                        <input type="date" id="season-end" name="season-end" required>
                                    </div>
                                </div>

                                <div class="field">
                                    <label for="season-name" class="field-label">Name</label>
                                    <input type="text" id="season-name" name="season-name" placeholder="e.g. Spring 2026" autocomplete="off" required>
                                </div>

                                <div class="field">
                                    <label for="season-desc" class="field-label">Description</label>
                                    <textarea id="season-desc" name="season-desc" placeholder="What is this season about?" autocomplete="off" required></textarea>
                                </div>

                                <div class="field">
                                    <label for="season-sickleave" class="field-label">Season sick leave</label>
                                    <input type="number" id="season-sickleave" name="season-sickleave" value="0" min="0" max="99" required>
                                </div>

                                <div class="field">
                                    <label for="season-prize" class="field-label">Season prize</label>
                                    <select id="season-prize" name="season-prize" required></select>
                                </div>

                                <div class="field-check">
                                    <input type="checkbox" id="join_anytime" name="join_anytime" value="join_anytime">
                                    <label for="join_anytime" title="Should people be able to join after season start?">Let users join the season at any point.</label>
                                </div>

                                <button type="submit" id="add-season-button" class="btn btn--block"><img src="assets/done.svg" class="color-invert">Add season</button>

                            </form>

                        </div>

                    </div>

                    <div class="admin-module" id="achievement-module">

                        <h3 id="achievement-module-title">Give achievement:</h3>

                        <form action="" onsubmit="event.preventDefault(); giveAchievement();">

                            <div class="field">
                                <label for="achievement-user" class="field-label">User</label>
                                <select id="achievement-user" class="admin-user-select" data-placeholder="Choose user" required></select>
                            </div>

                            <div class="field">
                                <label for="achievement-select" class="field-label">Achievement</label>
                                <select id="achievement-select" required></select>
                            </div>

                            <div class="field">
                                <label for="achievement-date" class="field-label">Given at <span class="field-hint">(optional, defaults to now)</span></label>
                                <input type="date" id="achievement-date" name="achievement-date">
                            </div>

                            <button type="submit" id="give-achievement-button" class="btn btn--block"><img src="assets/done.svg" class="color-invert">Give achievement</button>

                        </form>

                    </div>

                    <div class="admin-module" id="push-module">

                        <h3 id="push-module-title">Push notification:</h3>

                        <form action="" onsubmit="event.preventDefault(); pushNotification();">

                            <div class="field">
                                <label for="push-user" class="field-label">User</label>
                                <select id="push-user" class="admin-user-select" data-placeholder="Choose user" required></select>
                            </div>

                            <div class="field">
                                <label for="push-title" class="field-label">Title</label>
                                <input type="text" id="push-title" name="push-title" autocomplete="off" required>
                            </div>

                            <div class="field">
                                <label for="push-body" class="field-label">Body</label>
                                <textarea id="push-body" name="push-body" autocomplete="off" required></textarea>
                            </div>

                            <div class="field">
                                <label for="push-category" class="field-label">Opens</label>
                                <select id="push-category">
                                    <option value="general">Front page</option>
                                    <option value="news">News</option>
                                    <option value="achievement">Achievements</option>
                                    <option value="account">Account</option>
                                </select>
                            </div>

                            <button type="submit" id="push-button" class="btn btn--block"><img src="assets/done.svg" class="color-invert">Push to all devices</button>

                        </form>

                    </div>

                    <div class="admin-module" id="sync-module" style="display: none;">

                        <h3 id="sync-module-title">Integration sync:</h3>

                        <form action="" id="strava-sync-form" style="display: none;" onsubmit="event.preventDefault(); syncIntegration('strava');">

                            <div class="field">
                                <label for="strava-sync-user" class="field-label">Strava: re-fetch activities for</label>
                                <select id="strava-sync-user" class="admin-user-select" data-placeholder="All connected users"></select>
                            </div>

                            <button type="submit" id="strava-sync-button" class="btn btn--block"><img src="assets/done.svg" class="color-invert">Sync Strava</button>

                        </form>

                        <form action="" id="media-sync-form" style="display: none;" onsubmit="event.preventDefault(); syncIntegration('media');">

                            <div class="field">
                                <label for="media-sync-user" class="field-label">Media: re-sync playback for</label>
                                <select id="media-sync-user" class="admin-user-select" data-placeholder="All connected users"></select>
                            </div>

                            <button type="submit" id="media-sync-button" class="btn btn--block"><img src="assets/done.svg" class="color-invert">Sync media</button>

                        </form>

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Ultimate power.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();

        if(!admin) {
            document.getElementById('content').innerHTML = "...";
            error("You are not an admin.")
        } else {
            getServerInfo();
            getAdminStats();
            getAdminUsers();
            getInvites();
            getPrizes();
            getAchievements();
        }

    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

// adminRequest sends a JSON request to the API and hands the parsed result to onSuccess.
// API errors and unparseable responses are shown through the shared helpers instead.
function adminRequest(method, path, body, onSuccess) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState != 4) {
            return;
        }

        var result;
        try {
            result = JSON.parse(this.responseText);
        } catch(e) {
            console.log(e + ' - Response: ' + this.responseText);
            error("Could not reach API.");
            return;
        }

        if(result.error) {
            error(result.error);
            return;
        }

        onSuccess(result);
    };
    xhttp.withCredentials = true;
    xhttp.open(method, api_url + path);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(body === null ? null : JSON.stringify(body));
    return false;
}

function escapeHTML(value) {
    return String(value == null ? "" : value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

function getServerInfo() {
    adminRequest("get", "admin/server-info", null, function(result) {
        placeServerInfo(result.server);
        placeIntegrationSync(result.server);
    });
}

function serverInfoBadge(on, onText, offText) {
    onText = onText || "Enabled";
    offText = offText || "Disabled";
    return on
        ? `<span class="info-badge info-badge--on">` + onText + `</span>`
        : `<span class="info-badge info-badge--off">` + offText + `</span>`;
}

// serverInfoRow renders a key/value row. Values are trusted markup (badges) or config
// values the admin set themselves.
function serverInfoRow(key, value) {
    if(value === undefined || value === null || value === "") {
        value = "—";
    }
    return `<div class="info-row"><span class="info-key">` + key + `</span><span class="info-value">` + value + `</span></div>`;
}

function placeServerInfo(serverInfo) {
    var s = serverInfo || {};
    var db = s.database || {};
    var smtp = s.smtp || {};
    var strava = s.strava || {};
    var hevy = s.hevy || {};
    var mcp = s.mcp || {};
    var ai = s.ai || {};
    var push = s.push || {};
    var media = s.media || {};

    var dbTarget = "—";
    if((db.type || "").toLowerCase() === "sqlite") {
        dbTarget = db.location || "—";
    } else if(db.host) {
        dbTarget = db.host + (db.port ? ":" + db.port : "") + (db.name ? " / " + db.name : "");
    }

    var environmentBadge = `<span class="info-badge info-badge--neutral">` + (s.environment || "unknown") + `</span>`;

    var html = `
        <h3 id="server-info-title">Server info</h3>
        <div class="info-sections">

            <div class="info-section">
                <div class="info-section-head">Application ` + environmentBadge + `</div>
                ` + serverInfoRow("Name", s.name) + `
                ` + serverInfoRow("Version", s.treningheten_version) + `
                ` + serverInfoRow("Timezone", s.timezone) + `
                ` + serverInfoRow("Port", s.port) + `
                ` + serverInfoRow("Log level", s.log_level) + `
                ` + serverInfoRow("External URL", s.external_url) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Database</div>
                ` + serverInfoRow("Type", db.type) + `
                ` + serverInfoRow("Target", dbTarget) + `
                ` + serverInfoRow("SSL", serverInfoBadge(db.ssl, "On", "Off")) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Email (SMTP) ` + serverInfoBadge(smtp.enabled) + `</div>
                ` + serverInfoRow("Host", smtp.host) + `
                ` + serverInfoRow("Port", smtp.port) + `
                ` + serverInfoRow("From", smtp.from) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Strava ` + serverInfoBadge(strava.enabled) + `</div>
                ` + serverInfoRow("Credentials", serverInfoBadge(strava.configured, "Configured", "Missing")) + `
                ` + serverInfoRow("Redirect URI", strava.redirect_uri) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Hevy ` + serverInfoBadge(hevy.enabled) + `</div>
            </div>

            <div class="info-section">
                <div class="info-section-head">Media ` + serverInfoBadge(media.enabled) + `</div>
                ` + serverInfoRow("Plex", serverInfoBadge(media.plex)) + `
                ` + serverInfoRow("Spotify", serverInfoBadge(media.spotify)) + `
                ` + serverInfoRow("Audiobookshelf", serverInfoBadge(media.audiobookshelf)) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">MCP server ` + serverInfoBadge(mcp.enabled) + `</div>
            </div>

            <div class="info-section">
                <div class="info-section-head">AI (Ollama) ` + serverInfoBadge(ai.enabled) + `</div>
                ` + serverInfoRow("URL", ai.url) + `
                ` + serverInfoRow("Model", ai.model) + `
                ` + serverInfoRow("API key", serverInfoBadge(ai.api_key_set, "Set", "None")) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Push (VAPID) ` + serverInfoBadge(push.configured, "Configured", "Missing") + `</div>
                ` + serverInfoRow("Contact", push.contact) + `
            </div>

        </div>
    `;

    document.getElementById('server-info').innerHTML = html;
}

function getAdminStats() {
    adminRequest("get", "admin/stats", null, function(result) {
        placeAdminStats(result.stats);
    });
}

function adminStatsMetric(key, count, total, pct) {
    var value = count;
    if(total !== undefined && total !== null) {
        value += " / " + total;
    }
    if(pct !== undefined && pct !== null) {
        value += ` <span class="info-badge info-badge--neutral">` + pct + `%</span>`;
    }
    return serverInfoRow(key, value);
}

function placeAdminStats(stats) {
    var s = stats || {};

    var html = `
        <h3 id="admin-stats-title">Statistics</h3>
        <div class="info-sections">

            <div class="info-section">
                <div class="info-section-head">Users</div>
                ` + serverInfoRow("Total users", s.total_users) + `
                ` + adminStatsMetric("In a season now", s.users_in_season_now, s.total_users, s.users_in_season_now_pct) + `
                ` + adminStatsMetric("Notifications enabled", s.users_with_notifications, s.total_users, s.users_with_notifications_pct) + `
                ` + adminStatsMetric("Strava connected", s.users_with_strava, s.total_users, s.users_with_strava_pct) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Achievements</div>
                ` + serverInfoRow("Total achievements", s.achievements_total) + `
                ` + serverInfoRow("Avg. completion", `<span class="info-badge info-badge--neutral">` + s.avg_achievement_completion_pct + `%</span>`) + `
            </div>

        </div>
    `;

    document.getElementById('admin-stats').innerHTML = html;
}

function getAdminUsers() {
    adminRequest("get", "admin/users", null, function(result) {
        adminUsers = result.users || [];
        placeAdminUsers();
        placeUserSelects();
    });
}

function adminUserName(user) {
    return user.first_name + " " + user.last_name;
}

function placeAdminUsers() {
    var html = "";

    for(var i = 0; i < adminUsers.length; i++) {
        var user = adminUsers[i];

        var badges = "";
        if(user.admin) {
            badges += `<span class="info-badge info-badge--neutral">Admin</span>`;
        }
        if(!user.verified) {
            badges += `<span class="info-badge info-badge--off">Unverified</span>`;
        }
        if(!user.enabled) {
            badges += `<span class="info-badge info-badge--off">Disabled</span>`;
        }

        var action = "";
        if(user.id !== currentAdminID) {
            action = user.enabled
                ? `<button type="button" class="btn" onclick="setUserEnabled('` + user.id + `', false);">Disable</button>`
                : `<button type="button" class="btn" onclick="setUserEnabled('` + user.id + `', true);">Enable</button>`;
        }

        html += `
            <div class="admin-user` + (user.enabled ? `` : ` admin-user--disabled`) + `">
                <div class="admin-user-identity">
                    <div class="admin-user-name">` + escapeHTML(adminUserName(user)) + ` ` + badges + `</div>
                    <div class="admin-user-email">` + escapeHTML(user.email) + `</div>
                </div>
                ` + action + `
            </div>
        `;
    }

    document.getElementById("admin-user-list").innerHTML = html || "No users.";
}

// placeUserSelects fills every user picker on the page with the enabled users, keeping
// each picker's placeholder option and current choice.
function placeUserSelects() {
    var selects = document.getElementsByClassName("admin-user-select");

    for(var i = 0; i < selects.length; i++) {
        var select = selects[i];
        var previous = select.value;

        var options = `<option value="">` + escapeHTML(select.dataset.placeholder) + `</option>`;
        for(var j = 0; j < adminUsers.length; j++) {
            if(adminUsers[j].enabled) {
                options += `<option value="` + adminUsers[j].id + `">` + escapeHTML(adminUserName(adminUsers[j])) + `</option>`;
            }
        }

        select.innerHTML = options;
        select.value = previous;
        if(select.value !== previous) {
            select.value = "";
        }
    }
}

// setUserEnabled disables or re-enables a user. Disabling cuts off their access at once and
// takes them out of every ongoing and upcoming season; re-enabling restores access only.
function setUserEnabled(userID, enabled) {
    var user = adminUsers.find(function(candidate) { return candidate.id === userID; });
    var name = user ? adminUserName(user) : "this user";

    var question = enabled
        ? "Re-enable " + name + "? They regain access, but must rejoin seasons themselves."
        : "Disable " + name + "? They lose access immediately and leave all ongoing and upcoming seasons. Finished seasons are kept.";
    if(!confirm(question)) {
        return false;
    }

    return adminRequest("put", "admin/users/" + userID + "/enabled", { "enabled": enabled }, function(result) {
        success(result.message);
        for(var i = 0; i < adminUsers.length; i++) {
            if(adminUsers[i].id === result.user.id) {
                adminUsers[i] = result.user;
            }
        }
        placeAdminUsers();
        placeUserSelects();
    });
}

function getInvites() {
    adminRequest("get", "admin/invites", null, function(result) {
        placeInvites(result.invites);
    });
}

function placeInvites(invitesArray) {
    var html = ``;

    for(var i = 0; i < invitesArray.length; i++) {
        html += `
            <div id="" class="invitation-object">
                <div class="leaderboard-object-code">
                    Code: ` + escapeHTML(invitesArray[i].code) + `
                </div>
        `;

        if(invitesArray[i].used) {
            html += `
                    <div class="leaderboard-object-user">
                        Used by: ` + escapeHTML(invitesArray[i].recipient.first_name + ` ` + invitesArray[i].recipient.last_name) + `
                    </div>
                `;
        } else {
            html += `
                    <div class="leaderboard-object-user">
                        Not used
                    </div>
                    <img class="btn btn--icon clickable" onclick="deleteInvite('` + invitesArray[i].id + `')" src="/assets/trash-2.svg"></img>
                `;
        }

        html += `</div>`;
    }

    document.getElementById("invite-list").innerHTML = html;
}

function generateInvite() {
    return adminRequest("post", "admin/invites", null, function(result) {
        success(result.message);
        placeInvites(result.invites);
    });
}

// deleteInvite withdraws an unused invite. The route is DELETE; a used invite is refused
// with a 409 and its message shown.
function deleteInvite(inviteID) {
    if(!confirm("Are you sure you want to delete this invite?")) {
        return false;
    }

    return adminRequest("delete", "admin/invites/" + inviteID, null, function(result) {
        success(result.message);
        placeInvites(result.invites);
    });
}

function generateDebt() {
    if(!confirm("Are you sure you want to generate debt for the chosen week?")) {
        return false;
    }

    // Sent as UTC midnight of the picked day, as it always has been; the server finds the
    // week from that date.
    var debtWeekValue = document.getElementById("debt-week").value;
    if(!parsePickedDate(debtWeekValue)) {
        error("Failed to parse date request.");
        return false;
    }

    var debtTargetUser = document.getElementById("debt-user").value || null;

    return adminRequest("post", "admin/debts", { "date": debtWeekValue + "T00:00:00Z", "target_user": debtTargetUser }, function(result) {
        success(result.message);
    });
}

function getPrizes() {
    adminRequest("get", "admin/prizes", null, function(result) {
        placePrizes(result.prizes);
    });
}

function placePrizes(prizesArray) {
    var selectObject = document.getElementById("season-prize");

    for(var i = selectObject.options.length-1; i >= 0; i--) {
        selectObject.remove(i)
    }

    for(var i = 0; i < prizesArray.length; i++) {
        var option = document.createElement("option");
        option.text = prizesArray[i].quantity + " " + prizesArray[i].name;
        option.value = prizesArray[i].id;
        selectObject.add(option);
    }
}

// parsePickedDate turns an <input type="date"> value ("YYYY-MM-DD") into a Date at local
// midnight on that calendar day. new Date("YYYY-MM-DD") would be UTC midnight instead, so
// getDay() west of UTC lands on the day before; returns null for a malformed value.
function parsePickedDate(value) {
    var parts = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
    if(!parts) {
        return null;
    }
    return new Date(parseInt(parts[1], 10), parseInt(parts[2], 10) - 1, parseInt(parts[3], 10));
}

// addSeason validates the season form and posts it. The weekday and "in the future" checks run
// on the picked calendar dates in the browser's zone; the dates are sent as UTC midnight of the
// same calendar day, and the server builds the boundaries in its own zone from that date.
function addSeason() {
    clearResponse();

    var seasonStartValue = document.getElementById("season-start").value;
    var seasonEndValue = document.getElementById("season-end").value;
    var seasonJoinAnytime = document.getElementById("join_anytime").checked;
    var seasonName = document.getElementById("season-name").value;
    var seasonDescription = document.getElementById("season-desc").value;
    var seasonSickleave = parseInt(document.getElementById("season-sickleave").value);

    var seasonPrize;
    try {
        var seasonPrizeSelect = document.getElementById("season-prize");
        seasonPrize = seasonPrizeSelect[seasonPrizeSelect.selectedIndex].value;
    } catch(e) {
        console.log("Failed to parse prize. Error: " + e);
        error("Failed to parse prize.");
        return false;
    }

    var seasonStart = parsePickedDate(seasonStartValue);
    var seasonEnd = parsePickedDate(seasonEndValue);
    if(!seasonStart || !seasonEnd) {
        error("Failed to parse date object.");
        return false;
    }

    if(seasonStart.getDay() != 1) {
        error("Season start must be a monday.");
        return false;
    }

    if(seasonEnd.getDay() != 0) {
        error("Season end must be a sunday.");
        return false;
    }

    if(seasonEnd < seasonStart) {
        error("Season start must be before season end.");
        return false;
    }

    if(seasonStart < new Date()) {
        error("Season start must be later than now.");
        return false;
    }

    var form = {
        "start": seasonStartValue + "T00:00:00Z",
        "end": seasonEndValue + "T00:00:00Z",
        "name": seasonName,
        "description": seasonDescription,
        "prize_id": seasonPrize,
        "sickleave": seasonSickleave,
        "join_anytime": seasonJoinAnytime
    };

    return adminRequest("post", "admin/seasons", form, function(result) {
        success(result.message);
        document.getElementById("season-name").value = "";
        document.getElementById("season-desc").value = "";
        document.getElementById("season-sickleave").value = 0;
    });
}

function addPrize() {
    var form = {
        "name": document.getElementById("prize-name").value,
        "quantity": parseInt(document.getElementById("prize-quantity").value)
    };

    return adminRequest("post", "admin/prizes", form, function(result) {
        success(result.message);
        document.getElementById("prize-name").value = "";
        document.getElementById("prize-quantity").value = "";
        getPrizes();
    });
}

function getAchievements() {
    adminRequest("get", "auth/achievements", null, function(result) {
        placeAchievements(result.achievements || []);
    });
}

function placeAchievements(achievements) {
    achievements.sort(function(a, b) { return a.name.localeCompare(b.name); });

    var options = `<option value="">Choose achievement</option>`;
    for(var i = 0; i < achievements.length; i++) {
        options += `<option value="` + achievements[i].id + `">` + escapeHTML(achievements[i].name) + `</option>`;
    }

    document.getElementById("achievement-select").innerHTML = options;
}

function giveAchievement() {
    var userID = document.getElementById("achievement-user").value;
    var achievementID = document.getElementById("achievement-select").value;
    var givenAtValue = document.getElementById("achievement-date").value;

    var form = { "achievement_id": achievementID };
    if(givenAtValue) {
        var givenAt = parsePickedDate(givenAtValue);
        if(!givenAt) {
            error("Failed to parse date.");
            return false;
        }
        form.given_at = givenAt.toISOString();
    }

    return adminRequest("post", "admin/users/" + userID + "/achievement-delegations", form, function(result) {
        success(result.message);
        document.getElementById("achievement-date").value = "";
    });
}

function pushNotification() {
    var form = {
        "user": document.getElementById("push-user").value,
        "title": document.getElementById("push-title").value,
        "body": document.getElementById("push-body").value,
        "category": document.getElementById("push-category").value
    };

    return adminRequest("post", "admin/notifications/push/all-devices", form, function(result) {
        if(result.amount === 0) {
            info("The user has no devices subscribed to notifications.");
            return;
        }
        success("Pushed to " + result.amount + " device(s).");
        document.getElementById("push-title").value = "";
        document.getElementById("push-body").value = "";
    });
}

// placeIntegrationSync shows a sync form for each integration the server has switched on,
// and hides the module entirely when none are.
function placeIntegrationSync(serverInfo) {
    var stravaEnabled = !!(serverInfo && serverInfo.strava && serverInfo.strava.enabled);
    var mediaEnabled = !!(serverInfo && serverInfo.media && serverInfo.media.enabled);

    document.getElementById("strava-sync-form").style.display = stravaEnabled ? "" : "none";
    document.getElementById("media-sync-form").style.display = mediaEnabled ? "" : "none";
    document.getElementById("sync-module").style.display = (stravaEnabled || mediaEnabled) ? "" : "none";
}

// syncIntegration starts a background re-sync for one user, or for every connected user
// when none is picked. The server answers as soon as the sync is queued.
function syncIntegration(integration) {
    var paths = {
        "strava": "admin/strava/sync-activities-for-users",
        "media": "admin/media/sync-for-users"
    };

    var userID = document.getElementById(integration + "-sync-user").value;
    if(!userID && !confirm("Sync every connected user? This can take a while.")) {
        return false;
    }

    return adminRequest("post", paths[integration], { "user_ids": userID ? [userID] : [] }, function(result) {
        success(result.message);
    });
}
