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
        
        get_goals();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_goals(){

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
                place_goals(goals);

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

function place_goals(goals_array) {

    if(goals_array.length == 0) {
        info("No goals found.");
        error_splash_image();
        return;
    } else {
        document.getElementById("goals-title").style.display = "inline-block"
    }

    var html = ''

    for(var i = 0; i < goals_array.length; i++) {

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
                html += 'Season ID: ' + goals_array[i].season
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

                html += '<div id="goal-exercises" class="goal-exercises">';
                html += 'Exercises: '
                html += '</div>';

            html += '</div>'

            html += '<div class="goal-leaderboard" id="goal-leaderboard-' + goals_array[i].ID + '">'
            html += '</div>'

        html += '</div>'

    }

    goals_object = document.getElementById("goals-box")
    goals_object.innerHTML = html

    get_exercises();

}

function get_exercises(){

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
            info("Loading exercises...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercise/");
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