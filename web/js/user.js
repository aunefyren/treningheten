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

        </div>

        <div class="module color-invert" id="achievements-hr" style="display: none;">
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
    xhttp.open("post", api_url + "auth/user/get/" + userID + "/image");
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
    xhttp.open("post", api_url + "auth/user/get/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceUserData(user_object) {

    document.getElementById("user_name").innerHTML = user_object.first_name + " " + user_object.last_name

    // parse date object
    try {
        var date = new Date(Date.parse(user_object.CreatedAt));
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

    document.getElementById("user_admin").innerHTML = "Administrator: " + admin_string

    GetUserAhievements(user_object.ID);

}

function GetUserAhievements(userID) {

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

                PlaceUserAhievements(result.achivements)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/achievement/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceUserAhievements(achivementArray) {

    if(achivementArray.length > 0) {
        document.getElementById("achievements-module").style.display = "flex"
        document.getElementById("achievements-hr").style.display = "flex"
    } else {
        return;
    }

    for(var i = 0; i < achivementArray.length; i++) {

        // parse date object
        try {
            var date = new Date(Date.parse(achivementArray[i].given_at));
            var date_string = GetDateString(date, false)
        } catch {
            var date_string = "Error"
        }

        var html = `

        <div class="achievement" title="${achivementArray[i].description}" tabindex="1">

            <div class="achievement-base">

                <div class="achievement-image">
                    <img style="width: 100%; height: 100%;" class="achievement-img" id="achievement-img-${achivementArray[i].id}" src="/assets/images/barbell.gif">
                </div>

                <div class="achievement-title">
                    ${achivementArray[i].name}
                </div>

                <div class="achievement-date">
                    ${date_string}
                </div>     

            </div>     

            <div class="overlay">
                <div class="text-achievement">${achivementArray[i].description}</div>
            </div>

        </div>
        `;

        document.getElementById("achievements-box").innerHTML += html

        GetAchievementImage(achivementArray[i].id)

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
    xhttp.open("post", api_url + "auth/achievement/get/" + achievementID + "/image");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceAchievementImage(imageBase64, achievementID) {

    document.getElementById("achievement-img-" + achievementID).src = imageBase64

}