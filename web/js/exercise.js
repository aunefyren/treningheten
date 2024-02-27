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
        string_index = document.URL.lastIndexOf('/');
        exerciseDayID = document.URL.substring(string_index+1);
        console.log(exerciseDayID);
    } catch {
        exerciseDayID = 0;
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="text-body" style="text-align: center;">
                            <div class="exerciseDayWrapper" id="exerciseDayWrapper">
                                <p id="exercise-day-date" style="text-align: center;">...</p>
                                <p id="exercise-day-exercise-goal" style="text-align: center;">...</p>
                                <p id="exercise-day-note" style="text-align: center;"></p>
                            </div>
                        </div>

                        <hr class="invert" style="border: 0.025em solid var(--white); margin: 2em 0;">

                        <div class="exercisesWrapper" id="exercisesWrapper"></div>

                        <div class="addExerciseWrapper clickable hover" id="addExerciseWrapper" title="Add session" onclick="addExercise();">
                            <img src="/assets/plus.svg" class="button-icon" style="height: 100%; margin: 1em;">
                        </div>

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Write every detail.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        getExerciseDay(exerciseDayID);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }

    alert("This page is not fully functional yet. Coming soon.");
}

function getExerciseDay(exerciseDayID) {
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
                placeExerciseDay(result.exercise_day);
            }

        } else {
            info("Loading exercise day...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/exercises/" + exerciseDayID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeExerciseDay(exerciseDay) {
    var dateString = "Error"
    try {
        var resultDate = new Date(Date.parse(exerciseDay.date));
        dateString = date_start_string = GetDateString(resultDate, true)
    } catch(e) {
        console.log("Error: " + e)
    }

    document.getElementById('exercise-day-date').innerHTML = "Date: " + dateString;
    document.getElementById('exercise-day-exercise-goal').innerHTML = "Exercise goal for week: " + exerciseDay.goal.exercise_interval;

    if(exerciseDay.note) {
        document.getElementById('exercise-day-note').innerHTML = "Note: " + exerciseDay.note;
    }

    placeExercises(exerciseDay.exercises);
}

function placeExercises(exercises) {
    exercisesHTML = "";
    counter = 1; 

    exercises.forEach(exercise => {
        operationsHTML = generateOperationsHTML(exercise.operations, exercise.id)

        onHTML = ""
        restoreHTML = ""
        if(!exercise.on) {
            onHTML = "transparent";
            restoreHTML = `
                <div class="restoreExerciseWrapper" id="exercise-restore-${exercise.id}">
                    <button type="submit" onclick="restoreExercise('${exercise.id}');" id="" style="margin-bottom: 0em; width: 12em; margin: 0.5em;"><img src="/assets/done.svg" class="btn_logo color-invert"><p2>Restore session</p2></button>
                    <button type="submit" onclick="restoreExercise('${exercise.id}');" id="" style="margin-bottom: 0em; width: 12em; margin: 0.5em;"><img src="/assets/done.svg" class="btn_logo color-invert"><p2>Erase session</p2></button>
                </div>
            `;
        }

        noteHTML = "";
        if(exercise.note) {
            noteHTML = `
                <p>${exercise.note}</p>
            `;
        }

        var exerciseHTML = `
            ${restoreHTML}
            <div class="exerciseWrapper ${onHTML}" id="exercise-${exercise.id}">
                <h2>Session ${counter}</h2>
                ${noteHTML}
                <div class="operationsWrapper" id="operationsWrapper">${operationsHTML}</div>
            </div>
            <hr class="invert" style="border: 0.025em solid var(--white); margin: 2em 0;">
        `;
        exercisesHTML += exerciseHTML;
        counter += 1;
    });

    document.getElementById('exercisesWrapper').innerHTML = exercisesHTML;
}

function generateOperationsHTML(operations, exerciseID) {
    operationsHTML = "";

    operations.forEach(operation => {
        operationSetsHTML = generateOperationSetsHTML(operation.operation_sets, operation.id)

        var operationHTML = `
            <div class="operationWrapper" id="exercise-${operation.id}">
                ${operationSetsHTML}
            </div>
        `;
        operationsHTML += operationHTML;
    });

    operationsHTML += `
        <div class="addOperationWrapper clickable hover" id="addOperationWrapper-${exerciseID}" title="Add exercise" onclick="addExercise('${exerciseID}');">
            <img src="/assets/plus.svg" class="button-icon" style="height: 100%; margin: 4em;">
        </div>
    `;

    return operationsHTML
}

function generateOperationSetsHTML(operationSets, operationID) {
    operationSetsHTML = ""; 

    console.log(operationSets)

    operationSets.forEach(operationSet => {
        operationSetHTML = `
            <div class="operationSetWrapper" id="operation-set-${operationSet.id}">
                <div class="operation-set">${operationSet.action}</div>
                <div class="operation-set">${operationSet.repetitions}</div>
                <div class="operation-set">${operationSet.weight}</div>
            </div>
        `;
        operationSetsHTML += operationSetHTML;
    });

    operationSetsHTML += `
        <div class="addOperationSetWrapper clickable hover" id="addOperationSetWrapper-${operationID}" title="Add set" onclick="addExercise('${operationID}');">
            <img src="/assets/plus.svg" class="button-icon" style="height: 100%; margin: 0.25em;">
        </div>
    `;

    return operationSetsHTML;
}