var api_url = window.location.origin + "/api/";

// Load service worker
if('serviceWorker' in navigator) {
    navigator.serviceWorker.register('service-worker.js')
    .then((reg) => {
        // registration worked
        console.log('Registration succeeded. Scope is ' + reg.scope);
    }
)};

// Make XHTTP requests
function makeRequest (method, url, data) {
    return new Promise(function (resolve, reject) {
    var xhr = new XMLHttpRequest();
    xhr.open(method, url);
    xhr.onload = function () {
      if (this.status >= 200 && this.status < 300) {
        resolve(xhr.response);
      } else {
        reject({
          status: this.status,
          statusText: xhr.statusText
        });
      }
    };
    xhr.onerror = function () {
      reject({
        status: this.status,
        statusText: xhr.statusText
      });
    };
    if(method=="POST" && data){
        xhr.send(data);
    }else{
        xhr.send();
    }
    });
}

// Set new browser cookie
function set_cookie(cname, cvalue, exdays) {
    var d = new Date();
    d.setTime(d.getTime() + (exdays*24*60*60*1000));
    var expires = "expires="+ d.toUTCString();
    document.cookie = cname + "=" + cvalue + "; " + expires + "; path=/; samesite=strict;";
}

// Get cookie from browser
function get_cookie(cname) {
    var name = cname + "=";
    var decodedCookie = decodeURIComponent(document.cookie);
    var ca = decodedCookie.split(';');
    for(var i = 0; i <ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == ' '){
            c = c.substring(1);
        }

        if (c.indexOf(name) == 0) {
            return c.substring(name.length, c.length);
        }
    }
    return "";
}

// Validate login token and get login details
function get_login(cookie) {

    if(jwt == "") {
        load_page(false);
        return
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

        if (this.readyState == 4) {

            // Try to parse API response
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error === "You must verify your account." && window.location.pathname !== "/verify") {
                verifyPageRedirect();
                return;
            } else if(result.error === "Failed to validate token.") {
                jwt = "";
                if(window.location.pathname !== "/login") {
                    logInPageRedirect();
                    return;
                } else {
                    load_page(false);
                }
            } else if (result.error) {
                error(result.error)
                showLoggedOutMenu();
                return;
                
            } else {

                // If new token, save it
                if(result.token != null && result.token != "") {
                    // store jwt to cookie
                    console.log("Refreshed login token.")
                    set_cookie("treningheten", result.token, 7);
                }

                // Load page
                load_page(this.responseText)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/tokens/validate");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", cookie);
    xhttp.send();
    return;
}

// Called when login session was rejected, showing no content and an error
function invalid_session() {
    showLoggedOutMenu();
    document.getElementById('content').innerHTML = `
    <div class="" id="front-page">
        <div class="module">
            <div class="text-body" id="front-page-text" style="text-align: center;">
                Log in to use the platform.
            </div>
            <div id="log-in-button" style="margin-top: 2em; display: flex; width: 10em;">
                <button id="update-button" type="submit" href="#" onclick="window.location = '/login';">Log in</button>
            </div>
        </div>
    </div>
    `;
    document.getElementById('card-header').innerHTML = 'Log in...';
    error('No access.');
}

// Call given URL, get image from API, call place_image() which is a local function, not here
function get_image(url, cookie, info, iteration) {

    if(iteration > 5) {
        return;
    } if (iteration =! null) {
        iteration++;
    }

    var json_jwt = JSON.stringify({});
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {
            var result;
            if(result = JSON.parse(this.responseText)) { 
                place_image(result.image, info);
            } else {
                place_image(false);
            }
        } else if(this.readyState == 4 && this.status == 404) {
            console.log('Image returned 404. Info: ' . info);

            setTimeout(() => {
                get_image(url, cookie, info, iteration);
            }, 5000);

        } else if(this.readyState == 4 && this.status !== 200 && this.status !== 404) {
            place_image(false);
        }
    };
    xhttp.withCredentials = false;
    xhttp.open("post", url);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", "Bearer " + cookie);
    xhttp.send(json_jwt);
    return;
}

/// receives file and returns Base64 string of file
function get_base64(file, onLoadCallback) {
    return new Promise(function(resolve, reject) {
        var reader = new FileReader();
        reader.onload = function() { resolve(reader.result); };
        reader.onerror = reject;
        reader.readAsDataURL(file);
    });
}

// Show options that can be access when logged in
function showLoggedInMenu() {
    document.getElementById('login').classList.add('disabled');
    document.getElementById('login').classList.remove('enabled');

    document.getElementById('logout').classList.add('enabled');
    document.getElementById('logout').classList.remove('disabled');

    document.getElementById('news').classList.add('enabled');
    document.getElementById('news').classList.remove('disabled');

    document.getElementById('seasons').classList.add('enabled');
    document.getElementById('seasons').classList.remove('disabled');

    document.getElementById('exercises-tab').classList.add('enabled');
    document.getElementById('exercises-tab').classList.remove('disabled');

    document.getElementById('achievements-tab').classList.add('enabled');
    document.getElementById('achievements-tab').classList.remove('disabled');

    document.getElementById('account').classList.add('enabled');
    document.getElementById('account').classList.remove('disabled');

    document.getElementById('register').classList.add('disabled');
    document.getElementById('register').classList.remove('enabled');
}

// Remove options not accessable when not logged in
function showLoggedOutMenu() {
    document.getElementById('login').classList.add('enabled');
    document.getElementById('login').classList.remove('disabled');

    document.getElementById('logout').classList.add('disabled');
    document.getElementById('logout').classList.remove('enabled');

    document.getElementById('news').classList.add('disabled');
    document.getElementById('news').classList.remove('enabled');

    document.getElementById('seasons').classList.add('disabled');
    document.getElementById('seasons').classList.remove('enabled');

    document.getElementById('exercises-tab').classList.add('disabled');
    document.getElementById('exercises-tab').classList.remove('enabled');

    document.getElementById('achievements-tab').classList.add('disabled');
    document.getElementById('achievements-tab').classList.remove('enabled');

    document.getElementById('account').classList.add('disabled');
    document.getElementById('account').classList.remove('enabled');

    document.getElementById('register').classList.add('enabled');
    document.getElementById('register').classList.remove('disabled');
}

function showAdminMenu(admin) {
    if(admin) {
        document.getElementById('admin').classList.add('enabled');
        document.getElementById('admin').classList.remove('disabled');
    } else {
        document.getElementById('admin').classList.add('disabled');
        document.getElementById('admin').classList.remove('enabled');
    }
}

// Toggle navar expansion
function toggle_navbar() {
    var x = document.getElementById("navbar");
    var y = document.getElementById("nav-logo");
    if (!x.classList.contains("responsive")) {
        x.classList.add("responsive");
        x.classList.add("responsive");
        x.classList.remove("unresponsive");
        x.classList.remove("unresponsive");

        var cols = document.getElementsByClassName('nav-item');
        for(i = 0; i < cols.length; i++) {
            cols[i].style.backgroundColor = 'var(--lightblue)';
        }
    } else {
        x.classList.add("unresponsive");
        x.classList.add("unresponsive");
        x.classList.remove("responsive");
        x.classList.remove("responsive");

        var cols = document.getElementsByClassName('nav-item');
        for(i = 0; i < cols.length; i++) {
            cols[i].style.backgroundColor = 'none';
        }
    }
}

// Toggle navbar if clicked outside
document.addEventListener('click', function(event) {
    var isClickInsideElement = ignoreNav.contains(event.target);
    if (!isClickInsideElement) {
        var nav_classlist = document.getElementById('navbar').classList;
        if (nav_classlist.contains('responsive')) {
            toggle_navbar();
        }
    }
});

// Function for checking file extension of file
function return_file_extension(filename) {
    return (/[.]/.exec(filename)) ? /[^.]+$/.exec(filename) : undefined;
}

// Removes notification from response bar
function clearResponse(){
    document.getElementById("response").innerHTML = '';
}

// Displays a blue notification
function info(message) {
    document.getElementById("response").innerHTML = "<div class='alert alert-info'>" + message + "</div>";
    window.scrollTo(0, 0);
}

// Displays a green notification
function success(message) {
    document.getElementById("response").innerHTML = "<div class='alert alert-success'>" + message + "</div>";
    window.scrollTo(0, 0);
}

// Displays a red notification
function error(message) {
    document.getElementById("response").innerHTML = "<div class='alert alert-danger'>" + message + "</div>";
    window.scrollTo(0, 0);
}

// When log out button is pressed, remove cookie and redirect to home page
function logout() {
    set_cookie("treningheten", "", 1);
    window.location.href = '/';
}

// Return GET parameters in a given URL
function get_url_parameters(url) {
    
    const parameters = {}
    try {
        let paramString = url.split('?')[1];
        let params_arr = paramString.split('&');
        for(let i = 0; i < params_arr.length; i++) {
            let pair = params_arr[i].split('=');
            parameters[pair[0]] = pair[1];
        }
    }
    catch {
        return false
    }

    return parameters;
}

function trigger_fireworks(number) {
    if(number > 0) {
        document.getElementById('pyro').style.display = 'block';
        setTimeout(function () {
            trigger_fireworks(number-1);
        }, 5000);
    } else {
        document.getElementById('pyro').style.display = 'none';
    }
}

function error_splash_image() {
    try {
        var html = `
            <div class="module">
                <img src="/assets/images/barbell.gif">
            </div>
        `;

        document.getElementById("content").innerHTML = html;
    } catch(e) {
        console.log("Failed to add splash error. Error: " + e)
    }
}

function padNumber(num, size) {
    var s = "000000000" + num;
    return s.substr(s.length-size);
}

function gcdOfArray(input) {
    if (toString.call(input) !== "[object Array]")  
        return  false;  
    var len, a, b;
    len = input.length;
    if ( !len ) {
        return null;
    }
    a = input[ 0 ];
    for ( var i = 1; i < len; i++ ) {
        b = input[ i ];
        a = gcdOfTwoNumbers( a, b );
    }
    return a;
}

function gcdOfTwoNumbers(x, y) {
    if ((typeof x !== 'number') || (typeof y !== 'number')) 
      return false;
    x = Math.abs(x);
    y = Math.abs(y);
    while(y) {
      var t = y;
      y = x % y;
      x = t;
    }
    return x;
}

function GetDateString(dateTime, giveWeekday) {

    try {

        var weekDayArray = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"]
        var monthArray = ["January", "February", "March", "April", "May", "June", "July", "August", "September", "October", "November", "December"]
        var weekDay = "";
        var month = "";
        var day = "";
        var year = "";

        var weekDayInt = dateTime.getDay();
        var monthInt = dateTime.getMonth();
        var dayInt = dateTime.getDate();
        var yearInt = dateTime.getYear();

        weekDay = weekDayArray[weekDayInt]
        month = monthArray[monthInt]
        day = padNumber(dayInt, 2)

        if(yearInt >= 100) {
            year = yearInt + 1900
        } else {
            year = 1900 + yearInt
        }

        if(giveWeekday) {
            return weekDay + ", " + day + ". " + month + ", " + year;
        } else {
            return day + ". " + month + ", " + year;
        }

    } catch(e) {
        console.log("Failed to generate string for date time. Error: " + e)
        return "Error"
    }

}

function GetDayOfTheWeek(dateTime) {

    try {

        var weekDayArray = ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"]
        var weekDayInt = dateTime.getDay();
        weekDay = weekDayArray[weekDayInt]

        return weekDay

    } catch(e) {
        console.log("Failed to generate string for date time. Error: " + e)
        return "Error"
    }

}

function GetShortDate(dateTime) {

    try {

        var month = "";
        var day = "";
        var year = "";

        var monthInt = dateTime.getMonth()+1;
        var dayInt = dateTime.getDate();
        var yearInt = dateTime.getYear();

        day = padNumber(dayInt, 2)
        month = padNumber(monthInt, 2)

        if(yearInt >= 100) {
            year = yearInt + 1900
        } else {
            year = 1900 + yearInt
        }

        return year + "-" + month + "-" + day;


    } catch(e) {
        console.log("Failed to generate string for date time. Error: " + e)
        return "Error"
    }

}

/**
 * Returns the week number for this date.  dowOffset is the day of week the week
 * "starts" on for your locale - it can be from 0 to 6. If dowOffset is 1 (Monday),
 * the week returned is the ISO 8601 week number.
 * @param int dowOffset
 * @return int
 */
Date.prototype.getWeek = function (dowOffset) {
    /*getWeek() was developed by Nick Baicoianu at MeanFreePath: http://www.meanfreepath.com */
    
    dowOffset = typeof(dowOffset) == 'number' ? dowOffset : 0; //default dowOffset to zero
    var newYear = new Date(this.getFullYear(),0,1);
    var day = newYear.getDay() - dowOffset; //the day of week the year begins on
    day = (day >= 0 ? day : day + 7);
    var daynum = Math.floor((this.getTime() - newYear.getTime() - 
    (this.getTimezoneOffset()-newYear.getTimezoneOffset())*60000)/86400000) + 1;
    var weeknum;
    //if the year starts before the middle of a week
    if(day < 4) {
        weeknum = Math.floor((daynum+day-1)/7) + 1;
        if(weeknum > 52) {
            nYear = new Date(this.getFullYear() + 1,0,1);
            nday = nYear.getDay() - dowOffset;
            nday = nday >= 0 ? nday : nday + 7;
            /*if the next year starts before the middle of
                the week, it is week #1 of that year*/
            weeknum = nday < 4 ? 1 : 53;
        }
    }
    else {
        weeknum = Math.floor((daynum+day-1)/7);
    }
    return weeknum;
};

Date.prototype.addDays = function(days) {
    var date = new Date(this.valueOf());
    date.setDate(date.getDate() + days);
    return date;
}

function HTMLDecode(text) {
    var txt = document.createElement("textarea");
    txt.innerHTML = text;
    return txt.value
}

function HTMLAddNewLines(text) {
    return text.replace('\n', "<br>")
}

function IncreaseNumberInput(input_id, min, max) {
    var input_element = document.getElementById(input_id)
    var old_number = Number(input_element.innerHTML)
    var new_number = old_number + 1
    if(new_number <= max && new_number >= min) {
        input_element.innerHTML = new_number
    }
}

function DecreaseNumberInput(input_id, min, max) {
    var input_element = document.getElementById(input_id)
    var old_number = Number(input_element.innerHTML)
    var new_number = old_number - 1
    if(new_number <= max && new_number >= min) {
        input_element.innerHTML = new_number
    }
}

function getDebtOverview() {

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

                if(result.overview.debt_lost.length > 0) {

                    placeDebtSpin(result.overview);

                } else if(result.overview.debt_unviewed.length > 0 || result.overview.debt_won.length > 0 || result.overview.debt_unpaid.length > 0) {

                    placeDebtOverview(result.overview);

                } else {

                    document.getElementById("debt-module").style.display = "none";

                }

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/debts");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function setPrizeReceived(debt_id) {

    if(!confirm("Are you sure?")) {
        return;
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

                getDebtOverview();

            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/debts/" + debt_id + "/received");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function weeksBetween(dt2, dt1) {

    var startYear = dt2.getFullYear()
    var startWeek = dt2.getWeek(1)
    const endYear = dt1.getFullYear()
    const endWeek = dt1.getWeek(1)

    let weeksBetween = 1;

    while(startYear < endYear || startWeek < endWeek) {
        dt2 = dt2.addDays(7);

        startYear = dt2.getFullYear()
        startWeek = dt2.getWeek(1)

        weeksBetween += 1;
    }

    return weeksBetween;
}

function secondsToDurationString(originalSeconds) {
    var hourString = '';
    var minutes = originalSeconds
    var hours = Math.floor(originalSeconds / 3600)
    if(hours != 0) {
        hourString = padNumber(hours, 2) + ":"
        minutes = originalSeconds % 3600
    }

    var minutesString = padNumber(Math.floor(minutes / 60), 2)
    var secondsString = padNumber((originalSeconds % 60), 2)
    var time = hourString + minutesString + ':' + secondsString

    return time
}

function parseDurationStringToSeconds(duration) {
    timeFinal = null
    try {
        if(duration.includes(':')) {
            timeArray = duration.split(':')
            
            if(timeArray.length == 2) {
                var minutes = parseFloat(timeArray[0])
                var seconds = parseFloat(timeArray[1])
                timeFinal = (minutes * 60) + seconds
            } else if (timeArray.length == 3){
                var hours = parseFloat(timeArray[0])
                var minutes = parseFloat(timeArray[1])
                var seconds = parseFloat(timeArray[2])
                timeFinal = (hours * 3600) + (minutes * 60) + seconds
            }

        } else if(duration != ''){
            timeFinal = parseFloat(duration)
        }
    } catch (e) {
        console.log("Failed to parse time. Error: " + e)
    }
    return timeFinal
}

function showSelectDropdown(operationID, toggle) {
    if(toggle) {
        document.getElementById("operation-action-text-list-" + operationID).style.display = "flex";
    } else {
        document.getElementById("operation-action-text-list-" + operationID).style.display = "none";
    }
    filterFunction(operationID);
}

function filterFunction(operationID) {
    var input, filter, ul, li, a, i;
    input = document.getElementById("operation-action-text-" + operationID);
    filter = input.value.toUpperCase();
    div = document.getElementById("operation-action-text-list-" + operationID);
    a = div.getElementsByTagName("a");

    for (i = 0; i < a.length; i++) {
        txtValue = a[i].textContent || a[i].innerText;
        if (txtValue.toUpperCase().indexOf(filter) > -1) {
            a[i].style.display = "";
        } else {
            a[i].style.display = "none";
        }
    }
    toggleActionBorder(operationID, 'none');
}

function logInPageRedirect() {
    if(window.location.pathname !== "/login") {
        window.location = '/login';
        return true
    }
    return false
}

function verifyPageRedirect() {
    if(window.location.pathname !== "/verify") {
        window.location = '/verify';
        return true
    }
    return false
}