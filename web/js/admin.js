function load_page(result) {

    var admin = false

    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            admin = login_data.data.admin
        } catch {
            admin = false
        }

        showAdminMenu(admin)
    }

    var html = `
                <div class="modules" id="admin-page">
                    
                    <div class="server-info-module" id="server-info-module">

                        <div class="server-info" id="server-info">
                            <h3 id="server-info-title">Server info:</h3>
                            <p id="server-treningheten-version-title" style="">Version: <a id="server-treningheten-version">...</a></p>
                            <p id="server-timezone-title" style="">Timezone: <a id="server-timezone">...</a></p>
                        </div>

                    </div>

                    <div class="invitation-module" id="invitation-module">

                        <div class="invites" id="invites">

                            <h3 id="invitation-module-title">Invites:</h3>

                            <div class="invite-list" id="invite-list">
                            </div>

                            <button type="submit" onclick="generate_invite();" id="generate_invite_button" style=""><img src="assets/plus.svg" class="btn_logo color-invert"><p2>Generate</p2></button>
                        
                        </div>

                    </div>

                    <div class="debt-module" id="debt-module">

                        <div class="debt-form" id="debt-form">

                            <h3 id="debt-module-title">Debt:</h3>

                            <form action="" onsubmit="event.preventDefault(); generate_debt();">
                                
                                <label for="debt-week" class="clickable">Week with debt</label><br>
                                <input style="" class="" type="date" id="debt-week" name="debt-week" value="" required>

                                <button type="submit" onclick="" id="generate-debt-button" style=""><img src="assets/plus.svg" class="btn_logo color-invert"><p2>Generate debt</p2></button>

                            </form>

                        </div>

                    </div>

                    <div class="prize-module" id="prize-module">

                        <div class="prize-form" id="prize-form">

                            <h3 id="prize-module-title">Prize:</h3>

                            <form action="" onsubmit="event.preventDefault(); add_prize();">
                                
                                <label for="prize-name" class="clickable">Name of prize</label><br>
                                <input style="" class="" type="text" id="prize-name" name="prize-name" value="" autocomplete="off" required>

                                <label for="prize-quantity" class="clickable">Quantity of prize</label><br>
                                <input style="" class="" type="number" id="prize-quantity" name="prize-quantity" value="1" min="1" required>

                                <button type="submit" onclick="" id="add-prize-button" style=""><img src="assets/done.svg" class="btn_logo color-invert"><p2>Add prize</p2></button>

                            </form>

                        </div>

                    </div>

                    <div class="add-season-module" id="add-season-module">

                        <div class="season-form" id="season-form">

                            <h3 id="season-module-title">Season:</h3>

                            <form action="" onsubmit="event.preventDefault(); add_season();">

                                <label for="season-start" class="clickable">Start of season (monday)</label><br>
                                <input style="" class="" type="date" id="season-start" name="season-start" value="" required>

                                <label for="season-end" class="clickable">End of season (sunday)</label><br>
                                <input style="" class="" type="date" id="season-end" name="season-end" value="" required>
                                
                                <input style="" class="" type="text" id="season-name" name="season-name" value="" placeholder="Name" autocomplete="off" required>
                                <input style="" class="" type="text" id="season-desc" name="season-desc" value="" placeholder="Description" autocomplete="off" required>

                                <label for="season-sickleave" class="clickable">Season sick leave</label><br>
                                <input style="" class="" type="number" id="season-sickleave" name="season-sickleave" value="0" min="0" max="99" placeholder="" autocomplete="off" required>

                                <label for="season-prize" class="clickable">Season prize</label><br>
                                <select style="" class="form-control" id="season-prize" name="season-prize" value="" required>
                                </select>

                                <button type="submit" onclick="" id="add-season-button" style=""><img src="assets/done.svg" class="btn_logo color-invert"><p2>Add season</p2></button>

                            </form>

                        </div>

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Ultimate power';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();

        if(!admin) {
            document.getElementById('content').innerHTML = "...";
            error("You are not an admin.")
        } else {
            get_server_info();
            get_invites();
            get_prizes();
        }

    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_server_info() {

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

                place_server_info(result.server)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/server-info");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_server_info(server_info) {
    document.getElementById('server-treningheten-version').innerHTML = server_info.treningheten_version
    document.getElementById('server-timezone').innerHTML = server_info.timezone
}

function get_invites() {

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

                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/invite/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_invites(invites_array) {
    var html = ``;
    
    if(invites_array.length == 0) {
        html = `
            <div id="" class="invitation-object">
                <p id="" style="margin: 0.5em; text-align: center;">...</p>
            </div>
        `;
    } else {
        for(var i = 0; i < invites_array.length; i++) {
            html += `
                <div id="" class="invitation-object">
                    <div class="leaderboard-object-code">
                        Code: ` + invites_array[i].invite_code + `
                    </div>
            `;

            if(invites_array[i].invite_used) {
                html += `
                        <div class="leaderboard-object-user">
                            Used by: ` + invites_array[i].user.first_name + ` ` + invites_array[i].user.last_name + `
                        </div>
                    `;
            } else {
                html += `
                        <div class="leaderboard-object-user">
                            Not used
                        </div>
                        <img class="icon-img clickable" onclick="delete_invite(` + invites_array[i].ID + `)" src="../../assets/trash-2.svg"></img>
                    `;
            }

            html += `</div>`;

        }
        
    }

    document.getElementById("invite-list").innerHTML = html

    return
}

function generate_invite() {

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

                success(result.message)
                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/invite/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function delete_invite(invide_id) {

    if(!confirm("Are you sure you want to delete this invite?")) {
        return
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

                success(result.message)
                place_invites(result.invites)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/invite/" + invide_id + "/delete");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function generate_debt() {

    if(!confirm("Are you sure you want to generate debt for the chosen week?")) {
        return
    }

    var debt_week = document.getElementById("debt-week").value;

    try {
        var debt_week_object = new Date(debt_week);
        var debt_week_string = debt_week_object.toISOString()
        console.log(debt_week_string)
    } catch(e) {
        error("Failed to parse date request.")
        console.log("Error: " + e)
        return;
    }

    var form_obj = { 
            "date" : debt_week_string
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

                success(result.message);
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/debt/generate");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function get_prizes() {

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

                place_prizes(result.prizes)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/prize");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_prizes(prizesArray) {

    var selectObject = document.getElementById("season-prize");

    for(var i = selectObject.options.length-1; i >= 0; i--) {
        selectObject.remove(i)
    }

    for(var i = 0; i < prizesArray.length; i++) {
        var option = document.createElement("option");
        option.text = prizesArray[i].quantity + " " + prizesArray[i].name;
        option.value = prizesArray[i].ID;
        selectObject.add(option); 
    }

}

function add_season() {

    clearResponse();

    var now = new Date;

    var season_start = document.getElementById("season-start").value;
    var season_end = document.getElementById("season-end").value;
    var season_start_string = "";
    var season_end_string = "";

    try {
        var season_prize_select = document.getElementById("season-prize");
        var season_prize = parseInt(season_prize_select[season_prize_select.selectedIndex].value);
    } catch(e) {
        console.log("Failed to parse prize. Error: " + e)
        error("Failed to parse prize.")
        return
    }

    var season_name = document.getElementById("season-name").value;
    var season_desc = document.getElementById("season-desc").value;
    var season_sickleave = parseInt(document.getElementById("season-sickleave").value);

    try {

        var season_start_object = new Date(season_start);
        season_start_string = season_start_object.toISOString()
        var season_end_object = new Date(season_end);
        season_end_string = season_end_object.toISOString()

        if(season_start_object.getDay() != 1) {
            console.log("Day: " + season_start_object.getDay())
            error("Season start must be a monday.");
            return;
        }

        if(season_end_object.getDay() != 0) {
            error("Season end must be a sunday.");
            return;
        }

        if(season_end_object < season_start_object) {
            error("Season start must be before season end.");
            return;
        }

        if(season_start_object < now) {
            error("Season start must later than now.");
            return;
        }
        
    } catch(e) {
        error("Failed to parse date object.")
        console.log("Error: " + e)
        return;
    }

    var form_obj = { 
        "start" : season_start_string,
        "end" : season_end_string,
        "name" : season_name,
        "description" : season_desc,
        "prize" : season_prize,
        "sickleave" : season_sickleave
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

                success(result.message);

                document.getElementById("season-name").value = "";
                document.getElementById("season-desc").value = "";
                document.getElementById("season-sickleave").value = 0;
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/season/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function add_prize() {

    var prize_name = document.getElementById("prize-name").value;
    var prize_quantity = parseInt(document.getElementById("prize-quantity").value);

    var form_obj = { 
        "name" : prize_name,
        "quantity" : prize_quantity
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

                success(result.message);

                document.getElementById("prize-name").value = "";
                document.getElementById("prize-quantity").value = "";

                get_prizes();
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "admin/prize/register");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}