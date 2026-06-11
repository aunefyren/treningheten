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
                            Here is your participation data from {{.appName}}.
                        </div>

                    </div>

                    <div class="module">

                        <div id="goals-title" class="title" style="display: none;">
                            Years:
                        </div>

                        <div id="goals-box" class="goals">
                        </div>
                        
                    </div>

                    <div class="module">

                        <div id="exercises-title" class="title" style="display: none;">
                            Exercises:
                        </div>

                        <div id="exercises-box" class="exercises-box">
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'I\'m sure your notes are really useful.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        getAllExerciseDays();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function getAllExerciseDays(){

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

                // The years endpoint returns the distinct years directly, so the page
                // no longer enriches the whole exercise history just to list years.
                yearArray = result.years || [];

                placeExerciseYears(yearArray)
            }

        } else {
            info("Loading years...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/exercise-days/years");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeExerciseYears(yearArray) {

    if(yearArray.length == 0) {
        info("No exercise found.");
        error_splash_image();
        return;
    } else {
        document.getElementById("goals-title").style.display = "inline-block"
    }

    var html = ''

    for(var i = 0; i < yearArray.length; i++) {

        html += '<div class="goal-object-wrapper">'

            html += '<div class="goal-object">'
            
                html += '<div id="season-title" class="season-title">';
                html += yearArray[i]
                html += '</div>';

                html += '<div id="goal-button-expand-' + yearArray[i] + '" class="goal-button minimized">';
                    html += `<button type="submit" onclick="get_exercises('${yearArray[i]}');" id="goal_amount_button" style=""><p2 style="margin: 0 0 0 0.5em;">Expand</p2><img id="goal-button-image-${yearArray[i]}" src="assets/chevron-right.svg" class="btn_logo color-invert" style="padding: 0; margin: 0 0.5em 0 0;"></button>`;
                html += '</div>';

            html += '</div>'

            html += '<div class="goal-leaderboard" id="goal-leaderboard-' + yearArray[i] + '">'
            html += '</div>'

        html += '</div>'

    }

    goals_object = document.getElementById("goals-box")
    goals_object.innerHTML = html

}

function get_exercises(year){

    button = document.getElementById("goal-button-expand-" + year)

    if(button.classList.contains("minimized")) {
        button.classList.remove("minimized")
        button.classList.add("expand")
        document.getElementById("goal-button-image-" + year).src = "assets/chevron-down.svg"
    } else {
        button.classList.add("minimized")
        button.classList.remove("expand")
        document.getElementById("goal-leaderboard-" + year).innerHTML = ""
        document.getElementById("goal-leaderboard-" + year).style.margin = "0"
        document.getElementById("goal-button-image-" + year).src = "assets/chevron-right.svg"
        return
    }

    // Show a loading spinner inside the year wrapper while the exercises load.
    var leaderboard = document.getElementById("goal-leaderboard-" + year)
    leaderboard.innerHTML = '<div class="exercise-loading"><div class="trh-spinner"></div></div>'
    leaderboard.style.margin = "1em 0"

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

                document.getElementById("goal-leaderboard-" + year).innerHTML = ""
                error(result.error);

            } else {

                place_exercises(result.exercise, year);

            }

        } else {
            //info("Loading exercises...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/exercise-days/summary?year=" + year);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_exercises(exercise_array, year) {

    clearResponse();

    var leaderboard = document.getElementById("goal-leaderboard-" + year)
    leaderboard.style.margin = "1em 0"

    // Group days by week, keeping the source (date-descending) order so weeks appear
    // newest-first. Each day's HTML is built once and the whole block is assigned in a
    // single innerHTML write, instead of repeatedly appending (which re-parses the
    // growing markup on every iteration).
    var weekOrder = [];
    var weekDays = {};

    for(var i = 0; i < exercise_array.length; i++) {

        var day = exercise_array[i];

        if(day.exercise_interval == 0 && day.note == "") {
            continue;
        }

        var note_html = "";
        var note_text = "";
        if(day.note != "") {
            note_html = `
                <div class="exercise-notes">
                    Notes
                </div>
            `;
            note_text = `
            <div class="overlay">
                <div class="text-exercise">${HTMLAddNewLines(day.note)}</div>
            </div>
            `;
        }

        // parse date object
        try {
            var date = new Date(Date.parse(day.date));
            var dateFullString = GetDateString(date, false);
            var date_string = GetDayOfTheWeek(date)
            var dateStringDetailed = GetDateString(date, false)
            var week = date.getWeek(1);
        } catch {
            var dateFullString = "Error"
            var date_string = "Error"
            var dateStringDetailed = "Error"
            var week = "Error"
        }

        var dayHTML = `
            <div class="exercise-object clickable" title="${dateStringDetailed}" onclick="exerciseRedirect('${day.id}')">

                <div class="exercise-base">

                    <div class="exercise-date">
                        <img style="width: 100%;" src="assets/calendar.svg" class="btn_logo"></img>
                        <div class="exercise-date-string">${dateFullString}</div>
                        <div class="exercise-date-string"><b>${date_string}</b></div>
                    </div>

                    <div class="exercise-details" id="">
                        <div class="exercise-exercise-number">
                            Exercise amount: <b>${day.exercise_interval}</b>
                        </div>

                        ${note_html}

                    </div>

                </div>

                ${note_text}

            </div>
        `;

        if(!(week in weekDays)) {
            weekDays[week] = [];
            weekOrder.push(week);
        }
        weekDays[week].push(dayHTML);

    }

    if(weekOrder.length == 0) {
        leaderboard.innerHTML = `
            <div style="margin: 1em 0">
                None...
            </div>
        `;
        return;
    }

    var html = '';
    for(var w = 0; w < weekOrder.length; w++) {
        var week = weekOrder[w];
        html += `
            <hr style="margin: 0.25em;">
            <div class="exercise-week">
                <b>Week: ${week}</b>
            </div>
            <div class="exercises-group">
        `;
        // Days were collected newest-first; reverse so the week reads oldest-to-newest
        // (matching the previous prepend behaviour).
        html += weekDays[week].slice().reverse().join('');
        html += `</div>`;
    }

    leaderboard.innerHTML = html;

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

function exerciseRedirect(exerciseDayID) {
    window.location = '/exercises/' + exerciseDayID
}