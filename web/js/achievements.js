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
                            These are the achievements you can win. Most achievements are given the week after, season-based are given when the season is over.
                        </div>

                    </div>

                    <div class="module">
                        Progress:
                        <div class="progress-bar-number u-mt-sm" id="progress-bar-number"></div>
                        <div class="progress-bar-wrapper">
                            <div id="progress-bar" class="progress-bar"></div>
                        </div>
                    </div>

                    <div class="module" id="achievements-module" style="display: none;">

                        <div id="achievements-title" class="title u-mb-1">
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
            categoryText = `<div class="meta-tag u-mb-1">${achievementArray[i].category}</div>`;
        }

        var stackableHTML = ``
        if(achievementArray[i].multiple_delegations) {
            stackableHTML = `<div class="meta-tag u-mt-1">Stackable</div>`;
        }

        alternativeDescription = achievementArray[i].description
        if(achieved) {
            achieved_sum += 1

            try {
                alternativeDescription = achievementArrayPersonal[achievedIndex].description
            } catch (error) {
                console.log("Failed to change achievement description. Error: " + e)
            }

            // parse date object
            try {
                var date = new Date(Date.parse(achievementArrayPersonal[achievedIndex].last_given_at));
                var date_string = GetDateString(date, false)
            } catch {
                var date_string = "Error"
            }
            var date_string_html = date_string
            var class_string_html = ""

            for(var j = 0; j < achievementArrayPersonal[achievedIndex].achievement_delegations.length; j++) {
                if(!achievementArrayPersonal[achievedIndex].achievement_delegations[j].seen){
                    class_string_html += " new-achievement"
                    break;
                }
            }

            var delegationSum = achievementArrayPersonal[achievedIndex].achievement_delegations.length
            var delegationSumHTML = ``;

            if(delegationSum > 1) {
                delegationSumHTML = `
                    <div class="achievement-delegation-sum">
                        <b>${delegationSum}</b>
                    </div>
                `;
            }
        } else {
            var date_string_html = "Locked";
            var class_string_html = "transparent"
            var delegationSumHTML = ``
        }

        var html = `

        <div class="achievement unselectable" title="${achievementArray[i].description}" tabindex="1">

            <div class="achievement-base ${class_string_html}">

                ${delegationSumHTML}

                <div class="achievement-image" style="--cat-color: ${categoryColor};">
                    <img class="achievement-img achievement-img-logo" id="achievement-img-${achievementArray[i].id}" src="/assets/images/barbell.gif">
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
                    <div class="u-mb-2"> 
                        ${achievementArray[i].name}
                    </div>
                    <div class="achievement-description" id="achievement-description-${achievementArray[i].id}"> 
                        ${achievementArray[i].description}
                    </div>
                    ${stackableHTML}
                </div>
            </div>

        </div>
        `;

        document.getElementById("achievements-box").innerHTML += html

        if(achieved) {
            var achImg = document.getElementById("achievement-img-" + achievementArray[i].id);
            achImg.style.padding = "0"
            achImg.style.borderRadius = "10em"
            document.getElementById("achievement-description-" + achievementArray[i].id).innerHTML = alternativeDescription
            achImg.onerror = function() { this.onerror = null; this.src = '/assets/images/barbell.gif'; };
            achImg.src = achievementImageURL(achievementArray[i].id, true)
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

