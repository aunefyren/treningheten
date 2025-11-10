function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            var email = login_data.data.email
            var first_name = login_data.data.first_name
            var last_name = login_data.data.last_name
            var sunday_alert = login_data.data.sunday_alert
            user_id = login_data.data.id
            admin = login_data.data.admin
        } catch {
            var email = ""
            var first_name = ""
            var last_name = ""
            var sunday_alert = false
            user_id = 0
            admin = false
        }

        showAdminMenu(admin)

    } else {
        var email = ""
        var first_name = ""
        var last_name = ""
        var admin = false;
        var sunday_reminder = false
        user_id = 0
    }

    try {
        string_index = document.URL.lastIndexOf('/');
        requested_user_id = document.URL.substring(string_index+1);
    }
    catch {
        requested_user_id = 0
    }

    console.log("Wanted user: " + requested_user_id)

    var html = `

        <div class="module">

            <div class="user-active-profile-photo">
                <img style="width: 100%; height: 100%;" class="user-active-profile-photo-img" id="user-active-profile-photo-img" src="/assets/images/barbell.gif">
            </div>

            <b><p id="user_name" style="margin-top: 1em; font-size: 1.25em;"></p></b>
            <p id="join_date" style=""></p>
            <p id="user_admin" style=""></p>

            <div class="user-links" id="user-links" style="display:none; margin-top: 1em;">
            </div>

        </div>

        <div class="module" id="achievements-hr" style="display: none;">
            <hr>
        </div>

        <div class="module" id="achievements-module" style="display: none;">

            <div id="achievements-title" class="title" style="margin-bottom: 1em;">
                Achievements:
            </div>

            <div id="achievements-box" class="achievements-box">
            </div>

        </div>

    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'A real person.';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        GetUserData(requested_user_id);
        GetProfileImage(requested_user_id);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function GetProfileImage(userID) {

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

                PlaceProfileImage(result.image)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID + "/image");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImage(imageBase64) {

    document.getElementById("user-active-profile-photo-img").src = imageBase64

}

function GetUserData(userID) {

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

                PlaceUserData(result.user)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceUserData(user_object) {

    document.getElementById("user_name").innerHTML = user_object.first_name + " " + user_object.last_name

    // parse date object
    try {
        var date = new Date(Date.parse(user_object.created_at));
        var date_string = GetDateString(date)
    } catch {
        var date_string = "Error"
    }

    document.getElementById("join_date").innerHTML = "Joined: " + date_string

    if(user_object.admin) {
        var admin_string = "Yes"
    } else {
        var admin_string = "No"
    }

    if(user_object.strava_id && user_object.strava_public) {
        userLinks = document.getElementById("user-links")
        
        userLinks.style.display = "flex"

        userLinks.innerHTML += `
            <div onclick="window.open('https://www.strava.com/athletes/${user_object.strava_id}', '_blank');" class="clickable" style="width: 2em; height: 2em;" title="Strava profile">
                <img src="/assets/strava-logo.svg" style="" class="">
            </div>
        `;
    }

    document.getElementById("user_admin").innerHTML = "Administrator: " + admin_string

    GetUserAchievements(user_object.id);

}

function GetUserAchievements(userID) {

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

                PlaceUserAchievements(result.achievements)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/achievements?user=" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceUserAchievements(achievementArray) {

    if(achievementArray.length > 0) {
        document.getElementById("achievements-module").style.display = "flex"
        document.getElementById("achievements-hr").style.display = "flex"
    } else {
        return;
    }

    for(var i = 0; i < achievementArray.length; i++) {

        // parse date object
        try {
            var date = new Date(Date.parse(achievementArray[i].last_given_at));
            var date_string = GetDateString(date, false)
        } catch {
            var date_string = "Error"
        }

        var classString = ""
        for(var j = 0; j < achievementArray[i].achievement_delegations.length; j++) {
            if(!achievementArray[i].achievement_delegations[j].seen && achievementArray[i].achievement_delegations[j].user_id == user_id) {
                classString += " new-achievement"
            }
        }

        var categoryColor = `var(--${achievementArray[i].category_color})`;
        var categoryText = ""
        if(achievementArray[i].category !== "Default") {
            categoryText = `
            <div style="font-size: 0.70em; margin-bottom: 1em; border: solid 0.12rem; border-radius: 0.5em; width: auto; padding: 0.12rem 0.25rem;"> 
                ${achievementArray[i].category}
            </div>
            `;
        }

        var stackableHTML = ``
        if(achievementArray[i].multiple_delegations) {
            stackableHTML = `
                <div style="font-size: 0.70em; margin-top: 1em; border: solid 0.12rem; border-radius: 0.5em; width: auto; padding: 0.12rem 0.25rem;"> 
                    Stackable
                </div>
            `;
        }

        var delegationSum = achievementArray[i].achievement_delegations.length
        var delegationSumHTML = ``;

        if(delegationSum > 1) {
            delegationSumHTML = `
                <div class="achievement-delegation-sum">
                    <b>${delegationSum}</b>
                </div>
            `;
        }

        var html = `

        <div class="achievement unselectable" title="${achievementArray[i].description}" tabindex="1">

            <div class="achievement-base ${classString}">

                ${delegationSumHTML}

                <div class="achievement-image" style="border: solid 0.2em ${categoryColor};">
                    <img style="width: 100%; height: 100%;" class="achievement-img" id="achievement-img-${achievementArray[i].id}" src="/assets/images/barbell.gif">
                </div>

                <div class="achievement-title">
                    ${achievementArray[i].name}
                </div>

                <div class="achievement-date">
                    ${date_string}
                </div>     

            </div>     

            <div class="overlay">
                <div class="text-achievement"> 
                    ${categoryText}
                    <div style="margin-bottom: 0.5em;"> 
                        ${achievementArray[i].name}
                    </div>
                    <div style="" class="achievement-description"> 
                        ${achievementArray[i].description}
                    </div>
                    ${stackableHTML}
                </div>
            </div>

        </div>
        `;

        document.getElementById("achievements-box").innerHTML += html

        GetAchievementImage(achievementArray[i].id)

    }

}

function GetAchievementImage(achievementID) {

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

                PlaceAchievementImage(result.image, achievementID)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/achievements/" + achievementID + "/image?thumbnail=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceAchievementImage(imageBase64, achievementID) {

    document.getElementById("achievement-img-" + achievementID).src = imageBase64

}