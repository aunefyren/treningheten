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
                    
                        <div class="text-body" style="text-align: center;">
                            Here are all the seasons of Treningheten.
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

                <div class="module color-invert">
                    <hr>
                </div>

                <div class="module" id="stats_module">

                    <div class="title">
                        Season statistics
                    </div>

                    <div class="form-group">
                        <select id='select_season' class='form-control' onchange="choose_season()">
                            <option value="null">Choose season</option>
                        </select>
                    </div>

                    <div>

                        <div class="module" id="loading-dumbell" style="display: none;">
                            <img src="/assets/images/barbell.gif">
                        </div>

                        <div id="season-statistics-element-wrapper-div" class="season-statistics-element-wrapper-div">
                        </div>

                        <div id="chart-canvas-div" style="max-width: 40em; margin: 1em auto; padding: 0 0.5em;">
                            <canvas id="myChart" style="max-width: 100%; width: 1000px; display:none;"></canvas>
                        </div>

                        <div id="chart-canvas-div-two" style="max-width: 40em; margin: 1em auto; padding: 0 0.5em;">
                            <canvas id="myChartTwo" style="max-width: 100%; width: 1000px; display:none;"></canvas>
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

                console.log("Placing intial seasons: ")
                place_seasons(seasons);
                place_seasons_input(seasons);

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
                    html += `<button type="submit" onclick="get_leaderboard('${seasons_array[i].id}');" id="goal_amount_button" style=""><p2 style="margin: 0 0 0 0.5em;">Expand</p2><img id="season-button-image-${seasons_array[i].id}" src="assets/chevron-right.svg" class="btn_logo color-invert" style="padding: 0; margin: 0 0.5em 0 0;"></button>`;
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

            } else {

                //clearResponse();
                weeks = result.leaderboard;

                console.log("Placing weeks: ")
                place_leaderboard(weeks, season_id);
                
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
                <p id="" style="margin: 0.5em; text-align: center;">Season has not started yet.</p>
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
            for(var j = 0; j < weeks_array[i].users.length; j++) {
                var completion = "❌"
                if(weeks_array[i].users[j].sickleave) {
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
                    <div class="leaderboard-week-result-user clickable grey-underline" style="" onclick="location.href='/users/${weeks_array[i].users[j].user.id}'">
                        ` + weeks_array[i].users[j].user.first_name + `
                    </div>
                    <div class="leaderboard-week-result-exercise ` + clickable_str  + `" onclick="` + onclick_command_str  + `">
                        ` + completion  + `
                    </div>
                </div>
                `;
                results_html += result_html;

                var userFound = false;
                for(var l = 0; l < memberPhotoIDArray.length; l++) {
                    if(memberPhotoIDArray[l] == weeks_array[i].users[j].user.id) {
                        userFound = true;
                        break;
                    }
                }

                if(!userFound) {
                    var joined_image = `
                    <div class="leaderboard-week-member" style="cursor:hover;" id="member-${season_id}-${weeks_array[i].users[j].user.id}" title="${weeks_array[i].users[j].user.first_name} ${weeks_array[i].users[j].user.last_name}" onclick="location.href='/users/${weeks_array[i].users[j].user.id}'">
                        <div class="leaderboard-week-member-image">
                            <img style="width: 100%; height: 100%;" class="leaderboard-week-member-image-img" id="member-img-${season_id}-${weeks_array[i].users[j].user.id}" src="/assets/images/barbell.gif">
                        </div>
                        ${weeks_array[i].users[j].user.first_name}
                    </div>
                    `;
                    members += joined_image
                    memberPhotoIDArray.push(weeks_array[i].users[j].user.id)
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

    for(var i = 0; i < memberPhotoIDArray.length; i++) {
        GetProfileImageForUserOnLeaderboard(memberPhotoIDArray[i], season_id)
    }

    return

}

function GetProfileImageForUserOnLeaderboard(userID, seasonID) {

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

                PlaceProfileImageForUserOnLeaderboard(result.image, userID, seasonID)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImageForUserOnLeaderboard(imageBase64, userID, seasonID) {

    document.getElementById("member-img-" + seasonID + "-" + userID).src = imageBase64

}

function place_seasons_input(seasons_array) {

    var select_season = document.getElementById("select_season");
    var seasons = []
    var now = new Date();

    for(var i = 0; i < seasons_array.length; i++) {
        var user_found = false;
        for(var j = 0; j < seasons_array[i].goals.length; j++) {
            if(seasons_array[i].goals[j].user.id == user_id) {
                user_found = true;
                break
            }
        }

        var futureSeason = true;
        try {
            var season_start_object = new Date(seasons_array[i].start);
            if(now > season_start_object) {
                futureSeason = false;
            }
        } catch(e) {
            console.log("Failed to parse season start. Error: " + e)
        }

        if(user_found && !futureSeason) {
            seasons.push(seasons_array[i])
        }
    }

    for(var i = 0; i < seasons.length; i++) {
        
        var option = document.createElement("option");
        option.text = seasons[i].name
        option.value = seasons[i].id
        select_season.add(option); 

    }

}

function choose_season() {

    var select_season = document.getElementById("select_season");

    // Show loading gif
    document.getElementById("loading-dumbell").style.display = "inline-block";

    // Purge data
    canvas_div = document.getElementById("chart-canvas-div");
    canvas_div.innerHTML = "";
    canvas_div.innerHTML = '<canvas id="myChart" style="max-width: 100%; width: 1000px; display:none;"></canvas>';

    canvas_div_two = document.getElementById("chart-canvas-div-two");
    canvas_div_two.innerHTML = "";
    canvas_div_two.innerHTML = '<canvas id="myChartTwo" style="max-width: 100%; width: 1000px; display:none;"></canvas>';

    document.getElementById("season-statistics-element-wrapper-div").innerHTML = "";

    if(select_season.value == null || select_season.value == 0 || select_season.value == "null") {

        // Show loading gif
        document.getElementById("loading-dumbell").style.display = "none";

        var myChartElement = document.getElementById("myChart");
        myChartElement.style.display = "none"

    } else {
        
        get_season_leaderboard(select_season.value)

    }

}

function get_season_leaderboard(seasonID){

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

                place_statistics(result.leaderboard, result.weekdays, result.wheel_statistics);
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/" + seasonID + "/leaderboard-personal");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_statistics(leaderboard_array, weekday_array, wheel_statistics) {

    var myChartElement = document.getElementById("myChart");
    myChartElement.style.display = "inline-block"

    var myChartElement = document.getElementById("myChartTwo");
    myChartElement.style.display = "inline-block"

    leaderboard_array = leaderboard_array.reverse();

    var xValues = [];
    var yValues = [];
    var goals = [];
    var pointBackgroundColorArray = [];
    var borderColorArray = [];
    var longest_streak = 0;
    var highest_week = 0;
    var exercise_amount = 0;
    var sickleave_amount = 0;
    var goal = 0;
    var week_count = 0;
    var complete_weeks = 0;
    var incomplete_weeks = 0;
    var wheels_won = wheel_statistics.wheels_won;
    var wheels_lost = wheel_statistics.wheel_spins;

    // Look through array of data
    for (var i = 0; i < leaderboard_array.length; i++) {

        xValues.push("" + leaderboard_array[i].week_number + " (" + leaderboard_array[i].week_year + ")");

        var exercise = leaderboard_array[i].user.week_completion_interval
        var sickleave = leaderboard_array[i].user.sickleave
        goal = leaderboard_array[i].user.exercise_goal
        var streak = leaderboard_array[i].user.current_streak
        exercise_amount = exercise_amount + exercise

        if(streak > longest_streak) {
            longest_streak = streak;
        }

        if(exercise > highest_week) {
            highest_week = exercise;
        }
        
        if(sickleave) {
            pointBackgroundColorArray.push("rgba(215, 20, 20, 1)")
            borderColorArray.push("rgba(215, 20, 20, 1)")
            sickleave_amount = sickleave_amount + 1
        } else {
            pointBackgroundColorArray.push("rgba(119,141,169,1)")
            borderColorArray.push("rgba(119,141,169,1)")
        }

        yValues.push(eval(exercise));
        goals.push(eval(goal));

        if(exercise >= goal) {
            complete_weeks = complete_weeks + 1
        } else {
            incomplete_weeks = incomplete_weeks + 1
        }

        week_count = week_count + 1
           
    }

    console.log("goal: " + goal)
    console.log("weeks: " + week_count)
    console.log("complete weeks: " + complete_weeks)

    week_completion_percentage = Math.floor((complete_weeks / week_count) * 100)

    const lineChart = new Chart("myChart", {
        type: "line",
        data: {
            labels: xValues,
            datasets: [
                {
                    fill: true,
                    borderColor: borderColorArray,
                    pointBackgroundColor: pointBackgroundColorArray,
                    backgroundColor: "rgba(119,141,169,0.5)",
                    responsive: true,
                    data: yValues,
                    tension: 0,
                    label: "Exercise count",
                },
                {
                    fill: true,
                    borderColor: "rgba(119,141,169,0.25)",
                    responsive: false,
                    data: goals,
                    tension: 0,
                    label: "Goal",
                }
            ]
        },    
        options: {
            legend: {display: false},
            title: {
                display: true,
                text: "Weekly exercise graph",
                fontSize: 16
            },
            scales: {
                yAxes: [
                    {
                        beginAtZero: true,
                        min: 0,
                        ticks: {
                            beginAtZero: true,
                            precision: 0
                        }
                    }
                ]
            }
        }
    });


    var xValues2 = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"]
    var yValues2 = [weekday_array.monday, weekday_array.tuesday, weekday_array.wednesday, weekday_array.thursday, weekday_array.friday, weekday_array.saturday, weekday_array.sunday]

    const lineChartTwo = new Chart("myChartTwo", {
        type: "line",
        data: {
            labels: xValues2,
            datasets: [
                {
                    fill: true,
                    borderColor: "rgba(119,141,169,1)",
                    pointBackgroundColor: "rgba(119,141,169,1)",
                    backgroundColor: "rgba(119,141,169,0.5)",
                    responsive: true,
                    data: yValues2,
                    tension: 0,
                    label: "Exercise count",
                }
            ]
        },    
        options: {
            legend: {display: false},
            title: {
                display: true,
                text: "Weekday exercise graph",
                fontSize: 16
            },
            scales: {
                yAxes: [
                    {
                        beginAtZero: true,
                        min: 0,
                        ticks: {
                            beginAtZero: true,
                            precision: 0
                        }
                    }
                ]
            }
        }
    });

    if(goal > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Weekly exercise goal: ${goal}🏆
            </div>
        `;
    }

    if(week_completion_percentage > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Average weekly goal completion: ${week_completion_percentage}%📊
            </div>
        `;
    }

    if(longest_streak > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Longest week streak: ${longest_streak}🔥
            </div>
        `;
    }

    if(highest_week > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Most exercise in a week: ${highest_week}🏋️
            </div>
        `;
    }

    if(exercise_amount > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                All exercise combined: ${exercise_amount}💰
            </div>
        `;
    }

    if(sickleave_amount > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Weeks of sick leave: ${sickleave_amount}🤢
            </div>
        `;
    }

    if(wheels_lost > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Wheels spun: ${wheels_lost}🎡
            </div>
        `;
    }

    if(wheels_won > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Wheels won: ${wheels_won}⭐
            </div>
        `;
    }
    
    // Remove loading gif
    document.getElementById("loading-dumbell").style.display = "none";

}
