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

                                <textarea class="day-note-area" id="exercise-day-note" name="exercise-day-exercise-note" rows="3" cols="33" placeholder="Notes" style="margin-top: 1em;"></textarea>
                            
                                <button type="submit" onclick="updateExerciseDay('${exerciseDayID}');" id="updateExerciseDayButton" style="margin-bottom: 0em;"><img src="/assets/done.svg" class="btn_logo color-invert"><p2>Save</p2></button>
                            </div>
                        </div>

                        <hr class="invert" style="border: 0.025em solid var(--white); margin: 4em 0;">

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
    xhttp.open("get", api_url + "auth/exercise-days/" + exerciseDayID);
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

    document.getElementById('exercise-day-date').innerHTML = "<b>Date: " + dateString + "</b>";
    document.getElementById('exercise-day-exercise-goal').innerHTML = "Exercise goal for week: " + exerciseDay.goal.exercise_interval;
    document.getElementById('exercise-day-note').innerHTML = exerciseDay.note;

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
                <h2 style="">Session ${counter}</h2>

                <textarea class="day-note-area" id="exercise-note-${exercise.id}" name="exercise-exercise-note" rows="3" cols="33" placeholder="Notes" style="margin-top: 1em; width: 20em;"></textarea>

                <div class="operationsWrapper" id="operationsWrapper-${exercise.id}">${operationsHTML}</div>

                <button type="submit" onclick="addExercise('${exercise.id}');" id="updateExerciseButton" style="margin-bottom: 0em; margin-top: 2em; width: 8em;"><img src="/assets/done.svg" class="btn_logo color-invert"><p2>Save</p2></button>
            </div>
            <hr class="invert" style="border: 0.025em solid var(--white); margin: 4em 0;">
        `;
        exercisesHTML += exerciseHTML;
        counter += 1;
    });

    document.getElementById('exercisesWrapper').innerHTML = exercisesHTML;
}

function generateOperationsHTML(operations, exerciseID) {
    operationsHTML = `<div class="operationsWrapperSub" id="operationsWrapper-sub-${exerciseID}">`;

    operations.forEach(operation => {
        operationHTML = generateOperationHTML(operation, exerciseID)
        operationsHTML += operationHTML;
    });

    operationsHTML += `</div>`

    operationsAddButtonHTML = generateOperationAddButtonHTML(operations, exerciseID)
    operationsHTML += operationsAddButtonHTML;

    return operationsHTML
}

function generateOperationAddButtonHTML(operations, exerciseID) {
    return `
        <div class="addOperationWrapper clickable hover" id="addOperationWrapper-${exerciseID}" title="Add exercise" onclick="addOperation('${exerciseID}');">
            <img src="/assets/plus.svg" class="button-icon" style="height: 100%; margin: 4em;">
        </div>
    `;
}

function generateOperationHTML(operation, exerciseID) {
    operationSetsHTML = generateOperationSetsHTML(operation.operation_sets, operation.id, operation.weight_unit)

    var operationHTML = `
        <div class="operationWrapper" id="operation-${operation.id}">

            <div class="operation-selectors">
                <div class="operationType" id="operation-type-${operation.id}">
                    <input style="" class="operation-type-input" type="text" id="operation-type-text" name="operation-type-text" value="${operation.type}">
                </div>
                <div class="operationAction" id="operation--action-${operation.id}">
                    <input style="" class="operation-action-input" type="text" id="operation-action-text" name="operation-action-text" value="${operation.action}">
                </div>
            </div>

            ${operationSetsHTML}
        </div>
    `;

    return operationHTML;
}

function generateOperationSetsHTML(operationSets, operationID, weightUnit) {
    operationSetsHTML = ""; 

    operationSetsHTML += `
        <div class="operationSetWrapper" id="operation-set-titles" style="justify-content:space-between;">
            <div class="operation-set-title">
                sets
            </div>
            <div class="operation-set-title">
                reps
            </div>
            <div class="operation-set-title">
                ${weightUnit}
            </div>
        </div>

        <div class="operationSetWrapperSub" id="operation-set-wrapper-sub-${operationID}">
    `;

    setCounter = 1;
    operationSets.forEach(operationSet => {
        operationSetHTML = generateOperationSetHTML(operationSet, weightUnit, setCounter)
        operationSetsHTML += operationSetHTML;
        setCounter += 1;
    });

    operationSetsHTML += `
        </div>
        <div class="addOperationSetWrapper clickable hover" id="addOperationSetWrapper-${operationID}" title="Add set" onclick="addOperationSet('${operationID}', '${weightUnit}');" style="margin: 0.5em 0;">
            <img src="/assets/plus.svg" class="button-icon" style="height: 100%; margin: 0.25em;">
        </div>
    `;

    return operationSetsHTML;
}

function generateOperationSetHTML(operationSet, weightUnit, setCounter) {
    return `
        <div class="operationSetWrapper" id="operation-set-${operationSet.id}">
            <div class="operation-set unselectable">
                Set ${setCounter}
            </div>
            <div class="operation-set-input">
                <input style="" class="operation-set-rep-input" type="number" id="operation-set-rep-input" name="operation-set-rep-input" placeholder="reps" value="${operationSet.repetitions}">
            </div>
            <div class="operation-set-input">
                <input style="" class="operation-set-weight-input" type="number" id="operation-set-weight-input" name="operation-set-weight-input" placeholder="${weightUnit}" value="${operationSet.weight}">
            </div>
        </div>
    `;
}

function updateExerciseDay(exerciseDayID) {
    var exerciseDayNote = document.getElementById("exercise-day-note").value;

    console.log(exerciseDayNote)

    var form_obj = {
        "note": exerciseDayNote
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
                document.getElementById('exercise-day-note').innerHTML = result.exercise_day.note;
                success("Saved.")
            }

        } else {
            info("Updating...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercise-days/" + exerciseDayID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function getOperation(operationID) {
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
                console.log(result)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/operations/" + operationID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function getOperationSets(operationID) {
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
                console.log(result)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/operation-sets?operation_id=" + encodeURI(operationID));
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function addOperation(exerciseID) {
    var form_obj = {
        "exercise_id": exerciseID,
        "type": "lifting",
        "weight_unit": "kg",
        "distance_unit": "km"
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
                placeOperation(result.operation, exerciseID)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/operations");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeOperation(operation, exerciseID) {
    document.getElementById('operationsWrapper-sub-' + exerciseID).innerHTML += generateOperationHTML(operation, exerciseID)
}

function addOperationSet(operationID, weightUnit) {
    var form_obj = {
        "operation_id": operationID,
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
                placeOperationSet(result.operation_set, operationID, weightUnit)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/operation-sets");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeOperationSet(operationSet, operationID, weightUnit) {
    count = document.getElementById('operation-set-wrapper-sub-' + operationID).children.length
    document.getElementById('operation-set-wrapper-sub-' + operationID).innerHTML += generateOperationSetHTML(operationSet, weightUnit, count + 1)
}