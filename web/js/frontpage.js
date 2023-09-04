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
                    
                    <div class="module">
                    
                        <div class="title">
                            Treningheten
                        </div>

                        <div class="text-body" id="front-page-text" style="text-align: center;">
       
                        </div>

                        <div id="log-in-button" style="margin-top: 2em; display: none; width: 10em;">
                            <button id="update-button" type="submit" href="#" onclick="window.location = './login';">Log in</button>
                        </div>

                        <div class="module" id="barbell-gif" style="display: none;">
                            <img src="./assets/images/barbell.gif">
                        </div>

                    </div>

                    <div class="module" id="registergoal" style="display: none;">

                        <div id="season" class="season">

                            <h3 id="register_season_title" style="margin: 0 0 0.5em 0;">Loading...</h3>
                            <p id="register_season_start">...</p>
                            <p id="register_season_end">...</p>
                            <p style="margin-top: 1em; text-align: center;" id="register_season_desc">...</p>

                            <hr style="margin: 1em 0;">

                            <label for="commitment" title="How many days a week are you going to work out?">Weekly exercise goal</label>
                            <div class="number-box" id="commitment">
                                0
                            </div>
                            <div class="two-buttons">
                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('commitment', 1, 21);">
                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('commitment', 1, 21);">
                            </div>

                            <hr style="margin: 1em 0;">

                            <input style="" type="checkbox" id="compete" class="clickable" name="compete" value="compete" required>
                            <label for="compete" class="clickable" style="user-select: none; text-align: center;" title="If I fail to complete my goal, I must spin a wheel of fortune and provide a prize to the winner."> I want to compete with others to uphold my workout streak.</label><br>

                            <p id="prize-title" style="margin-top: 1em;">Potential prize:</p>
                            <div class="prize-wrapper">
                                <div id="register-prize-text" class="prize-text">...</div>
                            </div>

                            <hr style="margin: 1em 0;">

                            <button type="submit" onclick="register_goal();" id="register_goal_button" style=""><img src="assets/done.svg" class="btn_logo color-invert"><p2>Join season</p2></button>

                        </div>

                    </div>

                    <div class="module" id="countdownseason" style="display: none;">

                        <div id="season" class="season">

                            <h3 id="countdown_season_title" style="margin: 0 0 0.5em 0;">Loading...</h3>
                            <p id="countdown_season_start">...</p>
                            <p id="countdown_season_end">...</p>
                            <p style="margin-top: 1em; text-align: center;" id="countdown_season_desc">...</p>

                            <hr style="margin: 1em 0;">

                            <p style="text-align: center;" id="countdown_goal">...</p>

                            <hr style="margin: 1em 0;">

                            <p id="countdown_title">Starting in:</p>
                            
                            <p style="font-size: 2em; text-align: center;" id="countdown_number" class="countdown_number">00d 00h 00m 00s</p>

                            <a style="margin: 1em 0 0 0; font-size:0.75em; cursor: pointer;" onclick="delete_goal();">I changed my mind!</i></a>

                        </div>

                    </div>

                    <div class="module" id="ongoingseason" style="display: none;">

                        <div class="modules">

                            <div id="exercises" class="exercises">

                                <div class="week_days" id='calendar'>

                                    <div class="calender_status unselectable" id="calender_status">
                                        <a id="workout_this_week">...</a>
                                        /
                                        <a id="goal_this_week">...</a>
                                        this week
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_1_group">
                                        <div class="day-check">
                                            <label for="day_1_check" title="Have you been working out?">Monday</label>
                                            <div class="number-box" id="day_1_check">
                                                0
                                            </div>
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_1_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_1_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_1_note" name="day_1_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_1_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_2_group">
                                        <div class="day-check">
                                            <label for="day_2_check" title="Have you been working out?">Tuesday</label>
                                            <div class="number-box" id="day_2_check">
                                                0
                                            </div>
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_2_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_2_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_2_note" name="day_2_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_2_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_3_group">
                                        <div class="day-check">
                                            <label for="day_3_check" title="Have you been working out?">Wednesday</label>
                                            <div class="number-box" id="day_3_check">
                                                0
                                            </div>
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_3_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_3_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_3_note" name="day_3_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_3_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_4_group">
                                        <div class="day-check">
                                            <label for="day_4_check" title="Have you been working out?">Thursday</label>
                                            <div class="number-box" id="day_4_check">
                                                0
                                            </div>
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_4_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_4_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_4_note" name="day_4_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_4_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_5_group">
                                        <div class="day-check">
                                            <label for="day_5_check" title="Have you been working out?">Friday</label>
                                            <div class="number-box" id="day_5_check">
                                                0
                                            </div>
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_5_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_5_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_5_note" name="day_5_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_5_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_6_group">
                                        <div class="day-check">
                                            <label for="day_6_check" title="Have you been working out?">Saturday</label>
                                            <div class="number-box" id="day_6_check">
                                                0
                                            </div>
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_6_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_6_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_6_note" name="day_6_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_6_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_7_group">
                                        <div class="day-check">
                                            <label for="day_7_check" title="Have you been working out?">Sunday</label>
                                            <div class="number-box" id="day_7_check">
                                                0
                                            </div>
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_7_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_7_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_7_note" name="day_7_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_7_date">
                                    </div>

                                    <button type="submit" onclick="update_exercises();" id="goal_amount_button" style="margin-bottom: 0em;"><img src="assets/done.svg" class="btn_logo color-invert"><p2>Save</p2></button>

                                    <a style="margin: 0.5em; font-size:0.75em;cursor:pointer;" onclick="use_sickleave();">Use sick leave</i></a>

                                </div>

                            </div>

                            <div class="module-two">

                                <div id="season-module" class="season">

                                    <h3 id="season_title">Loading...</h3>
                                    <p id="season_desc" style="text-align: center;">...</p>

                                    <p id="season_start_title" style="margin-top: 1em;">Season start: <a id="season_start">...</a></p>
                                    <p id="season_end_title" style="">Season end: <a id="season_end">...</a></p>

                                    <p id="week_goal_title" style="margin: 1em 0 0 0;">Weekly goal: <b><a id="week_goal">0</a></b></p>
                                    <p id="goal_sickleave_title" style="">Sick leave left: <b><a id="goal_sickleave">0</a></b></p>

                                    <p id="prize-title" style="margin-top: 1em;">Potential prize:</p>
                                    <div class="prize-wrapper">
                                        <div id="prize-text" class="prize-text">...</div>
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
                                    </div>


                                </div>

                            </div>
                            

                            <div id="leaderboard" class="leaderboard">

                                <h3 style="margin: 0.5em;">Leaderboard</h3>

                                <div id="leaderboard-weeks" class="leaderboard-weeks">
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
        get_season(user_id);
        document.getElementById('front-page-text').innerHTML = 'Remember to log your workouts.';
    } else {
        showLoggedOutMenu();
        document.getElementById('front-page-text').innerHTML = 'Log in to use the platform.';
        document.getElementById('log-in-button').style.display = 'inline-block';
    }
}

function load_verify_account() {

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="title">
                            Treningheten
                        </div>

                        <div class="text-body" style="text-align: center;">
                            You must verify your account by giving us the access code we e-mailed you.
                        </div>

                    </div>

                    <div class="module">

                        <form action="" onsubmit="event.preventDefault(); verify_account();">
                            <label for="email_code">Code:</label><br>
                            <input type="text" name="email_code" id="email_code" placeholder="Code" autocomplete="one-time-code" required />
                            <button id="verify-button" type="submit" href="/">Verify</button>
                        </form>

                    </div>

                    <div class="module">
                        <a style="font-size:0.75em;cursor:pointer;" onclick="new_code();">Send me a new code!</i>
                    </div>

                </div>

    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Robot or human?';
    clearResponse();
    showLoggedInMenu();
    document.getElementById('navbar').style.display = 'none';

}

function verify_account(){

    var email_code = document.getElementById("email_code").value;

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

                // store jwt to cookie
                set_cookie("treningheten", result.token, 7);
                location.reload();

            }

        } else {
            info("Verifying account...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/user/verify/" + email_code);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
    
}

function new_code(){

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

            }

        } else {
            info("Sending new code...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/user/verification");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
    
}

function get_season(user_id){

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
            
            if(result.error == "No active or future seasons found.") {

                info(result.error);
                document.getElementById('front-page-text').innerHTML = 'An administrator must plan a new season.';

                get_debtoverview_prepare(true);

            } else if(result.error) {

                error(result.error);
                error_splash_image();

            } else {

                clearResponse();
                season = result.season;

                user_found = false;
                for(var i = 0; i < season.goals.length; i++) {
                    if(season.goals[i].user.ID == user_id) {
                        user_found = true
                        var goal = season.goals[i].exercise_interval
                        break
                    }
                }

                var date_start = new Date(season.start);
                var now = Date.now();

                if(user_found && now < date_start) {
                    countdown_module(season, goal, result.timezone)
                    get_debtoverview_prepare(false);
                } else if(user_found) {
                    document.getElementById("ongoingseason").style.display = "flex"
                    get_calendar(false);
                    place_season(season);
                    get_leaderboard();
                    get_debtoverview();
                } else {
                    registergoal_module(season)
                    get_debtoverview_prepare(false);
                }

            }

        } else {
            info("Loading season...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/season/getongoing");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function countdown_module(season_object, exercise_goal, timezone) {

    var date_start = new Date(season_object.start);
    var date_end = new Date(season_object.end);

    document.getElementById("countdownseason").style.display = "flex"
    document.getElementById("countdown_season_title").innerHTML = season_object.name
    document.getElementById("countdown_season_start").innerHTML = "Season start: " + GetDateString(date_start, true)
    document.getElementById("countdown_season_end").innerHTML = "Season end: " + GetDateString(date_end, true)
    document.getElementById("countdown_season_desc").innerHTML = season_object.description
    document.getElementById("countdown_goal").innerHTML = "You are signed up for " + exercise_goal + " exercises a week."

    var partici_string = "participants"
    if(season_object.goals.length == 1) {
        partici_string = "participant"
    }

    document.getElementById("countdown_title").innerHTML = season_object.goals.length + " " + partici_string + ". Starting in: "

    StartCountDown(date_start, timezone);
}

function registergoal_module(season_object) {

    var date_start = new Date(season_object.start);
    var date_end = new Date(season_object.end);

    document.getElementById("registergoal").style.display = "flex"
    document.getElementById("register_season_title").innerHTML = season_object.name
    document.getElementById("register_season_start").innerHTML = "Season start: " + GetDateString(date_start, true)
    document.getElementById("register_season_end").innerHTML = "Season end: " + GetDateString(date_end, true)
    document.getElementById("register_season_desc").innerHTML = season_object.description
    document.getElementById("register-prize-text").innerHTML = season_object.prize.quantity + " " + season_object.prize.name
}

function place_season(season_object) {

    document.getElementById("season_title").innerHTML = season_object.name
    document.getElementById("season_desc").innerHTML = season_object.description
    document.getElementById("prize-text").innerHTML = season_object.prize.quantity + " " + season_object.prize.name

}

function register_goal() {

    var exercise_goal = Number(document.getElementById("commitment").innerHTML);
    var goal_compete = document.getElementById("compete").checked

    var form_obj = {
        "exercise_interval": exercise_goal,
        "competing": goal_compete

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

                location.reload()

            }

        } else {
            info("Saving goal...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/goal/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function get_calendar(fireworks){

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
                week = result.week;

                console.log(week);

                console.log("Placing intial week: ")
                place_week(week, fireworks);

            }

        } else {
            info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercise/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_week(week, fireworks) {

    if(fireworks) {
        console.log("Triggered fireworks.")
        trigger_fireworks(1);
    }

    fireworks_int = week.days[0].exercise_interval + week.days[1].exercise_interval + week.days[2].exercise_interval + week.days[3].exercise_interval + week.days[4].exercise_interval + week.days[5].exercise_interval + week.days[6].exercise_interval

    document.getElementById("day_1_check").innerHTML = week.days[0].exercise_interval
    document.getElementById("day_1_note").value = HTMLDecode(week.days[0].note)
    document.getElementById("day_1_date").value = week.days[0].date

    document.getElementById("day_2_check").innerHTML = week.days[1].exercise_interval
    document.getElementById("day_2_note").value = HTMLDecode(week.days[1].note)
    document.getElementById("day_2_date").value = week.days[1].date

    document.getElementById("day_3_check").innerHTML = week.days[2].exercise_interval
    document.getElementById("day_3_note").value = HTMLDecode(week.days[2].note)
    document.getElementById("day_3_date").value = week.days[2].date

    document.getElementById("day_4_check").innerHTML = week.days[3].exercise_interval
    document.getElementById("day_4_note").value = HTMLDecode(week.days[3].note)
    document.getElementById("day_4_date").value = week.days[3].date

    document.getElementById("day_5_check").innerHTML = week.days[4].exercise_interval
    document.getElementById("day_5_note").value = HTMLDecode(week.days[4].note)
    document.getElementById("day_5_date").value = week.days[4].date

    document.getElementById("day_6_check").innerHTML = week.days[5].exercise_interval
    document.getElementById("day_6_note").value = HTMLDecode(week.days[5].note)
    document.getElementById("day_6_date").value = week.days[5].date

    document.getElementById("day_7_check").innerHTML = week.days[6].exercise_interval
    document.getElementById("day_7_note").value = HTMLDecode(week.days[6].note)
    document.getElementById("day_7_date").value = week.days[6].date

    // Place the exercise sum
    document.getElementById("workout_this_week").innerHTML = fireworks_int;

    return

}

function update_exercises() {

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

                clearResponse();
                week = result.week;

                console.log(week);

                new_fireworks_int = 0;
                new_fireworks = false
                for(var i = 0; i < week.days.length; i++) {
                    new_fireworks_int += week.days[i].exercise_interval
                }

                if (new_fireworks_int > fireworks_int) {
                    new_fireworks = true
                }

                console.log("Placing intial week: ")
                place_week(week, new_fireworks);
                get_leaderboard();

                success(result.message)

            }

        } else {
            info("Saving week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercise/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function IncreaseNumberInput(input_id, min, max) {
    var input_element = document.getElementById(input_id)
    var old_number = Number(input_element.innerHTML)
    var new_number = old_number + 1
    if(new_number <= max && new_number >= min) {
        input_element.innerHTML = new_number
    }
}

function DecreaseNumberInput(input_id, min, max) {
    var input_element = document.getElementById(input_id)
    var old_number = Number(input_element.innerHTML)
    var new_number = old_number - 1
    if(new_number <= max && new_number >= min) {
        input_element.innerHTML = new_number
    }
}

function StartCountDown(countdownDate){

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
            document.getElementById("countdown_number").innerHTML = padNumber(days, 2) + "d " + padNumber(hours, 2) + "h "
            + padNumber(minutes, 2) + "m " + padNumber(seconds, 2) + "s ";
        
            // If the count down is finished, write some text
        } else {
            clearInterval(x);
            document.getElementById("countdown_number").innerHTML = "...";

            setTimeout(() => {
                location.reload();
            }, 5000);
              
        }
        
    }, 1000);
}

function get_leaderboard(fireworks){

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
                past_weeks = result.leaderboard.past_weeks;
                this_week = result.leaderboard.this_week;

                console.log("Placing weeks: ")
                place_current_week(this_week);
                place_season_details(result.leaderboard.goal.exercise_interval, result.leaderboard.goal.sickleave_left, result.leaderboard.season.start, result.leaderboard.season.end);
                place_leaderboard(past_weeks);
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/season/leaderboard");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_leaderboard(weeks_array) {

    var html = ``;
    
    if(weeks_array.length == 0) {
        html = `
            <div id="" class="leaderboard-weeks">
                <p id="" style="margin: 0.5em; text-align: center;">No past weeks.</p>
            </div>
        `;
    } else {
        for(var i = 0; i < weeks_array.length; i++) {
            var week_html = `
                <div class="leaderboard-week" id="">
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
                if(weeks_array[i].users[j].debt !== null && weeks_array[i].users[j].debt.winner.ID !== 0) {
                    onclick_command_str = "location.replace('./wheel?debt_id=" + weeks_array[i].users[j].debt.ID + "'); "
                    clickable_str = "clickable"
                    completion += "🎡"
                }


                var result_html = `
                <div class="leaderboard-week-result" id="">
                    <div class="leaderboard-week-result-user" style="cursor: pointer;" onclick="location.href='./user/${weeks_array[i].users[j].user.ID}'">
                        ` + weeks_array[i].users[j].user.first_name + `
                    </div>
                    <div class="leaderboard-week-result-exercise ` + clickable_str  + `" onclick="` + onclick_command_str  + `">
                        ` + completion  + `
                    </div>
                </div>
                `;
                results_html += result_html;

            }

            week_html += results_html + `</div></div>`;

            html += week_html

        }
        
    }

    document.getElementById("leaderboard-weeks").innerHTML = html

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
    xhttp.open("post", api_url + "auth/user/get/" + userID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImageForUserOnLeaderboard(imageBase64, userID) {

    document.getElementById("member-img-" + userID).src = imageBase64

}

function place_season_details(goal, sickleave, seasonStart, SeasonEnd) {

    try {
        var date_start = new Date(seasonStart);
        var date_end = new Date(SeasonEnd);

        var date_start_string = GetDateString(date_start, true)
        var date_end_string = GetDateString(date_end, true)
    } catch {
        var date_start_string = "Error"
        var date_end_string = "Error"
    }

    document.getElementById("week_goal").innerHTML = goal
    document.getElementById("goal_this_week").innerHTML = goal;
    document.getElementById("goal_sickleave").innerHTML = sickleave
    document.getElementById("season_start").innerHTML = date_start_string
    document.getElementById("season_end").innerHTML = date_end_string
}

function place_current_week(week_array) {

    week_array.users = week_array.users.sort((a,b) => b.week_completion - a.week_completion);
    
    var html = ``;

    for(var i = 0; i < week_array.users.length; i++) {

        var completion = Math.trunc((week_array.users[i].week_completion * 100))
        var transparent = ""

        if(week_array.users[i].sickleave) {
            var current_streak = week_array.users[i].current_streak + "🤢"
            transparent = "transparent"

            if(week_array.users[i].user.ID == user_id){
                document.getElementById("calendar").classList.add("transparent")
                document.getElementById("calendar").classList.add("unselectable")
                document.getElementById("calendar").classList.add("noninteractive")
            } else {
                console.log(user_id)
            }

        } else if(week_array.users[i].current_streak > 0) {
            var current_streak = week_array.users[i].current_streak + "🔥"
        } else {
            var current_streak = week_array.users[i].current_streak + "💀"
        }

        var week_html = `
            <div class="current-week-user unselectable" id="">

                <div style="cursor: pointer;" onclick="location.href='./user/${week_array.users[i].user.ID}'">
                    ${week_array.users[i].user.first_name}

                    <div class="current-week-user-photo" title="` + week_array.users[i].user.first_name + ` ` + week_array.users[i].user.last_name + `">
                        <img style="width: 100%; height: 100%;" class="current-week-user-photo-img" id="current-week-user-photo-` + week_array.users[i].user.ID + `-` + i + `" src="/assets/images/barbell.gif">
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

        html += week_html

    }

    document.getElementById("current-week-users").innerHTML = html

    for(var i = 0; i < week_array.users.length; i++) {
        GetProfileImagesForCurrentWeek(week_array.users[i].user.ID, i)
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
    xhttp.open("post", api_url + "auth/user/get/" + userID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImagesForCurrentWeek(imageBase64, userID, index) {

    document.getElementById("current-week-user-photo-" + userID + "-" + index).src = imageBase64

}

function use_sickleave() {

    if(!confirm("Are you sure you want to use sick leave? The week will be marked as sick leave, no workouts can be logged, the current streak will be perserved.")) {
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

                location.reload();
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/sickleave/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function get_debtoverview() {

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

                if(result.overview.debt_lost.length > 0) {

                    var date_str = ""
                    try {
                        var date = new Date(result.overview.debt_lost[0].date);
                        var date_week = date.GetWeek();
                        var date_year = date.getFullYear();
                        var date_str = date_week + " (" + date_year + ")"
                    } catch {
                        date_str = "Error"
                    }
    
                    document.getElementById("ongoingseason").style.display = "none";
                    document.getElementById("unspun-wheel").style.display = "flex";
                    document.getElementById("unspun-wheel").innerHTML = `
                        You failed to reach your goal for week ${date_str} and must spin the wheel.
                        <div id="canvas-buttons" class="canvas-buttons">
                            <button id="go-to-wheel" onclick="location.replace('./wheel?debt_id=${result.overview.debt_lost[0].ID}');">Take me there</button>
                        </div>
                        `;

                    return;

                } else if(result.overview.debt_unviewed.length > 0 || result.overview.debt_won.length > 0 || result.overview.debt_unpaid.length > 0) {

                    place_debtoverview(result.overview);

                } else {

                    document.getElementById("debt-module").style.display = "none";

                }

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/debt");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_debtoverview(overviewArray) {

    var html = "";

    document.getElementById("debt-module").style.display = "flex";

    for(var i = 0; i < overviewArray.debt_unviewed.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_unviewed[i].debt.date);
            var date_week = date.GetWeek();
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        html += `
            <div class="debt-module-notification-view" id="">
                ${overviewArray.debt_unviewed[i].debt.loser.first_name} spun the wheel for week ${date_str}.<br>See if you won!<br>
                <img src="assets/arrow-right.svg" class="small-button-icon" onclick="location.replace('./wheel?debt_id=${overviewArray.debt_unviewed[i].debt.ID}'); ">
            </div>
            `;
    }

    for(var i = 0; i < overviewArray.debt_won.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_won[i].date);
            var date_week = date.GetWeek();
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        console.log(overviewArray.debt_won)

        html += `
            <div class="debt-module-notification-prize" id="">
                ${overviewArray.debt_won[i].loser.first_name} spun the wheel for week ${date_str} and you won <b>${overviewArray.debt_won[i].season.prize.quantity} ${overviewArray.debt_won[i].season.prize.name}</b>!<br>Have you received it?<br>
                <img src="assets/done.svg" class="small-button-icon" onclick="set_prizereceived(${overviewArray.debt_won[i].ID});">
            </div>
            `;
    }

    for(var i = 0; i < overviewArray.debt_unpaid.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_unpaid[i].date);
            var date_week = date.GetWeek();
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        console.log(overviewArray.debt_unpaid)

        html += `
            <div class="debt-module-notification-debt" id="">
                You spun the wheel for week ${date_str} and ${overviewArray.debt_unpaid[i].winner.first_name} won ${overviewArray.debt_unpaid[i].season.prize.quantity} ${overviewArray.debt_unpaid[i].season.prize.name}!<br>Provide the prize as soon as possible!<br>
            </div>
            `;
    }

    document.getElementById("debt-module-notifications").innerHTML = html;

}

function set_prizereceived(debt_id) {

    if(!confirm("Are you sure?")) {
        return;
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

                get_debtoverview();

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/debt/" + debt_id + "/received");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function get_debtoverview_prepare(show_barbell) {

    document.getElementById("exercises").style.display = "none";
    document.getElementById("leaderboard").style.display = "none";
    document.getElementById("season-module").style.display = "none";
    document.getElementById("current-week").style.display = "none";

    document.getElementById("ongoingseason").style.display = "flex";

    if(show_barbell) {
        document.getElementById("barbell-gif").style.display = "flex";
    }

    get_debtoverview();

}

function delete_goal() {

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

                location.reload();
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/goal/delete");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}
