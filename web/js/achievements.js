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
                            These are the achievements you can win.
                        </div>

                    </div>

                    <div class="module" id="achievements-module" style="display: none;">

                        <div id="achievements-title" class="title" style="margin-bottom: 1em;">
                            Achievements:
                        </div>

                        <div id="achievements-box" class="achievements-box">
                        </div>

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Hall of fame.';
    clearResponse();

    if(result !== false) {

        showLoggedInMenu();
        GetAllAchievements(user_id);

    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function GetAllAchievements(userID) {

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

                GetUserAhievements(result.achivements, userID)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/achievement/");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function GetUserAhievements(achivementArray, userID) {

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

                PlaceUserAhievements(result.achivements, achivementArray, userID)
                
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

function PlaceUserAhievements(achivementArrayPersonal, achivementArray, userID) {

    if(achivementArray.length > 0) {
        document.getElementById("achievements-module").style.display = "flex"
    } else {
        return;
    }

    for(var i = 0; i < achivementArray.length; i++) {

        var achieved = false;
        var achievedIndex = 0;
        for(var j = 0; j < achivementArrayPersonal.length; j++) {
            if(achivementArrayPersonal[j].id == achivementArray[i].ID) {
                achieved = true;
                achievedIndex = j;
                break;
            }
        }

        if(achieved) {
            // parse date object
            try {
                var date = new Date(Date.parse(achivementArrayPersonal[achievedIndex].given_at));
                var date_string = GetDateString(date, false)
            } catch {
                var date_string = "Error"
            }
            var date_string_html = date_string
            var class_string_html = ""
        } else {
            var date_string_html = "Locked";
            var class_string_html = "transparent"
        }

        var html = `

        <div class="achievement unselectable" title="${achivementArray[i].description}" tabindex="1">

            <div class="achievement-base ${class_string_html}">

                <div class="achievement-image">
                    <img style="width: 100%; height: 100%;" class="achievement-img" id="achievement-img-${achivementArray[i].ID}" src="/assets/lock.svg">
                </div>

                <div class="achievement-title">
                    ${achivementArray[i].name}
                </div>

                <div class="achievement-date">
                    ${date_string_html}
                </div>

            </div>

            <div class="overlay">
                <div class="text-achievement">${achivementArray[i].description}</div>
            </div>

        </div>
        `;

        document.getElementById("achievements-box").innerHTML += html

        if(achieved) {
            GetAchievementImage(achivementArray[i].ID)
        } else {
            document.getElementById("achievement-img-" + achivementArray[i].ID).style.padding  = "1.5em"
            document.getElementById("achievement-img-" + achivementArray[i].ID).style.borderRadius  = "0"
        }

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