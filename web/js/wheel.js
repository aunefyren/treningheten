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

            <div id="spinner-info" class="spinner-info">
            </div>

            <div style="overflow: hidden; max-width: 40em;">
                <canvas id='canvas' width='1000' height='1000' style="max-width: 100%;">
                    Canvas not supported, use another browser.
                </canvas>
            </div>

            <div id="canvas-buttons" class="canvas-buttons">

                <button id="bigButton" class="bigButton" onclick="calculatePrize(); this.disabled=true;">Spin the Wheel</button>
                <a href="javascript:void(0);" id="reset-button" style="display: none;" onclick="theWheel.stopAnimation(false); theWheel.rotationAngle=0; theWheel.draw(); drawTriangle(); bigButton.disabled=false; clearResponse();">Reset</a>
                <a href="javascript:void(0);" id="reload-button" style="display: none;" onclick="location.reload();">Replay</a>
            </div>
        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Your very own page...';
    clearResponse();

    console.log("user id: " + user_id)

    // Check URL for parameters
    const query_string = window.location.search;
    parameters = get_url_parameters(query_string)
    console.log(parameters)
    if(parameters != false && "debt_id" in parameters) {
        debt_id = Number(decodeURI(parameters.debt_id))
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

                if(result.debt.winner.ID != 0) {
                    winner = result.debt.winner
                    replay = true;
                    document.getElementById('bigButton').innerHTML = "See the result";
                    document.getElementById('reset-button').style.display = "inline-block";
                } else {
                    if(result.debt.loser.ID != user_id) {
                        error("This wheel has not been spun yet.")
                        return;
                    }
                }

                var date_str = ""
                try {
                    var date = new Date(result.debt.date);
                    var date_week = date.GetWeek();
                    var date_year = date.getFullYear();
                    date_str = date_week + " (" + date_year + ")"
                } catch {
                    date_str = "Error"
                }

                document.getElementById('spinner-info').innerHTML = result.debt.loser.first_name + " is spinning for week " + date_str + "!";

                loser = result.debt.loser;
                prize = result.debt.season.prize;

                placeWheel(result.winners, replay);
                
            }

        } else {
            info("Loading spin...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/debt/" + debt_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeWheel(canidateArray) {

    var colors = [
        "#800000", "#9A6324", "#808000", "#469990", "#e6194B", "#f58231", "#ffe119", "#bfef45", "#3cb44b", "#42d4f4", "#4363d8", "#911eb4", "#f032e6", "#a9a9a9", "#fabed4", "#ffd8b1", "#fffac8", "#aaffc3", "#dcbeff", "#ffffff"
    ]

    document.getElementById('wheel-module').style.display = "flex";

    var ticketAmount = 0;

    for(var i = 0; i < canidateArray.length; i++) {
        ticketAmount += canidateArray[i].tickets;

        if(colors.length > 0) {
            var index = Math.floor(Math.random()*colors.length)
            canidateArray[i].color = colors[index]
            var colors2 = []
            for(var j = 0; j < colors.length; j++) {
                if(j != index) {
                    colors2.push(colors[j]);
                }
            }
            colors = colors2;
        } else {
            canidateArray[i].color = "#" + Math.floor(Math.random()*16777215).toString(16);
        }
    }

    console.log("Placing " + ticketAmount + " tickets.");

    // Establish all the different integers of tickets for users
    var ticketAmounts = []
    for(var i = 0; i < canidateArray.length; i++) {
        ticketAmounts.push(canidateArray[i].tickets)
    }

    // Calculate Greatest Common Divisor
    var GCD = gcdOfArray(ticketAmounts)
    console.log("GCD: " + GCD)

    // Divide all tickets by GCD
    if(GCD > 1) {
        console.log("Dividing tickets by GCD.");
        for(var i = 0; i < canidateArray.length; i++) {
            canidateArray[i].tickets = Math.floor(canidateArray[i].tickets / GCD)
        }
    }

    // Add tickets to wheel dict
    for(var i = 0; i < canidateArray.length; i++) {
        for(var j = 0; j < canidateArray[i].tickets; j++) {
            placementArray.push({'fillStyle' : canidateArray[i].color, 'textStrokeStyle' : '#000000', 'text' : canidateArray[i].user.first_name, 'user_id' : canidateArray[i].user.ID})
        }
    }

    // Shuffle array order
    for(var i = 0; i < 10; i++) {
        placementArray = placementArray.sort((a, b) => 0.5 - Math.random());
    }

    theWheel = new Winwheel({
        'numSegments'    : placementArray.length,
        'outerRadius'    : 450,
        'centerX'        : 500,    // correctly position the wheel
        'centerY'        : 500,
        'segments'       : placementArray,
        'textAlignment'  : 'outer',
        'textFontSize'   : 30,
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
            if(placementArray[i].user_id == winner.ID) {
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

    ctx.strokeStyle = 'navy';     // Set line colour.
    ctx.fillStyle   = 'aqua';     // Set fill colour.
    ctx.lineWidth   = 2;
    ctx.beginPath();              // Begin path.
    ctx.moveTo(460, 5);           // Move to initial position.
    ctx.lineTo(540, 5);           // Draw lines to make the shape.
    ctx.lineTo(500, 75);
    ctx.lineTo(460, 5);
    ctx.stroke();                 // Complete the path by stroking (draw lines).
    ctx.fill();                   // Then fill.
}

function spinFinished() {
    

    if(winner.ID == user_id) {
        info("You won " + prize.quantity + " " + prize.name + " from " + loser.first_name + ".")
        trigger_fireworks(1);
    } else {
        info(winner.first_name + " won " + prize.quantity + " " + prize.name + " from " + loser.first_name + ".")
    }
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

                error(result.error);

            } else {

                winner = result.winner;
                
            }

            resolve();

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/debt/" + debt_id + "/choose");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}