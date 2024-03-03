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
        loadExerciseList()
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

                <textarea class="day-note-area" id="exercise-note-${exercise.id}" name="exercise-exercise-note" rows="3" cols="33" placeholder="Notes" style="margin-top: 1em; width: 20em;" onchange="updateExercise('${exercise.id}')">${exercise.note}</textarea>

                <div class="operationsWrapper" id="operationsWrapper-${exercise.id}">${operationsHTML}</div>
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
        operationsHTML += `
            <div class="operationWrapper" id="operation-${operation.id}">
                ${operationHTML}
            </div>
        `;
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

function generateOperationHTML(operation) {
    liftingHTML = ''
    timingHTML = ''
    movingHTML = ''
    if(operation.type == 'lifting') {
        liftingHTML = 'selected'
    } else if(operation.type == 'timing') {
        timingHTML = 'selected'
    } else if(operation.type == 'moving') {
        movingHTML = 'selected'
    }

    var operationHTML = `
        <div class="operation-selectors">
            <div class="operationType" id="operation-type-${operation.id}">
                <select class="operation-type-input" type="text" id="operation-type-text-${operation.id}" name="operation-type-text" onchange="updateOperation('${operation.id}')">
                    <option value="lifting" ${liftingHTML}>💪</option>
                    <option value="moving" ${movingHTML}>🏃‍♂️</option>
                    <option value="timing" ${timingHTML}>⏱️</option>
                </select>  
            </div>
            <div class="operationAction" id="operation-action-${operation.id}">
                <input style="" class="operation-action-input" type="text" list="operation-action-text-list-${operation.id}" id="operation-action-text-${operation.id}" name="operation-action-text" placeholder="exercise" value="${operation.action}" onchange="updateOperation('${operation.id}')">
                <datalist id="operation-action-text-list-${operation.id}">
                    ${exerciseListHTML}
                </datalist>
            </div>
            <input type="hidden" id="operation-distance-unit-${operation.id}" value="${operation.distance_unit}">
            <input type="hidden" id="operation-weight-unit-${operation.id}" value="${operation.weight_unit}">
        </div>

        ${generateOperationSetsHTML(operation.operation_sets, operation)}
    `;

    return operationHTML;
}

function generateOperationSetsHTML(operationSets, operation) {
    operationSetsHTML = ""; 

    repsHTML = 'block'
    distanceHTML = 'block'
    timingHTML = 'block'
    weightHTML = 'block'
    if(operation.type == 'lifting') {
        distanceHTML = 'none'
        timingHTML = 'none'
    } else if(operation.type == 'timing') {
        repsHTML = 'none'
        distanceHTML = 'none'
        weightHTML = 'none'
    } else if(operation.type == 'moving') {
        repsHTML = 'none'
        weightHTML = 'none'
    }

    operationSetsHTML += `
        <div class="operationSetWrapper" id="operation-set-titles" style="justify-content:space-between;">
            <div class="operation-set-title">
                sets
            </div>
            <div class="operation-set-title" style="display: ${repsHTML};">
                reps
            </div>
            <div class="operation-set-title" style="display: ${weightHTML};">
                ${operation.weight_unit}
            </div>
            <div class="operation-set-title" style="display: ${timingHTML};">
                time
            </div>
            <div class="operation-set-title" style="display: ${distanceHTML};">
                ${operation.distance_unit}
            </div>
        </div>

        <div class="operationSetWrapperSub" id="operation-set-wrapper-sub-${operation.id}">
    `;

    setCounter = 1;
    operationSets.forEach(operationSet => {
        operationSetsHTML += `
            <div class="operationSetWrapper" id="operation-set-${operationSet.id}">
                ${generateOperationSetHTML(operationSet, operation, setCounter)}
            </div>
        `;
        setCounter += 1;
    });

    operationSetsHTML += `
        </div>
        <div class="addOperationSetWrapper clickable hover" id="addOperationSetWrapper-${operation.id}" title="Add set" onclick="addOperationSet('${operation.id}');" style="margin: 0.5em 0;">
            <img src="/assets/plus.svg" class="button-icon" style="height: 100%; margin: 0.25em;">
        </div>
    `;

    return operationSetsHTML;
}

function generateOperationSetHTML(operationSet, operation, setCounter) {
    repsHTML = 'block'
    distanceHTML = 'block'
    timingHTML = 'block'
    weightHTML = 'block'
    if(operation.type == 'lifting') {
        distanceHTML = 'none'
        timingHTML = 'none'
    } else if(operation.type == 'timing') {
        repsHTML = 'none'
        distanceHTML = 'none'
        weightHTML = 'none'
    } else if(operation.type == 'moving') {
        repsHTML = 'none'
        weightHTML = 'none'
    }

    var reps = ""
    if(operationSet.repetitions) {
        reps = operationSet.repetitions
    }
    var weight = ""
    if(operationSet.weight) {
        weight = operationSet.weight
    }
    var time = ""
    if(operationSet.time) {
        var hourString = '';
        var minutes = operationSet.time
        var hours = Math.floor(operationSet.time / 3600)
        if(hours != 0) {
            hourString = padNumber(hours, 2) + ":"
            minutes = operationSet.time % 3600
        }

        var minutesString = padNumber(Math.floor(minutes / 60), 2)
        var secondsString = padNumber((operationSet.time % 60), 2)
        time = hourString + minutesString + ':' + secondsString
    }
    var distance = ""
    if(operationSet.distance != null) {
        distance = operationSet.distance
    }

    return `
        <div class="operation-set unselectable" id="operation-set-counter-${operationSet.id}">
            Set ${setCounter}
        </div>
        <div class="operation-set-input" id="operation-set-rep-${operationSet.id}" style="display: ${repsHTML};">
            <input style="" min="0" class="operation-set-rep-input" type="number" id="operation-set-rep-input-${operationSet.id}" name="operation-set-rep-input" placeholder="reps" value="${reps}" onchange="updateOperationSet('${operationSet.id}', '${setCounter}')">
        </div>
        <div class="operation-set-input" id="operation-set-weight-${operationSet.id}" style="display: ${weightHTML};">
            <input style="" min="0" class="operation-set-weight-input" type="number" id="operation-set-weight-input-${operationSet.id}" name="operation-set-weight-input" placeholder="${operation.weight_unit}" value="${weight}" onchange="updateOperationSet('${operationSet.id}', '${setCounter}')">
        </div>
        <div class="operation-set-input operation-set-input-wide" id="operation-set-time-${operationSet.id}" style="display: ${timingHTML};">
            <input style="" class="operation-set-time-input" type="text" id="operation-set-time-input-${operationSet.id}" name="operation-set-time-input" placeholder="00:00" value="${time}" onchange="updateOperationSet('${operationSet.id}', '${setCounter}')">
        </div>
        <div class="operation-set-input" id="operation-set-distance-${operationSet.id}" style="display: ${distanceHTML};">
            <input style="" min="0" class="operation-set-distance-input" type="number" id="operation-set-distance-input-${operationSet.id}" name="operation-set-distance-input" placeholder="${operation.distance_unit}" value="${distance}" onchange="updateOperationSet('${operationSet.id}', '${setCounter}')">
        </div>
    `;
}

function updateExerciseDay(exerciseDayID) {
    var exerciseDayNote = document.getElementById("exercise-day-note").value;

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
                placeNewOperation(result.operation, exerciseID)
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

function placeNewOperation(operation) {
    document.getElementById('operationsWrapper-sub-' + operation.exercise).innerHTML += `
        <div class="operationWrapper" id="operation-${operation.id}">
            ${generateOperationHTML(operation, operation.exercise)}
        </div>
    `;
}

function addOperationSet(operationID) {
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
                placeOperationNewSet(result.operation_set, operationID, result.operation)
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

function placeOperationNewSet(operationSet, operationID, operation) {
    element = document.getElementById('operation-set-wrapper-sub-' + operationID)
    count = element.children.length
    operationSetHTML = `
        <div class="operationSetWrapper" id="operation-set-${operationSet.id}">
            ${generateOperationSetHTML(operationSet, operation, count + 1)}
        </div>
    `;
    element.insertAdjacentHTML("beforeend", operationSetHTML)
}

function updateOperation(operationID) {
    var type = document.getElementById('operation-type-text-' + operationID).value
    var action = document.getElementById('operation-action-text-' + operationID).value
    var weight_unit = document.getElementById('operation-weight-unit-' + operationID).value
    var distance_unit = document.getElementById('operation-distance-unit-' + operationID).value

    var form_obj = {
        "type": type,
        "action": action,
        "weight_unit": weight_unit,
        "distance_unit": distance_unit,
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
                placeOperation(result.operation)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/operations/" + operationID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeOperation(operation) {
    operationHTML = generateOperationHTML(operation)
    document.getElementById('operation-' + operation.id).innerHTML = operationHTML
}

function updateOperationSet(operationSetID, setCount) {
    var repetitions = parseFloat(document.getElementById('operation-set-rep-input-' + operationSetID).value)
    var weight = parseFloat(document.getElementById('operation-set-weight-input-' + operationSetID).value)
    var time = document.getElementById('operation-set-time-input-' + operationSetID).value
    var distance = parseFloat(document.getElementById('operation-set-distance-input-' + operationSetID).value)

    var timeFinal = null;
    try {
        if(time.includes(':')) {
            timeArray = time.split(':')
            
            if(timeArray.length == 2) {
                var minutes = parseFloat(timeArray[0])
                var seconds = parseFloat(timeArray[1])
                timeFinal = (minutes * 60) + seconds
            } else if (timeArray.length == 3){
                var hours = parseFloat(timeArray[0])
                var minutes = parseFloat(timeArray[1])
                var seconds = parseFloat(timeArray[2])
                timeFinal = (hours * 3600) + (minutes * 60) + seconds
            }

        } else if(time != ''){
            timeFinal = parseFloat(time)
        }
    } catch (e) {
        console.log("Failed to parse time. Error: " + e)
    }

    var form_obj = {
        "repetitions": repetitions,
        "weight": weight,
        "time": timeFinal,
        "distance": distance,
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
                placeOperationSet(result.operation_set, result.operation, setCount)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/operation-sets/" + operationSetID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeOperationSet(operationSet, operation, setCount) {
    operationSetHTML = generateOperationSetHTML(operationSet, operation, setCount)
    document.getElementById('operation-set-' + operationSet.id).innerHTML = operationSetHTML
}

function updateExercise(exerciseID) {
    var note = document.getElementById('exercise-note-' + exerciseID).value

    var form_obj = {
        "note": note,
        "on": true,
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
                placeExercise(result.exercise)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/exercises/" + exerciseID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeExercise(exercise) {
    document.getElementById('exercise-note-' + exercise.id).value = exercise.note
}

function loadExerciseList() {
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
                processExerciseList(result)
            }

        }
    };

    xhttp.open("get", "/json/exercises.json");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send();
    return false;
}

function processExerciseList(exercisesArray) {
    exercisesArray.forEach(function(item){
        exerciseListHTML += `
            <option value="${item.title}">
                ${item.title}
            </option>
        `
    });
}