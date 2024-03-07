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

                        <textarea onchange="updateExerciseDay('${exerciseDayID}')" class="day-note-area" id="exercise-day-note" name="exercise-day-exercise-note" rows="3" cols="33" placeholder="Notes" style="margin-top: 1em;"></textarea>
                    </div>
                </div>

                <hr class="invert" style="border: 0.025em solid var(--white); margin: 4em 0;">

                <div class="exercisesWrapper" id="exercisesWrapper"></div>

                <div class="addExerciseWrapper clickable hover" id="addExerciseWrapper" title="Add session" onclick="addExercise('${exerciseDayID}');">
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
        var exerciseGenerated = generateExerciseHTML(exercise, counter)
        var exerciseHTML = `
            <div class="exerciseWrapper" id="exercise-${exercise.id}">
                ${exerciseGenerated}
            </div>
        `;

        if(exerciseGenerated != null) {
            exercisesHTML += exerciseHTML;
            counter += 1;
        }
    });

    document.getElementById('exercisesWrapper').innerHTML = exercisesHTML;
}

function generateExerciseHTML(exercise, count) {
    var exerciseHTML = null;

    if(exercise.on) {
        exerciseHTML = `
            <div class="top-row">
                <img src="/assets/trash-2.svg" style="height: 1em; width: 1em; padding: 1em;" onclick="updateExercise('${exercise.id}', false, ${count})" class="btn_logo clickable">
            </div>

            <div class="exerciseSubWrapper" id="exercise-sub-${exercise.id}">
                <h2 style="">Session ${count}</h2>
                
                <div class="exercise-input" id="exercise-time-${exercise.id}">
                    <input style="" class="exercise-time-input" type="text" id="exercise-time-input-${exercise.id}" name="exercise-time-input" pattern="[0-9:]{0,}" placeholder="hh:mm:ss" value="${secondsToDurationString(exercise.duration)}" onchange="updateExercise('${exercise.id}', true, ${count})">
                </div>

                <textarea class="day-note-area" id="exercise-note-${exercise.id}" name="exercise-exercise-note" rows="3" cols="33" placeholder="Notes" style="margin-top: 1em; width: 20em;" onchange="updateExercise('${exercise.id}', true, ${count})">${exercise.note}</textarea>

                <div class="operationsWrapper" id="operationsWrapper-${exercise.id}">
                    ${generateOperationsHTML(exercise.operations, exercise.id)}
                </div>

                <hr class="invert" style="border: 0.025em solid var(--white); margin: 4em 0;">
            </div>
        `;
    } else if (exercise.operations.length > 0){
        exerciseHTML = `
            <div class="exerciseSubWrapper" id="exercise-sub-${exercise.id}">
                <h2 style="">Deleted session</h2>
                
                <p>
                    Contains ${exercise.operations.length} exercise(s).
                </p>

                <input style="" class="exercise-time-input" type="hidden" id="exercise-time-input-${exercise.id}" name="exercise-time-input" pattern="[0-9:]{0,}" placeholder="hh:mm:ss" value="${secondsToDurationString(exercise.duration)}">
                <textarea class="day-note-area" id="exercise-note-${exercise.id}" name="exercise-exercise-note" rows="3" cols="33" placeholder="Notes" style="margin-top: 1em; width: 20em; display: none;">${exercise.note}</textarea>

                <button type="submit" onclick="updateExercise('${exercise.id}', true, ${count});" id="restore-exercise-button-${exercise.id}" style="margin-bottom: 0em; width: 8em;"><img src="/assets/refresh-cw.svg" class="btn_logo color-invert"><p2>Restore</p2></button>

                <hr class="invert" style="border: 0.025em solid var(--white); margin: 4em 0;">
            </div>
        `;
    }

    return exerciseHTML;
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

        <div class="bottom-row">
            <img src="/assets/trash-2.svg" style="height: 1em; width: 1em; padding: 0.5em 1em 1em 1em;" onclick="deleteOperation('${operation.id}')" class="btn_logo clickable">
        </div>
    `;

    return operationHTML;
}

function generateOperationSetsHTML(operationSets, operation) {
    operationSetsHTML = ""; 

    repsHTML = 'block'
    distanceHTML = 'block'
    timingHTML = 'block'
    weightHTML = 'block'
    averageHTML = 'block'
    if(operation.type == 'lifting') {
        distanceHTML = 'none'
        timingHTML = 'none'
        averageHTML = 'none'
    } else if(operation.type == 'timing') {
        repsHTML = 'none'
        distanceHTML = 'none'
        weightHTML = 'none'
        averageHTML = 'none'
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
            <div class="operation-set-title" style="display: ${averageHTML};">
                ${operation.distance_unit}/t
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
    averageHTML = 'block'
    if(operation.type == 'lifting') {
        distanceHTML = 'none'
        timingHTML = 'none'
        averageHTML = 'none'
    } else if(operation.type == 'timing') {
        repsHTML = 'none'
        distanceHTML = 'none'
        weightHTML = 'none'
        averageHTML = 'none'
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
        time = secondsToDurationString(operationSet.time)
    }
    var distance = ""
    if(operationSet.distance != null) {
        distance = operationSet.distance
    }
    var average = ""
    if(operationSet.distance != null && operationSet.time != null) {
        average = parseFloat(operationSet.distance / (operationSet.time / 3600)).toFixed(2);
    }

    return `
        <div class="operation-set clickable" id="operation-set-counter-${operationSet.id}"  onclick="deleteOperationSet('${operationSet.id}')">
            Set ${setCounter}
        </div>
        <div class="operation-set-input" id="operation-set-weight-${operationSet.id}" style="display: ${weightHTML};">
            <input style="" min="0" class="operation-set-weight-input" type="number" id="operation-set-weight-input-${operationSet.id}" name="operation-set-weight-input" placeholder="${operation.weight_unit}" value="${weight}" onchange="updateOperationSet('${operationSet.id}', '${setCounter}')">
        </div>
        <div class="operation-set-input" id="operation-set-rep-${operationSet.id}" style="display: ${repsHTML};">
            <input style="" min="0" class="operation-set-rep-input" type="number" id="operation-set-rep-input-${operationSet.id}" name="operation-set-rep-input" placeholder="reps" value="${reps}" onchange="updateOperationSet('${operationSet.id}', '${setCounter}')">
        </div>
        <div class="operation-set-input operation-set-input-wide" id="operation-set-time-${operationSet.id}" style="display: ${timingHTML};">
            <input style="" class="operation-set-time-input" type="text" id="operation-set-time-input-${operationSet.id}" name="operation-set-time-input" pattern="[0-9:]{0,}" placeholder="hh:mm:ss" value="${time}" onchange="updateOperationSet('${operationSet.id}', '${setCounter}')">
        </div>
        <div class="operation-set-input" id="operation-set-distance-${operationSet.id}" style="display: ${distanceHTML};">
            <input style="" min="0" class="operation-set-distance-input" type="number" id="operation-set-distance-input-${operationSet.id}" name="operation-set-distance-input" placeholder="${operation.distance_unit}" value="${distance}" onchange="updateOperationSet('${operationSet.id}', '${setCounter}')">
        </div>
        <div class="operation-set" id="operation-set-average-${operationSet.id}" style="display: ${averageHTML};">
            ${average}
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
            }

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
                placeNewOperationSet(result.operation_set, operationID, result.operation)
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

function placeNewOperationSet(operationSet, operationID, operation) {
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

    var timeFinal = parseDurationStringToSeconds(time);
    
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

function updateExercise(exerciseID, on, count) {
    if(!on && !confirm("Are you sure you want to delete this set?")) {
        return;
    }

    var note = document.getElementById('exercise-note-' + exerciseID).value
    var time = document.getElementById('exercise-time-input-' + exerciseID).value

    var form_obj = {
        "note": note,
        "on": on,
        "duration": parseDurationStringToSeconds(time)
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
                placeExercise(result.exercise, count)
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

function placeExercise(exercise, count) {
    document.getElementById('exercise-' + exercise.id).innerHTML = generateExerciseHTML(exercise, count)
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

function addExercise(exerciseDayID) {
    var form_obj = {
        "exercise_day_id": exerciseDayID,
        "on" : true,
        "note": "",
        "duration": null
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
                placeNewExercise(result.exercise)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function placeNewExercise(exercise) {
    element = document.getElementById('exercisesWrapper')
    count = element.children.length
    var exerciseHTML = `
        <div class="exerciseWrapper" id="exercise-${exercise.id}">
            ${generateExerciseHTML(exercise, count + 1)}
        </div>
    `;
    element.insertAdjacentHTML("beforeend", exerciseHTML)
}

function deleteExercise() {

}

function deleteOperation(operationID) {
    if(!confirm("Are you sure you want to delete this exercise?")) {
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
                document.getElementById('operation-' + operationID).remove();
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/operations/" + operationID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function deleteOperationSet(operationSetID) {
    if(!confirm("Are you sure you want to delete this set?")) {
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
                removeOperationSet(result.operation)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/operation-sets/" + operationSetID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function removeOperationSet(operation) {
    document.getElementById('operation-' + operation.id).innerHTML = generateOperationHTML(operation)
}