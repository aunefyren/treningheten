function load_page(result) {

    var admin = false

    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            admin = login_data.data.admin
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

                    <div class="invitation-module" id="invitation-module">

                        <div class="invites" id="invites">

                            <h3 id="invitation-module-title">Invites:</h3>

                            <div class="invite-list" id="invite-list">
                            </div>

                            <button type="submit" onclick="generate_invite();" id="generate_invite_button" class="btn"><img src="assets/plus.svg" class="color-invert">Generate</button>
                        
                        </div>

                    </div>

                    <div class="debt-module" id="debt-module">

                        <div class="debt-form" id="debt-form">

                            <h3 id="debt-module-title">Debt:</h3>

                            <form action="" onsubmit="event.preventDefault(); generate_debt();">

                                <div class="field">
                                    <label for="debt-week" class="field-label">Week with debt</label>
                                    <input type="date" id="debt-week" name="debt-week" required>
                                </div>

                                <div class="field">
                                    <label for="selectUser" class="field-label">User <span class="field-hint">(optional)</span></label>
                                    <select id="selectUser" onchange="">
                                        <option value="null">Choose optional user</option>
                                    </select>
                                </div>

                                <button type="submit" id="generate-debt-button" class="btn btn--block"><img src="assets/plus.svg" class="color-invert">Generate debt</button>

                            </form>

                        </div>

                    </div>

                    <div class="prize-module" id="prize-module">

                        <div class="prize-form" id="prize-form">

                            <h3 id="prize-module-title">Prize:</h3>

                            <form action="" onsubmit="event.preventDefault(); add_prize();">

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

                            <form action="" onsubmit="event.preventDefault(); add_season();">

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
            get_server_info();
            get_admin_stats();
            get_invites();
            get_prizes();
        }

    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_server_info() {

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

                place_server_info(result.server)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "admin/server-info");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function server_info_badge(on, onText, offText) {
    onText = onText || "Enabled";
    offText = offText || "Disabled";
    return on
        ? `<span class="info-badge info-badge--on">` + onText + `</span>`
        : `<span class="info-badge info-badge--off">` + offText + `</span>`;
}

function server_info_row(key, value) {
    if(value === undefined || value === null || value === "") {
        value = "—";
    }
    return `<div class="info-row"><span class="info-key">` + key + `</span><span class="info-value">` + value + `</span></div>`;
}

function place_server_info(server_info) {
    var s = server_info || {};
    var db = s.database || {};
    var smtp = s.smtp || {};
    var strava = s.strava || {};
    var hevy = s.hevy || {};
    var mcp = s.mcp || {};
    var ai = s.ai || {};
    var push = s.push || {};

    var db_target = "—";
    if((db.type || "").toLowerCase() === "sqlite") {
        db_target = db.location || "—";
    } else if(db.host) {
        db_target = db.host + (db.port ? ":" + db.port : "") + (db.name ? " / " + db.name : "");
    }

    var environment_badge = `<span class="info-badge info-badge--neutral">` + (s.environment || "unknown") + `</span>`;

    var html = `
        <h3 id="server-info-title">Server info</h3>
        <div class="info-sections">

            <div class="info-section">
                <div class="info-section-head">Application ` + environment_badge + `</div>
                ` + server_info_row("Name", s.name) + `
                ` + server_info_row("Version", s.treningheten_version) + `
                ` + server_info_row("Timezone", s.timezone) + `
                ` + server_info_row("Port", s.port) + `
                ` + server_info_row("Log level", s.log_level) + `
                ` + server_info_row("External URL", s.external_url) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Database</div>
                ` + server_info_row("Type", db.type) + `
                ` + server_info_row("Target", db_target) + `
                ` + server_info_row("SSL", server_info_badge(db.ssl, "On", "Off")) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Email (SMTP) ` + server_info_badge(smtp.enabled) + `</div>
                ` + server_info_row("Host", smtp.host) + `
                ` + server_info_row("Port", smtp.port) + `
                ` + server_info_row("From", smtp.from) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Strava ` + server_info_badge(strava.enabled) + `</div>
                ` + server_info_row("Credentials", server_info_badge(strava.configured, "Configured", "Missing")) + `
                ` + server_info_row("Redirect URI", strava.redirect_uri) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Hevy ` + server_info_badge(hevy.enabled) + `</div>
            </div>

            <div class="info-section">
                <div class="info-section-head">MCP server ` + server_info_badge(mcp.enabled) + `</div>
            </div>

            <div class="info-section">
                <div class="info-section-head">AI (Ollama) ` + server_info_badge(ai.enabled) + `</div>
                ` + server_info_row("URL", ai.url) + `
                ` + server_info_row("Model", ai.model) + `
                ` + server_info_row("API key", server_info_badge(ai.api_key_set, "Set", "None")) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Push (VAPID) ` + server_info_badge(push.configured, "Configured", "Missing") + `</div>
                ` + server_info_row("Contact", push.contact) + `
            </div>

        </div>
    `;

    document.getElementById('server-info').innerHTML = html;
}

function get_admin_stats() {

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

                place_admin_stats(result.stats)

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "admin/stats");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function admin_stats_metric(key, count, total, pct) {
    var value = count;
    if(total !== undefined && total !== null) {
        value += " / " + total;
    }
    if(pct !== undefined && pct !== null) {
        value += ` <span class="info-badge info-badge--neutral">` + pct + `%</span>`;
    }
    return server_info_row(key, value);
}

function place_admin_stats(stats) {
    var s = stats || {};

    var html = `
        <h3 id="admin-stats-title">Statistics</h3>
        <div class="info-sections">

            <div class="info-section">
                <div class="info-section-head">Users</div>
                ` + server_info_row("Total users", s.total_users) + `
                ` + admin_stats_metric("In a season now", s.users_in_season_now, s.total_users, s.users_in_season_now_pct) + `
                ` + admin_stats_metric("Notifications enabled", s.users_with_notifications, s.total_users, s.users_with_notifications_pct) + `
                ` + admin_stats_metric("Strava connected", s.users_with_strava, s.total_users, s.users_with_strava_pct) + `
            </div>

            <div class="info-section">
                <div class="info-section-head">Achievements</div>
                ` + server_info_row("Total achievements", s.achievements_total) + `
                ` + server_info_row("Avg. completion", `<span class="info-badge info-badge--neutral">` + s.avg_achievement_completion_pct + `%</span>`) + `
            </div>

        </div>
    `;

    document.getElementById('admin-stats').innerHTML = html;
}

function get_invites() {

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

                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "admin/invites");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_invites(invites_array) {
    var html = ``;
    
    if(invites_array.length == 0) {
        html = "";
    } else {
        for(var i = 0; i < invites_array.length; i++) {
            html += `
                <div id="" class="invitation-object">
                    <div class="leaderboard-object-code">
                        Code: ` + invites_array[i].code + `
                    </div>
            `;

            if(invites_array[i].used) {
                html += `
                        <div class="leaderboard-object-user">
                            Used by: ` + invites_array[i].recipient.first_name + ` ` + invites_array[i].recipient.last_name + `
                        </div>
                    `;
                addUserToDebtSelection(invites_array[i].recipient)
            } else {
                html += `
                        <div class="leaderboard-object-user">
                            Not used
                        </div>
                        <img class="btn btn--icon clickable" onclick="delete_invite(` + invites_array[i].id + `)" src="/assets/trash-2.svg"></img>
                    `;
            }

            html += `</div>`;

        }
        
    }

    document.getElementById("invite-list").innerHTML = html

    return
}

function generate_invite() {

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

                success(result.message)
                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/invites");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function delete_invite(invite_id) {

    if(!confirm("Are you sure you want to delete this invite?")) {
        return
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

                success(result.message)
                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/invites/" + invite_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function generate_debt() {
    if(!confirm("Are you sure you want to generate debt for the chosen week?")) {
        return
    }

    var debt_week = document.getElementById("debt-week").value;
    var debtTargetUser = document.getElementById("selectUser").value;

    if(debtTargetUser == "null") {
        debtTargetUser = null
    }

    try {
        var debt_week_object = new Date(debt_week);
        var debt_week_string = debt_week_object.toISOString()
        console.log(debt_week_string)
    } catch(e) {
        error("Failed to parse date request.")
        console.log("Error: " + e)
        return;
    }

    var form_obj = { 
            "date" : debt_week_string,
            "target_user": debtTargetUser
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

                success(result.message);
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/debts");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function get_prizes() {

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

                place_prizes(result.prizes)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "admin/prizes");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_prizes(prizesArray) {

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

function add_season() {

    clearResponse();

    var now = new Date;

    var season_start = document.getElementById("season-start").value;
    var season_end = document.getElementById("season-end").value;
    var season_start_string = "";
    var season_end_string = "";
    var season_join_anytime = document.getElementById("join_anytime").checked;

    try {
        var season_prize_select = document.getElementById("season-prize");
        var season_prize = season_prize_select[season_prize_select.selectedIndex].value;
    } catch(e) {
        console.log("Failed to parse prize. Error: " + e)
        error("Failed to parse prize.")
        return
    }

    var season_name = document.getElementById("season-name").value;
    var season_desc = document.getElementById("season-desc").value;
    var season_sickleave = parseInt(document.getElementById("season-sickleave").value);

    try {

        var season_start_object = new Date(season_start);
        season_start_string = season_start_object.toISOString()
        var season_end_object = new Date(season_end);
        season_end_string = season_end_object.toISOString()

        if(season_start_object.getDay() != 1) {
            console.log("Day: " + season_start_object.getDay())
            error("Season start must be a monday.");
            return;
        }

        if(season_end_object.getDay() != 0) {
            error("Season end must be a sunday.");
            return;
        }

        if(season_end_object < season_start_object) {
            error("Season start must be before season end.");
            return;
        }

        if(season_start_object < now) {
            error("Season start must later than now.");
            return;
        }
        
    } catch(e) {
        error("Failed to parse date object.")
        console.log("Error: " + e)
        return;
    }

    var form_obj = { 
        "start" : season_start_string,
        "end" : season_end_string,
        "name" : season_name,
        "description" : season_desc,
        "prize_id" : season_prize,
        "sickleave" : season_sickleave,
        "timezone" : Intl.DateTimeFormat().resolvedOptions().timeZone,
        "join_anytime": season_join_anytime
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

                success(result.message);

                document.getElementById("season-name").value = "";
                document.getElementById("season-desc").value = "";
                document.getElementById("season-sickleave").value = 0;
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/seasons");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function add_prize() {

    var prize_name = document.getElementById("prize-name").value;
    var prize_quantity = parseInt(document.getElementById("prize-quantity").value);

    var form_obj = { 
        "name" : prize_name,
        "quantity" : prize_quantity
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

                success(result.message);

                document.getElementById("prize-name").value = "";
                document.getElementById("prize-quantity").value = "";

                get_prizes();
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/prizes");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function addUserToDebtSelection(user) {
    select = document.getElementById('selectUser');
    var opt = document.createElement('option');
    opt.value = user.id;
    opt.innerHTML = user.first_name + " " + user.last_name;
    select.appendChild(opt);
}