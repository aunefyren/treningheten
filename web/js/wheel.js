function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            var sunday_alert = login_data.data.sunday_alert
            admin = login_data.data.admin
            user_id = login_data.data.id
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            var sunday_alert = false
            admin = false
            user_id = 0;
        }

        showAdminMenu(admin)

    } else {
        var email = ""
        var first_name = ""
        var last_name = ""
        var admin = false;
        var sunday_reminder = false
        user_id = 0;
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
                    
        <div class="module" id="wheel-module" style="display: none;">

            <div id="spinner-winner-image-wrapper" class="spinner-winner-image-wrapper">
                <div class="title">
                    Winner!
                </div>
                <div class="spinner-winner-image-div" id="spinner-winner-image-div">
                    <img src="/assets/images/barbell.gif" id="spinner-winner-image" class="shiny-image"></img>
                </div>
            </div>

            <div id="spinner-info" class="spinner-info">
            </div>

            <div class="wheel-canvas-wrap">
                <canvas id='canvas' width='1000' height='1000'>
                    Canvas not supported, use another browser.
                </canvas>
            </div>

            <div id="canvas-buttons" class="canvas-buttons">

                <button id="bigButton" class="btn btn--primary" onclick="calculatePrize(); this.disabled=true;">Spin the Wheel</button>
                <a href="javascript:void(0);" id="reset-button" class="btn btn--ghost" style="display: none;" onclick="theWheel.stopAnimation(false); theWheel.rotationAngle=0; theWheel.draw(); drawTriangle(); bigButton.disabled=false; clearResponse(); resetPage();">Reset</a>
                <a href="javascript:void(0);" id="reload-button" class="btn btn--ghost" style="display: none;" onclick="location.reload();">Replay</a>
            </div>
        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'It\'s not gambling...';
    clearResponse();

    console.log("user id: " + user_id)

    // Check URL for parameters
    const query_string = window.location.search;
    parameters = get_url_parameters(query_string)
    console.log(parameters)
    if(parameters != false && "debt_id" in parameters) {
        debt_id = decodeURI(parameters.debt_id)
    } else {
        console.log("Debt ID not found in parameters.")
    }

    if(result !== false) {
        showLoggedInMenu();
        // Usual pointer drawing code.
        get_debt(debt_id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_debt(debt_id) {

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

                if(result.debt.winner != null) {
                    winner = result.debt.winner
                    replay = true;
                    document.getElementById('bigButton').innerHTML = "See the result";
                    document.getElementById('reset-button').style.display = "inline-block";
                } else {
                    if(result.debt.loser.id != user_id) {
                        error("This wheel has not been spun yet.")
                        return;
                    }
                }

                var date_str = ""
                try {
                    var date = new Date(result.debt.date);
                    var date_week = date.getWeek(1);
                    var date_year = date.getFullYear();
                    date_str = date_week + " (" + date_year + ")"
                } catch {
                    date_str = "Error"
                }

                document.getElementById('spinner-info').innerHTML = result.debt.loser.first_name + " " + result.debt.loser.last_name + " is spinning for week " + date_str + "!";

                loser = result.debt.loser;
                prize = result.debt.season.prize;

                placeWheel(result.winners, replay);
                
            }

        } else {
            info("Loading spin...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/debts/" + debt_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function isHexColor(value) {
    return typeof value === "string" && /^#[0-9a-fA-F]{6}$/.test(value);
}

// readableTextColor returns black or white depending on the perceived brightness of
// the background, so segment labels stay legible on any fill color.
function readableTextColor(hex) {
    var r = parseInt(hex.substr(1, 2), 16);
    var g = parseInt(hex.substr(3, 2), 16);
    var b = parseInt(hex.substr(5, 2), 16);
    var yiq = (r * 299 + g * 587 + b * 114) / 1000;
    return yiq >= 140 ? "#000000" : "#ffffff";
}

// fitSegmentFontSize returns the largest font size (down to a floor) at which a label
// fits the radial run of a segment. Winwheel draws 'outer'/horizontal text from the rim
// inward with no fitting of its own, so a long first name (or a wide emoji) would spill
// past the wide part of the wedge toward the pointy center — the "name placed outside the
// slice" bug. We never truncate: a long name only gets a smaller font. Measured in the
// wheel's base (logical) units — Winwheel multiplies both geometry and font by scaleFactor
// internally, so the ratio (and therefore the fitted size) is scale-independent.
function fitSegmentFontSize(ctx, label, baseFontSize) {
    var outerRadius = 450;                          // matches Winwheel config below
    var textMargin = baseFontSize / 1.7;            // matches Winwheel's default margin
    var innerKeepRadius = outerRadius * 0.34;       // keep the label out of the pointy center
    var radialBudget = (outerRadius - textMargin) - innerKeepRadius;
    var minFontSize = 16;

    ctx.font = "bold " + baseFontSize + "px Arial"; // matches Winwheel's textFontWeight/Family
    var width = ctx.measureText(label).width;
    if (width <= radialBudget) {
        return baseFontSize;
    }
    // Text width scales linearly with font size, so the fitting size is a closed form.
    var fitted = Math.floor(baseFontSize * (radialBudget / width));
    return Math.max(minFontSize, fitted);
}

// hslToHex spreads auto colors deterministically once the curated palette is used up.
function hslToHex(h, s, l) {
    s /= 100; l /= 100;
    var c = (1 - Math.abs(2 * l - 1)) * s;
    var x = c * (1 - Math.abs((h / 60) % 2 - 1));
    var m = l - c / 2;
    var r = 0, g = 0, b = 0;
    if (h < 60) { r = c; g = x; }
    else if (h < 120) { r = x; g = c; }
    else if (h < 180) { g = c; b = x; }
    else if (h < 240) { g = x; b = c; }
    else if (h < 300) { r = x; b = c; }
    else { r = c; b = x; }
    var toHex = function(v) { return ("0" + Math.round((v + m) * 255).toString(16)).slice(-2); };
    return "#" + toHex(r) + toHex(g) + toHex(b);
}

// assignWheelColors sets candidateArray[i].color. A user's chosen color is honored
// verbatim (duplicates allowed). Everyone else gets a stable, distinct color: ordered
// by user id (deterministic across spins) and drawn from the unused palette, falling
// back to a golden-angle spread if the palette is exhausted.
function assignWheelColors(candidateArray, palette) {
    var used = {};

    for (var i = 0; i < candidateArray.length; i++) {
        var pick = candidateArray[i].user ? candidateArray[i].user.wheel_color : null;
        if (isHexColor(pick)) {
            candidateArray[i].color = pick;
            used[pick.toLowerCase()] = true;
        }
    }

    var auto = candidateArray.filter(function(c) { return !c.color; });
    auto.sort(function(a, b) {
        var ai = (a.user && a.user.id) || "";
        var bi = (b.user && b.user.id) || "";
        return ai < bi ? -1 : ai > bi ? 1 : 0;
    });

    var goldenIndex = 0;
    for (var i = 0; i < auto.length; i++) {
        var chosen = null;
        for (var j = 0; j < palette.length; j++) {
            if (!used[palette[j].toLowerCase()]) { chosen = palette[j]; break; }
        }
        if (!chosen) {
            chosen = hslToHex((goldenIndex * 137.508) % 360, 65, 55);
            goldenIndex++;
        }
        auto[i].color = chosen;
        used[chosen.toLowerCase()] = true;
    }
}

function placeWheel(candidateArray) {

    var palette = [
        "#800000", "#9A6324", "#808000", "#469990", "#e6194B", "#f58231", "#ffe119", "#bfef45", "#3cb44b", "#42d4f4", "#4363d8", "#911eb4", "#f032e6", "#a9a9a9", "#fabed4", "#ffd8b1", "#fffac8", "#aaffc3", "#dcbeff", "#ffffff"
    ];

    document.getElementById('wheel-module').style.display = "flex";

    // Render the canvas backing store at the device pixel ratio so segment borders and
    // names stay crisp on high-DPI / mobile screens. The displayed (CSS) size is
    // unchanged — only the backing resolution and Winwheel's scaleFactor grow. Capped
    // at 3 to bound the canvas size. Stroke widths (not scaled by Winwheel) and the
    // manual pointer are multiplied by wheelScale to keep their on-screen thickness.
    var wheelBaseSize = 1000;
    var wheelScale = Math.min(Math.max(window.devicePixelRatio || 1, 1), 3);
    var wheelCanvas = document.getElementById('canvas');
    if (wheelCanvas) {
        wheelCanvas.width = wheelBaseSize * wheelScale;
        wheelCanvas.height = wheelBaseSize * wheelScale;
        wheelCanvas.style.width = wheelBaseSize + 'px';
        wheelCanvas.style.height = 'auto';
        wheelCanvas.style.maxWidth = '100%';
    }

    // Honor each user's chosen color; auto-assign a stable, distinct color otherwise.
    assignWheelColors(candidateArray, palette);

    var ticketAmount = 0;
    for(var i = 0; i < candidateArray.length; i++) {
        ticketAmount += candidateArray[i].tickets;
    }

    console.log("Placing " + ticketAmount + " tickets.");

    // Establish all the different integers of tickets for users
    var ticketAmounts = []
    for(var i = 0; i < candidateArray.length; i++) {
        ticketAmounts.push(candidateArray[i].tickets)
    }

    // Calculate Greatest Common Divisor
    var GCD = gcdOfArray(ticketAmounts)
    console.log("GCD: " + GCD)

    // Divide all tickets by GCD
    if(GCD != null && GCD > 1) {
        console.log("Dividing tickets by GCD.");
        for(var i = 0; i < candidateArray.length; i++) {
            candidateArray[i].tickets = Math.floor(candidateArray[i].tickets / GCD)
        }
    }

    console.log(candidateArray.length + " candidates for wheel")

    // Shared 2D context used only to measure label widths for per-segment font fitting.
    var measureCtx = wheelCanvas ? wheelCanvas.getContext('2d') : null;
    var baseFontSize = 34;

    // Add tickets to wheel dict
    for(var i = 0; i < candidateArray.length; i++) {
        var user = candidateArray[i].user;
        var fill = candidateArray[i].color;
        var textColor = readableTextColor(fill);

        var label = user.first_name;
        if (user.wheel_emoji && user.wheel_emoji.length > 0 && user.wheel_emoji.length <= 16) {
            label = user.wheel_emoji + " " + label;
        }

        // Shrink long labels so they stay within the segment instead of spilling toward
        // the wheel center. Short names keep the full base size.
        var fontSize = measureCtx ? fitSegmentFontSize(measureCtx, label, baseFontSize) : baseFontSize;

        var segment = {
            'fillStyle'       : fill,
            'text'            : label,
            'textFontSize'    : fontSize,
            // Contrast comes from the luminance-picked fill color (black on light fills,
            // white on dark). We deliberately omit a text outline: Winwheel strokes the
            // outline ON TOP of the fill with miter joins, which looks rough/aliased on
            // rotated glyphs. Fill-only gives clean, smoothly anti-aliased text.
            'textFillStyle'   : textColor,
            'textStrokeStyle' : null,
            'user_id'         : user.id
        };

        // Optional per-user segment border.
        if (isHexColor(user.wheel_border_color)) {
            segment.strokeStyle = user.wheel_border_color;
            segment.lineWidth = 8 * wheelScale;
        }

        for(var j = 0; j < candidateArray[i].tickets; j++) {
            placementArray.push(Object.assign({}, segment));
        }
    }

    // Shuffle array order
    for(var i = 0; i < 10; i++) {
        placementArray = placementArray.sort((a, b) => 0.5 - Math.random());
    }

    theWheel = new Winwheel({
        'numSegments'    : placementArray.length,
        'scaleFactor'    : wheelScale,
        'outerRadius'    : 450,
        'centerX'        : 500,    // correctly position the wheel
        'centerY'        : 500,
        'segments'       : placementArray,
        'textAlignment'  : 'outer',
        'textFontSize'   : 34,
        'animation'      :
        {
            'type'          : 'spinToStop',
            'duration'      : 8,
            'spins'         : 16,
            'callbackAfter' : 'drawTriangle()',
            'callbackFinished' : 'spinFinished()'
        }
    });

    // Usual pointer drawing code.
    drawTriangle();
}

// Function with formula to work out stopAngle before spinning animation.
// Called from Click of the Spin button.
async function calculatePrize()
{   

    var segment = 0;

    if(!replay) {
        try {
            info("Preparing spin...");
            await new Promise((resolve) => {
                choose_winner(resolve, debt_id);
            });
        } catch(e) {
          console.log(`Failed API call. Error: '${e}'.`);
          response.infoLog += `\nFailed API call. Error: '${e}'.`;
          return response;
        }
    }

    if(winner != false) {
        for(var i = 0; i < placementArray.length; i++) {
            if(placementArray[i].user_id == winner.id) {
                segment = i+1;
            }
        }
    } else {
        error("No winner.");
        return
    }

    if(segment == 0) {
        error("Failed to find correct segment.");
        return
    }

    let stopAt = theWheel.getRandomForSegment(segment);
 
    // Important thing is to set the stopAngle of the animation before stating the spin.
    theWheel.animation.stopAngle = stopAt;

    // Start the spin animation here.
    theWheel.startAnimation();

    if(!replay) {
        document.getElementById('reload-button').style.display = "inline-block";
    }

}

function drawTriangle()
{
    // Get the canvas context the wheel uses.
    let ctx = theWheel.ctx;

    // The triangle is drawn outside Winwheel, so scale its coords by the same factor
    // as the wheel to stay aligned with the DPR-scaled backing store.
    let s = theWheel.scaleFactor || 1;

    ctx.strokeStyle = 'navy';     // Set line color.
    ctx.fillStyle   = 'aqua';     // Set fill color.
    ctx.lineWidth   = 2 * s;
    ctx.beginPath();              // Begin path.
    ctx.moveTo(460 * s, 5 * s);   // Move to initial position.
    ctx.lineTo(540 * s, 5 * s);   // Draw lines to make the shape.
    ctx.lineTo(500 * s, 75 * s);
    ctx.lineTo(460 * s, 5 * s);
    ctx.stroke();                 // Complete the path by stroking (draw lines).
    ctx.fill();                   // Then fill.
}

function spinFinished() {

    GetProfileImage(winner.id);
    document.getElementById('spinner-winner-image-div').onclick = function(){location.href=`/users/${winner.id}`};
    document.getElementById('spinner-winner-image-wrapper').style.animation = "slide 0.25s ease 0.5s forwards";
    setTimeout(function () {
        document.getElementById('spinner-winner-image-wrapper').style.animation = "slide 0.25s ease 0.5s forwards, smooth-appear 0.5s ease forwards";
    }, 1000);
    
    if(winner.id == user_id) {
        info("You won " + prize.quantity + " " + prize.name + " from " + loser.first_name + ".")
    } else {
        info(winner.first_name + " " + winner.last_name + " won " + prize.quantity + " " + prize.name + " from " + loser.first_name + ".")
    }

    trigger_fireworks(2);

}

async function choose_winner(resolve, debt_id) {

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

                clearResponse();
                error(result.error);

            } else {

                clearResponse();
                winner = result.winner;
                
            }

            resolve();

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/debts/" + debt_id + "/choose");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function GetProfileImage(userID) {

    var img = document.getElementById("spinner-winner-image");
    if (!img) {
        return;
    }
    img.onerror = function() { this.onerror = null; this.src = '/assets/images/barbell.gif'; };
    img.src = profileImageURL(userID, false);

    setInterval(function () {
        document.getElementById('spinner-winner-image-div').classList.add('shine')
        setTimeout(function () {
            document.getElementById('spinner-winner-image-div').classList.remove('shine')
        }, 2000);
    }, 3000);

}

function resetPage() {
    document.getElementById("spinner-winner-image").src = "/assets/images/barbell.gif"
    document.getElementById('spinner-winner-image-wrapper').style.animation = "";
    document.getElementById('spinner-winner-image-wrapper').style.height = "0";
}