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
                            Here is your participation data from Treningheten.
                        </div>

                    </div>

                    <div class="module">

                        <div id="goals-title" class="title" style="display: none;">
                            Goals:
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
        get_seaons();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_seaons(){

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

                get_goals(seasons);

            }

        } else {
            info("Loading goals...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/season");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function get_goals(seasonsArray){

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
                goals = result.goals;

                console.log(goals);

                console.log("Placing intial goals: ")
                place_goals(goals, seasonsArray);

            }

        } else {
            info("Loading goals...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/goal");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_goals(goals_array, seasonsArray) {

    if(goals_array.length == 0) {
        info("No goals found.");
        error_splash_image();
        return;
    } else {
        document.getElementById("goals-title").style.display = "inline-block"
    }

    var html = ''

    for(var i = 0; i < goals_array.length; i++) {

        var seasonIndex = 0;
        var seasonFound = false;
        for(var j = 0; j < seasonsArray.length; j++) {
            if(goals_array[i].season == seasonsArray[j].ID) {
                seasonIndex = j;
                seasonFound = true;
                break
            }
        }

        if(!seasonFound) {
            continue;
        }

        if(goals_array[i].exercise_interval) {
            var compete_string = "Yes"
        } else {
            var compete_string = "No"
        }

        // parse date object
        try {
            var date = new Date(Date.parse(goals_array[i].CreatedAt));
            var date_string = GetDateString(date)
        } catch {
            var date_string = "Error"
        }

        html += '<div class="goal-object-wrapper">'

            html += '<div class="goal-object">'
            
                html += '<div id="season-title" class="season-title">';
                html += seasonsArray[seasonIndex].name
                html += '</div>';

                html += '<div id="goal-exercise" class="goal-exercise">';
                html += 'Exercise goal: ' + goals_array[i].exercise_interval
                html += '</div>';

                html += '<div id="goal-compete" class="goal-compete">';
                html += 'Competing: ' + compete_string
                html += '</div>';

                html += '<div id="goal-join-date" class="goal-join-date">';
                html += '<img src="assets/calendar.svg" class="btn_logo"></img> Join date: ' + date_string
                html += '</div>';

                html += '<div id="goal-button-expand-' + goals_array[i].ID + '" class="goal-button minimized">';
                    html += '<button type="submit" onclick="get_exercises(' + goals_array[i].ID + ');" id="goal_amount_button" style=""><p2 style="margin: 0 0 0 0.5em;">Expand</p2><img id="goal-button-image-' + goals_array[i].ID + '" src="assets/chevron-right.svg" class="btn_logo color-invert" style="padding: 0; margin: 0 0.5em 0 0;"></button>';
                html += '</div>';

            html += '</div>'

            html += '<div class="goal-leaderboard" id="goal-leaderboard-' + goals_array[i].ID + '">'
            html += '</div>'

        html += '</div>'

    }

    goals_object = document.getElementById("goals-box")
    goals_object.innerHTML = html

}

function get_exercises(goalID){

    button = document.getElementById("goal-button-expand-" + goalID)

    if(button.classList.contains("minimized")) {
        button.classList.remove("minimized")
        button.classList.add("expand")
        document.getElementById("goal-button-image-" + goalID).src = "assets/chevron-down.svg"
    } else {
        button.classList.add("minimized")
        button.classList.remove("expand")
        document.getElementById("goal-leaderboard-" + goalID).innerHTML = ""
        document.getElementById("goal-button-image-" + goalID).src = "assets/chevron-right.svg"
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
                exercise = result.exercise;

                console.log(exercise)

                console.log("Placing exercises: ")
                place_exercises(exercise);
                
            }

        } else {
            //info("Loading exercises...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercise/" + goalID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_exercises(exercise_array) {

    clearResponse();
    
    for(var i = 0; i < exercise_array.length; i++) {

        if(exercise_array[i].exercise_interval == 0 && exercise_array[i].note == "") {
            continue;
        } else if (exercise_array[i].note == "") {
            note = "None"
        } else {
            note = exercise_array[i].note
        }

        // parse date object
        try {
            var date = new Date(Date.parse(exercise_array[i].date));
            var date_string = GetDateString(date)
        } catch {
            var date_string = "Error"
        }

        var html = `

            <div class="exercise-object">

                <div class="exercise-date">
                    <img src="assets/calendar.svg" class="btn_logo"></img> ${date_string}
                </div>

                <div class="exercise-details" id="">
                    <div class="exercise-exercise-number">
                        Exercise amount: ${exercise_array[i].exercise_interval}
                    </div>

                    <div class="exercise-notes">
                        Notes: ${note}
                    </div>
                    
                </div>

            </div>
        `;

        document.getElementById("goal-leaderboard-" + exercise_array[i].goal).innerHTML += html

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
    xhttp.open("post", api_url + "auth/user/get/" + userID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImageForUserOnLeaderboard(imageBase64, userID, seasonID) {

    document.getElementById("member-img-" + seasonID + "-" + userID).src = imageBase64

}