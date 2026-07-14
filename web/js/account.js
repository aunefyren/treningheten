function load_page(result) {
    if(result !== false) {
        var login_data = JSON.parse(result);

        try {
            // Server data
            var vapid_public_key = login_data.vapid_public_key;
            var strava_client_id = login_data.strava_client_id;
            var strava_redirect_uri = login_data.strava_redirect_uri;
            var strava_enabled = login_data.strava_enabled;
            var hevy_enabled = login_data.hevy_enabled;
            plex_enabled = login_data.plex_enabled;
            spotify_enabled = login_data.spotify_enabled;
            spotify_client_id = login_data.spotify_client_id;
            spotify_redirect_uri = login_data.spotify_redirect_uri;
            audiobookshelf_enabled = login_data.audiobookshelf_enabled;

            // Premade variables
            user_id = login_data.data.id
            admin = login_data.data.admin

        } catch {
            vapid_public_key = "";
            strava_client_id = "";
            strava_redirect_uri = "";
            strava_enabled = false;
            hevy_enabled = false;
            plex_enabled = false;
            spotify_enabled = false;
            spotify_client_id = "";
            spotify_redirect_uri = "";
            audiobookshelf_enabled = false;

            user_id = 0
            admin = false
        }

        showAdminMenu(admin)

    } else {
        vapid_public_key = "";
        strava_client_id = "";
        strava_redirect_uri = "";
        strava_enabled = false;
        hevy_enabled = false;
        plex_enabled = false;
        spotify_enabled = false;
        spotify_client_id = "";
        spotify_redirect_uri = "";
        audiobookshelf_enabled = false;

        user_id = 0
        admin = false
    }

    var strava_oauth = `http://www.strava.com/oauth/authorize?client_id=${encodeURI(strava_client_id)}&response_type=code&redirect_uri=${encodeURI(strava_redirect_uri)}&approval_prompt=force&scope=activity:read_all`
    spotify_oauth = `https://accounts.spotify.com/authorize?client_id=${encodeURIComponent(spotify_client_id)}&response_type=code&redirect_uri=${encodeURIComponent(spotify_redirect_uri)}&scope=${encodeURIComponent('user-read-recently-played')}&state=spotify`

    var html = `

        <div class="module">

            <div class="user-active-profile-photo">
                <img class="user-active-profile-photo-img u-fill" id="user-active-profile-photo-img" src="/assets/images/barbell.gif">
            </div>

            <b><p id="user_name" style="margin-top: 1em; font-size: 1.25em;"></p></b>
            <p id="join_date"></p>
            <p id="user_admin"></p>

            <div class="button-collection">
                <button onclick="window.location.href = '/users/${user_id}';"class="btn" type="submit" href="">Public profile</button>
                <button onclick="window.location.href = '/gear';" class="btn" type="submit" href="">Manage gear</button>
            </div>

            <div class="account-section-wrapper">

                <div class="account-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('notifications-wrapper', 'section-button-notifications')">
                        <div class="">Device Notifications</div>
                        <img id="section-button-notifications"  src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="notifications-wrapper" class="notifications-wrapper minimized">
                        <div class="notification-options" id="">
                            <div class="notification-option" id="">
                                <input class="clickable" type="checkbox" id="notification-reminder-toggle" name="notification-reminder-toggle" value="">
                                <label for="notification-reminder-toggle" class="clickable u-m-0">Logging reminders</label><br>
                            </div>

                            <div class="notification-option" id="">
                                <input class="clickable" type="checkbox" id="notification-achievement-toggle" name="notification-achievement-toggle" value="">
                                <label for="notification-achievement-toggle" class="clickable u-m-0">Achievements</label><br>
                            </div>

                            <div class="notification-option" id="">
                                <input class="clickable" type="checkbox" id="notification-news-toggle" name="notification-news-toggle" value="">
                                <label for="notification-news-toggle" class="clickable u-m-0">News</label><br>
                            </div>
                        
                        </div>

                        <div id="notification_button_div" style="margin-top: 2em; display: flex; height: 3em; flex-direction: row; flex-wrap: nowrap; align-content: center; justify-content: center;align-items: center;">
                            <button type="submit" class="btn btn--primary" style="float: none !important;" id="" onclick="create_push('${vapid_public_key}'); return false;">
                                Notify me on this device
                            </button>
                        </div>
                    </div>
                </div>

                <div class="account-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('mail-notifications-wrapper', 'section-button-mail-notifications')">
                        <div class="">E-mail Notifications</div>
                        <img id="section-button-mail-notifications" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="mail-notifications-wrapper" class="mail-notifications-wrapper minimized">
                        
                        <div class="notification-options" id="">

                            <div class="notification-option" id="">
                                <input class="clickable u-mt-3" type="checkbox" id="sunday_alert" name="sunday_alert" value="sunday_alert" onchange="updateAccountValue('sunday_alert');">
                                <label for="sunday_alert" class="clickable u-m-0">Send me e-mail logging reminders on Sundays.</label><br>
                            </div>

                        </div>

                    </div>
                </div>

                <div class="account-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('settings-wrapper', 'section-button-settings')">
                        <div class="">Account Settings</div>
                        <img id="section-button-settings" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>
                    
                    <div id="settings-wrapper" class="settings-wrapper minimized">
                        <form action="" style="margin: 0;" onsubmit="event.preventDefault(); send_update('${user_id}');">

                            <label id="form-input-icon" for="email">Replace email:</label>
                            <input type="email" name="email" id="email" placeholder="Email" value="" required/>

                            <label id="form-input-icon" for="birth_date">Birth date:</label>
                            <input type="date" name="birth_date" id="birth_date" placeholder="dd-mm-yyyy" value="" />

                            <label id="form-input-icon" for="new_profile_image" class="u-mt-2">Replace profile image:</label>
                            <input type="file" name="new_profile_image" id="new_profile_image" style="height:2.5em;" placeholder="" value="" accept="image/png, image/jpeg" />

                            <input onclick="change_password_toggle();" class="clickable u-mt-3" type="checkbox" id="password-toggle" name="confirm" value="confirm" >
                            <label for="password-toggle" class="clickable u-m-0">Change my password.</label><br>

                            <div id="change-password-box" style="display:none;">

                                <label id="form-input-icon" for="password"></label>
                                <input type="password" name="password" id="password" placeholder="New password" />

                                <label id="form-input-icon" for="password_repeat"></label>
                                <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" />

                            </div>

                            <input class="clickable u-mt-3" type="checkbox" id="share_activities" name="share_activities" value="share_activities">
                            <label for="share_activities" class="clickable u-m-0">Share my activities on the activity feed.</label><br>

                            <input class="clickable u-mt-3" type="checkbox" id="share_statistics" name="share_statistics" value="share_statistics">
                            <label for="share_statistics" class="clickable u-m-0">Share my statistics on my page.</label><br>

                            <label style="margin-top: 5em;" id="form-input-icon" for="password_old">Current password:</label>
                            <input type="password" name="password_old" id="password_old" placeholder="To save your changes, type your current password." required />

                            <button class="btn u-mt-2" id="update-button" type="submit" href="/">Update account</button>

                        </form>
                    </div>

                </div>

                <div class="account-section" style="display: none;" id="strava-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('strava-wrapper', 'section-button-strava')">
                        <div class="">Strava</div>
                        <img id="section-button-strava" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="strava-wrapper" class="strava-wrapper minimized">
                    </div>
                </div>

                <div class="account-section" style="display: none;" id="hevy-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('hevy-wrapper', 'section-button-hevy')">
                        <div class="">Hevy</div>
                        <img id="section-button-hevy" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="hevy-wrapper" class="hevy-wrapper minimized">
                    </div>
                </div>

                <div class="account-section" style="display: none;" id="plex-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('plex-wrapper', 'section-button-plex')">
                        <div class="">Plex</div>
                        <img id="section-button-plex" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="plex-wrapper" class="plex-wrapper minimized">
                    </div>
                </div>

                <div class="account-section" style="display: none;" id="spotify-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('spotify-wrapper', 'section-button-spotify')">
                        <div class="">Spotify</div>
                        <img id="section-button-spotify" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="spotify-wrapper" class="spotify-wrapper minimized">
                    </div>
                </div>

                <div class="account-section" style="display: none;" id="audiobookshelf-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('audiobookshelf-wrapper', 'section-button-audiobookshelf')">
                        <div class="">Audiobookshelf</div>
                        <img id="section-button-audiobookshelf" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="audiobookshelf-wrapper" class="audiobookshelf-wrapper minimized">
                    </div>
                </div>

                <div class="account-section" id="wheel-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('wheel-wrapper', 'section-button-wheel')">
                        <div class="">Wheel appearance</div>
                        <img id="section-button-wheel" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="wheel-wrapper" class="wheel-wrapper minimized">
                    </div>
                </div>

                <div class="account-section" id="pat-section">

                    <div class="account-section-tab clickable" onclick="toggleSection('pat-wrapper', 'section-button-pat')">
                        <div class="">Developer access tokens</div>
                        <img id="section-button-pat" src="assets/chevron-right.svg" class="color-invert u-m-2">
                    </div>

                    <div id="pat-wrapper" class="pat-wrapper minimized">
                    </div>
                </div>


            </div>


            <div class="module" id="">
                <hr>
            </div>

            <div class="button-collection">

                <button onclick="leave_season();"class="btn btn--danger" type="submit" href="">Leave season</button>

                <button onclick="delete_account();" class="btn btn--danger" type="submit" href="">Delete account</button>

            </div>

        </div>

    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Your very own page...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        GetUserData(user_id, strava_oauth, strava_enabled, hevy_enabled);
        GetProfileImage(user_id);
        CheckForSubscription();
        renderPATSection(admin);
        if(plex_enabled || spotify_enabled || audiobookshelf_enabled) {
            renderMediaSection();
        }
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
    var share_statistics = document.getElementById("share_statistics").checked;
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
                "share_activities": share_activities,
                "share_statistics": share_statistics
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
            "share_activities": share_activities,
            "share_statistics": share_statistics
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

                // store the refreshed OAuth token pair
                if(result.data) {
                    store_tokens(result.data.access_token, result.data.refresh_token);
                }

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

    var img = document.getElementById("user-active-profile-photo-img");
    if (!img) {
        return;
    }
    img.onerror = function() { this.onerror = null; this.src = '/assets/images/barbell.gif'; };
    // Cache-buster: this is the user's own account page, where they may have just changed
    // their photo, so bypass the image cache to always show the current one.
    img.src = profileImageURL(userID, false) + "?v=" + Date.now();

}

function GetUserData(userID, stravaOauth, stravaEnabled, hevyEnabled) {

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

                PlaceUserData(result.user, stravaOauth, stravaEnabled, hevyEnabled)
                
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

function PlaceUserData(user_object, stravaOauth, stravaEnabled, hevyEnabled) {

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

    if(user_object.share_statistics) {
        document.getElementById("share_statistics").checked = true;
    }

    if(stravaEnabled) {
        // Promote to globals so the section can be re-rendered after disconnect
        // without re-fetching server config.
        stravaOauthURL = stravaOauth;
        stravaHevyEnabled = hevyEnabled;
        renderStravaSection(user_object);
    }

    if(hevyEnabled) {
        renderHevySection(user_object);
    }

    renderWheelSection(user_object);
}

function renderStravaSection(user_object) {
    var stravaOauth = stravaOauthURL;
    var hevyEnabled = stravaHevyEnabled;

    var stravaHTML = `
        <p class="u-w-full u-text-center">
            Strava exercises sync automatically every hour. Be careful to only log your sessions to either Strava or {{.appName}}.
        </p>

        <button onclick="window.location.href='${stravaOauth}';" class="btn u-w-10" type="submit" href="">Connect Strava</button>
    `;

    if(user_object.strava_code && user_object.strava_code != "") {
        var walksHTML = user_object.strava_walks ? "checked" : "";
        var publicHTML = user_object.strava_public ? "checked" : "";

        // Only relevant when Hevy is available on this server.
        var skipHevyOption = "";
        if(hevyEnabled) {
            var skipHevyHTML = user_object.strava_skip_hevy ? "checked" : "";
            skipHevyOption = `
                <div class="strava-option" id="">
                    <input class="clickable" type="checkbox" id="strava_skip_hevy" name="strava_skip_hevy" value="" onchange="updateAccountValue('strava_skip_hevy');" ${skipHevyHTML}>
                    <label for="strava_skip_hevy" class="clickable u-m-0">Skip Strava activities already in Hevy</label><br>
                </div>
            `;
        }

        stravaHTML = `
            <p class="u-w-full u-text-center">
                Strava is connected. Exercises sync automatically every hour. Be careful to only log your sessions to either Strava or {{.appName}}.
            </p>

            <div class="notification-options" id="">
                <button onclick="syncStrava('${user_object.id}');" class="btn integration-btn" type="submit" href="">Sync Strava now</button>
                <button onclick="disconnectStrava('${user_object.id}');" class="btn btn--danger integration-btn" type="submit" href="">Disconnect Strava</button>
            </div>

            <div class="notification-options" id="">
                <div class="strava-option" id="">
                    <input class="clickable" type="checkbox" id="strava_walks" name="strava_walks" value="" onchange="updateAccountValue('strava_walks');" ${walksHTML}>
                    <label for="strava_walks" class="clickable u-m-0">Ignore walks</label><br>
                </div>

                <div class="strava-option" id="">
                    <input class="clickable" type="checkbox" id="strava_public" name="strava_public" value="" onchange="updateAccountValue('strava_public');" ${publicHTML}>
                    <label for="strava_public" class="clickable u-m-0">Show my Strava on my profile</label><br>
                </div>

                ${skipHevyOption}
            </div>
        `;
    }

    document.getElementById("strava-wrapper").innerHTML = stravaHTML
    document.getElementById('strava-section').style.display = 'flex'
}

// Re-fetch the user and re-render only the Strava section (used after disconnect
// so the rest of the page is left untouched).
function refreshStravaSection(user_id) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                return;
            }
            if(!result.error && result.user) {
                renderStravaSection(result.user);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + user_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
}

function renderHevySection(user_object) {
    var hevyHTML = `
        <p class="u-w-full u-text-center">
            Connect Hevy to sync your workouts automatically. Your Hevy API key is found under Settings in the Hevy app and requires a Hevy PRO subscription.
        </p>

        <div class="notification-options" id="">
            <input id="hevy_api_key" type="password" placeholder="Hevy API key" autocomplete="off" class="u-w-16">
            <button onclick="setHevy('${user_object.id}');" class="btn integration-btn" type="submit" href="">Connect Hevy</button>
        </div>
    `;

    if(user_object.hevy_connected) {
        var hevyPublicHTML = user_object.hevy_public ? "checked" : "";

        hevyHTML = `
            <p class="u-w-full u-text-center">
                Hevy is connected. Workouts sync automatically. Be careful to only log your sessions to either Hevy or {{.appName}}.
            </p>

            <div class="notification-options" id="">
                <input id="hevy_api_key" type="password" placeholder="Replace Hevy API key" autocomplete="off" class="u-w-16">
                <button onclick="setHevy('${user_object.id}');" class="btn integration-btn" type="submit" href="">Update key</button>
                <button onclick="syncHevy('${user_object.id}');" class="btn integration-btn" type="submit" href="">Sync Hevy now</button>
                <button onclick="disconnectHevy('${user_object.id}');" class="btn btn--danger integration-btn" type="submit" href="">Disconnect Hevy</button>
            </div>

            <div class="notification-options" id="">
                <div class="strava-option" id="">
                    <input class="clickable" type="checkbox" id="hevy_public" name="hevy_public" value="" onchange="updateAccountValue('hevy_public');" ${hevyPublicHTML}>
                    <label for="hevy_public" class="clickable u-m-0">Show my Hevy on my profile</label><br>
                </div>
            </div>
        `;
    }

    document.getElementById("hevy-wrapper").innerHTML = hevyHTML
    document.getElementById('hevy-section').style.display = 'flex'
}

// --- Media / Plex -----------------------------------------------------------

// Tracks the active Plex PIN poll so a second connect attempt cancels the first.
var plexPollTimer = null;

// renderMediaSection fetches the user's media connections and renders the Plex
// section's connected/disconnected state. Other providers reuse the same endpoint
// when they land.
function renderMediaSection() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                return;
            }
            if(result.error) {
                return;
            }

            var plexConnection = null;
            var spotifyConnection = null;
            var audiobookshelfConnection = null;
            (result.connections || []).forEach(function(connection) {
                if(connection.provider == "plex") {
                    plexConnection = connection;
                } else if(connection.provider == "spotify") {
                    spotifyConnection = connection;
                } else if(connection.provider == "audiobookshelf") {
                    audiobookshelfConnection = connection;
                }
            });

            if(plex_enabled) {
                renderPlexSection(plexConnection);
            }
            if(spotify_enabled) {
                renderSpotifySection(spotifyConnection);
            }
            if(audiobookshelf_enabled) {
                renderAudiobookshelfSection(audiobookshelfConnection);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/media/connections");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
}

function renderPlexSection(connection) {
    var plexHTML = `
        <p class="u-w-full u-text-center">
            Connect Plex to overlay what you listened to onto your workouts. You will be sent to Plex to approve the connection.
        </p>

        <div class="notification-options" id="">
            <button onclick="connectPlex();" class="btn integration-btn" type="submit" href="">Connect Plex</button>
        </div>
    `;

    if(connection && connection.connected) {
        var serverValue = connection.server_url ? escapeHTML(connection.server_url) : "";
        var serverHint = connection.server_url
            ? "If syncing fails, your server may be behind a reverse proxy — set the URL you actually reach it on (e.g. https://plex.example.com)."
            : "No server auto-detected. Enter the URL you reach Plex on, e.g. https://plex.example.com";

        plexHTML = `
            <p class="u-w-full u-text-center">
                Plex is connected. Your listening history is matched onto activities by time.
            </p>

            <div class="notification-options" id="">
                <input id="plex_server_url" type="text" placeholder="https://plex.example.com" autocomplete="off" value="${serverValue}" class="u-w-18">
                <button onclick="savePlexServerURL();" class="btn integration-btn" type="submit" href="">Save server URL</button>
            </div>

            <p style="width: 100%; text-align: center; opacity: 0.7; font-size: 0.85em;">
                ${serverHint}
            </p>

            <div class="notification-options" id="">
                <button onclick="connectPlex();" class="btn integration-btn" type="submit" href="">Reconnect Plex</button>
                <button onclick="disconnectPlex();" class="btn btn--danger integration-btn" type="submit" href="">Disconnect Plex</button>
            </div>
        `;
    }

    document.getElementById("plex-wrapper").innerHTML = plexHTML;
    document.getElementById('plex-section').style.display = 'flex';
}

// connectPlex starts the plex.tv PIN flow: it asks the API for a PIN, opens the
// Plex approval page in a new tab, and then polls until the PIN is approved.
function connectPlex() {
    if(plexPollTimer) {
        clearTimeout(plexPollTimer);
        plexPollTimer = null;
    }

    // Open the window synchronously inside the click so the browser doesn't treat
    // it as a pop-up; the URL is filled in once the API returns the PIN.
    var plexWindow = window.open("", "_blank");

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                if(plexWindow) { plexWindow.close(); }
                return;
            }

            if(result.error || !result.pin) {
                error(result.error || "Failed to start Plex connection.");
                if(plexWindow) { plexWindow.close(); }
                return;
            }

            if(plexWindow) {
                plexWindow.location.href = result.pin.auth_url;
            } else {
                window.location.href = result.pin.auth_url;
            }

            info("Waiting for Plex approval...");
            pollPlexPin(result.pin.pin_id, 0);
        } else {
            info("Connecting...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/media/plex/pin");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// pollPlexPin checks the PIN every few seconds until it is approved or the attempt
// budget (~2.5 minutes) is exhausted.
function pollPlexPin(pinID, attempts) {
    var maxAttempts = 50;
    if(attempts >= maxAttempts) {
        error("Plex connection timed out. Please try again.");
        return;
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }

            if(result.error) {
                error(result.error);
                return;
            }

            if(result.result && result.result.authorized) {
                success("Plex connected.");
                renderMediaSection();
                return;
            }

            // Not approved yet — check again shortly.
            plexPollTimer = setTimeout(function() {
                pollPlexPin(pinID, attempts + 1);
            }, 3000);
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/media/plex/pin/" + pinID + "/check");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
}

function disconnectPlex() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }

            if(result.error) {
                error(result.error);
            } else {
                success(result.message);
                renderMediaSection();
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/media/plex");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function savePlexServerURL() {
    var serverURL = document.getElementById("plex_server_url").value.trim();
    if(serverURL == "") {
        error("Please enter your Plex server URL.");
        return false;
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }

            if(result.error) {
                error(result.error);
            } else {
                // The save succeeds even when unreachable; warn vs confirm on the flag.
                if(result.reachable === false) {
                    info(result.message);
                } else {
                    success(result.message);
                }
                renderMediaSection();
            }
        } else {
            info("Saving...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/media/plex/server");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(JSON.stringify({ server_url: serverURL }));
    return false;
}

// --- Media / Spotify --------------------------------------------------------

function renderSpotifySection(connection) {
    var spotifyHTML = `
        <p class="u-w-full u-text-center">
            Connect Spotify to overlay what you listened to onto your workouts. Spotify only keeps the last ~24 hours of listening, so connect it before (or soon after) you train.
        </p>

        <div class="notification-options" id="">
            <button onclick="connectSpotify();" class="btn integration-btn" type="submit" href="">Connect Spotify</button>
        </div>
    `;

    if(connection && connection.connected) {
        spotifyHTML = `
            <p class="u-w-full u-text-center">
                Spotify is connected. Recent listening is matched onto activities by time. Because Spotify only exposes the last ~24 hours, older workouts can't be back-filled.
            </p>

            <div class="notification-options" id="">
                <button onclick="connectSpotify();" class="btn integration-btn" type="submit" href="">Reconnect Spotify</button>
                <button onclick="disconnectSpotify();" class="btn btn--danger integration-btn" type="submit" href="">Disconnect Spotify</button>
            </div>
        `;
    }

    document.getElementById("spotify-wrapper").innerHTML = spotifyHTML;
    document.getElementById('spotify-section').style.display = 'flex';
}

// connectSpotify sends the user to Spotify's consent screen; the /oauth page relays
// the authorization code back to the API (state=spotify routes it there).
function connectSpotify() {
    if(!spotify_oauth) {
        error("Spotify is not configured.");
        return false;
    }
    window.location.href = spotify_oauth;
    return false;
}

function disconnectSpotify() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }

            if(result.error) {
                error(result.error);
            } else {
                success(result.message);
                renderMediaSection();
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/media/spotify");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// --- Media / Audiobookshelf -------------------------------------------------

function renderAudiobookshelfSection(connection) {
    var absHTML = `
        <p class="u-w-full u-text-center">
            Connect Audiobookshelf to overlay the audiobooks and podcasts you listened to onto your workouts. Enter your server URL and an API token from your Audiobookshelf account settings.
        </p>

        <div class="notification-options" id="">
            <input id="abs_server_url" type="text" placeholder="https://abs.example.com" autocomplete="off" value="" class="u-w-18">
        </div>
        <div class="notification-options" id="">
            <input id="abs_token" type="password" placeholder="API token" autocomplete="off" value="" class="u-w-18">
            <button onclick="connectAudiobookshelf();" class="btn integration-btn" type="submit" href="">Connect</button>
        </div>
    `;

    if(connection && connection.connected) {
        var serverValue = connection.server_url ? escapeHTML(connection.server_url) : "";
        absHTML = `
            <p class="u-w-full u-text-center">
                Audiobookshelf is connected. Your listening history is matched onto activities by time.
            </p>

            <div class="notification-options" id="">
                <input id="abs_server_url" type="text" placeholder="https://abs.example.com" autocomplete="off" value="${serverValue}" class="u-w-18">
            </div>
            <div class="notification-options" id="">
                <input id="abs_token" type="password" placeholder="New API token (to update)" autocomplete="off" value="" class="u-w-18">
                <button onclick="connectAudiobookshelf();" class="btn integration-btn" type="submit" href="">Reconnect</button>
            </div>

            <div class="notification-options" id="">
                <button onclick="disconnectAudiobookshelf();" class="btn btn--danger integration-btn" type="submit" href="">Disconnect</button>
            </div>
        `;
    }

    document.getElementById("audiobookshelf-wrapper").innerHTML = absHTML;
    document.getElementById('audiobookshelf-section').style.display = 'flex';
}

// connectAudiobookshelf validates the entered server URL + API token server-side and
// stores the connection.
function connectAudiobookshelf() {
    var serverURL = document.getElementById("abs_server_url").value.trim();
    var token = document.getElementById("abs_token").value.trim();
    if(serverURL == "") {
        error("Please enter your Audiobookshelf server URL.");
        return false;
    }
    if(token == "") {
        error("Please enter an Audiobookshelf API token.");
        return false;
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }

            if(result.error) {
                error(result.error);
            } else {
                success(result.message);
                renderMediaSection();
            }
        } else {
            info("Connecting...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/media/audiobookshelf/connect");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(JSON.stringify({ server_url: serverURL, token: token }));
    return false;
}

function disconnectAudiobookshelf() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }

            if(result.error) {
                error(result.error);
            } else {
                success(result.message);
                renderMediaSection();
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/media/audiobookshelf");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// Re-fetch the user and re-render only the Hevy section (used after connect/disconnect
// so the rest of the page is left untouched).
function refreshHevySection(user_id) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                return;
            }
            if(!result.error && result.user) {
                renderHevySection(result.user);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + user_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return;
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

function setHevy(user_id) {
    var apiKey = document.getElementById("hevy_api_key").value.trim();
    if(apiKey == "") {
        error("Please enter your Hevy API key.");
        return false;
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
                success(result.message);
                refreshHevySection(user_id);
            }
        } else {
            info("Connecting...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/" + user_id + "/hevy");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(JSON.stringify({ hevy_api_key: apiKey }));
    return false;
}

function disconnectStrava(user_id) {
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
                refreshStravaSection(user_id);
            }
        } else {
            info("Disconnecting...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/users/" + user_id + "/strava");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function disconnectHevy(user_id) {
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
                refreshHevySection(user_id);
            }
        } else {
            info("Disconnecting...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/users/" + user_id + "/hevy");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function syncHevy(user_id) {
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
    xhttp.open("post", api_url + "auth/users/" + user_id + "/hevy-sync");
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

// --- Personal Access Tokens ---

var wheelPalette = [
    "#800000", "#9A6324", "#808000", "#469990", "#e6194B", "#f58231", "#ffe119", "#bfef45", "#3cb44b", "#42d4f4", "#4363d8", "#911eb4", "#f032e6", "#a9a9a9", "#fabed4", "#ffd8b1", "#fffac8", "#aaffc3", "#dcbeff", "#ffffff"
];
var wheelState = { color: null, border: null, emoji: null, firstName: "You" };

function isAccountHex(value) {
    return typeof value === "string" && /^#[0-9a-fA-F]{6}$/.test(value);
}

// Black or white text depending on background brightness (mirrors the wheel).
function accountReadableTextColor(hex) {
    if (!isAccountHex(hex)) return "#000000";
    var r = parseInt(hex.substr(1, 2), 16), g = parseInt(hex.substr(3, 2), 16), b = parseInt(hex.substr(5, 2), 16);
    return ((r * 299 + g * 587 + b * 114) / 1000) >= 140 ? "#000000" : "#ffffff";
}

function renderWheelSection(user_object) {
    wheelState.color = isAccountHex(user_object.wheel_color) ? user_object.wheel_color : null;
    wheelState.border = isAccountHex(user_object.wheel_border_color) ? user_object.wheel_border_color : null;
    wheelState.emoji = (typeof user_object.wheel_emoji === "string" && user_object.wheel_emoji) ? user_object.wheel_emoji : null;
    wheelState.firstName = user_object.first_name || "You";

    var swatches = function(kind, current) {
        var html = "";
        for (var i = 0; i < wheelPalette.length; i++) {
            var c = wheelPalette[i];
            var sel = (current && current.toLowerCase() === c.toLowerCase()) ? " wheel-swatch-selected" : "";
            html += `<button type="button" data-color="${c}" class="wheel-swatch${sel}" style="background:${c};" title="${c}" onclick="selectWheelColor('${kind}','${c}')"></button>`;
        }
        return html;
    };

    var colorCustom = isAccountHex(wheelState.color) ? wheelState.color : "#4363d8";
    var borderCustom = isAccountHex(wheelState.border) ? wheelState.border : "#000000";

    var html = `
        <p class="u-w-full u-text-center">Choose how you appear on the prize wheel. Leave a value unset to be auto-assigned.</p>

        <div id="wheel-preview" class="wheel-preview"></div>

        <div class="wheel-option">
            <label>Color</label>
            <div id="wheel-swatches-color" class="wheel-swatches">${swatches('color', wheelState.color)}</div>
            <div class="wheel-controls">
                <input type="color" id="wheel-color-custom" value="${colorCustom}" onchange="selectWheelColor('color', this.value)">
                <button type="button" class="wheel-clear" onclick="selectWheelColor('color', null)">Auto</button>
            </div>
        </div>

        <div class="wheel-option">
            <label>Border</label>
            <div id="wheel-swatches-border" class="wheel-swatches">${swatches('border', wheelState.border)}</div>
            <div class="wheel-controls">
                <input type="color" id="wheel-border-custom" value="${borderCustom}" onchange="selectWheelColor('border', this.value)">
                <button type="button" class="wheel-clear" onclick="selectWheelColor('border', null)">None</button>
            </div>
        </div>

        <div class="wheel-option">
            <label for="wheel-emoji-input">Emoji</label>
            <div class="wheel-controls">
                <input type="text" id="wheel-emoji-input" maxlength="16" placeholder="🔥" value="${wheelState.emoji ? escapeHTML(wheelState.emoji) : ""}" oninput="previewWheelEmoji(this.value)" onchange="saveWheelEmoji(this.value)">
                <button type="button" class="wheel-clear" onclick="document.getElementById('wheel-emoji-input').value=''; saveWheelEmoji('');">Clear</button>
            </div>
        </div>
    `;

    document.getElementById("wheel-wrapper").innerHTML = html;
    updateWheelPreview();
}

function selectWheelColor(kind, value) {
    if (value !== null && !isAccountHex(value)) return;
    if (kind === 'color') {
        wheelState.color = value;
        saveWheelValue('wheel_color', value === null ? "" : value);
    } else {
        wheelState.border = value;
        saveWheelValue('wheel_border_color', value === null ? "" : value);
    }
    highlightSwatch(kind, value);
    updateWheelPreview();
}

// firstGrapheme returns the first user-perceived character (one emoji), correctly
// handling multi-codepoint emoji (flags, skin tones, ZWJ sequences) where supported.
function firstGrapheme(value) {
    value = (value || "").trim();
    if (!value) return "";
    if (typeof Intl !== "undefined" && Intl.Segmenter) {
        var first = new Intl.Segmenter(undefined, { granularity: "grapheme" }).segment(value)[Symbol.iterator]().next();
        return first.done ? "" : first.value.segment;
    }
    // Fallback for older browsers: first code point.
    return Array.from(value)[0] || "";
}

function previewWheelEmoji(value) {
    var one = firstGrapheme(value);
    var input = document.getElementById('wheel-emoji-input');
    if (input && input.value !== one) input.value = one;
    wheelState.emoji = one === "" ? null : one;
    updateWheelPreview();
}

function saveWheelEmoji(value) {
    var one = firstGrapheme(value);
    var input = document.getElementById('wheel-emoji-input');
    if (input && input.value !== one) input.value = one;
    wheelState.emoji = one === "" ? null : one;
    saveWheelValue('wheel_emoji', one);
    updateWheelPreview();
}

function highlightSwatch(kind, value) {
    var container = document.getElementById('wheel-swatches-' + kind);
    if (!container) return;
    var buttons = container.querySelectorAll('button');
    for (var i = 0; i < buttons.length; i++) {
        var dc = buttons[i].getAttribute('data-color');
        if (value && dc && dc.toLowerCase() === value.toLowerCase()) {
            buttons[i].classList.add('wheel-swatch-selected');
        } else {
            buttons[i].classList.remove('wheel-swatch-selected');
        }
    }
}

function updateWheelPreview() {
    var preview = document.getElementById('wheel-preview');
    if (!preview) return;
    var fill = isAccountHex(wheelState.color) ? wheelState.color : "#cccccc";
    var text = accountReadableTextColor(fill);
    var outline = text === "#000000" ? "#ffffff" : "#000000";
    var label = (wheelState.emoji ? wheelState.emoji + " " : "") + wheelState.firstName;

    preview.style.background = fill;
    preview.style.color = text;
    preview.style.border = "4px solid " + (isAccountHex(wheelState.border) ? wheelState.border : "transparent");
    preview.style.textShadow = "0 0 2px " + outline + ", 0 0 2px " + outline;
    preview.textContent = label + (isAccountHex(wheelState.color) ? "" : " (auto color)");
}

function saveWheelValue(property, value) {
    var form_obj = {};
    form_obj[property] = value;
    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e + ' - Response: ' + this.responseText);
                error("Could not reach API.");
                return;
            }
            if (result.error) {
                error(result.error);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("PATCH", api_url + "auth/users/" + user_id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
}

function renderPATSection(isAdmin) {
    var adminCheckbox = "";
    if(isAdmin) {
        adminCheckbox = `
            <div class="pat-option">
                <input class="clickable" type="checkbox" id="pat_admin" name="pat_admin">
                <label for="pat_admin" class="clickable u-m-0">Include admin access</label>
            </div>`;
    }

    var html = `
    <div class="text-body u-mb-1">
        Personal access tokens let your own scripts and integrations use the API on your behalf.
        Treat them like passwords &mdash; anyone with a token can act as you.
    </div>

    <form class="pat-form" onsubmit="event.preventDefault(); submitPAT();">
        <input type="text" id="pat_name" placeholder="Token name (e.g. my laptop script)" required>

        <select id="pat_scope" class="clickable">
            <option value="api:read">Read-only</option>
            <option value="api:write">Read &amp; write</option>
        </select>

        <select id="pat_expiry" class="clickable">
            <option value="30">Expires in 30 days</option>
            <option value="90" selected>Expires in 90 days</option>
            <option value="180">Expires in 180 days</option>
            <option value="365">Expires in 365 days</option>
        </select>
        ` + adminCheckbox + `
        <button class="btn u-w-12" type="submit">Create token</button>
    </form>

    <div id="pat-new-token" style="margin-top:1em;"></div>

    <div id="pat-list" style="margin-top:1em; width:100%;">Loading...</div>
    `;

    document.getElementById("pat-wrapper").innerHTML = html;
    loadPATs();
}

function loadPATs() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if(this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                document.getElementById("pat-list").innerHTML = "Could not load tokens.";
                return;
            }
            if(result.error) {
                document.getElementById("pat-list").innerHTML = "Could not load tokens.";
                return;
            }
            renderPATList(result.data || []);
        }
    };
    xhttp.open("get", api_url + "auth/pats");
    xhttp.setRequestHeader("Authorization", "Bearer " + jwt);
    xhttp.send();
}

function renderPATList(pats) {
    if(!pats.length) {
        document.getElementById("pat-list").innerHTML = `<div class="text-body">No active tokens.</div>`;
        return;
    }

    var rows = "";
    for(var i = 0; i < pats.length; i++) {
        var p = pats[i];
        var expires = new Date(p.expires_at).toLocaleDateString();
        var lastUsed = p.last_used_at ? new Date(p.last_used_at).toLocaleDateString() : "never";
        rows += `
        <div class="pat-list-item">
            <div class="pat-list-info">
                <b>${escapeHTML(p.name)}</b>
                <span class="pat-meta">${escapeHTML(p.scope)} &middot; expires ${expires} &middot; last used ${lastUsed}</span>
            </div>
            <button class="btn btn--danger pat-revoke" onclick="revokePAT('${p.id}')">Revoke</button>
        </div>`;
    }
    document.getElementById("pat-list").innerHTML = rows;
}

function submitPAT() {
    var name = document.getElementById("pat_name").value;
    var scope = document.getElementById("pat_scope").value;
    var expiry = parseInt(document.getElementById("pat_expiry").value, 10);
    var adminEl = document.getElementById("pat_admin");
    var admin = adminEl ? adminEl.checked : false;

    var form_data = JSON.stringify({
        "name": name,
        "scope": scope,
        "admin": admin,
        "expires_in_days": expiry
    });

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if(this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                error("Could not reach API.");
                return;
            }
            if(result.error) {
                error(result.error);
                return;
            }
            success("Token created.");
            document.getElementById("pat_name").value = "";
            showNewPAT(result.data.token);
            loadPATs();
        }
    };
    xhttp.open("post", api_url + "auth/pats");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", "Bearer " + jwt);
    xhttp.send(form_data);
}

function showNewPAT(token) {
    var html = `
    <div class="pat-new-token-box">
        <div class="text-body u-mb-2">
            Copy your new token now &mdash; you won't be able to see it again.
        </div>
        <code class="pat-token-value" id="pat-token-value">${escapeHTML(token)}</code>
        <button class="btn" type="button" style="width:8em; margin-top:0.5em;" onclick="copyPAT()">Copy</button>
    </div>`;
    document.getElementById("pat-new-token").innerHTML = html;
}

function copyPAT() {
    var value = document.getElementById("pat-token-value").innerText;
    navigator.clipboard.writeText(value).then(function() {
        success("Token copied to clipboard.");
    }, function() {
        error("Could not copy token.");
    });
}

function revokePAT(patID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if(this.readyState == 4) {
            var result;
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                error("Could not reach API.");
                return;
            }
            if(result.error) {
                error(result.error);
                return;
            }
            success("Token revoked.");
            loadPATs();
        }
    };
    xhttp.open("DELETE", api_url + "auth/pats/" + patID);
    xhttp.setRequestHeader("Authorization", "Bearer " + jwt);
    xhttp.send();
}

function escapeHTML(value) {
    return String(value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;");
}