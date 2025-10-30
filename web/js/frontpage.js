function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        if(login_data.error === "You must verify your account.") {
            load_verify_account();
            return;
        }

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
        var admin = false;
    }

    console.log(user_id)

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module" id="top-module">
                    
                        <div class="title">
                            Treningheten
                        </div>

                        <div class="text-body" id="front-page-text" style="text-align: center;">
       
                        </div>

                        <div id="log-in-button" style="margin-top: 2em; display: none; width: 10em;">
                            <button id="update-button" type="submit" href="#" onclick="window.location = '/login';">Log in</button>
                        </div>

                        <div class="module" id="barbell-gif" style="display: none;">
                            <img src="/assets/images/barbell.gif">
                        </div>

                        <button type="submit" onclick="" id="add-exercise-button" style="display: none; margin-bottom: 0em; width: 12em;"><img src="assets/plus.svg" class="btn_logo color-invert"><p2>Start new workout</p2></button>

                    </div>

                    <div class="module" id="ongoingseason" style="display: none;">

                        <div class="modules">

                            <div id="exercises" class="exercises">

                                <div class="week_days" id='calendar'>

                                    <div id="week-progress-bar-wrapper" class="week-progress-bar-wrapper" style="width: 20em;">
                                        <div id="week-progress-bar" class="week-progress-bar" style="">
                                            <div class="calender_status unselectable" id="calender_status">
                                                <a id="workout_this_week">...</a>
                                                /
                                                <a id="goal_this_week">...</a>
                                                this week
                                            </div>
                                        </div>
                                    </div>

                                    <div class="form-group" style="" id="day_1_group">
                                        <div class="day-check">
                                            <label style="margin: 0;" for="day_1_check" title="Have you been working out?">Monday</label>
                                            <div class="number-box" id="day_1_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_1_buttons">
                                                <img src="assets/minus.svg" class="small-button-icon clickable" onclick="DecreaseNumberInput('day_1_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon clickable" onclick="IncreaseNumberInput('day_1_check', 0, 3);">
                                                <img src="assets/edit-3.svg" style="padding: 0.40em;" class="small-button-icon clickable" onclick="EditExercise(1);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_1_note" name="day_1_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_1_date">
                                        <input type="hidden" value="" id="day_1_id">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_2_group">
                                        <div class="day-check">
                                            <label style="margin: 0;" for="day_2_check" title="Have you been working out?">Tuesday</label>
                                            <div class="number-box" id="day_2_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_2_buttons">
                                                <img src="assets/minus.svg" class="small-button-icon clickable" onclick="DecreaseNumberInput('day_2_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon clickable" onclick="IncreaseNumberInput('day_2_check', 0, 3);">
                                                <img src="assets/edit-3.svg" style="padding: 0.40em;" class="small-button-icon clickable" onclick="EditExercise(2);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_2_note" name="day_2_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_2_date">
                                        <input type="hidden" value="" id="day_2_id">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_3_group">
                                        <div class="day-check">
                                            <label style="margin: 0;" for="day_3_check" title="Have you been working out?">Wednesday</label>
                                            <div class="number-box" id="day_3_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_3_buttons">
                                                <img src="assets/minus.svg" class="small-button-icon clickable" onclick="DecreaseNumberInput('day_3_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon clickable" onclick="IncreaseNumberInput('day_3_check', 0, 3);">
                                                <img src="assets/edit-3.svg" style="padding: 0.40em;" class="small-button-icon clickable" onclick="EditExercise(3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_3_note" name="day_3_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_3_date">
                                        <input type="hidden" value="" id="day_3_id">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_4_group">
                                        <div class="day-check">
                                            <label style="margin: 0;" for="day_4_check" title="Have you been working out?">Thursday</label>
                                            <div class="number-box" id="day_4_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_4_buttons">
                                                <img src="assets/minus.svg" class="small-button-icon clickable" onclick="DecreaseNumberInput('day_4_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon clickable" onclick="IncreaseNumberInput('day_4_check', 0, 3);">
                                                <img src="assets/edit-3.svg" style="padding: 0.40em;" class="small-button-icon clickable" onclick="EditExercise(4);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_4_note" name="day_4_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_4_date">
                                        <input type="hidden" value="" id="day_4_id">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_5_group">
                                        <div class="day-check">
                                            <label style="margin: 0;" for="day_5_check" title="Have you been working out?">Friday</label>
                                            <div class="number-box" id="day_5_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_5_buttons">
                                                <img src="assets/minus.svg" class="small-button-icon clickable" onclick="DecreaseNumberInput('day_5_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon clickable" onclick="IncreaseNumberInput('day_5_check', 0, 3);">
                                                <img src="assets/edit-3.svg" style="padding: 0.40em;" class="small-button-icon clickable" onclick="EditExercise(5);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_5_note" name="day_5_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_5_date">
                                        <input type="hidden" value="" id="day_5_id">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_6_group">
                                        <div class="day-check">
                                            <label style="margin: 0;" for="day_6_check" title="Have you been working out?">Saturday</label>
                                            <div class="number-box" id="day_6_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_6_buttons">
                                                <img src="assets/minus.svg" class="small-button-icon clickable" onclick="DecreaseNumberInput('day_6_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon clickable" onclick="IncreaseNumberInput('day_6_check', 0, 3);">
                                                <img src="assets/edit-3.svg" style="padding: 0.40em;" class="small-button-icon clickable" onclick="EditExercise(6);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_6_note" name="day_6_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_6_date">
                                        <input type="hidden" value="" id="day_6_id">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_7_group">
                                        <div class="day-check">
                                            <label style="margin: 0;" for="day_7_check" title="Have you been working out?">Sunday</label>
                                            <div class="number-box" id="day_7_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_7_buttons">
                                                <img src="assets/minus.svg" class="small-button-icon clickable" onclick="DecreaseNumberInput('day_7_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon clickable" onclick="IncreaseNumberInput('day_7_check', 0, 3);">
                                                <img src="assets/edit-3.svg" style="padding: 0.40em;" class="small-button-icon clickable" onclick="EditExercise(7);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_7_note" name="day_7_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_7_date">
                                        <input type="hidden" value="" id="day_7_id">
                                    </div>

                                    <input type="hidden" value="" id="calendar_user_id">
                                    <input type="hidden" value="" id="calendar_season_id">

                                    <button type="submit" onclick="update_exercises(false, 0);" id="goal_amount_button" style="margin-bottom: 0em; transition: 1s;"><img src="assets/done.svg" class="btn_logo color-invert"><p2>Save</p2></button>

                                    <a style="margin: 0.5em; font-size:0.75em;cursor:pointer;" onclick="use_sickleave();">Use sick leave</i></a>

                                </div>

                            </div>

                            <div class="module-two">

                                <div id="season-module" class="season" style="padding: 0 1em 1em 1em;">

                                    <div id="season-progress-bar-wrapper" class="season-progress-bar-wrapper" style="width: 20em;">
                                        <div id="season-progress-bar" class="season-progress-bar" style="">
                                            <div class="calender_status unselectable" id="season_status">
                                                <a id="weeks_so_far">...</a>
                                                /
                                                <a id="weeks_total">...</a>
                                                weeks
                                            </div>
                                        </div>
                                    </div>

                                    <h3 id="season_title">Loading...</h3>
                                    <p id="season_desc" style="text-align: center;">...</p>

                                    <p id="season_start_title" style="margin-top: 1em;">Season start: <a id="season_start">...</a></p>
                                    <p id="season_end_title" style="">Season end: <a id="season_end">...</a></p>

                                    <p id="week_goal_title" style="margin: 1em 0 0 0;">Weekly goal: <b><a id="week_goal">0</a></b></p>
                                    <p id="goal_sickleave_title" style="">Sick leave left: <b><a id="goal_sickleave">0</a></b></p>

                                    <p id="prize-title" style="margin-top: 1em;">Potential prize:</p>
                                    <div class="prize-wrapper" id="prize-wrapper">
                                        <div id="prize-text" class="prize-text">...</div>
                                    </div>

                                    <hr id="seasonDivider" style="display:none;">

                                    <div id="potentialSeasonsWrapper" style="display:none;">
                                    </div>

                                    <div id="countdownSeasonsWrapper" style="display:none;">
                                    </div>

                                </div>

                                <div id="debt-module" class="debt-module" style="display: none;">

                                    <h3 id="debt-module-title">Prizes</h3>

                                    <div id="debt-module-notifications" class="debt-module-notifications">
                                    </div>

                                </div>

                                <div id="current-week" class="current-week">

                                    <h3 id="current-week-title">Current week</h3>

                                    <div id="current-week-users" class="current-week-users">
                                        ...
                                    </div>


                                </div>

                            </div>

                            <div class="module-two">

                                <div id="activities" class="activities">

                                    <h3 style="margin: 0.5em;">Activities</h3>

                                    <div id="activities-week" class="activities-week">
                                        <p style="margin-bottom: 0.5em; text-align:center;">No public activities yet this week...</p>
                                    </div>

                                </div>
                                

                                <div id="leaderboard" class="leaderboard">

                                    <h3 style="margin: 0.5em;">Previous weeks</h3>

                                    <div id="leaderboard-weeks" class="leaderboard-weeks">
                                        ...
                                    </div>

                                </div>

                            </div>


                        </div>

                    </div>

                    <div class="module" id="unspun-wheel" style="display: none;">

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Welcome to the frontpage!';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        get_season(user_id, true);
        document.getElementById('front-page-text').innerHTML = 'Remember to log your workouts.';
    } else {
        showLoggedOutMenu();
        document.getElementById('front-page-text').innerHTML = 'Log in to use the platform.';
        document.getElementById('log-in-button').style.display = 'inline-block';
    }
}

function get_season(user_id, loadingMessage){

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
            
            /*
            if(result.error == "No active or future seasons found." || result.seasons.length == 0) {
                clearResponse();
                document.getElementById('top-module').innerHTML = `
                <div class="module">
                    <div class="title">
                        Treningheten
                    </div>

                    <div class="picture-box" style="height: 12em; width: 12em; margin: 1em 0;">
                        <img class="max-size" src="/assets/images/bored.svg">
                    </div>
                

                    <div class="text-body" id="front-page-text" style="text-align: center;">
                        <b>There currently is no season of Treningheten planned or ongoing.</b>
                        <br><br>
                        Contact your local Treningheten administrator to plan a new season.
                        <br><br>
                        Meanwhile, feel free to check out the <a href="/seasons">past seasons and your statistics</a>.
                        <br><br>
                        Or perhaps check out <a href="/achievements">your own achievements</a>?
                    </div>

                    <div id="debt-module" class="debt-module" style="display: none; margin-top: 5em;">
                        <h3 id="debt-module-title">Prizes</h3>
                        <div id="debt-module-notifications" class="debt-module-notifications">
                        </div>
                    </div>

                </div>
                `;

                getDebtOverview();

            } else */if(result.error) {

                error(result.error);
                error_splash_image();

            } else {

                clearResponse();

                document.getElementById("ongoingseason").style.display = "flex"
                getDebtOverview();
                
                // If one or more seasons were found
                if(result.seasons.length > 0) {
                    var season = result.seasons[0];
                    var goal = null;

                    var user_found = false;
                    for(var i = 0; i < season.goals.length; i++) {
                        if(season.goals[i].user.id == user_id) {
                            user_found = true
                            goal = season.goals[i]
                            break
                        }
                    }

                    for(var i = 0; i < season.goals.length; i++) {
                        userList[season.goals[i].user.id] = season.goals[i].user
                    }

                    var date_start = new Date(season.start);
                    var now = Date.now();

                    if(user_found && now < date_start) {
                        countdownRedirect()
                    } else if(user_found) {
                        get_calendar(false, user_id, loadingMessage);
                        place_season(season, user_id);
                        get_leaderboard(season, goal, true, false);
                        getActivities(season);
                    }
                } else {
                    get_calendar(false, user_id, loadingMessage);
                    place_season(false, user_id);

                    try {
                        document.getElementById('current-week').outerHTML = ""
                        document.getElementById('leaderboard').outerHTML = ""
                    } catch (e) {
                        console.log('Removing divs threw error: ' + e)
                    }

                    document.getElementById('goal_this_week').innerHTML = "0"
                }
            }

        } else {
            if(loadingMessage) {
                info("Loading season...");
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/get-on-going");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_season(season_object, userID) {
    if(season_object) {
        document.getElementById("calendar_season_id").value = season_object.id
        document.getElementById("season_title").innerHTML = season_object.name
        document.getElementById("season_desc").innerHTML = season_object.description
        document.getElementById("prize-text").innerHTML = season_object.prize.quantity + " " + season_object.prize.name

        try {
            var date_start = new Date(season_object.start);
            var date_end = new Date(season_object.end);
    
            var date_start_string = GetDateString(date_start, true)
            var date_end_string = GetDateString(date_end, true)
        } catch {
            var date_start_string = "Error"
            var date_end_string = "Error"
        }
        
        document.getElementById("season_start").innerHTML = date_start_string
        document.getElementById("season_end").innerHTML = date_end_string
    
        placeSeasonProgress(date_start, date_end);
        getPotentialSeasons();
        getCountdownSeasons(userID);
    } else {
        document.getElementById("season_title").innerHTML = "No active season"
        document.getElementById("season_desc").innerHTML = "You can join or create a season to start competing."
        getPotentialSeasons();
        getCountdownSeasons(userID);

        try {
            document.getElementById("season_start_title").outerHTML = ""
            document.getElementById("season_end_title").outerHTML = ""
            document.getElementById("week_goal_title").outerHTML = ""
            document.getElementById("goal_sickleave_title").outerHTML = ""
            document.getElementById("prize-title").outerHTML = ""
            document.getElementById("prize-wrapper").outerHTML = ""
        } catch(e) {
            console.log('Removing div\'s threw error: ' + e)
        }

        var currentYear = new Date().getFullYear()  // returns the current yea
        var startOfYear = new Date(currentYear + "-01-01")
        var endOfYear = new Date(currentYear + "-12-24")

        placeSeasonProgress(startOfYear, endOfYear);
    }
}

function get_calendar(fireworks, user_id, loadingMessage){

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
                var week = result.week;

                console.log(week);

                console.log("Placing intial week: ")
                place_week(week, fireworks, user_id);

            }

        } else {
            if(loadingMessage) {
                info("Loading week...");
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/exercises/week");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_week(week, fireworks, user_id) {

    if(fireworks) {
        console.log("Triggered fireworks.")
        trigger_fireworks(1);
    }

    // Sum of exercise to decide fireworks
    fireworks_int = 0
    if(week.days && week.days.length > 0) {
        for(var i = 0; i < week.days.length; i++) {
            if(week.days[i].exercise_interval) {
                fireworks_int += week.days[i].exercise_interval
            }
        }
    }

    document.getElementById("calendar_user_id").value = user_id

    document.getElementById("day_1_check").innerHTML = week.days[0].exercise_interval
    document.getElementById("day_1_note").value = HTMLDecode(week.days[0].note)
    document.getElementById("day_1_date").value = week.days[0].date
    document.getElementById("day_1_id").value = week.days[0].id

    document.getElementById("day_2_check").innerHTML = week.days[1].exercise_interval
    document.getElementById("day_2_note").value = HTMLDecode(week.days[1].note)
    document.getElementById("day_2_date").value = week.days[1].date
    document.getElementById("day_2_id").value = week.days[1].id

    document.getElementById("day_3_check").innerHTML = week.days[2].exercise_interval
    document.getElementById("day_3_note").value = HTMLDecode(week.days[2].note)
    document.getElementById("day_3_date").value = week.days[2].date
    document.getElementById("day_3_id").value = week.days[2].id

    document.getElementById("day_4_check").innerHTML = week.days[3].exercise_interval
    document.getElementById("day_4_note").value = HTMLDecode(week.days[3].note)
    document.getElementById("day_4_date").value = week.days[3].date
    document.getElementById("day_4_id").value = week.days[3].id

    document.getElementById("day_5_check").innerHTML = week.days[4].exercise_interval
    document.getElementById("day_5_note").value = HTMLDecode(week.days[4].note)
    document.getElementById("day_5_date").value = week.days[4].date
    document.getElementById("day_5_id").value = week.days[4].id

    document.getElementById("day_6_check").innerHTML = week.days[5].exercise_interval
    document.getElementById("day_6_note").value = HTMLDecode(week.days[5].note)
    document.getElementById("day_6_date").value = week.days[5].date
    document.getElementById("day_6_id").value = week.days[5].id

    document.getElementById("day_7_check").innerHTML = week.days[6].exercise_interval
    document.getElementById("day_7_note").value = HTMLDecode(week.days[6].note)
    document.getElementById("day_7_date").value = week.days[6].date
    document.getElementById("day_7_id").value = week.days[6].id

    // Find day int
    const now = new Date(Date.now());
    var day = now.getDay();
    if(day == 0) {
        day = 7
    }

    // Add class to current day
    document.getElementById("day_" + day + "_check").classList.add("active-day") 

    // Enable workout buttons
    addExerciseButton = document.getElementById("add-exercise-button")
    addExerciseButton.addEventListener("click", function(e) {
        addExercise(week.days[day-1].id)
    }, false);
    addExerciseButton.style.display = 'flex'

    // Place editing icon for exercise
    for(var i = 1; i <= day; i++) {
        document.getElementById("day_" + i + "_buttons").style.display = "flex" 
        console.log("index: " + i)
        let checkObject = document.getElementById(`day_${i}_check`)
        let dayInteger = i;
        checkObject.onclick = function(){EditExercise(dayInteger)};
        checkObject.classList.add('clickable')
    }

    document.getElementById("workout_this_week").innerText  = fireworks_int

    return

}

function update_exercises(go_to_exercise, weekDayInt) {
    var user_id = document.getElementById("calendar_user_id").value

    var day_1_check = document.getElementById("day_1_check").innerHTML
    var day_1_note = document.getElementById("day_1_note").value
    var day_1_date = document.getElementById("day_1_date").value

    var day_2_check = document.getElementById("day_2_check").innerHTML
    var day_2_note = document.getElementById("day_2_note").value
    var day_2_date = document.getElementById("day_2_date").value

    var day_3_check = document.getElementById("day_3_check").innerHTML
    var day_3_note = document.getElementById("day_3_note").value
    var day_3_date = document.getElementById("day_3_date").value

    var day_4_check = document.getElementById("day_4_check").innerHTML
    var day_4_note = document.getElementById("day_4_note").value
    var day_4_date = document.getElementById("day_4_date").value

    var day_5_check = document.getElementById("day_5_check").innerHTML
    var day_5_note = document.getElementById("day_5_note").value
    var day_5_date = document.getElementById("day_5_date").value

    var day_6_check = document.getElementById("day_6_check").innerHTML
    var day_6_note = document.getElementById("day_6_note").value
    var day_6_date = document.getElementById("day_6_date").value

    var day_7_check = document.getElementById("day_7_check").innerHTML
    var day_7_note = document.getElementById("day_7_note").value
    var day_7_date = document.getElementById("day_7_date").value

    var form_obj = {
        "days": [
            {
                "date": day_1_date,
                "note": day_1_note,
                "exercise_interval": Number(day_1_check)
            },
            {
                "date": day_2_date,
                "note": day_2_note,
                "exercise_interval": Number(day_2_check)
            },
            {
                "date": day_3_date,
                "note": day_3_note,
                "exercise_interval": Number(day_3_check)
            },
            {
                "date": day_4_date,
                "note": day_4_note,
                "exercise_interval": Number(day_4_check)
            },
            {
                "date": day_5_date,
                "note": day_5_note,
                "exercise_interval": Number(day_5_check)
            },
            {
                "date": day_6_date,
                "note": day_6_note,
                "exercise_interval": Number(day_6_check)
            },
            {
                "date": day_7_date,
                "note": day_7_note,
                "exercise_interval": Number(day_7_check)
            },
        ],
        "timezone" : Intl.DateTimeFormat().resolvedOptions().timeZone
    };

    var form_data = JSON.stringify(form_obj);

    console.log("Saving new week: ")
    console.log(form_data)

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
                blinkCalendar();

                week = result.week;

                if(go_to_exercise === true) {
                    GoToExercise(week.days[weekDayInt-1].id)
                }

                console.log(week);

                new_fireworks_int = 0;
                new_fireworks = false
                for(var i = 0; i < week.days.length; i++) {
                    new_fireworks_int += week.days[i].exercise_interval
                }

                if (new_fireworks_int > fireworks_int) {
                    new_fireworks = true
                }

                console.log("Placing initial week: ")
                place_week(week, new_fireworks);
                get_season(user_id, false);

                //success(result.message)
            }

        } else {
            //info("Saving week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises/week");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function get_leaderboard(season, goal, refresh, fireworks){
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
                if(result.leaderboard.length < 1) {
                    return;
                }
                
                this_week = result.leaderboard[0];

                if(result.leaderboard.length > 1) {
                    past_weeks = result.leaderboard.splice(1, result.leaderboard.length-1);
                } else {
                    past_weeks = [];
                }

                place_current_week(this_week);

                if(refresh) {
                    place_season_details(goal, season);
                    place_leaderboard(past_weeks);
                }
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/" + season.id + "/weeks");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function place_leaderboard(weeks_array) {

    var leaderboardWeeks = document.getElementById("leaderboard-weeks")
    var html = ``;

    if(weeks_array.length == 0) {
        html = `
            <div id="" class="leaderboard-weeks">
                <p id="" style="margin: 0.5em; text-align: center;">No past weeks.</p>
            </div>
        `;
        leaderboardWeeks.innerHTML = html
    } else {
        leaderboardWeeks.innerHTML = ""

        for(var i = 0; i < weeks_array.length; i++) {
            var week_html = `
                <div class="leaderboard-week" id="">
                    <div class="leaderboard-week-number">
                        Week ` + weeks_array[i].week_number + ` (` + weeks_array[i].week_year + `)
                    </div>
                    <div class="leaderboard-week-results">
            `;

            var results_html = "";

            // Sort users
            week_array[i].users = week_array[i].users.sort((a,b) => b.user_id.localeCompare(a.user_id));

            for(var j = 0; j < weeks_array[i].users.length; j++) {
                var completion = "❌"

                if(!weeks_array[i].users[j].full_week_participation && weeks_array[i].users[j].week_completion < 1.0) {
                    completion = "🕙"
                } else if(weeks_array[i].users[j].sick_leave) {
                    completion = "🤢"
                } else if(weeks_array[i].users[j].week_completion >= 1.0) {
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
                    <div class="leaderboard-week-result-user clickable" style="cursor: pointer;" onclick="location.href='/users/${weeks_array[i].users[j].user_id}'">
                        ` + userList[weeks_array[i].users[j].user_id].first_name + `
                    </div>
                    <div class="leaderboard-week-result-exercise ` + clickable_str  + `" onclick="` + onclick_command_str  + `">
                        ` + completion  + `
                    </div>
                </div>
                `;
                results_html += result_html;

            }

            week_html += results_html + `</div></div>`;

            leaderboardWeeks.innerHTML += week_html
        }
        
    }

    return
}

function GetProfileImageForUserOnLeaderboard(userID) {

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

                PlaceProfileImageForUserOnLeaderboard(result.image, userID)
                
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

function PlaceProfileImageForUserOnLeaderboard(imageBase64, userID) {

    document.getElementById("member-img-" + userID).src = imageBase64

}

function place_season_details(goal) {
    document.getElementById("week_goal").innerHTML = goal.exercise_interval
    document.getElementById("goal_this_week").innerText  = goal.exercise_interval
    document.getElementById("goal_sickleave").innerHTML = goal.sickleave_left
}

function place_current_week(week_array) {
    var currentWeekUsers = document.getElementById("current-week-users")

    // Sort users
    week_array.users = week_array.users.sort((a,b) => b.week_completion - a.week_completion);
    
    // Remove initial data
    currentWeekUsers.innerHTML = ""

    document.getElementById('current-week-title').innerHTML = `Current week (${week_array.week_number})`

    for(var i = 0; i < week_array.users.length; i++) {

        var completion = Math.trunc((week_array.users[i].week_completion * 100))
        var transparent = ""

        if(!week_array.users[i].full_week_participation && completion < 100) {
            var current_streak = week_array.users[i].current_streak + "🕙"
        } else if(week_array.users[i].sick_leave) {
            var current_streak = week_array.users[i].current_streak + "🤢"
            transparent = "transparent"

            if(week_array.users[i].user_id == user_id){
                document.getElementById("calendar").classList.add("transparent")
                document.getElementById("calendar").classList.add("unselectable")
                document.getElementById("calendar").classList.add("noninteractive")
                document.getElementById("add-exercise-button").style.display = 'none';
            } else {
                console.log(user_id)
            }

        } else if(week_array.users[i].current_streak > 0) {
            var current_streak = week_array.users[i].current_streak + "🔥"
        } else {
            var current_streak = week_array.users[i].current_streak + "💀"
        }

        if(completion >= 100) {
            transparent += " bold-font "
        }

        if(week_array.users[i].user_id == user_id) {
            placeWeekProgress(completion)
        }

        var week_html = `
            <div class="current-week-user unselectable" id="">

                <div style="" class="">
                    
                    <div class="" style="font-size: 0.8em;">
                        <b>${userList[week_array.users[i].user_id].first_name}</b>
                    </div>

                    <div class="current-week-user-photo" title="` + userList[week_array.users[i].user_id].first_name + ` ` + userList[week_array.users[i].user_id].last_name + `" onclick="location.href='/users/${week_array.users[i].user_id}'">
                        <img style="width: 100%; height: 100%;" class="current-week-user-photo-img" id="current-week-user-photo-` + week_array.users[i].user_id + `-` + i + `" src="/assets/images/barbell.gif">
                    </div>
                </div>

                <div class="current-week-user-results">

                    <div class="current-week-user-completion ` + transparent + `" title="How much of the goal for this week is finished.">
                        ` + completion + `%
                    </div>

                    <div class="current-week-user-completion" title="How many weeks in a row have been at least 100%.">
                        ` + current_streak + ` 
                    </div>

                </div>

            </div>
        `;

        currentWeekUsers.innerHTML += week_html
        GetProfileImagesForCurrentWeek(week_array.users[i].user_id, i);
    }

    return
}

function GetProfileImagesForCurrentWeek(userID, index) {

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

                PlaceProfileImagesForCurrentWeek(result.image, userID, index)
                
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

function PlaceProfileImagesForCurrentWeek(imageBase64, userID, index) {

    document.getElementById("current-week-user-photo-" + userID + "-" + index).src = imageBase64

}

function use_sickleave() {

    if(!confirm("Are you sure you want to use sick leave? The week will be marked as sick leave, no workouts can be logged, the current streak will be preserved.")) {
        return
    }

    seasonID = document.getElementById("calendar_season_id").value;

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

                location.reload();
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/sickleave/" + seasonID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function placeDebtOverview(overviewArray) {

    var html = "";

    document.getElementById("debt-module").style.display = "flex";

    for(var i = 0; i < overviewArray.debt_unviewed.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_unviewed[i].debt.date);
            var date_week = date.getWeek(1);
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        html += `
            <div class="debt-module-notification-view" id="">
                ${overviewArray.debt_unviewed[i].debt.loser.first_name} ${overviewArray.debt_unviewed[i].debt.loser.last_name} spun the wheel for week ${date_str}.<br>See if you won!<br>
                <img src="assets/arrow-right.svg" class="small-button-icon clickable" onclick="location.replace('/wheel?debt_id=${overviewArray.debt_unviewed[i].debt.id}'); ">
            </div>
        `;
    }

    for(var i = 0; i < overviewArray.debt_won.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_won[i].date);
            var date_week = date.getWeek(1);
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        console.log(overviewArray.debt_won)

        html += `
            <div class="debt-module-notification-prize" id="">
                ${overviewArray.debt_won[i].loser.first_name} ${overviewArray.debt_won[i].loser.last_name} spun the wheel for week ${date_str} and you won <b>${overviewArray.debt_won[i].season.prize.quantity} ${overviewArray.debt_won[i].season.prize.name}</b>!<br>Have you received it?<br>
                <img src="assets/done.svg" class="small-button-icon clickable" onclick="setPrizeReceived('${overviewArray.debt_won[i].id}');">
            </div>
        `;
    }

    for(var i = 0; i < overviewArray.debt_unpaid.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_unpaid[i].date);
            var date_week = date.getWeek(1);
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        console.log(overviewArray.debt_unpaid)

        html += `
            <div class="debt-module-notification-debt" id="">
                You spun the wheel for week ${date_str} and ${overviewArray.debt_unpaid[i].winner.first_name} ${overviewArray.debt_unpaid[i].winner.last_name} won ${overviewArray.debt_unpaid[i].season.prize.quantity} ${overviewArray.debt_unpaid[i].season.prize.name}!<br>Provide the prize as soon as possible!<br>
            </div>
            `;
    }

    document.getElementById("debt-module-notifications").innerHTML = html;

}

function placeDebtSpin(overview) {
    
    var date_str = ""
    try {
        var date = new Date(overview.debt_lost[0].date);
        var date_week = date.getWeek(1);
        var date_year = date.getFullYear();
        var date_str = date_week + " (" + date_year + ")"
    } catch {
        date_str = "Error"
    }   
    
    try {
        document.getElementById("ongoingseason").style.display = "none";
    } catch (e) {
        console.log("Failed to hide ongoing season. Error: " + e)
    }  

    try {
        document.getElementById("top-module").style.display = "none";
    } catch (e) {
        console.log("Failed to hide top module. Error: " + e)
    }  
    
    document.getElementById("unspun-wheel").style.display = "flex";
    document.getElementById("unspun-wheel").innerHTML = `
        You failed to reach your goal for week ${date_str} and must spin the wheel.
        <div id="canvas-buttons" class="canvas-buttons">
            <button id="go-to-wheel" onclick="location.replace('/wheel?debt_id=${overview.debt_lost[0].id}');">Take me there</button>
        </div>
    `;
    return;
}

function EditExercise(weekdayInt) {

    update_exercises(true, weekdayInt);

}

function GoToExercise(exerciseID) {

    window.location = '/exercises/' + exerciseID

}

function placeSeasonProgress(seasonStartObject, seasonEndObject) {

    const weekSum = weeksBetween(seasonStartObject, seasonEndObject)

    const now = new Date();
    const weekAmount = weeksBetween(seasonStartObject, now)

    document.getElementById("weeks_so_far").innerHTML = weekAmount
    document.getElementById("weeks_total").innerHTML = weekSum

    var ach_percentage = Math.floor((weekAmount / weekSum) * 100)
    
    if(ach_percentage > 100) {
        ach_percentage = 100;
    }

    document.getElementById("season-progress-bar").style.width  = ach_percentage + "%"

    if(ach_percentage > 99) {
        document.getElementById("season-progress-bar-wrapper").classList.remove('transparent');
        setTimeout(function() {
            document.getElementById("season-progress-bar").classList.add("blink")
        }, 1500);
        setTimeout(function() {
            document.getElementById("season-progress-bar").classList.remove('blink');
        }, 2500);
    }
}

function placeWeekProgress(percentage, exercise, exerciseGoal) {

    console.log("This weeks progress: " + percentage)

    if(percentage > 100) {
        percentage = 100;
    }

    document.getElementById("week-progress-bar").style.width  = percentage + "%"

    if(percentage > 99) {
        document.getElementById("week-progress-bar-wrapper").classList.remove('transparent');
        setTimeout(function() {
            document.getElementById("week-progress-bar").classList.add("blink")
        }, 1500);
        setTimeout(function() {
            document.getElementById("week-progress-bar").classList.remove('blink');
        }, 2500);
    }

}

function verifyPageRedirect() {

    window.location = '/verify'

}

function countdownRedirect() {

    window.location = '/countdown'
    
}

function registerGoalRedirect() {

    window.location = '/registergoal'
    
}

function exerciseRedirect(exerciseDayID) {
    window.location = '/exercises/' + exerciseDayID
}

function addExercise(exerciseDayID) {
    var form_obj = {
        "exercise_day_id": exerciseDayID,
        "on" : true,
        "note": "",
        "duration": null
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
                exerciseRedirect(exerciseDayID)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function getPotentialSeasons(user_id){

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
                return;
            } else {
                placePotentialSeasons(result.seasons);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons?potential=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function placePotentialSeasons(seasonsArray) {
    try {
        if(!seasonsArray || seasonsArray.length == 0) {
            return;
        }

        var html = "Seasons you can join:<br><div class='potentialSeasonList'><hr>";
        for(var i = 0; i < seasonsArray.length; i++) {
            html += `
                <div class="potentialSeason hover clickable" onclick="document.location.href = '/registergoal?season_id=${seasonsArray[i].id}'">
                    ${seasonsArray[i].name}
                </div>
            `
        }
        html += '</div>'

        potentialSeasonsWrapper = document.getElementById("potentialSeasonsWrapper");
        potentialSeasonsWrapper.innerHTML = html;
        potentialSeasonsWrapper.style.display = "flex";
        document.getElementById("seasonDivider").style.display = "flex";
    } catch (e) {
        console.log("Failed to place potential seasons. Error: " + e)
        error("Failed to place potential seasons.")
        return;
    }
}

function getCountdownSeasons(userID){

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
                return;
            } else {
                placeCountdownSeasons(result.seasons, userID);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons?countdown=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function placeCountdownSeasons(seasonsArray, userID) {
    console.log(seasonsArray)

    try {
        if(!seasonsArray || seasonsArray.length == 0) {
            return;
        }

        var html = "Seasons you are waiting for:<br><div class='countdownSeasonList'><hr>";
        for(var i = 0; i < seasonsArray.length; i++) {
            var goalID = ''
            for(var j = 0; j < seasonsArray[i].goals.length; j++) {
                if(seasonsArray[i].goals[j].user.id == userID) {
                    goalID = seasonsArray[i].goals[j].id;
                    break;
                }
            }

            var date_start = new Date(seasonsArray[i].start);
            var date_end = new Date(seasonsArray[i].end);

            var joinText = "..."
            if(seasonsArray[i].join_anytime) {
                joinText = "<b>You can join at any point in the season.</b>"
            } else {
                joinText = "<b>You must join before the start date.</b>"
            }

            var partici_string = "participants"
            if(seasonsArray[i].goals.length == 1) {
                partici_string = "participant"
            }

            html += `
                <div class="countdownSeason">
                    <h3 id="countdown_season_title" style="margin: 0 0 0.5em 0;">${seasonsArray[i].name}</h3>
                    <p id="countdown_season_start">${GetDateString(date_start, true)}</p>
                    <p id="countdown_season_end">${GetDateString(date_end, true)}</p>
                  
                    <p id="countdown_title" style="margin-top: 0.25em;">${seasonsArray[i].goals.length + " " + partici_string}. Starting in:</p>
                    
                    <p style="font-size: 2em; text-align: center;" id="countdown_number_${seasonsArray[i].id}" class="countdown_number">00d 00h 00m 00s</p>

                    <a class="clickable hover" style="margin: 1em 0 0 0; font-size:0.75em;" onclick="deleteGoal('${goalID}');">I changed my mind!</i></a>
                    <hr>
                </div>
            `
        }
        html += '</div>'

        countdownSeasonsWrapper = document.getElementById("countdownSeasonsWrapper");
        countdownSeasonsWrapper.innerHTML = html;
        countdownSeasonsWrapper.style.display = "flex";

        for(var i = 0; i < seasonsArray.length; i++) {
            var date_start = new Date(seasonsArray[i].start);
            activateCountdown(date_start, seasonsArray[i].id);
        }

        document.getElementById("seasonDivider").style.display = "flex";
    } catch (e) {
        console.log("Failed to place countdown seasons. Error: " + e)
        error("Failed to place countdown seasons.")
        return;
    }
}

function activateCountdown(countdownDate, seasonID){

    // Set the date we're counting down to
    var countDownDate = countdownDate.getTime();

    // Update the count down every 1 second
    var x = setInterval(function() {

        // Get today's date
        var now = new Date();

        // Find the distance between now and the count down date
        var distance = Math.floor(countDownDate - now.getTime());
    
        // Time calculations for days, hours, minutes and seconds
        var days = Math.floor(distance / (1000 * 60 * 60 * 24));
        var hours = Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
        var minutes = Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60));
        var seconds = Math.floor((distance % (1000 * 60)) / 1000);

        if (distance > 0) {
            // Display the result in the element with id="demo"
            document.getElementById("countdown_number_" + seasonID).innerHTML = padNumber(days, 2) + "d " + padNumber(hours, 2) + "h "
            + padNumber(minutes, 2) + "m " + padNumber(seconds, 2) + "s ";
        
            // If the count down is finished, write some text
        } else {
            clearInterval(x);
            document.getElementById("countdown_number").innerHTML = "...";

            setTimeout(() => {
                frontPageRedirect(true);
            }, 5000);
              
        }
        
    }, 1000);
}

function deleteGoal(goalID) {
    if(!confirm("Are you sure you want to delete your goal? You are free to create another one afterward as long as the season has not started.")) {
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
                frontPageRedirect(true);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/goals/" + goalID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function blinkCalendar() {
    try {
        document.getElementById("goal_amount_button").classList.add("blink")
        setTimeout(function() {
            document.getElementById("goal_amount_button").classList.remove('blink');
        }, 1500);
    } catch (e) {
        console.log("Failed to blink the calendar. Error: " + e)
    }
}

function getActivities(season){
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
                placeActivities(result.activities);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/" + season.id + "/activities");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeActivities(activitiesArray) {
    if(activitiesArray.length < 1) {
        return
    }

    var usersToGetArray = [];
    var activitiesHTML = `
        <div class="activityWrapper">
    `;

    activitiesArray.forEach(activity => {
        activityHTML = generateActivityHTML(activity);
        activitiesHTML += activityHTML;
        usersToGetArray.push({"userID": activity.user.id, "activityID": activity.id});
    });

    
    activitiesHTML += `
        </div>
    `;

    try {
        document.getElementById("activities-week").innerHTML = activitiesHTML
    } catch (error) {
        console.log(error)
    }

    console.log("yoo: " + usersToGetArray.length)
    for(var i = 0; i < usersToGetArray.length; i++) {
        console.log(i)
        GetProfileImageForActivity(usersToGetArray[i].userID, usersToGetArray[i].activityID);
    }
}

function generateActivityHTML(activity) {
    var activitySentence = "worked out";
    if(activity.actions && activity.actions.length == 1) {
        if(activity.actions[0].past_tense_verb && activity.actions[0].past_tense_verb != "") {
            activitySentence = activity.actions[0].past_tense_verb;
        }
    } else if(activity.actions && activity.actions.length > 1) {
        activitySentence = "did a range of workouts"
    }

    var activityLogoBarHTML = "";
    if(activity.actions) {
        activity.actions.forEach(action => {
            if(action.has_logo) {
                activityLogoBarHTML += `
                    <div class="activity-logo">
                        <img style="width: 100%;" src="assets/actions/${action.name}.svg" class="" title="${action.name}"></img>
                    </div>
                `;
            }
        });
    }

    var activityStravaHTML = "";
    if(activity.strava_ids && activity.strava_ids.length > 0) {
        activity.strava_ids.forEach(stravaID => {
            activityStravaHTML += `
                <img class="strava-logo-img clickable" src="/assets/strava-logo.svg" onclick="window.open('https://www.strava.com/activities/${stravaID}', '_blank')">
                </img>
            `;
        });
    }

    

    var dateObject = new Date(activity.time)

    activityHTML = `
        <div class="activity">

            <div class="activity-date">
                ${GetDateString(dateObject, true)}
            </div>

            <div class="activity-sections">
                <div class="activity-photo-wrapper">
                    <div class="activity-user-photo" title="` + activity.user.first_name + ` ` + activity.user.last_name + `" onclick="location.href='/users/${activity.user.id}'">
                        <img style="width: 100%; height: 100%; border-radius: 100%; object-fit: cover; overflow: hidden;" class="activity-user-photo-img" id="activity-user-photo-` + activity.user.id + `-` + activity.id + `" src="/assets/images/barbell.gif">
                    </div>
                </div>

                <div class="activity-details">
                    <div>
                        ${activity.user.first_name} ${activitySentence}!
                    </div>

                    <div class="activity-logo-wrapper">
                        ${activityLogoBarHTML}
                    </div>

                </div>

                <div class="activity-strava-wrapper">
                    ${activityStravaHTML}
                </div>

            </div>
        </div>
    `;

    return activityHTML;
}

function GetProfileImageForActivity(userID, index) {
    console.log(index)
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
                PlaceProfileImageForActivity(result.image, userID, index)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImageForActivity(imageBase64, userID, index) {
    document.getElementById("activity-user-photo-" + userID + "-" + index).src = imageBase64
}