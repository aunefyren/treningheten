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

            <div style="margin-top: 1em; display: flex; flex-direction: row; flex-wrap: wrap; align-content: center; justify-content: center; align-items: center;">
                <p id="first_name" style="margin: 0.25em"></p>
                <p id="last_name" style="margin: 0.25em"></p>
            </div>

            <p id="join_date" style="margin: 0.25em"></p>

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

    document.getElementById("first_name").innerHTML = user_object.first_name
    document.getElementById("last_name").innerHTML = user_object.last_name

    // parse date object
    try {
        var date = new Date(Date.parse(user_object.CreatedAt));
        var date_string = GetDateString(date)
    } catch {
        var date_string = "Error"
    }

    document.getElementById("join_date").innerHTML = "Joined: " + date_string

}