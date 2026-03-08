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

            <div id="myModal" class="modal">
                <span class="close selectable clickable" onclick="closeModal()">&times;</span>

                <div class="modal-content" id="modal-content">
                </div>

                <div id="caption"></div>
            </div>
            
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

                <div class="" style="margin-top: 5em; display: none;" id="stravaCombineButtonWrapper">
                    <button type="submit" class="btn btn-primary" style="width: 15em; background-color: salmon; font-size:0.75em;" id="" onclick="combineStravaExercises(); return false;">
                        Combine Strava exercises
                    </button>
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
    //document.getElementById('exercise-day-exercise-goal').innerHTML = "Exercise goal for week: " + exerciseDay.goal.exercise_interval;
    document.getElementById('exercise-day-note').innerHTML = exerciseDay.note;

    placeExercises(exerciseDay.exercises);
}

var exerciseCache = {};

function isSimpleActivity(exercise) {
    return exercise.operations.length === 1 &&
           exercise.operations[0].operation_sets.length === 1 &&
           exercise.operations[0].type === 'moving';
}

function placeExercises(exercises) {
    exercisesHTML = "";
    counter = 1;
    exerciseCache = {};

    try {
        document.getElementById('stravaCombineButtonWrapper').style.display = "none";
    } catch(e) {
        console.log("Failed to remove Strava button. Error: " + e)
    }

    exercises.forEach(exercise => {
        exercise._count = counter;
        exerciseCache[exercise.id] = exercise;

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

function generateExerciseHTML(exercise, count, forceFullEditor = false) {
    var exerciseHTML = null;

    if(exercise.is_on) {

        if(!forceFullEditor && isSimpleActivity(exercise)) {
            return generateSimpleActivityHTML(exercise, count);
        }

        durationHTML = ""
        if(exercise.duration) {
            durationHTML = secondsToDurationString(exercise.duration)
        }

        timeHTML = ""
        if(exercise.time) {
            // Parse and extract HH:MM in local time
            const date = new Date(exercise.time);
            timeHTML = date.toLocaleTimeString('en-GB', {
                hour: '2-digit',
                minute: '2-digit',
                hour12: false
            });
        }

        stravaHTML = ""
        stravaCombineHTML = ""
        stravaDivide = ""
        if(exercise.strava_id && exercise.strava_id.length > 0) {
            stravaHTML += `<div class="strava-stack">`
            stravaIDString = ""
            for(var i = 0; i < exercise.strava_id.length; i++) {
                stravaHTML += `
                    <p class="strava-text clickable" onclick="window.open('https://www.strava.com/activities/${exercise.strava_id[i]}', '_blank')">
                        Strava session (${exercise.strava_id[i]})
                        <img src="/assets/external-link.svg" class="btn_logo" style="width: 1.25em; height: 1.25em; padding: 0; margin: 0.25em 0.5em;">
                    </p>
                `;

                if(i != 0) {
                    stravaIDString += ";"
                }
                stravaIDString += exercise.strava_id[i]
            }
            stravaHTML += `</div>`
            
            stravaCombineHTML += `
                <input style="margin: 0.5em;" id="${stravaIDString}" class="clickable stravaCombineCheck" type="checkbox" name="" value="">
            `;

            if(exercise.strava_id.length > 1) {
                stravaDivide += `
                    <img src="/assets/scissors.svg" style="height: 1em; width: 1em; padding: 1em;" onclick="divideStravaExercises('${exercise.id}')" class="btn_logo clickable color-invert">
                `;
            }

            try {
                document.getElementById('stravaCombineButtonWrapper').style.display = "flex";
            } catch(e) {
                console.log("Failed to show Strava button. Error: " + e)
            }
        }

        exerciseHTML = `
            <div class="top-row">
                ${stravaDivide}
                ${stravaCombineHTML}
                <img src="/assets/trash-2.svg" style="height: 1em; width: 1em; padding: 1em;" onclick="updateExercise('${exercise.id}', false, ${count}, '${exercise.time}', true)" class="btn_logo clickable color-invert">
            </div>

            <div class="exerciseSubWrapper" id="exercise-sub-${exercise.id}">
                ${stravaHTML}

                <h2 style="">Session ${count}</h2>
                
                <div class="exercise-input" id="exercise-timeofday-${exercise.id}">
                    <label style="margin: 0;" for="exercise-timeofday-input-${exercise.id}"" title="Time of day">Time</label>
                    <input style="" class="exercise-time-input" type="time" id="exercise-timeofday-input-${exercise.id}" name="exercise-timeofday-input" placeholder="hh:mm" value="${timeHTML}" onchange="updateExercise('${exercise.id}', true, ${count}, '${exercise.time}', true)">
                </div>
                <div class="exercise-input" id="exercise-time-${exercise.id}">
                    <label style="margin: 0;" for="exercise-time-input-${exercise.id}"" title="Total session duration">Duration</label>
                    <input style="" class="exercise-time-input" type="text" id="exercise-time-input-${exercise.id}" name="exercise-time-input" pattern="[0-9:]{0,}" placeholder="hh:mm:ss" value="${durationHTML}" onchange="updateExercise('${exercise.id}', true, ${count}, '${exercise.time}', true)">
                </div>

                <textarea class="day-note-area" id="exercise-note-${exercise.id}" name="exercise-exercise-note" rows="3" cols="33" placeholder="Notes" style="margin-top: 1em; width: 20em;" onchange="updateExercise('${exercise.id}', true, ${count}, '${exercise.time}', true)">${exercise.note}</textarea>

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

                <button type="submit" onclick="updateExercise('${exercise.id}', true, ${count}, '${exercise.time});" id="restore-exercise-button-${exercise.id}" style="margin-bottom: 0em; width: 8em;"><img src="/assets/refresh-cw.svg" class="btn_logo color-invert"><p2>Restore</p2></button>

                <hr class="invert" style="border: 0.025em solid var(--white); margin: 4em 0;">
            </div>
        `;
    }

    return exerciseHTML;
}

function generateSimpleActivityHTML(exercise, count) {
    const operation = exercise.operations[0];
    const set = operation.operation_sets[0];
    const action = operation.action;

    var timeHTML = ""
    if (exercise.time) {
        const date = new Date(exercise.time);
        timeHTML = date.toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit', hour12: false });
    }

    var durationHTML = set.time
        ? secondsToDurationString(set.time)
        : (exercise.duration ? secondsToDurationString(exercise.duration) : "—");

    var distanceHTML = set.distance != null
        ? parseFloat(set.distance).toFixed(2) + " " + operation.distance_unit
        : "—";

    var avgHTML = "—"
    const speedTime = set.moving_time || set.time;
    if (set.distance != null && speedTime != null) {
        avgHTML = parseFloat(set.distance / (speedTime / 3600)).toFixed(2) + " " + operation.distance_unit + "/h";
    }

    var actionName = action ? action.name : "Activity"
    var activityTitle = (operation.note && operation.note.trim() !== '') ? operation.note.trim() : actionName
    var actionIcon
    if (action && action.has_logo) {
        actionIcon = `<img src="/assets/actions/${action.name}.svg" class="color-invert" style="height: 1em; width: 1em; vertical-align: middle; margin: 0.5rem;">`
    } else {
        actionIcon = operation.type === 'moving' ? '🏃‍♂️' : operation.type === 'timing' ? '⏱️' : '💪'
    }

    var stravaLinkHTML = ""
    const stravaActivityID = set.strava_id

    if (stravaActivityID) {
        stravaLinkHTML = `<a href="https://www.strava.com/activities/${stravaActivityID}" target="_blank" style="display: inline-flex; align-items: center; margin: 0.5rem; vertical-align: middle; opacity: 0.8;">
            <img src="/assets/strava-logo.svg" style="height: 0.85em; width: auto;">
        </a>`
    } else {
        console.log("no Strava ID")
    }

    const streams = set.strava_streams;
    const hasHeartrate = streams && streams.heartrate && streams.heartrate.data && streams.heartrate.data.length > 0;
    const latlngData = streams && (streams.latlng || streams.lat_lng);
    const hasRoute = latlngData && latlngData.data && latlngData.data.length > 0;

    // Elevation gain from altitude stream
    var elevationHTML = "";
    if (streams && streams.altitude && streams.altitude.data && streams.altitude.data.length > 1) {
        const altData = streams.altitude.data;
        let gain = 0;
        for (let i = 1; i < altData.length; i++) {
            const delta = altData[i] - altData[i - 1];
            if (delta > 0) gain += delta;
        }

        if(gain > 0) {
            elevationValue = Math.round(gain) + " m";

            elevationHTML = `
                <div class="simple-stat">
                    <span class="simple-stat-value">${elevationValue}</span>
                    <span class="simple-stat-label">Elevation</span>
                </div>
            `
        }
    }

    // Compute HR stats
    var hrStatsHTML = "";
    if (hasHeartrate) {
        const hrVals = streams.heartrate.data.filter(v => v > 0);
        if (hrVals.length > 0) {
            const hrAvg = Math.round(hrVals.reduce((a, b) => a + b, 0) / hrVals.length);
            const hrMin = Math.min(...hrVals);
            const hrMax = Math.max(...hrVals);
            hrStatsHTML = `
                <div class="simple-activity-stats" style="margin-top: 1em;">
                    <div class="simple-stat">
                        <span class="simple-stat-value">${hrMin} <span style="font-size:0.6em; opacity:0.7;">bpm</span></span>
                        <span class="simple-stat-label">Min HR</span>
                    </div>
                    <div class="simple-stat">
                        <span class="simple-stat-value">${hrAvg} <span style="font-size:0.6em; opacity:0.7;">bpm</span></span>
                        <span class="simple-stat-label">Avg HR</span>
                    </div>
                    <div class="simple-stat">
                        <span class="simple-stat-value">${hrMax} <span style="font-size:0.6em; opacity:0.7;">bpm</span></span>
                        <span class="simple-stat-label">Max HR</span>
                    </div>
                </div>
            `;
        }
    }

    // Debug: log what we have so we can track down missing strava/map data
    console.log('[simple card] exercise.strava_id:', exercise.strava_id, '| set.strava_activity_id:', set.strava_activity_id, '| hasRoute:', hasRoute, '| latlngKeys:', streams ? Object.keys(streams) : null);

    const hrCanvasID = `hr-chart-${set.id}`;
    const mapDivID = `route-map-${set.id}`;

    var hrHTML = hasHeartrate ? `<div class="simple-activity-hr-wrapper"><canvas id="${hrCanvasID}" class="simple-activity-hr-chart"></canvas>${hrStatsHTML}</div>` : "";
    var mapHTML = hasRoute ? `<div id="${mapDivID}" class="simple-activity-map" style="height: 300px; width: 100%; border-radius: 0.5em; overflow: hidden; position: relative; z-index: 0;"></div>` : "";

    stravaSyncButtonHTML = "";
    if (stravaActivityID) {
        stravaSyncButtonHTML = `
            <img src="/assets/refresh-cw.svg" style="height: 1em; width: 1em; padding: 1em;"
                onclick="stravaSyncOperationSet('${set.id}')"
                class="btn_logo clickable color-invert">
        `;
    }

    var html = `
        <div class="top-row">
            ${stravaSyncButtonHTML}
            <img src="/assets/edit.svg" style="height: 1em; width: 1em; padding: 1em;"
                onclick="switchToFullEditor('${exercise.id}')"
                class="btn_logo clickable color-invert">
            <img src="/assets/trash-2.svg" style="height: 1em; width: 1em; padding: 1em;"
                onclick="updateExercise('${exercise.id}', false, ${count}, '${exercise.time}', true)"
                class="btn_logo clickable color-invert">
        </div>
        <div class="exerciseSubWrapper" id="exercise-sub-${exercise.id}">
            <h2 class="simple-activity-title">${actionIcon} ${activityTitle}${stravaLinkHTML}</h2>
            <p style="opacity: 0.6; margin: 0.25em 0;">${timeHTML}</p>

            <div class="simple-activity-stats-wrapper">
                <div class="simple-activity-stats">
                    <div class="simple-stat">
                        <span class="simple-stat-value">${durationHTML}</span>
                        <span class="simple-stat-label">Duration</span>
                    </div>
                    <div class="simple-stat">
                        <span class="simple-stat-value">${distanceHTML}</span>
                        <span class="simple-stat-label">Distance</span>
                    </div>
                    <div class="simple-stat">
                        <span class="simple-stat-value">${avgHTML}</span>
                        <span class="simple-stat-label">Avg speed</span>
                    </div>
                    ${elevationHTML}
                </div>

                ${mapHTML}
            </div>

            ${hrHTML}

            <hr class="invert" style="border: 0.025em solid var(--white); margin: 4em 0;">
        </div>
    `;

    // Defer chart and map rendering - poll until divs are actually in the DOM
    if (hasHeartrate || hasRoute) {
        var renderAttempts = 0;
        var renderInterval = setInterval(function() {
            renderAttempts++;
            var hrReady = !hasHeartrate || document.getElementById(hrCanvasID);
            var mapReady = !hasRoute || document.getElementById(mapDivID);
            if (hrReady && mapReady) {
                clearInterval(renderInterval);
                if (hasHeartrate) {
                    // The streams are distance-sampled, not time-sampled, so
                    // streams.time.data is elapsed seconds and doesn't match moving_time.
                    // Fix: keep only samples where the runner is moving (velocity > 0.5 m/s),
                    // then space them evenly across set.moving_time seconds.
                    const hrRaw = streams.heartrate.data;
                    const velocity = streams.velocity_smooth && streams.velocity_smooth.data;
                    const movingTimeSecs = set.moving_time || set.time;
                    let chartHrData, chartTimeData;
                    if (velocity && velocity.length === hrRaw.length && movingTimeSecs) {
                        chartHrData = [];
                        const movingIndices = [];
                        for (let i = 0; i < hrRaw.length; i++) {
                            if (velocity[i] > 0.5) {
                                movingIndices.push(i);
                            }
                        }
                        const n = movingIndices.length;
                        chartTimeData = movingIndices.map((idx, j) => Math.round(j / (n - 1) * movingTimeSecs));
                        chartHrData = movingIndices.map(idx => hrRaw[idx]);
                    } else {
                        chartTimeData = streams.time.data;
                        chartHrData = hrRaw;
                    }
                    renderHeartrateChart(hrCanvasID, chartTimeData, chartHrData, movingTimeSecs);
                }
                if (hasRoute) {
                    console.log("[map] div found, rendering route with", latlngData.data.length, "points");
                    renderRouteMap(mapDivID, latlngData.data);
                }
            } else if (renderAttempts > 50) {
                clearInterval(renderInterval);
                console.warn("[simple card] DOM elements not found after 50 attempts. hr:", hrCanvasID, "map:", mapDivID);
            }
        }, 50);
    }

    return html;
}

function renderHeartrateChart(canvasID, timeData, hrData, maxSecs) {
    var canvas = document.getElementById(canvasID);
    if (!canvas) return;

    // Build {x, y} points — both arrays are already aligned (same length)
    var points = [];
    for (var i = 0; i < hrData.length; i++) {
        if (hrData[i] > 0) {
            points.push({
                x: timeData[i],
                y: hrData[i]
            });
        }
    }

    if (points.length === 0) return;

    var tickColor = "rgba(255,255,255,0.6)";

    new Chart(canvas, {
        type: "line",
        data: {
            datasets: [{
                label: "Heart Rate (bpm)",
                data: points,
                fill: true,
                borderColor: "rgba(220, 80, 80, 1)",
                pointBackgroundColor: "rgba(220, 80, 80, 1)",
                backgroundColor: "rgba(220, 80, 80, 0.2)",
                tension: 0.3,
                pointRadius: 0,
                borderWidth: 2
            }]
        },
        options: {
            legend: { display: false },
            scales: {
                xAxes: [{
                    type: "linear",
                    gridLines: { color: "rgba(255,255,255,0.1)" },
                    ticks: {
                        fontColor: tickColor,
                        autoSkip: true,
                        maxTicksLimit: 6,
                        min: 0,
                        max: maxSecs || undefined,
                        callback: function(value) {
                            return secondsToDurationString(value);
                        }
                    }
                }],
                yAxes: [{
                    gridLines: { color: "rgba(255,255,255,0.1)" },
                    ticks: {
                        fontColor: tickColor,
                        beginAtZero: false,
                        precision: 0,
                        callback: function(value) { return value + " bpm"; }
                    }
                }]
            },
            tooltips: {
                callbacks: {
                    title: function(items) {
                        return secondsToDurationString(items[0].xLabel);
                    },
                    label: function(item) {
                        return item.yLabel + " bpm";
                    }
                }
            }
        }
    });
}

function renderRouteMap(divID, latlngData) {
    var mapDiv = document.getElementById(divID);
    if (!mapDiv) { console.warn('[map] div not found:', divID); return; }
    if (!latlngData || latlngData.length === 0) { console.warn('[map] no latlng data'); return; }

    // Leaflet must be available globally
    if (typeof L === 'undefined') {
        console.warn('[map] Leaflet not loaded');
        mapDiv.style.display = 'none';
        return;
    }
    console.log('[map] rendering', latlngData.length, 'points, first point:', latlngData[0]);

    var map = L.map(divID, { zoomControl: true, scrollWheelZoom: false });

    L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
        attribution: '© OpenStreetMap contributors',
        maxZoom: 18
    }).addTo(map);

    var latLngs = latlngData.map(function(point) {
        return [point[0], point[1]];
    });

    var polyline = L.polyline(latLngs, {
        color: 'rgba(220, 80, 80, 1)',
        weight: 3,
        opacity: 0.85
    }).addTo(map);

    // Start marker
    L.circleMarker(latLngs[0], {
        radius: 6, color: '#4caf50', fillColor: '#4caf50', fillOpacity: 1
    }).addTo(map);

    // End marker
    L.circleMarker(latLngs[latLngs.length - 1], {
        radius: 6, color: '#f44336', fillColor: '#f44336', fillOpacity: 1
    }).addTo(map);

    map.fitBounds(polyline.getBounds(), { padding: [16, 16] });
}

function switchToFullEditor(exerciseID) {
    const exercise = exerciseCache[exerciseID];
    if (!exercise) return;
    document.getElementById('exercise-' + exerciseID).innerHTML = generateExerciseHTML(exercise, exercise._count, true);
    // Append a "back to summary" button after the full editor renders
    const subWrapper = document.getElementById('exercise-sub-' + exerciseID);
    if (subWrapper && isSimpleActivity(exercise)) {
        const backBtn = document.createElement('button');
        backBtn.textContent = '← Back to summary';
        backBtn.className = 'back-to-summary-btn';
        backBtn.style.cssText = 'margin-top: 1em; font-size: 0.75em; opacity: 0.6;';
        backBtn.onclick = function() {
            document.getElementById('exercise-' + exerciseID).innerHTML = generateSimpleActivityHTML(exercise, exercise._count);
        };
        subWrapper.insertBefore(backBtn, subWrapper.firstChild);
    }
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
            <img src="/assets/plus.svg" class="button-icon" style="height: 100%; margin: 1em;">
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

    actionHTML = ""
    if(operation.action) {
        actionHTML = operation.action.name
    }

    noneHTML = ''
    barbellsHTML = ''
    dumbbellsHTML = ''
    bandsHTML = ''
    ropeHTML = ''
    benchHTML = ''
    treadmillHTML = ''
    machineHTML = ''
    if(operation.equipment == 'barbells') {
        barbellsHTML = 'selected'
    } else if(operation.equipment == 'dumbbells') {
        dumbbellsHTML = 'selected'
    } else if(operation.equipment == 'bands') {
        bandsHTML = 'selected'
    } else if(operation.equipment == 'rope') {
        ropeHTML = 'selected'
    } else if(operation.equipment == 'bench') {
        benchHTML = 'selected'
    } else if(operation.equipment == 'treadmill') {
        treadmillHTML = 'selected'
    } else if(operation.equipment == 'machine') {
        machineHTML = 'selected'
    } else {
        noneHTML = 'selected'
    }

    var operationHTML = `
        <div class="operation-selectors">
            <div class="operationType" id="operation-type-${operation.id}">
                <select class="operation-type-input" type="text" id="operation-type-text-${operation.id}" name="operation-type-text" style="text-align: center; font-size: 0.9em !important; min-height: 2em; min-width: 3em; height: 100% !important;" onchange="updateOperation('${operation.id}')">
                    <option value="lifting" ${liftingHTML}>💪</option>
                    <option value="moving" ${movingHTML}>🏃‍♂️</option>
                    <option value="timing" ${timingHTML}>⏱️</option>
                </select>  
            </div>
            <div class="operationAction" id="operation-action-${operation.id}">
                <input style="" class="operation-action-input" type="text" list="operation-action-text-list-${operation.id}" id="operation-action-text-${operation.id}" name="operation-action-text" placeholder="exercise" value="${actionHTML}" onkeyup="filterFunction('${operation.id}')" onfocus="showSelectDropdown('${operation.id}', true)">
                <div id="operation-action-text-list-${operation.id}" class="dropdown-actions-wrapper" style="display: none;">
                    ${processExerciseList(operation.id)}
                </div>
            </div>
            <div class="operationEquipment" id="operation-equipment-${operation.id}">
                <select class="operation-equipment-input" type="text" id="operation-equipment-text-${operation.id}" name="operation-equipment-text" style="text-align: center; font-size: 0.9em !important; min-height: 2em; min-width: 3em;" onchange="updateOperation('${operation.id}')">
                    <option value="" ${noneHTML}></option>
                    <option value="barbells" ${barbellsHTML}>Barbells</option>
                    <option value="dumbbells" ${dumbbellsHTML}>Dumbbells</option>
                    <option value="bands" ${bandsHTML}>Bands</option>
                    <option value="rope" ${ropeHTML}>Rope</option>
                    <option value="bench" ${benchHTML}>Bench</option>
                    <option value="treadmill" ${treadmillHTML}>Treadmill</option>
                    <option value="machine" ${machineHTML}>Machine</option>
                </select>  
            </div>
            
            <div class="addActionWrapper clickable hover" id="addActionWrapper-${operation.id}" title="Add new exercise" onclick="addAction('${operation.id}');" style="">
                <img src="/assets/plus.svg" class="button-icon" style="width: 100%; margin: 0.25em;">
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
            <div class="operation-set-title" style="display: ${weightHTML};">
                ${operation.weight_unit}
            </div>
            <div class="operation-set-title" style="display: ${repsHTML};">
                reps
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
    if(operationSet.moving_time) {
        time = secondsToDurationString(operationSet.moving_time)
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
    updateBackButtonVisibility(operationID);
}

function updateOperation(operationID) {
    console.log('Updating operation...')
    toggleActionBorder(operationID, 'none');

    var type = document.getElementById('operation-type-text-' + operationID).value
    var action = document.getElementById('operation-action-text-' + operationID).value
    var weight_unit = document.getElementById('operation-weight-unit-' + operationID).value
    var distance_unit = document.getElementById('operation-distance-unit-' + operationID).value
    var equipment = document.getElementById('operation-equipment-text-' + operationID).value

    var form_obj = {
        "type": type,
        "action": action,
        "weight_unit": weight_unit,
        "distance_unit": distance_unit,
        "equipment": equipment,
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
                console.log(result.error);
                toggleActionBorder(operationID, 'salmon');
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
        "moving_time": timeFinal,
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

function updateExercise(exerciseID, on, count, originalTimeString, fromEditor = false) {
    if(!on && !confirm("Are you sure you want to delete this session?")) {
        return;
    }

    var note = document.getElementById('exercise-note-' + exerciseID).value
    var time = document.getElementById('exercise-time-input-' + exerciseID).value
    var newIso = ""

    try {
        var timeOfDay = document.getElementById('exercise-timeofday-input-' + exerciseID).value

        const localDate = new Date(originalTimeString);
        const [hours, minutes] = timeOfDay.split(':').map(Number);
        localDate.setHours(hours, minutes, 0, 0);
        newIso = toLocalISOString(localDate);
    } catch(e) {
        console.log("failed to parse original time ISO. error: " + e)
        return
    }

    var form_obj = {
        "note": note,
        "is_on": on,
        "duration": parseDurationStringToSeconds(time),
        "time": newIso
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
                if (fromEditor) {
                    placeExercise(result.exercise, count, true);
                } else {
                    placeExercise(result.exercise, count);
                }
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

function placeExercise(exercise, count, forceFullEditor = false) {
    document.getElementById('exercise-' + exercise.id).innerHTML = generateExerciseHTML(exercise, count, forceFullEditor)
    if (forceFullEditor && isSimpleActivity(exercise)) {
        // Re-inject the "back to summary" button just like switchToFullEditor does
        const subWrapper = document.getElementById('exercise-sub-' + exercise.id);
        if (subWrapper) {
            const backBtn = document.createElement('button');
            backBtn.textContent = '← Back to summary';
            backBtn.className = 'back-to-summary-btn';
            backBtn.style.cssText = 'margin-top: 1em; font-size: 0.75em; opacity: 0.6;';
            backBtn.onclick = function() {
                document.getElementById('exercise-' + exercise.id).innerHTML = generateSimpleActivityHTML(exercise, exercise._count);
            };
            subWrapper.insertBefore(backBtn, subWrapper.firstChild);
        }
    }
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
                exerciseList = result.actions
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/actions");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function processExerciseList(operationID) {
    exerciseListHTML = "";
    actions = exerciseList;

    actions.forEach(function(action){
        if(action.name && action.name != "") {
            exerciseListHTML += `
                <a value="" onclick="selectActionForOperation('${operationID}', '${action.name}')" class="dropdown-action clickable">
                    ${action.name}
                </a>
            `
        }
        if(action.norwegian_name && action.norwegian_name != "" && action.norwegian_name != action.name) {
            exerciseListHTML += `
                <a value="" onclick="selectActionForOperation('${operationID}', '${action.norwegian_name}')" class="dropdown-action clickable">
                    ${action.norwegian_name}
                </a>
            `
        }
    });

    return exerciseListHTML
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
    // operation.id is the operation ID; find its set wrapper to recount
    updateBackButtonVisibility(operation.id);
}

// Show/hide the "← Back to summary" button based on whether the exercise
// still qualifies as a simple activity (1 operation, 1 set, type=moving).
function updateBackButtonVisibility(operationID) {
    const setWrapper = document.getElementById('operation-set-wrapper-sub-' + operationID);
    if (!setWrapper) return;

    // Walk up to find the exercise wrapper: operation-set-wrapper-sub-X is inside
    // operationWrapper > operationsWrapperSub > operationsWrapper > exerciseSubWrapper
    const exerciseSubWrapper = setWrapper.closest('.exerciseSubWrapper');
    if (!exerciseSubWrapper) return;

    const exerciseID = exerciseSubWrapper.id.replace('exercise-sub-', '');
    const exercise = exerciseCache[exerciseID];
    if (!exercise) return;

    const backBtn = exerciseSubWrapper.querySelector('.back-to-summary-btn');
    if (!backBtn) return;

    const setCount = setWrapper.children.length;
    const operationCount = document.getElementById('operationsWrapper-sub-' + exerciseID)
        ? document.getElementById('operationsWrapper-sub-' + exerciseID).children.length
        : 1;

    // Simple = 1 operation, 1 set, type moving (mirrors isSimpleActivity logic)
    const typeEl = document.getElementById('operation-type-text-' + operationID);
    const currentType = typeEl ? typeEl.value : 'moving';
    const isSimple = (operationCount === 1 && setCount === 1 && currentType === 'moving');

    backBtn.style.display = isSimple ? '' : 'none';
}

function selectActionForOperation(operationID, actionName) {
    console.log('Selected: ' + actionName)
    document.getElementById('operation-action-text-' + operationID).value = actionName 
    showSelectDropdown(operationID, false)
    updateOperation(operationID)
}

function toggleActionBorder(operationID, color) {
    element = document.getElementById('operation-action-text-' + operationID)
    element.setAttribute('style', `border:0.2em solid ${color} !important`);
}

function addAction(operationID) {
    closeAllLists();
    myModal = document.getElementById("myModal")
    myModal.style.display = "flex";
    modalContent = document.getElementById("modal-content") 

    modalHTML = `
        <div class="addNewActionWrapper" id="add-action-wrapper-${operationID}">
            <h3 style="">Add new exercise</h3>
            
            <div class="add-new-exercise-names">
                <div class="add-new-exercise-name">
                    <b>English name</b>
                    <div class="exercise-input" id="action-${operationID}">
                        <input style="" class="new-action-name-english-input" type="text" id="new-action-name-english-input-${operationID}" name="new-action-name-english-input" placeholder="Running" value="">
                    </div>
                </div>
                <div class="add-new-exercise-name" style="min-width: 8em;">
                    OR/AND
                </div>
                <div class="add-new-exercise-name">
                    <hb>Norwegian name</b>
                    <div class="exercise-input" id="action-${operationID}">
                        <input style="" class="new-action-name-norwegian-input" type="text" id="new-action-name-norwegian-input-${operationID}" name="new-action-name-norwegian-input" placeholder="Løping" value="">
                    </div>
                </div>
            </div>

            <b style="margin-top: 1em;">Sets, distance or time-based?</b>
            <div class="operationType" id="new-action-type-${operationID}">
                <select class="new-action-type-input" type="text" id="new-action-type-input-${operationID}" name="new-action-type-input" style="text-align: center; font-size: 0.9em !important; min-height: 2em; min-width: 3em; height: 100% !important;">
                    <option value="lifting">💪</option>
                    <option value="moving">🏃‍♂️</option>
                    <option value="timing">⏱️</option>
                </select>  
            </div>

            <hr class="invert" style="border: 0.025em solid var(--grey); margin: 1em 0;">
            <h3 style="margin-bottom:1em; ">Optional</h3>
            
            <b>Description</b>
            <textarea class="new-action-description-input" id="new-action-description-input-${operationID}" name="new-action-description-input" rows="3" cols="33" placeholder="Fast paced moving which can be..." style="width: 20em;" ></textarea>

            <div class="add-new-exercise-name">
                <b>Body part/category</b>
                <div class="new-action-bodypart" id="new-action-bodypart-${operationID}">
                    <input style="" class="new-action-bodypart-input" type="text" id="new-action-bodypart-input-${operationID}" name="new-action-bodypart-input" placeholder="Cardio" value="">
                </div>
            </div>

            <button type="submit" onclick="createAction('${operationID}');" id="goal_amount_button" style="margin-bottom: 0em;"><img src="/assets/done.svg" class="btn_logo color-invert"><p2>Add and use</p2></button>

        </div>
    `;

    modalContent.innerHTML = modalHTML
}

function closeModal() {
    document.getElementById("myModal").style.display = "none";
}

function createAction(operationID) {
    var name = document.getElementById('new-action-name-english-input-' + operationID).value
    var norwegian_name = document.getElementById('new-action-name-norwegian-input-' + operationID).value
    var description = document.getElementById('new-action-description-input-' + operationID).value
    var type = document.getElementById('new-action-type-input-' + operationID).value
    var body_part = document.getElementById('new-action-bodypart-input-' + operationID).value

    var form_obj = {
        "name": name,
        "norwegian_name" : norwegian_name,
        "description": description,
        "type": type,
        "body_part": body_part
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                alert("Could not reach API.");
                return;
            }
            
            if(result.error) {
                alert(result.error);
            } else {
                selectActionForOperation(operationID, result.action.name)
                closeModal();
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/actions");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

window.addEventListener('click', function(e){   
    if (
        e.target.classList.contains('dropdown-action') ||
        e.target.classList.contains('dropdown-actions-wrapper') ||
        e.target.classList.contains('operation-action-input') ||
        e.target.classList.contains('operationAction')
    ){
        console.log('Inside div.')
    } else{
        closeAllLists();
    }
});

function closeAllLists() {
    lists = document.getElementsByClassName('dropdown-actions-wrapper');
    for(var i = 0; i < lists.length; i++) {
        if(lists[i].style.display == 'flex') {
            operationID = lists[i].id.replace('operation-action-text-list-', '')
            lists[i].style.display = 'none';
            updateOperation(operationID)
        }
    }
}

function combineStravaExercises() {
    checkButtons = document.getElementsByClassName('stravaCombineCheck')

    var stravaIDArray = []

    for(var i = 0; i < checkButtons.length; i++) {
        if(checkButtons[i].checked) {
            stravaIDArray.push(checkButtons[i].id)
        }
    }

    if(stravaIDArray.length < 2) {
        alert("Choose two or more exercises to combine.");
        return;
    }

    if(!confirm("Are you sure you want to combine these Strava exercises?")) {
        return;
    }

    combineStravaExercisesAPI(stravaIDArray);
}

function combineStravaExercisesAPI(stravaIDArray) {
    var form_data = JSON.stringify(stravaIDArray);

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
                alert(result.message)
                location.reload(); 
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises/strava-combine");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}

function divideStravaExercises(exerciseID) {
    if(!confirm("Are you sure you want to divide all these Strava exercises?")) {
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
                alert(result.message)
                location.reload(); 
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises/" + exerciseID + "/strava-divide");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function stravaSyncOperationSet(operationSetID) {
    if(!confirm("Are you sure you want to update this with data from Strava?")) {
        return
    }

    var form_obj = {};
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
                location.reload(); 
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/operation-sets/" + operationSetID + "/strava-sync");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}