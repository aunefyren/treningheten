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
                <!-- The Modal -->
                <div id="myModal" class="modal closed">
                    <span class="close clickable" onclick="toggleModal()">&times;</span>
                    <div class="modalContent" id="modalContent">
                    </div>
                    <div id="caption"></div>
                </div>

                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="text-body" style="text-align: center;">
                            Here you can see statistics and track certain health values.
                        </div>

                    </div>

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

                        <div class="module" id="loading-dumbbell" style="display: none;">
                            <img src="/assets/images/barbell.gif">
                        </div>

                        <div id="season-statistics-element-wrapper-div" class="season-statistics-element-wrapper-div">
                        </div>

                        <div id="chart-canvas-div" style="max-width: 40em; margin: 1em auto; padding: 0 0.5em; background-color: var(--white); border-radius: 1em;">
                            <canvas id="myChart" style="max-width: 100%; width: 1000px; display:none;"></canvas>
                        </div>

                        <div id="chart-canvas-div-two" style="max-width: 40em; margin: 1em auto; padding: 0 0.5em; background-color: var(--white); border-radius: 1em;">
                            <canvas id="myChartTwo" style="max-width: 100%; width: 1000px; display:none;"></canvas>
                        </div>

                    </div>

                </div>

                <div class="module color-invert">
                    <hr>
                </div>

                <div class="module" id="weight_module">
                    <div class="title">
                        Weight statistics
                    </div>

                    <div class="addActionWrapper clickable hover" id="" title="Weight data" onclick="getWeights(true);" style="">
                        <img src="/assets/database.svg" class="button-icon" style="width: 100%; margin: 0.25em;">
                    </div>

                    <div id="chart-canvas-div" style="max-width: 40em; margin: 1em auto; padding: 0 0.5em; background-color: var(--white); border-radius: 1em;">
                        <canvas id="myChartWeights" style="max-width: 100%; width: 1000px; display:none;"></canvas>
                    </div>
                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Math and stuff.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        getSeasons();
        getWeights();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function getSeasons(){
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
                placeSeasonsInput(seasons);
            }
        } else {
            info("Loading seasons...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeSeasonsInput(seasons_array) {
    var select_season = document.getElementById("select_season");
    var seasons = []
    var now = new Date();

    for(var i = 0; i < seasons_array.length; i++) {
        var user_found = false;
        for(var j = 0; j < seasons_array[i].goals.length; j++) {
            if(seasons_array[i].goals[j].user.id == user_id) {
                user_found = true;
                break
            }
        }

        var futureSeason = true;
        try {
            var season_start_object = new Date(seasons_array[i].start);
            if(now > season_start_object) {
                futureSeason = false;
            }
        } catch(e) {
            console.log("Failed to parse season start. Error: " + e)
        }

        if(user_found && !futureSeason) {
            seasons.push(seasons_array[i])
        }
    }

    for(var i = 0; i < seasons.length; i++) {
        var option = document.createElement("option");
        option.text = seasons[i].name
        option.value = seasons[i].id
        select_season.add(option); 
    }
}

function choose_season() {

    var select_season = document.getElementById("select_season");

    // Show loading gif
    document.getElementById("loading-dumbbell").style.display = "inline-block";

    // Purge data
    canvas_div = document.getElementById("chart-canvas-div");
    canvas_div.innerHTML = "";
    canvas_div.innerHTML = '<canvas id="myChart" style="max-width: 100%; width: 1000px; display:none;"></canvas>';

    canvas_div_two = document.getElementById("chart-canvas-div-two");
    canvas_div_two.innerHTML = "";
    canvas_div_two.innerHTML = '<canvas id="myChartTwo" style="max-width: 100%; width: 1000px; display:none;"></canvas>';

    document.getElementById("season-statistics-element-wrapper-div").innerHTML = "";

    if(select_season.value == null || select_season.value == 0 || select_season.value == "null") {

        // Show loading gif
        document.getElementById("loading-dumbbell").style.display = "none";

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

                place_statistics(result.leaderboard, result.weekdays, result.wheel_statistics);
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/" + seasonID + "/weeks-personal");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_statistics(leaderboard_array, weekday_array, wheel_statistics) {

    var myChartElement = document.getElementById("myChart");
    myChartElement.style.display = "inline-block"

    var myChartElement = document.getElementById("myChartTwo");
    myChartElement.style.display = "inline-block"

    leaderboard_array = leaderboard_array.reverse();

    var xValues = [];
    var yValues = [];
    var goals = [];
    var pointBackgroundColorArray = [];
    var borderColorArray = [];
    var longest_streak = 0;
    var highest_week = 0;
    var exercise_amount = 0;
    var sickleave_amount = 0;
    var goal = 0;
    var week_count = 0;
    var complete_weeks = 0;
    var incomplete_weeks = 0;
    var wheels_won = wheel_statistics.wheels_won;
    var wheels_lost = wheel_statistics.wheel_spins;

    // Look through array of data
    for (var i = 0; i < leaderboard_array.length; i++) {

        xValues.push("" + leaderboard_array[i].week_number + " (" + leaderboard_array[i].week_year + ")");

        var exercise = leaderboard_array[i].user.week_completion_interval
        var sickleave = leaderboard_array[i].user.sick_leave
        goal = leaderboard_array[i].user.exercise_goal
        var streak = leaderboard_array[i].user.current_streak
        exercise_amount = exercise_amount + exercise

        if(streak > longest_streak) {
            longest_streak = streak;
        }

        if(exercise > highest_week) {
            highest_week = exercise;
        }
        
        if(sickleave) {
            pointBackgroundColorArray.push("rgba(215, 20, 20, 1)")
            borderColorArray.push("rgba(215, 20, 20, 1)")
            sickleave_amount = sickleave_amount + 1
        } else {
            pointBackgroundColorArray.push("rgba(119,141,169,1)")
            borderColorArray.push("rgba(119,141,169,1)")
        }

        yValues.push(eval(exercise));
        goals.push(eval(goal));

        if(exercise >= goal) {
            complete_weeks = complete_weeks + 1
        } else {
            incomplete_weeks = incomplete_weeks + 1
        }

        week_count = week_count + 1
           
    }

    console.log("goal: " + goal)
    console.log("weeks: " + week_count)
    console.log("complete weeks: " + complete_weeks)

    week_completion_percentage = Math.floor((complete_weeks / week_count) * 100)

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
                    tension: 0,
                    label: "Exercise count",
                },
                {
                    fill: true,
                    borderColor: "rgba(119,141,169,0.25)",
                    responsive: false,
                    data: goals,
                    tension: 0,
                    label: "Goal",
                }
            ]
        },    
        options: {
            legend: {display: false},
            title: {
                display: true,
                text: "Weekly exercise graph",
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


    var xValues2 = ["Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"]
    var yValues2 = [weekday_array.monday, weekday_array.tuesday, weekday_array.wednesday, weekday_array.thursday, weekday_array.friday, weekday_array.saturday, weekday_array.sunday]

    const lineChartTwo = new Chart("myChartTwo", {
        type: "line",
        data: {
            labels: xValues2,
            datasets: [
                {
                    fill: true,
                    borderColor: "rgba(119,141,169,1)",
                    pointBackgroundColor: "rgba(119,141,169,1)",
                    backgroundColor: "rgba(119,141,169,0.5)",
                    responsive: true,
                    data: yValues2,
                    tension: 0,
                    label: "Exercise count",
                }
            ]
        },    
        options: {
            legend: {display: false},
            title: {
                display: true,
                text: "Weekday exercise graph",
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

    if(goal > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Weekly exercise goal: ${goal}🏆
            </div>
        `;
    }

    if(week_completion_percentage > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Average weekly goal completion: ${week_completion_percentage}%📊
            </div>
        `;
    }

    if(longest_streak > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Longest week streak: ${longest_streak}🔥
            </div>
        `;
    }

    if(highest_week > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Most exercise in a week: ${highest_week}🏋️
            </div>
        `;
    }

    if(exercise_amount > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                All exercise combined: ${exercise_amount}💰
            </div>
        `;
    }

    if(sickleave_amount > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Weeks of sick leave: ${sickleave_amount}🤢
            </div>
        `;
    }

    if(wheels_lost > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Wheels spun: ${wheels_lost}🎡
            </div>
        `;
    }

    if(wheels_won > 0) {
        document.getElementById("season-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Wheels won: ${wheels_won}⭐
            </div>
        `;
    }
    
    // Remove loading gif
    document.getElementById("loading-dumbbell").style.display = "none";

}

function getWeights(path) {
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
                weights = result.weights;

                if(!path) {
                    placeWeights(weights)
                } else {
                    viewWeight(weights)
                }
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/weights");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeWeights(weightsArray) {
    var myChartElement = document.getElementById("myChartWeights");
    myChartElement.style.display = "inline-block"

    weightsArray = weightsArray.reverse();

    var xValues = [];
    var yValues = [];
    var pointBackgroundColorArray = [];
    var borderColorArray = [];

    // Look through array of data
    for (var i = 0; i < weightsArray.length; i++) {
        try {
            var dateObject = new Date(weightsArray[i].date)
            var isoString = dateObject.toISOString();
            // Split at the "T" character to get the date part
            var formattedDate = isoString.split("T")[0];
        } catch (error) {
            continue
        }

        xValues.push(formattedDate);
        yValues.push(weightsArray[i].weight);
        pointBackgroundColorArray.push("rgba(119,141,169,1)")
        borderColorArray.push("rgba(119,141,169,1)")
    }

    const lineChart = new Chart("myChartWeights", {
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
                    tension: 0,
                    label: "Weight in KG",
                }
            ]
        },    
        options: {
            legend: {display: false},
            title: {
                display: true,
                text: "Weight over time",
                fontSize: 16
            },
            scales: {
                yAxes: [
                    {
                        ticks: {
                            beginAtZero: true,
                            precision: 0,
                            //suggestedMin: 50,
                            beginAtZero: false,
                        }
                    }
                ]
            }
        }
    });
}

function viewWeight(weights) {
    var now = new Date();

    weightsHTML = ``;
    for (let index = 0; index < weights.length; index++) {
        const weight = weights[index];
        dateObject = new Date(weight.date)
        const timeString = GetDateString(dateObject, false)
        weightsHTML += `
            <div class="weight-value">
                <div style="width: 8em;"><div style="font-size: 0.75em;">${timeString}</div></div>
                <div style="width: 5em;">${weight.weight} KG</div>
                <div style="width: 8em; display: flex; justify-content: flex-end;">
                    <div class="addActionWrapper clickable hover" id="" title="Weight data" onclick="deleteWeight('${weight.id}');" style="">
                        <img src="/assets/trash-2.svg" class="button-icon" style="width: 100%; margin: 0.25em;">
                    </div>
                </div>
            </div>
        `;
    }

    var htmlContent = `
        <div class="weight-input-wrapper">
            <div class="weight-input">
                <label for="weightValue">Weight (KG)</label><br>
                <input type="number" name="weightValue" id="weightValue" placeholder="" autocomplete="off" min="0" max="500" value="0" />
            </div>
            <div class="weight-input">
                <label for="weightTime">Time of weight</label><br>
                <input type="date" name="weightTime" id="weightTime" style="min-width: 10em;" placeholder="" autocomplete="off" value="${now.toISOString().split('T')[0]}" />
            </div>
            <div><button id="register-button" type="submit" href="/" onclick="addWeight()" style="width: 5em;">Save</button></div>
        </div>
        <hr>
        <div class="weight-values-wrapper">
            ${weightsHTML}
        </div>
    `;

    toggleModal(htmlContent);
}

function deleteWeight(weightID) {
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
                toggleModal();
                error(result.error);
            } else {
                getWeights(true);
                getWeights(false);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/weights/" + weightID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function addWeight() {
    var weightValueString = document.getElementById("weightValue").value;
    var weightTimeString = document.getElementById("weightTime").value;

    try {
        weightValue = parseFloat(weightValueString)
        weightDate = new Date(weightTimeString)
        weightTimeString = weightDate.toISOString()
    } catch (error) {
        console.log(error)
    }

    var form_obj = { 
        "weight" : weightValue,
        "date" : weightTimeString
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
                toggleModal();
                error(result.error);
            } else {
                getWeights(true);
                getWeights(false);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/weights");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}