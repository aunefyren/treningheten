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
                            These are the achievements you can win. Most achievements are given the week after, season-based are given when the season is over.
                        </div>

                    </div>

                    <div class="module">
                        Progress:
                        <div class="progress-bar-number" id="progress-bar-number" style="margin-top:0.5em;"></div>
                        <div class="progress-bar-wrapper">
                            <div id="progress-bar" class="progress-bar" style=""></div>
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

                GetUserAchievements(result.achievements, userID)
                
            }

        } else {
            // info("Loading week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/achievements");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function GetUserAchievements(achievementArray, userID) {

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

                PlaceUserAchievements(result.achievements, achievementArray, userID)
                
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

function PlaceUserAchievements(achievementArrayPersonal, achievementArray, userID) {

    if(achievementArray.length > 0) {
        document.getElementById("achievements-module").style.display = "flex"
    } else {
        return;
    }

    var achieved_sum = 0;
    var achievement_sum = 0;

    for(var i = 0; i < achievementArray.length; i++) {

        achievement_sum += 1

        var achieved = false;
        var achievedIndex = 0;
        for(var j = 0; j < achievementArrayPersonal.length; j++) {
            if(achievementArrayPersonal[j].id == achievementArray[i].id) {
                achieved = true;
                achievedIndex = j;
                break;
            }
        }

        var categoryColor = `var(--${achievementArray[i].category_color})`;
        var categoryText = ""
        if(achievementArray[i].category !== "Default") {
            categoryText = `
            <div style="font-size: 0.70em; margin-bottom: 1em;"> 
                ${achievementArray[i].category}
            </div>
            `;
        }

        if(achieved) {

            achieved_sum += 1

            // parse date object
            try {
                var date = new Date(Date.parse(achievementArrayPersonal[achievedIndex].achievement_delegation.given_at));
                var date_string = GetDateString(date, false)
            } catch {
                var date_string = "Error"
            }
            var date_string_html = date_string
            var class_string_html = ""

            if(!achievementArrayPersonal[achievedIndex].achievement_delegation.seen){
                class_string_html += " new-achievement"
            }
        } else {
            var date_string_html = "Locked";
            var class_string_html = "transparent"
        }

        var html = `

        <div class="achievement unselectable" title="${achievementArray[i].description}" tabindex="1">

            <div class="achievement-base ${class_string_html}">

                <div class="achievement-image" style="border: solid 0.2em ${categoryColor};">
                    <img style="width: 100%; height: 100%; padding: 1.5em; border-radius: 0;" class="achievement-img" id="achievement-img-${achievementArray[i].id}" src="/assets/images/barbell.gif">
                </div>

                <div class="achievement-title">
                    ${achievementArray[i].name}
                </div>

                <div class="achievement-date">
                    ${date_string_html}
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
                </div>
            </div>

        </div>
        `;

        document.getElementById("achievements-box").innerHTML += html

        if(achieved) {
            document.getElementById("achievement-img-" + achievementArray[i].id).style.padding  = "0"
            document.getElementById("achievement-img-" + achievementArray[i].id).style.borderRadius  = "10em"
            GetAchievementImage(achievementArray[i].id)
        } else {
            document.getElementById("achievement-img-" + achievementArray[i].id).src  = "/assets/lock.svg"
        }

    }

    var ach_percentage = Math.floor((achieved_sum / achievement_sum) * 100)
    console.log(ach_percentage)
    document.getElementById("progress-bar").style.width  = ach_percentage + "%"
    document.getElementById("progress-bar").title  = ach_percentage + "%"
    document.getElementById("progress-bar-number").innerHTML  = achieved_sum + "/" + achievement_sum
    
    if(ach_percentage > 99) {
        setTimeout(function() {
            document.getElementById("progress-bar").classList.add("blink")
        }, 1500);
        setTimeout(function() {
            document.getElementById("progress-bar").classList.remove('blink');
            //document.getElementById("progress-bar").style.backgroundColor  = "var(--lightgreen)"
        }, 2500);
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
    xhttp.open("get", api_url + "auth/achievements/" + achievementID + "/image");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceAchievementImage(imageBase64, achievementID) {

    document.getElementById("achievement-img-" + achievementID).src = imageBase64

}