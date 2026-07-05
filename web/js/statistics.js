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

                <div class="module" id="stats_module">

                    <div class="title">
                        Activity statistics
                    </div>

                    <div class="form-group">
                        <select id='select_activity' class='form-control' onchange="chooseActivity()">
                            <option value="null">Choose activity</option>
                        </select>

                        <input style="" class="" type="date" id="activityStartTime" name="activityStartTime" value="" onchange="chooseActivity()" required>
                        <input style="" class="" type="date" id="activityEndTime" name="activityEndTime" value="" onchange="chooseActivity()" required>
                    </div>

                    <div>

                        <div class="module" id="loading-dumbbell-activities" style="display: none;">
                            <img src="/assets/images/barbell.gif">
                        </div>

                        <div id="activity-statistics-element-wrapper-div" class="season-statistics-element-wrapper-div">
                        </div>

                        <div id="chart-canvas-div-activity" style="max-width: 40em; margin: 1em auto; padding: 0 0.5em; background-color: var(--white); border-radius: 1em;">
                            <canvas id="myActivityChart" style="max-width: 100%; width: 1000px; display:none;"></canvas>
                        </div>

                        <div id="activity-heatmap-wrapper" style="display: none;">
                            <div class="text-body" style="text-align: center; margin-top: 1em;">
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
    canvas_div.innerHTML = '<canvas id="myActivityChart" style="max-width: 100%; width: 1000px; display:none;"></canvas>';

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

function placeActivityStatistics(statistics) {
    if(statistics.statistics.sums.operations == 0) {
        // Remove loading gif
        document.getElementById("loading-dumbbell-activities").style.display = "none";
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                No activities in period :(
            </div>
        `;
    }

    // Sums
    if(statistics.statistics.sums.distance) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Distance in period: ${statistics.statistics.sums.distance.toFixed(2)} ${statistics.operations[0].distance_unit}
            </div>
        `;
    }

    if(statistics.statistics.sums.time) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Time spent in period: ${secondsToDurationString(statistics.statistics.sums.time)}
            </div>
        `;
    }

    if(statistics.statistics.sums.repetition) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Repetitions in period: ${statistics.statistics.sums.repetition}
            </div>
        `;
    }

    if(statistics.statistics.sums.weight) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Weight in period: ${statistics.statistics.sums.weight.toFixed(2)} ${statistics.operations[0].weight_unit}
            </div>
        `;
    }

    if(statistics.statistics.sums.operations) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Amount in period: ${statistics.statistics.sums.operations}
            </div>
        `;
    }

    // Averages
    if(statistics.statistics.averages.distance) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Average distance: ${statistics.statistics.averages.distance.toFixed(2)} ${statistics.operations[0].distance_unit}
            </div>
        `;
    }

    if(statistics.statistics.averages.time) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Average time: ${secondsToDurationString(statistics.statistics.averages.time)}
            </div>
        `;
    }

    if(statistics.statistics.averages.repetition) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Average repetitions: ${statistics.statistics.averages.repetition}
            </div>
        `;
    }

    if(statistics.statistics.averages.weight) {
        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Average weight: ${statistics.statistics.averages.weight.toFixed(2)} ${statistics.operations[0].weight_unit}
            </div>
        `;
    }

    // Tops
    if(statistics.statistics.tops.distance) {
        distance = 0;
        statistics.statistics.tops.distance.operation_sets.forEach(operationSet => {
            distance += operationSet.distance;
        });

        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Furthest activity: ${distance.toFixed(2)} ${statistics.statistics.tops.distance.distance_unit}
            </div>
        `;
    }

    if(statistics.statistics.tops.time) {
        time = 0;
        statistics.statistics.tops.time.operation_sets.forEach(operationSet => {
            time += operationSet.time;
        });

        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Longest activity: ${secondsToDurationString(time)}
            </div>
        `;
    }

    if(statistics.statistics.tops.repetition) {
        repetitions = 0;
        statistics.statistics.tops.repetition.operation_sets.forEach(operationSet => {
            repetitions += operationSet.repetitions;
        });

        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Most repetitions: ${repetitions}
            </div>
        `;
    }

    if(statistics.statistics.tops.weight) {
        weight = 0;
        statistics.statistics.tops.weight.operation_sets.forEach(operationSet => {
            weight += operationSet.weight;
        });

        document.getElementById("activity-statistics-element-wrapper-div").innerHTML += `
            <div class="season-statistics-element unselectable">
                Highest weight: ${weight.toFixed(2)} ${statistics.operations[0].weight_unit}
            </div>
        `;
    }

    // Soundtrack overlay: only rendered when media is enabled and the matched sessions
    // had playback (the backend leaves statistics.media null otherwise).
    placeActivityMediaStatistics(statistics.media);

    // Remove loading gif
    document.getElementById("loading-dumbbell-activities").style.display = "none";

    // Draw the GPS heatmap from the same operations, inheriting the activity + date filter.
    renderActivityHeatmap(statistics.operations);
}

// placeActivityMediaStatistics renders the aggregated soundtrack stats for the chosen
// activity + period as pills under a "Soundtrack" heading, marked with the amber audio
// accent used by the workout media rail. It is a no-op when there is no media block, so
// the section stays hidden unless relevant soundtracks were listened to in the period.
function placeActivityMediaStatistics(media) {
    if (!media) {
        return;
    }

    var wrapper = document.getElementById("activity-statistics-element-wrapper-div");
    var pills = "";

    if (media.top_track) {
        var trackArtist = media.top_track.artist ? " — " + media.top_track.artist : "";
        var trackPlays = media.top_track.count > 1 ? ` (${media.top_track.count} plays)` : "";
        pills += `
            <div class="season-statistics-element unselectable">
                Top song: ${media.top_track.title}${trackArtist}${trackPlays}🎵
            </div>
        `;
    }

    if (media.top_artist) {
        var artistPlays = media.top_artist.count > 1 ? ` (${media.top_artist.count} plays)` : "";
        pills += `
            <div class="season-statistics-element unselectable">
                Top artist: ${media.top_artist.title}${artistPlays}🎤
            </div>
        `;
    }

    if (media.songs > 0) {
        pills += `
            <div class="season-statistics-element unselectable">
                Songs played: ${media.songs}🎧
            </div>
        `;
    }

    if (media.unique_artists > 0) {
        pills += `
            <div class="season-statistics-element unselectable">
                Different artists: ${media.unique_artists}👥
            </div>
        `;
    }

    if (media.listening_time > 0) {
        pills += `
            <div class="season-statistics-element unselectable">
                Music listened: ${secondsToDurationString(media.listening_time)}⏱️
            </div>
        `;
    }

    if (media.spoken_time > 0) {
        pills += `
            <div class="season-statistics-element unselectable">
                Spoken audio: ${secondsToDurationString(media.spoken_time)}🎙️
            </div>
        `;
    }

    if (!pills) {
        return;
    }

    wrapper.innerHTML += `
        <div class="text-body activity-soundtrack-heading" style="text-align: center; width: 100%; margin-top: 1em;">
            Soundtrack
        </div>
        ${pills}
    `;
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