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

                            <button type="submit" onclick="registerGoal();" id="register_goal_button" style=""><img src="assets/done.svg" class="btn_logo color-invert"><p2>Join season</p2></button>

                        </div>

                    </div>

                    <div class="module" id="unspun-wheel" style="display: none;">

                    </div>

                    <div class="module">
                        <div id="debt-module" class="debt-module" style="display: none;">

                            <h3 id="debt-module-title">Prizes</h3>

                            <div id="debt-module-notifications" class="debt-module-notifications">
                            </div>

                        </div>
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Write every detail.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        getSeason(user_id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function getSeason(userID){

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

                getDebtoverview();

            } else if(result.error) {

                error(result.error);
                error_splash_image();

            } else {

                clearResponse();
                season = result.season;

                // Check if user has a goal
                user_found = false;
                for(var i = 0; i < season.goals.length; i++) {
                    if(season.goals[i].user.ID == userID) {
                        user_found = true
                        var goal = season.goals[i].exercise_interval
                        break
                    }
                }

                var date_start = new Date(season.start);
                var now = Date.now();

                if(user_found && now < date_start) {
                    countdownRedirect();
                } else if(user_found) {
                    frontPageRedirect();
                } else {
                    registerGoalModule(season)
                }

                getDebtOverview();

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

function countdownRedirect() {

    window.location = '/countdown'

}

function frontPageRedirect() {

    window.location = '/'

}

function registerGoalModule(season_object) {

    var date_start = new Date(season_object.start);
    var date_end = new Date(season_object.end);

    document.getElementById("registergoal").style.display = "flex"
    document.getElementById("register_season_title").innerHTML = season_object.name
    document.getElementById("register_season_start").innerHTML = "Season start: " + GetDateString(date_start, true)
    document.getElementById("register_season_end").innerHTML = "Season end: " + GetDateString(date_end, true)
    document.getElementById("register_season_desc").innerHTML = season_object.description
    document.getElementById("register-prize-text").innerHTML = season_object.prize.quantity + " " + season_object.prize.name
}

function registerGoal() {

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

                countdownRedirect();

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


function placeDebtOverview(overviewArray) {

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
                ${overviewArray.debt_unviewed[i].debt.loser.first_name} ${overviewArray.debt_unviewed[i].debt.loser.last_name} spun the wheel for week ${date_str}.<br>See if you won!<br>
                <img src="assets/arrow-right.svg" class="small-button-icon" onclick="location.replace('/wheel?debt_id=${overviewArray.debt_unviewed[i].debt.ID}'); ">
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
                ${overviewArray.debt_won[i].loser.first_name} ${overviewArray.debt_won[i].loser.last_name} spun the wheel for week ${date_str} and you won <b>${overviewArray.debt_won[i].season.prize.quantity} ${overviewArray.debt_won[i].season.prize.name}</b>!<br>Have you received it?<br>
                <img src="assets/done.svg" class="small-button-icon" onclick="setPrizeReceived(${overviewArray.debt_won[i].ID});">
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
        var date_week = date.GetWeek();
        var date_year = date.getFullYear();
        var date_str = date_week + " (" + date_year + ")"
    } catch {
        date_str = "Error"
    }
    
    document.getElementById("registergoal").style.display = "none";
    document.getElementById("unspun-wheel").style.display = "flex";
    document.getElementById("unspun-wheel").innerHTML = `
        You failed to reach your goal for week ${date_str} and must spin the wheel.
        <div id="canvas-buttons" class="canvas-buttons">
            <button id="go-to-wheel" onclick="location.replace('/wheel?debt_id=${overview.debt_lost[0].ID}');">Take me there</button>
        </div>
    `;
    return;
}

