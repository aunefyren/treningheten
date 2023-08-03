var api_url = window.location.origin + "/api/";

// Load service worker
if('serviceWorker' in navigator) {
    navigator.serviceWorker.register('/js/service-worker.js')
.then((reg) => {
    // registration worked
    console.log('Registration succeeded. Scope is ' + reg.scope);
})};

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
    document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/;samesite=strict";
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

            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            
            if(result.error === "You must verify your account.") {

                load_page(this.responseText)

            } else if (result.error) {
                
                error(result.error)
                showLoggedInMenu();
                return;
                
            } else {

                load_page(this.responseText)
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/token/validate");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", cookie);
    xhttp.send();
    return;
}

// Called when login session was rejected, showing no content and an error
function invalid_session() {
    showLoggedOutMenu();
    document.getElementById('content').innerHTML = '';
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
    } else {
        x.classList.add("unresponsive");
        x.classList.add("unresponsive");
        x.classList.remove("responsive");
        x.classList.remove("responsive");
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
    window.location.href = '../../';
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

Date.prototype.GetWeek = function() {
    var onejan = new Date(this.getFullYear(), 0, 1);
    return Math.ceil((((this - onejan) / 86400000) + onejan.getDay() + 1) / 7);
}

function error_splash_image() {
    try {
        var html = `
            <div class="module">
                <img src="./assets/images/barbell.gif">
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