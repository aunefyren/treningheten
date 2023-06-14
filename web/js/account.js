function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            var sunday_alert = login_data.data.sunday_alert
            user_id = login_data.data.id
            admin = login_data.data.admin
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            var sunday_alert = false
            user_id = 0
            admin = false
        }

        showAdminMenu(admin)

    } else {
        var email = ""
        var first_name = ""
        var last_name = ""
        var admin = false;
        var sunday_reminder = false
        user_id = 0
    }

    try {
        string_index = document.URL.lastIndexOf('/');
        wishlist_id = document.URL.substring(string_index+1);

        group_id = 0
    }
    catch {
        group_id = 0
        wishlist_id = 0
    }

    if(sunday_alert) {
        sunday_reminder_str = "checked"
    } else {
        sunday_reminder_str = ""
    }

    var html = `

        <div class="module">

            <div class="user-ative-profile-photo">
                <img style="width: 100%; height: 100%;" class="user-ative-profile-photo-img" id="user-ative-profile-photo-img" src="/assets/images/barbell.gif">
            </div>
        
            <form action="" onsubmit="event.preventDefault(); send_update();">

                <label id="form-input-icon" for="email"></label>
                <input type="email" name="email" id="email" placeholder="Email" value="` + email + `" required/>

                <label id="form-input-icon" for="first_name"></label>
                <input type="text" name="first_name" id="first_name" placeholder="First name" value="` + first_name + `" required disabled />

                <label id="form-input-icon" for="last_name"></label>
                <input type="text" name="last_name" id="last_name" placeholder="Last name" value="` + last_name + `" disabled required/>

                <label id="form-input-icon" for="new_profile_image" style="margin-top: 2em;">Replace profile image:</label>
                <input type="file" name="new_profile_image" id="new_profile_image" placeholder="" value="" accept="image/png, image/jpeg" />

                <input onclick="change_password_toggle();" style="margin-top: 2em;" class="clickable" type="checkbox" id="password-toggle" name="confirm" value="confirm" >
                <label for="password-toggle" class="unselectable clickable">Change my password.</label><br>

                <div id="change-password-box" style="display:none;">

                    <label id="form-input-icon" for="password"></label>
                    <input type="password" name="password" id="password" placeholder="New password" />

                    <label id="form-input-icon" for="password_repeat"></label>
                    <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" />

                </div>

                <input style="margin-top: 2em;" class="clickable" type="checkbox" id="reminder-toggle" name="reminder-toggle" value="reminder-toggle" ` + sunday_reminder_str + `>
                <label for="reminder-toggle" class="unselectable clickable">Send me logging reminders on Sundays.</label><br>

                <button id="update-button" style="margin-top: 2em;" type="submit" href="/">Update account</button>

            </form>

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
                    <img src="./assets/images/barbell.gif">
                </div>

                <div id="season-longest-streak-div" class="text-body">
                </div>

                <div id="season-highest-week-div" class="text-body">
                </div>

                <div id="chart-canvas-div" style="max-width: 40; margin: 1em;">
                    <canvas id="myChart" style="max-width: 100%; width: 1000px; display:none;"></canvas>
                </div>

            </div>

        </div>

    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Your very own page...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        GetProfileImage(user_id);
        get_seasons();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function change_password_toggle() {

    var check_box = document.getElementById("password-toggle").checked;
    var password_box = document.getElementById("change-password-box")

    if(check_box) {
        password_box.style.display = "inline-block"
    } else {
        password_box.style.display = "none"
    }

}

function send_update() {

    var email = document.getElementById("email").value;
    var password = document.getElementById("password").value;
    var password_repeat = document.getElementById("password_repeat").value;
    var sunday_alert = document.getElementById("reminder-toggle").checked;
    var new_profile_image = document.getElementById('new_profile_image').files[0];

    if(new_profile_image) {

        if(new_profile_image.size > 1048576) {
            error("Image exceeds 10MB size limit.")
            return;
        } else if(new_profile_image.size < 10485) {
            error("Image smaller than 0.01MB size requirement.")
            return;
        }

        new_profile_image = get_base64(new_profile_image);
        
        new_profile_image.then(function(result) {
            
            var form_obj = { 
                "email" : email,
                "password" : password,
                "password_repeat": password_repeat,
                "sunday_alert": sunday_alert,
                "profile_image": result
            };

            var form_data = JSON.stringify(form_obj);

            document.getElementById("user-ative-profile-photo-img").src = 'assets/images/barbell.gif';

            send_update_two(form_data);
        
        });

    } else {
        var form_obj = { 
                            "email" : email,
                            "password" : password,
                            "password_repeat": password_repeat,
                            "sunday_alert": sunday_alert,
                            "profile_image": ""
                        };

        var form_data = JSON.stringify(form_obj);
        
        send_update_two(form_data);
    }
}

function send_update_two(form_data) {

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

                success(result.message);

                // store jwt to cookie
                set_cookie("treningheten", result.token, 7);

                if(result.verified) {
                    location.reload();
                } else {
                    location.href = './';
                }
                
            }

        } else {
            info("Updating account...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/user/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

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

                place_seasons(result.seasons);
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/season");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_seasons(seasons_array) {

    var select_season = document.getElementById("select_season");
    var seasons = []

    for(var i = 0; i < seasons_array.length; i++) {
        var user_found = false;
        for(var j = 0; j < seasons_array[i].goals.length; j++) {
            if(seasons_array[i].goals[j].user.ID == user_id) {
                user_found = true;
                break
            }
        }
        if(user_found) {
            seasons.push(seasons_array[i])
        }
    }

    for(var i = 0; i < seasons.length; i++) {
        
        var option = document.createElement("option");
        option.text = seasons_array[i].name
        option.value = seasons_array[i].ID
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
    document.getElementById("season-longest-streak-div").innerHTML = "";
    document.getElementById("season-highest-week-div").innerHTML = "";

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

                place_statistics(result.leaderboard);
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/season/" + seasonID + "/leaderboard-personal");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_statistics(leaderboard_array) {

    var myChartElement = document.getElementById("myChart");
    myChartElement.style.display = "inline-block"

    leaderboard_array = leaderboard_array.reverse();

    var xValues = [];
    var yValues = [];
    var goals = [];
    var pointBackgroundColorArray = [];
    var borderColorArray = [];
    var longest_streak = 0;
    var highest_week = 0;

    // Look through array of data
    for (var i = 0; i < leaderboard_array.length; i++) {

        xValues.push("" + leaderboard_array[i].week_number + " (" + leaderboard_array[i].week_year + ")");

        var exercise = leaderboard_array[i].user.week_completion_interval
        var sickleave = leaderboard_array[i].user.sickleave
        var goal = leaderboard_array[i].user.exercise_goal
        var streak = leaderboard_array[i].user.current_streak

        if(streak > longest_streak) {
            longest_streak = streak;
        }

        if(exercise > highest_week) {
            highest_week = exercise;
        }
        
        if(sickleave) {
            pointBackgroundColorArray.push("rgba(215, 20, 20, 1)")
            borderColorArray.push("rgba(215, 20, 20, 1)")
        } else {
            pointBackgroundColorArray.push("rgba(119,141,169,1)")
            borderColorArray.push("rgba(119,141,169,1)")
        }

        yValues.push(eval(exercise));
        goals.push(eval(goal));
           
    }

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
                    tension: 0.25,
                    label: "Exercise count",
                },
                {
                    fill: true,
                    borderColor: "rgba(119,141,169,0.25)",
                    responsive: true,
                    data: goals,
                    tension: 0.25,
                    label: "Goal",
                }
            ]
        },    
        options: {
            legend: {display: false},
            title: {
                display: true,
                text: "Week exercise count",
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

    if(longest_streak > 0) {
        document.getElementById("season-longest-streak-div").innerHTML = "Longest streak: " + longest_streak + "🔥";
    }

    if(highest_week > 0) {
        document.getElementById("season-highest-week-div").innerHTML = "Most exercise in a week: " + highest_week + "🏋️";
    }

    // Remove loading gif
    document.getElementById("loading-dumbell").style.display = "none";

}

function GetProfileImage(userID) {

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

                PlaceProfileImage(result.image)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/user/get/" + user_id + "/image");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImage(imageBase64) {

    document.getElementById("user-ative-profile-photo-img").src = imageBase64

}