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
                    
                    <div class="module">
                    
                        <div class="text-body u-text-center">
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

                        <div id="chart-canvas-div" class="panel-card">
                            <canvas id="myChart" class="panel-wide" style="display:none;"></canvas>
                        </div>

                        <div id="chart-canvas-div-two" class="panel-card">
                            <canvas id="myChartTwo" class="panel-wide" style="display:none;"></canvas>
                        </div>

                    </div>

                </div>

                <div class="module color-invert">
                    <hr>
                </div>

                <div class="module" id="stats_module">

                    <div class="title">
                        Activity statistics
                    </div>

                    <div class="form-group">
                        <select id='select_activity' class='form-control' onchange="chooseActivity()">
                            <option value="null">Choose activity</option>
                        </select>

                        <input class="" type="date" id="activityStartTime" name="activityStartTime" value="" onchange="chooseActivity()" required>
                        <input class="" type="date" id="activityEndTime" name="activityEndTime" value="" onchange="chooseActivity()" required>
                    </div>

                    <div>

                        <div class="module" id="loading-dumbbell-activities" style="display: none;">
                            <img src="/assets/images/barbell.gif">
                        </div>

                        <div id="activity-statistics-element-wrapper-div" class="season-statistics-element-wrapper-div">
                        </div>

                        <div id="chart-canvas-div-activity" class="panel-card">
                            <canvas id="myActivityChart" class="panel-wide" style="display:none;"></canvas>
                        </div>

                        <div id="activity-heatmap-wrapper" style="display: none;">
                            <div class="text-body u-text-center u-mt-1">
                                Heatmap of this activity's GPS tracks.
                            </div>
                            <div id="activity-heatmap-canvas" class="heatmap-canvas" style="display: none;"></div>
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

                    <button type="button" class="btn btn--icon" title="Weight data" aria-label="Weight data" onclick="getWeights(true);">
                        <img src="/assets/database.svg">
                    </button>

                    <div id="chart-canvas-div" class="panel-card">
                        <canvas id="myChartWeights" class="panel-wide" style="display:none;"></canvas>
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
        getActivities();
        placeDefaultDates();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

var activityHeatmapInstance = null;
var activityHeatLayer = null;

// resetActivityHeatmap hides the heatmap and drops the current heat layer. Called when
// the activity/date filter changes so stale points are not shown while reloading.
function resetActivityHeatmap() {
    var wrapper = document.getElementById("activity-heatmap-wrapper");
    if (wrapper) {
        wrapper.style.display = "none";
    }
    if (activityHeatmapInstance && activityHeatLayer) {
        activityHeatmapInstance.removeLayer(activityHeatLayer);
        activityHeatLayer = null;
    }
}

// HEATMAP_GRID is the cell size (degrees, ~67 m) used to count how many distinct
// activities pass through an area. Coarse enough that two runs of the same route fall
// in the same cell, fine enough to keep routes distinct.
var HEATMAP_GRID = 0.0006;
// HEATMAP_BUCKETS quantizes visit frequency into colour bands, which also lets adjacent
// same-frequency segments merge into one polyline (fewer layers to render).
var HEATMAP_BUCKETS = 8;

// renderActivityHeatmap draws the GPS streams from the chosen activity's statistics
// response as route polylines tinted by visit frequency, so it inherits the activity-type
// and date filters for free. Drawing lines (not a point-density blob) keeps routes
// continuous, and colouring by how many distinct activities pass through each cell means a
// single run stays cool while genuinely frequented routes warm up — overlapping translucent
// lines reinforce that. Only activities with GPS movement carry latlng streams; everything
// else shows the "no GPS data" note.
function renderActivityHeatmap(operations) {

    var wrapper = document.getElementById("activity-heatmap-wrapper");
    var canvas = document.getElementById("activity-heatmap-canvas");
    if (!wrapper) {
        return;
    }

    var model = buildHeatmapModel(operations);

    // No GPS data for this activity/period (or the map library is unavailable): hide the
    // whole heatmap section, including its caption, so nothing is shown for non-GPS activities.
    if (!model.tracks.length || typeof L === "undefined") {
        if (typeof L === "undefined" && model.tracks.length) {
            console.log("Leaflet is not loaded.");
        }
        wrapper.style.display = "none";
        return;
    }

    wrapper.style.display = "block";
    canvas.style.display = "block";

    if (!activityHeatmapInstance) {
        activityHeatmapInstance = L.map("activity-heatmap-canvas");
        L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
            maxZoom: 19,
            attribution: "&copy; OpenStreetMap contributors"
        }).addTo(activityHeatmapInstance);
    }

    if (activityHeatLayer) {
        activityHeatmapInstance.removeLayer(activityHeatLayer);
    }

    // A shared canvas renderer keeps thousands of segments performant, and translucent
    // strokes composite where routes overlap, adding to the frequency tint. Each segment
    // gets a wide, soft glow underlay plus a crisp line on top; collecting them into two
    // groups (glow first, lines second) keeps every glow beneath every line so the halos
    // never paint over a neighbouring route's crisp stroke.
    var renderer = L.canvas({ padding: 0.5 });
    var glowLayers = [];
    var lineLayers = [];
    for (var t = 0; t < model.tracks.length; t++) {
        drawTrackSegments(model.tracks[t], model.counts, model.maxCount, renderer, glowLayers, lineLayers);
    }
    activityHeatLayer = L.layerGroup(glowLayers.concat(lineLayers)).addTo(activityHeatmapInstance);

    // Open on the densest cluster (usually the most-frequented area) rather than fitting
    // all points, which would zoom out to fit far-away one-off activities.
    var center = densestCenter(model.allPoints);
    activityHeatmapInstance.setView(center, 13);

    // The container starts hidden, so Leaflet may have measured a zero size; recompute
    // once it is visible.
    setTimeout(function() {
        activityHeatmapInstance.invalidateSize();
        activityHeatmapInstance.setView(center, 13);
    }, 200);

}

// heatmapCellKey rounds a coordinate to its grid cell so points from different
// activities that pass through the same place share a key.
function heatmapCellKey(lat, lng) {
    return Math.round(lat / HEATMAP_GRID) + ":" + Math.round(lng / HEATMAP_GRID);
}

// buildHeatmapModel turns the operations' latlng streams into per-activity tracks plus a
// grid of visit frequencies. Tracks are thinned by an adaptive stride (chosen so the total
// sample count stays under a cap) to keep rendering light. Frequency counts DISTINCT
// activities per cell — one activity passing through a cell many times still counts once —
// so the tint reflects how often a place is revisited, not how densely the GPS sampled a
// single track. Returns { tracks: [[lat,lng]...], counts, maxCount, allPoints }.
function buildHeatmapModel(operations) {

    var model = { tracks: [], counts: {}, maxCount: 0, allPoints: [] };
    if (!operations) {
        return model;
    }

    // Adaptive stride: choose it from the total sample count so the whole map stays under
    // a fixed cap regardless of how many/long the tracks are.
    var total = 0;
    for (var a = 0; a < operations.length; a++) {
        var aSets = operations[a].operation_sets || [];
        for (var b = 0; b < aSets.length; b++) {
            var aStreams = aSets[b].strava_streams;
            if (aStreams && aStreams.latlng && aStreams.latlng.data) {
                total += aStreams.latlng.data.length;
            }
        }
    }
    var cap = 60000;
    var stride = Math.max(3, Math.ceil(total / cap));

    for (var i = 0; i < operations.length; i++) {
        // Cells this single activity touches, deduplicated so it contributes at most once
        // per cell to the frequency count.
        var cellsThisActivity = {};
        var sets = operations[i].operation_sets || [];

        for (var j = 0; j < sets.length; j++) {
            var streams = sets[j].strava_streams;
            if (!streams || !streams.latlng || !streams.latlng.data) {
                continue;
            }
            var data = streams.latlng.data;
            var track = [];

            for (var k = 0; k < data.length; k += stride) {
                var coordinate = data[k];
                if (!coordinate || coordinate.length < 2) {
                    continue;
                }
                var pt = [coordinate[0], coordinate[1]];
                track.push(pt);
                model.allPoints.push(pt);
                cellsThisActivity[heatmapCellKey(pt[0], pt[1])] = true;
            }

            // Keep the final sample so the drawn line reaches the real end of the track.
            var last = data[data.length - 1];
            if (last && last.length >= 2 && (data.length - 1) % stride !== 0) {
                var endPt = [last[0], last[1]];
                track.push(endPt);
                model.allPoints.push(endPt);
                cellsThisActivity[heatmapCellKey(endPt[0], endPt[1])] = true;
            }

            if (track.length >= 2) {
                model.tracks.push(track);
            }
        }

        for (var key in cellsThisActivity) {
            model.counts[key] = (model.counts[key] || 0) + 1;
            if (model.counts[key] > model.maxCount) {
                model.maxCount = model.counts[key];
            }
        }
    }

    return model;
}

// heatmapColor maps a 0..1 frequency fraction to a colour, sweeping the hue from blue
// (rare) through green/yellow to red (most frequented). High saturation and a slightly
// deeper lightness give the crisp line punchy contrast.
function heatmapColor(fraction) {
    var hue = (1 - fraction) * 220;
    return "hsl(" + hue + ", 100%, 48%)";
}

// heatmapGlowColor is the same hue as the crisp line but lighter, used for the soft
// underlay that gives each route a coloured halo.
function heatmapGlowColor(fraction) {
    var hue = (1 - fraction) * 220;
    return "hsl(" + hue + ", 100%, 62%)";
}

// heatmapBucket converts a cell's visit count into a 0..(HEATMAP_BUCKETS-1) band. With a
// single activity (maxCount <= 1) everything lands in band 0, so a lone run stays cool.
function heatmapBucket(count, maxCount) {
    var fraction = maxCount > 1 ? (count - 1) / (maxCount - 1) : 0;
    return Math.round(fraction * (HEATMAP_BUCKETS - 1));
}

// drawTrackSegments splits one track into runs of consecutive points that share a
// frequency band and pushes each run as a single coloured polyline, so the line's colour
// follows how often each stretch is revisited while keeping the layer count down.
function drawTrackSegments(track, counts, maxCount, renderer, glowLayers, lineLayers) {
    if (track.length < 2) {
        return;
    }

    var bucketAt = function(pt) {
        return heatmapBucket(counts[heatmapCellKey(pt[0], pt[1])] || 1, maxCount);
    };

    var runStart = 0;
    var runBucket = bucketAt(track[0]);

    for (var i = 1; i < track.length; i++) {
        var bucket = bucketAt(track[i]);
        if (bucket !== runBucket) {
            // Include point i as the shared vertex so segments join without gaps.
            pushHeatmapSegment(track.slice(runStart, i + 1), runBucket, renderer, glowLayers, lineLayers);
            runStart = i;
            runBucket = bucket;
        }
    }
    pushHeatmapSegment(track.slice(runStart), runBucket, renderer, glowLayers, lineLayers);
}

function pushHeatmapSegment(points, bucket, renderer, glowLayers, lineLayers) {
    if (points.length < 2) {
        return;
    }
    var fraction = bucket / (HEATMAP_BUCKETS - 1);

    // Soft, wide underlay for the halo.
    glowLayers.push(L.polyline(points, {
        renderer: renderer,
        color: heatmapGlowColor(fraction),
        weight: 9,
        opacity: 0.2,
        lineCap: "round",
        lineJoin: "round"
    }));

    // Crisp line on top.
    lineLayers.push(L.polyline(points, {
        renderer: renderer,
        color: heatmapColor(fraction),
        weight: 4,
        opacity: 0.8,
        lineCap: "round",
        lineJoin: "round"
    }));
}

// densestCenter buckets points into coarse (~1 km) cells, finds the cell with the most
// samples, and returns the average coordinate of that cell — a cheap "biggest cluster".
function densestCenter(points) {

    var cells = {};
    var precision = 100; // round to 2 decimals -> ~1.1 km cells
    var bestKey = null;
    var bestCount = 0;

    for (var i = 0; i < points.length; i++) {
        var latRounded = Math.round(points[i][0] * precision) / precision;
        var lngRounded = Math.round(points[i][1] * precision) / precision;
        var key = latRounded + "," + lngRounded;

        var cell = cells[key];
        if (!cell) {
            cell = { count: 0, latSum: 0, lngSum: 0 };
            cells[key] = cell;
        }
        cell.count++;
        cell.latSum += points[i][0];
        cell.lngSum += points[i][1];

        if (cell.count > bestCount) {
            bestCount = cell.count;
            bestKey = key;
        }
    }

    if (!bestKey) {
        return [points[0][0], points[0][1]];
    }

    var best = cells[bestKey];
    return [best.latSum / best.count, best.lngSum / best.count];
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
    canvas_div.innerHTML = '<canvas id="myChart" class="panel-wide" style="display:none;"></canvas>';

    canvas_div_two = document.getElementById("chart-canvas-div-two");
    canvas_div_two.innerHTML = "";
    canvas_div_two.innerHTML = '<canvas id="myChartTwo" class="panel-wide" style="display:none;"></canvas>';

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
    var canvas = document.getElementById("myChartWeights");
    canvas.style.display = "inline-block";

    // New: build an array of points { t: Date, y: number }
    var points = [];

    // Reverse so newest is last, same as before
    weightsArray = weightsArray.reverse();

    for (var i = 0; i < weightsArray.length; i++) {
        try {
            var dateObject = new Date(weightsArray[i].date);
            if (isNaN(dateObject.getTime())) continue; // skip bad dates

            points.push({
                t: dateObject,                 // time value
                y: weightsArray[i].weight      // weight
            });
        } catch (error) {
            continue;
        }
    }

    // variable used inside the tick callback
    var lastLabel = null;

    new Chart(canvas, {
        type: "line",
        data: {
            datasets: [{
                label: "Weight in KG",
                data: points,
                fill: true,
                borderColor: "rgba(119,141,169,1)",
                pointBackgroundColor: "rgba(119,141,169,1)",
                backgroundColor: "rgba(119,141,169,0.5)",
                tension: 0
            }]
        },
        options: {
            scales: {
                xAxes: [{
                    type: "time",
                    distribution: "linear",
                    time: {
                        unit: "month",
                        tooltipFormat: "YYYY-MM-DD",
                        displayFormats: { month: "MMM YYYY" }
                    },
                    ticks: {
                        source: "data",
                        autoSkip: true,
                        callback: function(value, index, ticks) {
                            // hide duplicates (same formatted label)
                            if (value === lastLabel) {
                                return "";
                            }
                            lastLabel = value;
                            return value;
                        }
                    }
                }],
                yAxes: [{
                    ticks: { beginAtZero: false, precision: 0 }
                }]
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
                <div class="u-w-8"><div class="u-fs-sm">${timeString}</div></div>
                <div class="u-w-5">${weight.weight} KG</div>
                <div class="u-w-8 u-flex-end">
                    <button type="button" class="btn btn--icon" title="Delete weight" aria-label="Delete weight" onclick="deleteWeight('${weight.id}');">
                        <img src="/assets/trash-2.svg">
                    </button>
                </div>
            </div>
        `;
    }

    var htmlContent = `
        <div class="trm-row">
            <div class="trm-field">
                <label class="trm-label" for="weightValue">Weight (KG)</label>
                <input type="number" name="weightValue" id="weightValue" autocomplete="off" min="0" max="500" value="0" />
            </div>
            <div class="trm-field">
                <label class="trm-label" for="weightTime">Time of weight</label>
                <input type="date" name="weightTime" id="weightTime" autocomplete="off" value="${now.toISOString().split('T')[0]}" />
            </div>
        </div>
        <button class="btn btn--primary btn--block" id="register-button" type="submit" onclick="addWeight()">Save</button>
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

function getActivities(){
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
                placeActivitiesInput(result.actions);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/actions?experienced=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeActivitiesInput(actionsArray) {
    var selectActivity = document.getElementById("select_activity");
    for(var i = 0; i < actionsArray.length; i++) {
        var option = document.createElement("option");
        option.text = actionsArray[i].name
        option.value = actionsArray[i].id
        selectActivity.add(option); 
    }
}

function chooseActivity() {
    var selectActivity = document.getElementById("select_activity");
    var activityStartTime = document.getElementById("activityStartTime").value;
    var activityEndTime = document.getElementById("activityEndTime").value;

    if(!activityStartTime || activityStartTime == "" || !activityEndTime || activityEndTime == "") {
        // Show loading gif
        document.getElementById("loading-dumbbell").style.display = "none";

        var myChartElement = document.getElementById("myActivityChart");
        myChartElement.style.display = "none"
        return
    }

    console.log(activityStartTime)
    const date = new Date(activityStartTime);
    const activityStartTimeString = date.toISOString();
    const dateTwo = new Date(activityEndTime);
    const activityEndTimeString = dateTwo.toISOString();

    // Show loading gif
    document.getElementById("loading-dumbbell-activities").style.display = "inline-block";

    // Purge data
    canvas_div = document.getElementById("chart-canvas-div-activity");
    canvas_div.innerHTML = "";
    canvas_div.innerHTML = '<canvas id="myActivityChart" class="panel-wide" style="display:none;"></canvas>';

    document.getElementById("activity-statistics-element-wrapper-div").innerHTML = "";

    resetActivityHeatmap();

    if(selectActivity.value == null || selectActivity.value == 0 || selectActivity.value == "null") {
        // Show loading gif
        document.getElementById("loading-dumbbell-activities").style.display = "none";

        var myChartElement = document.getElementById("myActivityChart");
        myChartElement.style.display = "none"
    } else {
        getActivityStatistics(selectActivity.value, activityStartTimeString, activityEndTimeString)
    }
}

function getActivityStatistics(activityID, startTime, endTime){
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
                placeActivityStatistics(result.statistics);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/actions/" + activityID + "/statistics?start=" + startTime + "&end=" + endTime);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// escapeStat makes user-provided media text safe for innerHTML (titles, artists and
// artwork URLs come from third-party providers).
function escapeStat(value) {
    return String(value == null ? "" : value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

// statCard renders one metric as a dark readout card: a quiet label above a big
// condensed value. family sets the accent stripe (movement/effort/strength/time/audio/
// neutral) so a card's colour encodes what kind of metric it is, not decoration.
function statCard(label, value, unit, family) {
    var unitHtml = unit ? `<span class="stat-card-unit">${escapeStat(unit)}</span>` : "";
    return `
        <div class="stat-card" data-family="${family}">
            <div class="stat-card-label">${escapeStat(label)}</div>
            <div class="stat-card-value">${escapeStat(value)}${unitHtml}</div>
        </div>`;
}

// statSection wraps a group of cards under a quiet heading; renders nothing when the
// group is empty, so activities without that data simply omit the section.
function statSection(title, cards) {
    if (!cards.length) {
        return "";
    }
    return `
        <div class="stat-section">
            <div class="stat-section-title">${escapeStat(title)}</div>
            <div class="stat-grid">${cards.join("")}</div>
        </div>`;
}

// computeStreamMetrics folds the per-set Strava sensor streams (already loaded for the
// GPS heatmap) into period aggregates: heart rate is averaged and peaked, elevation is
// the summed positive altitude change, cadence and power are averaged over active
// samples. Each "has*" flag stays false when no activity in the period carried that
// stream, so the matching cards are skipped for activities that don't record it.
function computeStreamMetrics(operations) {
    var hrSum = 0, hrCount = 0, hrMax = 0;
    var cadSum = 0, cadCount = 0;
    var powSum = 0, powCount = 0;
    var elevation = 0;

    for (var i = 0; i < operations.length; i++) {
        var sets = operations[i].operation_sets || [];
        for (var j = 0; j < sets.length; j++) {
            var streams = sets[j].strava_streams;
            if (!streams) {
                continue;
            }

            if (streams.heartrate && streams.heartrate.data) {
                var hrData = streams.heartrate.data;
                for (var k = 0; k < hrData.length; k++) {
                    if (hrData[k] > 0) {
                        hrSum += hrData[k];
                        hrCount++;
                        if (hrData[k] > hrMax) {
                            hrMax = hrData[k];
                        }
                    }
                }
            }

            if (streams.cadence && streams.cadence.data) {
                var cadData = streams.cadence.data;
                for (var k = 0; k < cadData.length; k++) {
                    if (cadData[k] > 0) {
                        cadSum += cadData[k];
                        cadCount++;
                    }
                }
            }

            if (streams.watts && streams.watts.data) {
                var powData = streams.watts.data;
                for (var k = 0; k < powData.length; k++) {
                    if (powData[k] > 0) {
                        powSum += powData[k];
                        powCount++;
                    }
                }
            }

            if (streams.altitude && streams.altitude.data && streams.altitude.data.length > 1) {
                var altData = streams.altitude.data;
                for (var k = 1; k < altData.length; k++) {
                    var delta = altData[k] - altData[k - 1];
                    if (delta > 0) {
                        elevation += delta;
                    }
                }
            }
        }
    }

    return {
        hasHR: hrCount > 0,
        hrAvg: hrCount > 0 ? Math.round(hrSum / hrCount) : 0,
        hrMax: hrMax,
        hasCadence: cadCount > 0,
        cadenceAvg: cadCount > 0 ? Math.round(cadSum / cadCount) : 0,
        hasPower: powCount > 0,
        powerAvg: powCount > 0 ? Math.round(powSum / powCount) : 0,
        elevationGain: elevation
    };
}

// sumSetField totals a numeric field across a "top" record's operation sets (a top is a
// whole activity, so its distance/time/etc. is the sum of its sets).
function sumSetField(topRecord, field) {
    var total = 0;
    (topRecord.operation_sets || []).forEach(function(operationSet) {
        total += operationSet[field] || 0;
    });
    return total;
}

// placeActivityStatistics renders the chosen activity + period as a dark instrument
// readout: metrics grouped into Totals / Averages / Bests / Sensors, each card carrying
// a family accent, plus the soundtrack panel. Only the metrics an activity actually has
// are shown, so a run and a lift produce different readouts.
function placeActivityStatistics(statistics) {
    var wrapper = document.getElementById("activity-statistics-element-wrapper-div");
    var loading = document.getElementById("loading-dumbbell-activities");
    var stats = statistics.statistics;
    var operations = statistics.operations || [];

    if (!stats || !stats.sums || stats.sums.operations == 0) {
        wrapper.innerHTML = `<div class="stat-empty">No activities in this period.</div>`;
        placeActivityMediaStatistics(statistics.media);
        if (loading) { loading.style.display = "none"; }
        renderActivityHeatmap(operations);
        return;
    }

    var distanceUnit = operations.length ? (operations[0].distance_unit || "") : "";
    var weightUnit = operations.length ? (operations[0].weight_unit || "") : "";

    // Totals
    var totals = [];
    if (stats.sums.distance) { totals.push(statCard("Distance", stats.sums.distance.toFixed(2), distanceUnit, "movement")); }
    if (stats.sums.time) { totals.push(statCard("Time", secondsToDurationString(stats.sums.time), "", "time")); }
    if (stats.sums.repetition) { totals.push(statCard("Reps", stats.sums.repetition, "", "strength")); }
    if (stats.sums.weight) { totals.push(statCard("Weight moved", stats.sums.weight.toFixed(2), weightUnit, "strength")); }
    if (stats.sums.operations) { totals.push(statCard("Sessions", stats.sums.operations, "", "neutral")); }

    // Averages
    var averages = [];
    if (stats.averages.distance) { averages.push(statCard("Avg distance", stats.averages.distance.toFixed(2), distanceUnit, "movement")); }
    if (stats.averages.time) { averages.push(statCard("Avg time", secondsToDurationString(stats.averages.time), "", "time")); }
    if (stats.averages.repetition) { averages.push(statCard("Avg reps", Math.round(stats.averages.repetition), "", "strength")); }
    if (stats.averages.weight) { averages.push(statCard("Avg weight", stats.averages.weight.toFixed(2), weightUnit, "strength")); }

    // Bests
    var bests = [];
    if (stats.tops.distance) { bests.push(statCard("Furthest", sumSetField(stats.tops.distance, "distance").toFixed(2), stats.tops.distance.distance_unit || distanceUnit, "movement")); }
    if (stats.tops.time) { bests.push(statCard("Longest", secondsToDurationString(sumSetField(stats.tops.time, "time")), "", "time")); }
    if (stats.tops.repetition) { bests.push(statCard("Most reps", sumSetField(stats.tops.repetition, "repetitions"), "", "strength")); }
    if (stats.tops.weight) { bests.push(statCard("Heaviest", sumSetField(stats.tops.weight, "weight").toFixed(2), weightUnit, "strength")); }

    // Sensors (from Strava streams)
    var stream = computeStreamMetrics(operations);
    var sensors = [];
    if (stream.hasHR) {
        sensors.push(statCard("Avg heart rate", stream.hrAvg, "bpm", "effort"));
        sensors.push(statCard("Max heart rate", stream.hrMax, "bpm", "effort"));
    }
    if (stream.elevationGain > 0) { sensors.push(statCard("Elevation gain", Math.round(stream.elevationGain), "m", "movement")); }
    if (stream.hasCadence) { sensors.push(statCard("Avg cadence", stream.cadenceAvg, "rpm", "movement")); }
    if (stream.hasPower) { sensors.push(statCard("Avg power", stream.powerAvg, "W", "effort")); }

    var body = statSection("Totals", totals)
        + statSection("Averages", averages)
        + statSection("Bests", bests)
        + statSection("Sensors", sensors);

    wrapper.innerHTML = `<div class="stat-panel">${body}</div>`;

    // Soundtrack overlay: only rendered when media is enabled and the matched sessions
    // had playback (the backend leaves statistics.media null otherwise).
    placeActivityMediaStatistics(statistics.media);

    // Remove loading gif
    if (loading) { loading.style.display = "none"; }

    // Draw the GPS heatmap from the same operations, inheriting the activity + date filter.
    renderActivityHeatmap(operations);
}

// soundtrackSub joins an artist and a play count into a "Artist · N plays" subline.
function soundtrackSub(artist, count) {
    var parts = [];
    if (artist) { parts.push(artist); }
    if (count > 1) { parts.push(count + " plays"); }
    return parts.join(" · ");
}

// soundtrackTopRow renders a highlighted media item: cover art (or a typed glyph when the
// provider gave no artwork) beside a label / title / subline.
function soundtrackTopRow(label, title, sub, artwork, glyph) {
    var art = artwork
        ? `<div class="soundtrack-art" style="background-image:url('${escapeStat(artwork)}')"></div>`
        : `<div class="soundtrack-art soundtrack-art-glyph">${glyph}</div>`;
    var subHtml = sub ? `<div class="soundtrack-top-sub">${escapeStat(sub)}</div>` : "";
    return `
        <div class="soundtrack-top">
            ${art}
            <div class="soundtrack-top-body">
                <div class="soundtrack-top-label">${escapeStat(label)}</div>
                <div class="soundtrack-top-title">${escapeStat(title)}</div>
                ${subHtml}
            </div>
        </div>`;
}

// soundtrackColumn is one side of the soundtrack panel (Music or Spoken): its top rows
// and a row of pill-shaped totals. Returns "" when there is nothing to show.
function soundtrackColumn(title, rows, meta) {
    if (!rows && !meta.length) {
        return "";
    }
    var metaHtml = meta.length
        ? `<div class="soundtrack-meta">${meta.map(function(m) { return `<span>${escapeStat(m)}</span>`; }).join("")}</div>`
        : "";
    return `
        <div class="soundtrack-col">
            <div class="soundtrack-col-title">${escapeStat(title)}</div>
            ${rows}
            ${metaHtml}
        </div>`;
}

// placeActivityMediaStatistics renders the aggregated soundtrack as its own amber-marked
// panel, split into Music and Spoken columns so a run soundtracked by a podcast reads as
// richly as one soundtracked by songs. It is a no-op when there is no media block, so the
// panel stays hidden unless soundtracks were listened to in the period.
function placeActivityMediaStatistics(media) {
    if (!media) {
        return;
    }

    var wrapper = document.getElementById("activity-statistics-element-wrapper-div");

    // Music column.
    var musicRows = "";
    if (media.top_track) {
        musicRows += soundtrackTopRow("Top song", media.top_track.title, soundtrackSub(media.top_track.artist, media.top_track.count), media.top_track.artwork, "♪");
    }
    if (media.top_artist) {
        musicRows += soundtrackTopRow("Top artist", media.top_artist.title, soundtrackSub("", media.top_artist.count), media.top_artist.artwork, "♫");
    }
    var musicMeta = [];
    if (media.songs > 0) { musicMeta.push(media.songs + (media.songs == 1 ? " song" : " songs")); }
    if (media.unique_artists > 0) { musicMeta.push(media.unique_artists + (media.unique_artists == 1 ? " artist" : " artists")); }
    if (media.listening_time > 0) { musicMeta.push(secondsToDurationString(media.listening_time)); }
    var musicCol = soundtrackColumn("Music", musicRows, musicMeta);

    // Spoken column.
    var spokenRows = "";
    if (media.top_podcast) {
        var episodes = media.top_podcast.count;
        var episodesLabel = episodes > 0 ? (episodes + (episodes == 1 ? " episode" : " episodes")) : "";
        spokenRows += soundtrackTopRow("Top podcast", media.top_podcast.title, episodesLabel, media.top_podcast.artwork, "🎙");
    }
    if (media.top_audiobook) {
        spokenRows += soundtrackTopRow("Top audiobook", media.top_audiobook.title, media.top_audiobook.artist, media.top_audiobook.artwork, "📖");
    }
    var spokenMeta = [];
    if (media.podcast_episodes > 0) { spokenMeta.push(media.podcast_episodes + (media.podcast_episodes == 1 ? " episode" : " episodes")); }
    if (media.audiobooks > 0) { spokenMeta.push(media.audiobooks + (media.audiobooks == 1 ? " book" : " books")); }
    if (media.spoken_time > 0) { spokenMeta.push(secondsToDurationString(media.spoken_time)); }
    var spokenCol = soundtrackColumn("Spoken", spokenRows, spokenMeta);

    if (!musicCol && !spokenCol) {
        return;
    }

    wrapper.innerHTML += `
        <div class="soundtrack-panel">
            <div class="soundtrack-head"><span class="soundtrack-mark">♫</span>Soundtrack</div>
            <div class="soundtrack-cols">${musicCol}${spokenCol}</div>
        </div>`;
}

function placeDefaultDates() {
    // Get today's date
    const today = new Date();

    // Format to YYYY-MM-DD
    const todayStr = today.toISOString().split('T')[0];

    // Get a date one week ago
    const weekAgo = new Date();
    weekAgo.setDate(today.getDate() - 7);
    const weekAgoStr = weekAgo.toISOString().split('T')[0];

    // Set the input values
    document.getElementById('activityEndTime').value = todayStr;
    document.getElementById('activityStartTime').value = weekAgoStr;
}