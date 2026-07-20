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

    try {
        // Get parameters from URL string
        var url_string = window.location.href
        var url = new URL(url_string);
        var seasonID = url.searchParams.get("season_id");

        if(!seasonID) {
            error("Invalid season chosen...");
            return
        }
    } catch(e) {
        console.log("Invalid season chosen. Error: " + e);
        error("Invalid season chosen...");
        return
    }

    var html = `
        <div class="" id="front-page">
            
            <div class="module" id="registergoal" style="display: none;">

                <div id="season" class="season">

                    <h3 id="register_season_title" class="u-mb-2">Loading...</h3>
                    <p id="register_season_start">...</p>
                    <p id="register_season_end">...</p>
                    <p class="u-mt-1 u-text-center" id="register_season_desc">...</p>

                    <p class="u-mt-1 u-text-center" id="register_season_jointext">...</p>

                    <hr class="u-my-1">

                    <label for="commitment" class="field-label u-text-center" title="How many days a week are you going to work out?">Weekly exercise goal</label>
                    <div class="number-box" id="commitment">
                        0
                    </div>
                    <div class="two-buttons">
                        <img src="assets/minus.svg" class="btn btn--icon clickable" onclick="DecreaseNumberInput('commitment', 1, 21);">
                        <img src="assets/plus.svg" class="btn btn--icon clickable" onclick="IncreaseNumberInput('commitment', 1, 21);">
                    </div>

                    <hr class="u-my-1">

                    <div class="field-check">
                        <input type="checkbox" id="compete" name="compete" value="compete">
                        <label for="compete" title="If I fail to complete my goal, I must spin a wheel of fortune and provide a prize to the winner.">I want to compete with others to uphold my workout streak.</label>
                    </div>

                    <p id="prize-title" class="u-mt-2">Potential prize:</p>
                    <div class="prize-wrapper">
                        <div id="register-prize-text" class="prize-text">...</div>
                    </div>

                    <hr class="u-my-1">

                    <button type="submit" onclick="registerGoal('${seasonID}');" id="register_goal_button" class="btn btn--primary btn--block"><img src="assets/done.svg" class="color-invert">Join season</button>

                </div>

            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Write every detail.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        getSeason(user_id, seasonID);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function getSeason(userID, seasonID){

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

            } else if(result.error) {

                error(result.error);
                error_splash_image();

            } else {

                clearResponse();
                season = result.season;

                // Check if user has a goal
                user_found = false;
                for(var i = 0; i < season.goals.length; i++) {
                    if(season.goals[i].user.id == userID) {
                        user_found = true
                        var goal = season.goals[i].exercise_interval
                        break
                    }
                }

                var date_start = new Date(season.start);
                var date_end = new Date(season.end);
                var now = Date.now();

                if(user_found) {
                    frontPageRedirect();
                } else if((now > date_start && !season.join_anytime) && now < date_end) {
                    error("It is too late to join this season.");
                    return;
                } else if((now < date_start || season.join_anytime) && now < date_end) {
                    registerGoalModule(season)
                } else {
                    error("Logic error :/");
                    return;
                }

            }

        } else {
            info("Loading season...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/" + seasonID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function frontPageRedirect() {

    window.location = '/'

}

function registerGoalModule(season_object) {

    var date_start = new Date(season_object.start);
    var date_end = new Date(season_object.end);

    var joinText = "..."
    if(season_object.join_anytime) {
        joinText = "<b>You can join at any point in the season.</b>"
    } else {
        joinText = "<b>You must join before the start date.</b>"
    }

    document.getElementById("registergoal").style.display = "flex"
    document.getElementById("register_season_title").innerHTML = season_object.name
    document.getElementById("register_season_start").innerHTML = "Season start: " + GetDateString(date_start, true)
    document.getElementById("register_season_end").innerHTML = "Season end: " + GetDateString(date_end, true)
    document.getElementById("register_season_desc").innerHTML = season_object.description
    document.getElementById("register_season_jointext").innerHTML = joinText;
    document.getElementById("register-prize-text").innerHTML = season_object.prize.quantity + " " + season_object.prize.name
}

function registerGoal(seasonID) {

    var exercise_goal = Number(document.getElementById("commitment").innerHTML);
    var goal_compete = document.getElementById("compete").checked

    var form_obj = {
        "exercise_interval": exercise_goal,
        "competing": goal_compete,
        "season_id": seasonID

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

                frontPageRedirect();

            }

        } else {
            info("Saving goal...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/goals");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

