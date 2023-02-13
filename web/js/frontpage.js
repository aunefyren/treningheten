function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        if(login_data.error === "You must verify your account.") {
            load_verify_account();
            return;
        }

        user_id = login_data.data.id

    } else {
        var login_data = false;
        var user_id = 0
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="title">
                            Treningheten
                        </div>

                        <div class="text-body" style="text-align: center;">
                            Workout time.

                            <br>
                            <br>

                            Welcome to the front page. Not much to see here currently.
                        </div>

                    </div>

                    <div class="module" id="registergoal" style="display: none;">

                        <div id="season" class="season">

                            <h3 id="register_season_title">Loading...</h3>
                            <p id="register_season_start">...</p>
                            <p id="register_season_end">...</p>
                            <p style="margin-top: 1em;" id="register_season_desc">...</p>

                            <hr>

                            <label for="commitment" title="How many days a week are you going to work out?">Weekly exercise goal</label>
                            <input type="number" value="1" min="1" max="21" class="form-control-small" id="commitment">
                            <div class="two-buttons">
                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('commitment', 1, 21);">
                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('commitment', 1, 21);">
                            </div>

                            <hr>

                            <input style="" type="checkbox" id="compete" name="compete" value="compete" required>
                            <label for="compete" style="user-select: none; text-align: center;"> I want to compete with others to uphold my workout streak.</label><br>

                            <button type="submit" onclick="register_goal();" id="register_goal_button" style=""><img src="assets/done.svg" class="btn_logo color-invert"><p2>Join season</p2></button>

                        </div>

                    </div>

                    <div class="module" id="ongoingseason" style="display: none;">

                        <div class="modules">

                            <div id="exercises" class="exercises">

                                <div class="week_days" id='calendar'>

                                    <div class="form-group" style="" id="day_1_group">
                                        <div class="day-check">
                                            <label for="day_1_check" title="Have you been working out?">Monday</label>
                                            <input type="number" value="0" min="0" max="3" class="form-control-small" id="day_1_check">
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_1_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_1_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <input type="text" class="form-control" placeholder="Notes" id="day_1_note">
                                        </div>
                                        <input type="hidden" value="" id="day_1_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_2_group">
                                        <div class="day-check">
                                            <label for="day_2_check" title="Have you been working out?">Tuesday</label>
                                            <input type="number" value="0" min="0" max="3" class="form-control-small" id="day_2_check">
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_2_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_2_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <input type="text" class="form-control" placeholder="Notes" id="day_2_note">
                                        </div>
                                        <input type="hidden" value="" id="day_2_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_3_group">
                                        <div class="day-check">
                                            <label for="day_3_check" title="Have you been working out?">Wednesday</label>
                                            <input type="number" value="0" min="0" max="3" class="form-control-small" id="day_3_check">
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_3_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_3_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <input type="text" class="form-control" placeholder="Notes" id="day_3_note">
                                        </div>
                                        <input type="hidden" value="" id="day_3_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_4_group">
                                        <div class="day-check">
                                            <label for="day_4_check" title="Have you been working out?">Thursday</label>
                                            <input type="number" value="0" min="0" max="3" class="form-control-small" id="day_4_check">
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_4_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_4_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <input type="text" class="form-control" placeholder="Notes" id="day_4_note">
                                        </div>
                                        <input type="hidden" value="" id="day_4_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_5_group">
                                        <div class="day-check">
                                            <label for="day_5_check" title="Have you been working out?">Friday</label>
                                            <input type="number" value="0" min="0" max="3" class="form-control-small" id="day_5_check">
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_5_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_5_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <input type="text" class="form-control" placeholder="Notes" id="day_5_note">
                                        </div>
                                        <input type="hidden" value="" id="day_5_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_6_group">
                                        <div class="day-check">
                                            <label for="day_6_check" title="Have you been working out?">Saturday</label>
                                            <input type="number" value="0" min="0" max="3" class="form-control-small" id="day_6_check">
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_6_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_6_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <input type="text" class="form-control" placeholder="Notes" id="day_6_note">
                                        </div>
                                        <input type="hidden" value="" id="day_6_date">
                                    </div>

                                    <hr style="margin: 0.25em;">

                                    <div class="form-group" style="" id="day_7_group">
                                        <div class="day-check">
                                            <label for="day_7_check" title="Have you been working out?">Sunday</label>
                                            <input type="number" value="0" min="0" max="3" class="form-control-small" id="day_7_check">
                                            <div class="two-buttons">
                                                <img src="assets/minus.svg" class="small-button-icon" onclick="DecreaseNumberInput('day_7_check', 0, 3);">
                                                <img src="assets/plus.svg" class="small-button-icon" onclick="IncreaseNumberInput('day_7_check', 0, 3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <input type="text" class="form-control" placeholder="Notes" id="day_7_note">
                                        </div>
                                        <input type="hidden" value="" id="day_7_date">
                                    </div>

                                    <button type="submit" onclick="update_exercises();" id="goal_amount_button" style=""><img src="assets/done.svg" class="btn_logo color-invert"><p2>Save</p2></button>

                                </div>

                            </div>

                            <div class="module-two">

                                <div id="season" class="season">

                                    <h3 id="season_title">Loading...</h3>
                                    <p id="season_desc">...</p>

                                </div>

                                <div id="leaderboard" class="leaderboard">

                                    <h3>Leaderboard...</h3>
                                    <p id="season_desc">Coming soon...</p>

                                </div>

                            </div>

                        </div>

                    </div>

                    <div class="module">

                        <div id="divider-1" class="divider" style="display: none;">
                            <hr></hr>
                        </div>


                        <div id="news-title" class="title" style="display: none;">
                            News:
                        </div>

                        <div id="divider-2" class="divider" style="display: none;">
                            <hr></hr>
                        </div>

                        <div id="news-box" class="news">
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Welcome to the frontpage!';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        //get_news(login_data.admin);
        get_season(user_id);
    } else {
        showLoggedOutMenu();
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
                            <input type="text" name="email_code" id="email_code" placeholder="Code" autocomplete="off" required />
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
                set_cookie("poenskelisten", result.token, 7);
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
            
            if(result.error) {

                error(result.error);

            } else {

                clearResponse();
                season = result.season;

                user_found = false;
                for(var i = 0; i < season.goals.length; i++) {
                    if(season.goals[i].user.ID == user_id) {
                        user_found = true
                        break
                    }
                }

                if(user_found) {
                    document.getElementById("ongoingseason").style.display = "flex"
                    get_calendar(false);
                    place_season(season);
                } else {

                    var date_start = new Date(season.start);
                    var date_end = new Date(season.end);

                    const options = { weekday: 'long', year: 'numeric', month: 'long', day: 'numeric' };

                    document.getElementById("registergoal").style.display = "flex"
                    document.getElementById("register_season_title").innerHTML = season.name
                    document.getElementById("register_season_start").innerHTML = "Season start: " + date_start.toLocaleString("no-NO", options)
                    document.getElementById("register_season_end").innerHTML = "Season end: " + date_end.toLocaleString("no-NO", options)
                    document.getElementById("register_season_desc").innerHTML = season.description
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

function place_season(season_object) {

    document.getElementById("season_title").innerHTML = season_object.name
    document.getElementById("season_desc").innerHTML = season_object.description

}

function register_goal() {

    var exercise_goal = Number(document.getElementById("commitment").value);
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

    document.getElementById("day_1_check").value = week.days[0].exercise_interval
    document.getElementById("day_1_note").value = week.days[0].note
    document.getElementById("day_1_date").value = week.days[0].date

    document.getElementById("day_2_check").value = week.days[1].exercise_interval
    document.getElementById("day_2_note").value = week.days[1].note
    document.getElementById("day_2_date").value = week.days[1].date

    document.getElementById("day_3_check").value = week.days[2].exercise_interval
    document.getElementById("day_3_note").value = week.days[2].note
    document.getElementById("day_3_date").value = week.days[2].date

    document.getElementById("day_4_check").value = week.days[3].exercise_interval
    document.getElementById("day_4_note").value = week.days[3].note
    document.getElementById("day_4_date").value = week.days[3].date

    document.getElementById("day_5_check").value = week.days[4].exercise_interval
    document.getElementById("day_5_note").value = week.days[4].note
    document.getElementById("day_5_date").value = week.days[4].date

    document.getElementById("day_6_check").value = week.days[5].exercise_interval
    document.getElementById("day_6_note").value = week.days[5].note
    document.getElementById("day_6_date").value = week.days[5].date

    document.getElementById("day_7_check").value = week.days[6].exercise_interval
    document.getElementById("day_7_note").value = week.days[6].note
    document.getElementById("day_7_date").value = week.days[6].date

    return

}

function update_exercises() {

    var day_1_check = document.getElementById("day_1_check").value
    var day_1_note = document.getElementById("day_1_note").value
    var day_1_date = document.getElementById("day_1_date").value

    var day_2_check = document.getElementById("day_2_check").value
    var day_2_note = document.getElementById("day_2_note").value
    var day_2_date = document.getElementById("day_2_date").value

    var day_3_check = document.getElementById("day_3_check").value
    var day_3_note = document.getElementById("day_3_note").value
    var day_3_date = document.getElementById("day_3_date").value

    var day_4_check = document.getElementById("day_4_check").value
    var day_4_note = document.getElementById("day_4_note").value
    var day_4_date = document.getElementById("day_4_date").value

    var day_5_check = document.getElementById("day_5_check").value
    var day_5_note = document.getElementById("day_5_note").value
    var day_5_date = document.getElementById("day_5_date").value

    var day_6_check = document.getElementById("day_6_check").value
    var day_6_note = document.getElementById("day_6_note").value
    var day_6_date = document.getElementById("day_6_date").value

    var day_7_check = document.getElementById("day_7_check").value
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
        ]
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
    var old_number = Number(input_element.value)
    var new_number = old_number + 1
    if(new_number <= max && new_number >= min) {
        input_element.value = new_number
    }
}

function DecreaseNumberInput(input_id, min, max) {
    var input_element = document.getElementById(input_id)
    var old_number = Number(input_element.value)
    var new_number = old_number - 1
    if(new_number <= max && new_number >= min) {
        input_element.value = new_number
    }
}