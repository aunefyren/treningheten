function load_page(result) {
    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            // Server data
            var vapid_public_key = login_data.vapid_public_key;
            var strava_client_id = login_data.strava_client_id;
            var strava_redirect_uri = login_data.strava_redirect_uri;
            var strava_enabled = login_data.strava_enabled;

            // Premade variables
            user_id = login_data.data.id
            admin = login_data.data.admin
            
        } catch {
            vapid_public_key = "";
            strava_client_id = "";
            strava_redirect_uri = "";
            strava_enabled = false;

            user_id = 0
            admin = false
        }

        showAdminMenu(admin)

    } else {
        vapid_public_key = "";
        strava_client_id = "";
        strava_redirect_uri = "";
        strava_enabled = false;

        user_id = 0
        admin = false
    }   

    var strava_oauth = `http://www.strava.com/oauth/authorize?client_id=${encodeURI(strava_client_id)}&response_type=code&redirect_uri=${encodeURI(strava_redirect_uri)}&approval_prompt=force&scope=activity:read_all`

    var html = `

        <div class="module">

            <div class="user-active-profile-photo">
                <img style="width: 100%; height: 100%;" class="user-active-profile-photo-img" id="user-active-profile-photo-img" src="/assets/images/barbell.gif">
            </div>

            <b><p id="user_name" style="margin-top: 1em; font-size: 1.25em;"></p></b>
            <p id="join_date" style=""></p>
            <p id="user_admin" style=""></p>

            <div class="account-section-wrapper">

                <div class="account-section" style="">

                    <div class="account-section-tab clickable" onclick="toggleSection('notifications-wrapper', 'section-button-notifications')">
                        <div class="">Device Notifications</div>
                        <img id="section-button-notifications"  src="assets/chevron-right.svg" class="color-invert" style="margin: 0.5em;">
                    </div>

                    <div id="notifications-wrapper" class="notifications-wrapper minimized">
                        <div class="notification-options" id="" style="">
                            <div class="notification-option" id="" style="">
                                <input style="" class="clickable" type="checkbox" id="notification-reminder-toggle" name="notification-reminder-toggle" value="">
                                <label for="notification-reminder-toggle" style="margin: 0;" class="clickable">Logging reminders</label><br>
                            </div>

                            <div class="notification-option" id="" style="">
                                <input style="" class="clickable" type="checkbox" id="notification-achievement-toggle" name="notification-achievement-toggle" value="">
                                <label for="notification-achievement-toggle" style="margin: 0;" class="clickable">Achievements</label><br>
                            </div>

                            <div class="notification-option" id="" style="">
                                <input style="" class="clickable" type="checkbox" id="notification-news-toggle" name="notification-news-toggle" value="">
                                <label for="notification-news-toggle" style="margin: 0;" class="clickable">News</label><br>
                            </div>
                        
                        </div>

                        <div id="notification_button_div" style="margin-top: 2em; display: flex; height: 3em; flex-direction: row; flex-wrap: nowrap; align-content: center; justify-content: center;align-items: center;">
                            <button type="submit" class="btn btn-primary" style="float: none !important;" id="" onclick="create_push('${vapid_public_key}'); return false;">
                                Notify me on this device
                            </button>
                        </div>
                    </div>
                </div>

                <div class="account-section" style="">

                    <div class="account-section-tab clickable" onclick="toggleSection('mail-notifications-wrapper', 'section-button-mail-notifications')">
                        <div class="">E-mail Notifications</div>
                        <img id="section-button-mail-notifications" src="assets/chevron-right.svg" class="color-invert" style="margin: 0.5em;">
                    </div>

                    <div id="mail-notifications-wrapper" class="mail-notifications-wrapper minimized">
                        
                        <div class="notification-options" id="" style="">

                            <div class="notification-option" id="" style="">
                                <input style="margin-top: 3em;" class="clickable" type="checkbox" id="sunday_alert" name="sunday_alert" value="sunday_alert" onchange="updateAccountValue('sunday_alert');">
                                <label for="sunday_alert" style="margin: 0;" class="clickable">Send me e-mail logging reminders on Sundays.</label><br>
                            </div>

                        </div>

                    </div>
                </div>

                <div class="account-section" style="">

                    <div class="account-section-tab clickable" style="" onclick="toggleSection('settings-wrapper', 'section-button-settings')">
                        <div class="">Account Settings</div>
                        <img id="section-button-settings" src="assets/chevron-right.svg" class="color-invert" style="margin: 0.5em;">
                    </div>
                    
                    <div id="settings-wrapper" class="settings-wrapper minimized">
                        <form action="" style="margin: 0;" onsubmit="event.preventDefault(); send_update('${user_id}');">

                            <label id="form-input-icon" for="email">Replace email:</label>
                            <input type="email" name="email" id="email" placeholder="Email" value="" required/>

                            <label id="form-input-icon" for="birth_date">Birth date:</label>
                            <input type="date" name="birth_date" id="birth_date" placeholder="dd-mm-yyyy" value="" />

                            <label id="form-input-icon" for="new_profile_image" style="margin-top: 2em;">Replace profile image:</label>
                            <input type="file" name="new_profile_image" id="new_profile_image" style="height:2.5em;" placeholder="" value="" accept="image/png, image/jpeg" />

                            <input onclick="change_password_toggle();" style="margin-top: 3em;" class="clickable" type="checkbox" id="password-toggle" name="confirm" value="confirm" >
                            <label for="password-toggle" style="margin: 0;" class="clickable">Change my password.</label><br>

                            <div id="change-password-box" style="display:none;">

                                <label id="form-input-icon" style="" for="password"></label>
                                <input type="password" name="password" id="password" placeholder="New password" />

                                <label id="form-input-icon" for="password_repeat"></label>
                                <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" />

                            </div>

                            <input style="margin-top: 3em;" class="clickable" type="checkbox" id="share_activities" name="share_activities" value="share_activities">
                            <label for="share_activities" style="margin: 0;" class="clickable">Share my activities on the activity feed.</label><br>

                            <label style="margin-top: 5em;" id="form-input-icon" for="password_old">Current password:</label>
                            <input type="password" name="password_old" id="password_old" placeholder="To save your changes, type your current password." required />

                            <button id="update-button" style="margin-top: 2em;" type="submit" href="/">Update account</button>

                        </form>
                    </div>

                </div>

                <div class="account-section" style="display: none;" id="strava-section">

                    <div class="account-section-tab clickable" style="" onclick="toggleSection('strava-wrapper', 'section-button-strava')">
                        <div class="">Strava</div>
                        <img id="section-button-strava" src="assets/chevron-right.svg" class="color-invert" style="margin: 0.5em;">
                    </div>
                    
                    <div id="strava-wrapper" class="strava-wrapper minimized">
                    </div>
                </div>


            </div>


            <div class="module" id="" style="">
                <hr>
            </div>

            <div class="button-collection">

                <button onclick="leave_season();"class="danger-button" style="" type="submit" href="">Leave season</button>

                <button onclick="delete_account();" class="danger-button" style="" type="submit" href="">Delete account</button>

            </div>

        </div>

    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Your very own page...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        GetUserData(user_id, strava_oauth, strava_enabled);
        GetProfileImage(user_id);
        CheckForSubscription();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function change_password_toggle() {

    var check_box = document.getElementById("password-toggle").checked;
    var password_box = document.getElementById("change-password-box")

    if(check_box) {
        password_box.style.display = "inline-block"
    } else {
        password_box.style.display = "none"
    }

}

function send_update(user_id) {

    var password = ""
    var password_repeat = ""
    
    if(document.getElementById("password-toggle").checked) {
        password = document.getElementById("password").value;
        password_repeat = document.getElementById("password_repeat").value;
    }
    
    var email = document.getElementById("email").value;
    var password_old = document.getElementById("password_old").value;
    var share_activities = document.getElementById("share_activities").checked;
    var new_profile_image = document.getElementById('new_profile_image').files[0];
    var birth_date = document.getElementById('birth_date').value;

    try {
        var birth_date_object = new Date(birth_date);
        var birth_date_string = birth_date_object.toISOString()
    } catch(e) {
        var birth_date_string = null
    }

    if(new_profile_image) {

        if(new_profile_image.size > 10000000) {
            error("Image exceeds 10MB size limit.")
            return;
        } else if(new_profile_image.size < 10000) {
            error("Image smaller than 0.01MB size requirement.")
            document.getElementById("password_old").value = "";
            return;
        }

        new_profile_image = get_base64(new_profile_image);
        
        new_profile_image.then(function(result) {
            
            var form_obj = { 
                "email" : email,
                "password" : password,
                "password_repeat": password_repeat,
                "profile_image": result,
                "password_old": password_old,
                "birth_date": birth_date_string,
                "share_activities": share_activities
            };

            var form_data = JSON.stringify(form_obj);

            document.getElementById("user-active-profile-photo-img").src = 'assets/images/barbell.gif';

            send_update_two(form_data, user_id);
        
        });

    } else {
        var form_obj = { 
            "email" : email,
            "password" : password,
            "password_repeat": password_repeat,
            "profile_image": "",
            "password_old": password_old,
            "birth_date": birth_date_string,
            "share_activities": share_activities
        };

        var form_data = JSON.stringify(form_obj);
        
        send_update_two(form_data, user_id);
    }
}

function send_update_two(form_data, user_id) {

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                document.getElementById("password_old").value = "";
                return;
            }
            
            if(result.error) {

                error(result.error);
                document.getElementById("password_old").value = "";

            } else {

                success(result.message);

                // store jwt to cookie
                set_cookie("treningheten", result.token, 7);
                
                if(result.verified) {
                    location.reload();
                } else {
                    location.href = '/';
                }
                
            }

        } else {
            info("Updating account...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/" + user_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

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
    xhttp.open("get", api_url + "auth/users/" + user_id + "/image");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

    return;

}

function PlaceProfileImage(imageBase64) {

    document.getElementById("user-active-profile-photo-img").src = imageBase64

}

function GetUserData(userID, stravaOauth, stravaEnabled) {

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

                PlaceUserData(result.user, stravaOauth, stravaEnabled)
                
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

function PlaceUserData(user_object, stravaOauth, stravaEnabled) {

    document.getElementById("user_name").innerHTML = user_object.first_name + " " + user_object.last_name
    document.getElementById("email").value = user_object.email

    if(user_object.birth_date != null) {
        var birth_date_object = new Date(Date.parse(user_object.birth_date))
        var birth_date = GetShortDate(birth_date_object)
        document.getElementById("birth_date").value = birth_date
    }

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

    document.getElementById("user_admin").innerHTML = "Administrator: " + admin_string

    if(user_object.sunday_alert) {
        document.getElementById("sunday_alert").checked = true;
    }

    if(user_object.share_activities) {
        document.getElementById("share_activities").checked = true;
    }

    if(stravaEnabled) {
        var stravaHTML = `
            <p style="width: 100%; text-align: center;">
                Strava exercises sync automatically every hour. Be careful to only log your sessions to either Strava or Treningheten.
            </p>

            <button onclick="window.location.href='${stravaOauth}';" class="" style="width: 10em;" type="submit" href="">Connect Strava</button>
        `;

        if(user_object.strava_code && user_object.strava_code != "") {
            walksHTML = "";
            padelHTML = "";
            publicHTML = "";

            if(user_object.strava_padel) {
                padelHTML = "checked"
            }
            if(user_object.strava_walks) {
                walksHTML = "checked"
            }
            if(user_object.strava_public) {
                publicHTML = "checked"
            }

            stravaHTML = `
                <p style="width: 100%; text-align: center;">
                    Strava exercises sync automatically every hour. Be careful to only log your sessions to either Strava or Treningheten.
                </p>

                <div class="notification-options" id="" style="">
                    <button onclick="window.location.href='${stravaOauth}';" class="" style="width: 12em;" type="submit" href="">Connect Strava again</button>
                    <button onclick="syncStrava('${user_object.id}');" class="" style="width: 12em;" type="submit" href="">Sync Strava now</button>
                </div>

                <div class="notification-options" id="" style="">
                    <div class="strava-option" id="" style="">
                        <input style="" class="clickable" type="checkbox" id="strava_padel" name="strava_padel" value="" onchange="updateAccountValue('strava_padel');" ${padelHTML}>
                        <label for="strava_padel" style="margin: 0;" class="clickable">Convert Pickleball to Padel</label><br>
                    </div>

                    <div class="strava-option" id="" style="">
                        <input style="" class="clickable" type="checkbox" id="strava_walks" name="strava_walks" value="" onchange="updateAccountValue('strava_walks');" ${walksHTML}>
                        <label for="strava_walks" style="margin: 0;" class="clickable">Ignore walks</label><br>
                    </div>

                    <div class="strava-option" id="" style="">
                        <input style="" class="clickable" type="checkbox" id="strava_public" name="strava_public" value="" onchange="updateAccountValue('strava_public');" ${publicHTML}>
                        <label for="strava_public" style="margin: 0;" class="clickable">Show my Strava on my profile</label><br>
                    </div>
                </div>
            `;
        }

        document.getElementById("strava-wrapper").innerHTML = stravaHTML
        document.getElementById('strava-section').style.display = 'flex'
    }
}

function leave_season() {
    alert("Doesn't work yet :(");
}

function delete_account() {
    alert("Doesn't work yet :(");
}

function PlaceSubscriptionData(subscription) {

    document.getElementById("notification-reminder-toggle").checked = subscription.sunday_alert;
    document.getElementById("notification-achievement-toggle").checked = subscription.achievement_alert;
    document.getElementById("notification-news-toggle").checked = subscription.news_alert;

}

function syncStrava(user_id) {
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
        } else {
            info("Syncing...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/" + user_id + "/strava-sync");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function toggleSection(divID, buttonID) {
    section = document.getElementById(divID)

    if(section.classList.contains("minimized")) {
        section.classList.remove("minimized")
        section.classList.add("expand")
        section.style.display = 'flex';
        document.getElementById(buttonID).src = "assets/chevron-down.svg"
    } else {
        section.classList.add("minimized")
        section.classList.remove("expand")
        section.style.display = 'none';
        document.getElementById(buttonID).src = "assets/chevron-right.svg"
        return
    }
}

function updateAccountValue(property) {
    var value = document.getElementById(property).checked;

    var form_obj = {};
    form_obj[property] = value;

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
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("PATCH", api_url + "auth/users/" + user_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
}