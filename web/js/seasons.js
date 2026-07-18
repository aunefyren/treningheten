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

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="text-body u-text-center">
                            Here are all the seasons of {{.appName}}.
                        </div>

                    </div>

                    <div class="module">

                        <div id="seasons-title" class="title" style="display: none;">
                            Seasons:
                        </div>

                        <div id="seasons-box" class="seasons">
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'The archive.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        
        get_seasons();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_seasons(){
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
                seasons = result.seasons;

                console.log(seasons);

                console.log("Placing initial seasons: ")
                place_seasons(seasons);
            }

        } else {
            info("Loading seasons...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_seasons(seasons_array) {

    if(seasons_array.length == 0) {
        info("No seasons found.");
        error_splash_image();
        return;
    } else {
        document.getElementById("seasons-title").style.display = "inline-block"
    }

    var html = ''

    for(var i = 0; i < seasons_array.length; i++) {

        // parse date object
        try {
            var date = new Date(Date.parse(seasons_array[i].start));
            var date_string = GetDateString(date)

            var date2 = new Date(Date.parse(seasons_array[i].end));
            var date_string2 = GetDateString(date2)
        } catch {
            var date_string = "Error"
            var date_string2 = "Error"
        }

        html += '<div class="season-object-wrapper">'

            html += '<div class="season-object">'
            
                html += '<div id="season-title" class="season-title">';
                html += seasons_array[i].name
                html += '</div>';

                html += '<div id="season-body" class="season-body">';
                html += seasons_array[i].description
                html += '</div>';

                html += '<div id="season-date" class="season-date">';
                html += '<img src="assets/calendar.svg" class="btn_logo"></img>';
                html += date_string
                html += '</div>';

                html += '<div id="season-date2" class="season-date">';
                html += '<img src="assets/calendar.svg" class="btn_logo"></img>';
                html += date_string2
                html += '</div>';

                html += '<div id="season-button-expand-' + seasons_array[i].id + '" class="season-button minimized">';
                    html += `<button type="button" onclick="get_leaderboard('${seasons_array[i].id}');" class="btn btn--sm">Expand<img id="season-button-image-${seasons_array[i].id}" src="assets/chevron-right.svg" class="color-invert"></button>`;
                html += '</div>';

            html += '</div>'

            html += '<div class="season-leaderboard" id="season-leaderboard-' + seasons_array[i].id + '">'
            html += '</div>'

        html += '</div>'

    }

    seasons_object = document.getElementById("seasons-box")
    seasons_object.innerHTML = html

}

function get_leaderboard(season_id){

    button = document.getElementById("season-button-expand-" + season_id)

    if(button.classList.contains("minimized")) {
        button.classList.remove("minimized")
        button.classList.add("expand")
        document.getElementById("season-button-image-" + season_id).src = "assets/chevron-down.svg"
        // Show a loading spinner while the season's leaderboard is fetched; it is replaced
        // when place_leaderboard renders into the same container.
        document.getElementById("season-leaderboard-" + season_id).innerHTML = `
            <div class="exercise-loading"><div class="trh-spinner"></div></div>
        `;
    } else {
        button.classList.add("minimized")
        button.classList.remove("expand")
        document.getElementById("season-leaderboard-" + season_id).innerHTML = ""
        document.getElementById("season-button-image-" + season_id).src = "assets/chevron-right.svg"
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
                document.getElementById("season-leaderboard-" + season_id).innerHTML = "";

            } else {
                season = result.season;

                for(var i = 0; i < season.goals.length; i++) {
                    userList[season.goals[i].user.id] = season.goals[i].user
                }

                get_leaderboard_two(season_id);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/" + season_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function get_leaderboard_two(season_id) {
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
                document.getElementById("season-leaderboard-" + season_id).innerHTML = "";

            } else {

                //clearResponse();
                weeks = result.leaderboard;

                console.log("Placing weeks: ")
                place_leaderboard(weeks.past_weeks, season_id);
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/" + season_id + "/leaderboard");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_leaderboard(weeks_array, season_id) {

    var html = ``;
    var members = ``;
    var memberPhotoIDArray = [];
    
    if(weeks_array.length == 0) {
        html = `
            <div id="" class="leaderboard-weeks">
                <p id="" class="u-m-2 u-text-center">Season has not started yet.</p>
            </div>
        `;
    } else {
        for(var i = 0; i < weeks_array.length; i++) {
            var week_html = `
                <div class="leaderboard-week-two" id="">
                    <div class="leaderboard-week-number">
                        Week ` + weeks_array[i].week_number + ` (` + weeks_array[i].week_year + `)
                    </div>
                    <div class="leaderboard-week-results">
            `;

            var results_html = "";

            // Sort users
            weeks_array[i].users = weeks_array[i].users.sort((a,b) => b.user_id.localeCompare(a.user_id));

            for(var j = 0; j < weeks_array[i].users.length; j++) {
                var completion = "❌"

                if(!weeks_array[i].users[j].full_week_participation && weeks_array[i].users[j].week_completion < 1) {
                    completion = "🕙"
                } else if(weeks_array[i].users[j].sick_leave) {
                    completion = "🤢"
                } else if(weeks_array[i].users[j].week_completion >= 1) {
                    completion = "✅"
                }

                var onclick_command_str = "return;"
                var clickable_str = ""
                if(weeks_array[i].users[j].debt !== null && weeks_array[i].users[j].debt.winner !== null) {
                    onclick_command_str = "location.replace('/wheel?debt_id=" + weeks_array[i].users[j].debt.id + "'); "
                    clickable_str = "clickable grey-underline"
                    completion += "🎡"
                }

                var result_html = `
                <div class="leaderboard-week-result" id="">
                    <div class="leaderboard-week-result-user clickable grey-underline" onclick="location.href='/users/${weeks_array[i].users[j].user_id}'">
                        ` + userList[weeks_array[i].users[j].user_id].first_name + `
                    </div>
                    <div class="leaderboard-week-result-exercise ` + clickable_str  + `" onclick="` + onclick_command_str  + `">
                        ` + completion  + `
                    </div>
                </div>
                `;
                results_html += result_html;

                var userFound = false;
                for(var l = 0; l < memberPhotoIDArray.length; l++) {
                    if(memberPhotoIDArray[l] == weeks_array[i].users[j].user_id) {
                        userFound = true;
                        break;
                    }
                }

                if(!userFound) {
                    var joined_image = `
                    <div class="leaderboard-week-member" id="member-${season_id}-${weeks_array[i].users[j].user_id}" title="${userList[weeks_array[i].users[j].user_id].first_name}" onclick="location.href='/users/${weeks_array[i].users[j].user_id}'">
                        <div class="leaderboard-week-member-image">
                            <img class="u-fill leaderboard-week-member-image-img" src="${profileImageURL(weeks_array[i].users[j].user_id, true)}" onerror="${IMAGE_FALLBACK_ONERROR}">
                        </div>
                        ${userList[weeks_array[i].users[j].user_id].first_name}
                    </div>
                    `;
                    members += joined_image
                    memberPhotoIDArray.push(weeks_array[i].users[j].user_id)
                }

            }

            week_html += results_html + `</div></div>`;

            html += week_html

        }
        
    }

    if(members == "") {
        members = "None."
    }

    var members_html = `
    <div class="leaderboard-week-members-wrapper">
        Participants:
        <div class="leaderboard-week-members" id="">
            ${members}
        </div>
    </div>
    `;

    document.getElementById("season-leaderboard-" + season_id).innerHTML = members_html + html

    return

}