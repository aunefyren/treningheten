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

                            <a style="margin: 1em 0 0 0; font-size:0.75em; cursor: pointer;" onclick="deleteGoal();">I changed my mind!</i></a>

                        </div>

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

                getDebtOverview();

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
                    countdownModule(season, goal, result.timezone)
                    getDebtOverview();
                } else {
                    registerGoalRedirect();
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

function countdownModule(season_object, exercise_goal, timezone) {

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

    startCountDown(date_start, timezone);
}

function startCountDown(countdownDate){

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
                frontPageRedirect();
            }, 5000);
              
        }
        
    }, 1000);
}

function deleteGoal() {

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

                registerGoalRedirect();
                
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

function registerGoalRedirect() {

    window.location = '../registergoal'

}

function frontPageRedirect() {

    window.location = '../'

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