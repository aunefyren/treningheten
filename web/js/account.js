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

    var exerciseConnectionsEnabled = strava_enabled || hevy_enabled;
    var listeningConnectionsEnabled = plex_enabled || spotify_enabled || audiobookshelf_enabled;

    var html = `
        <div class="module">

            <div class="user-active-profile-photo">
                <img class="user-active-profile-photo-img u-fill" id="user-active-profile-photo-img" src="/assets/images/barbell.gif">
            </div>

            <p id="user_name" class="user-name"></p>
            <p id="join_date"></p>
            <p id="user_admin"></p>

            <div class="btn-group">
                <button onclick="window.location.href = '/users/${user_id}';" class="btn" type="button">Public profile</button>
                <button onclick="window.location.href = '/gear';" class="btn" type="button">Manage gear</button>
            </div>

            ${settingsGroupHTML("Account",
                accountSectionHTML("settings", "Account settings", accountSettingsBodyHTML(), "settings-body") +
                accountSectionHTML("notifications", "Notifications", notificationsBodyHTML(vapid_public_key), "settings-body") +
                accountSectionHTML("wheel", "Wheel appearance", "")
            )}

            ${settingsGroupHTML("Training",
                accountSectionHTML("training", "Training", trainingBodyHTML(exerciseConnectionsEnabled), "settings-body")
            )}

            ${exerciseConnectionsEnabled ? settingsGroupHTML("Exercise connections",
                (strava_enabled ? accountSectionHTML("strava", "Strava", "", "settings-body") : "") +
                (hevy_enabled ? accountSectionHTML("hevy", "Hevy", "", "settings-body") : "")
            ) : ""}

            ${listeningConnectionsEnabled ? settingsGroupHTML("Listening connections",
                (plex_enabled ? accountSectionHTML("plex", "Plex", "", "settings-body") : "") +
                (spotify_enabled ? accountSectionHTML("spotify", "Spotify", "", "settings-body") : "") +
                (audiobookshelf_enabled ? accountSectionHTML("audiobookshelf", "Audiobookshelf", "", "settings-body") : "")
            ) : ""}

            ${settingsGroupHTML("Developer",
                accountSectionHTML("pat", "Access tokens", "")
            )}

            <div class="module">
                <hr>
            </div>

            <div class="btn-group">
                <button onclick="leaveSeason();" class="btn btn--danger" type="button">Leave season</button>
                <button onclick="deleteAccount();" class="btn btn--danger" type="button">Delete account</button>
            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Your very own page...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        getUserData(user_id, strava_oauth, strava_enabled, hevy_enabled);
        GetProfileImage(user_id);
        CheckForSubscription();
        renderPATSection(admin);
        if(listeningConnectionsEnabled) {
            renderMediaSection();
        }
        if(exerciseConnectionsEnabled) {
            renderGoalCountingSection();
        }
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

// settingsGroupHTML wraps a run of accordion sections in one titled panel. The page is split
// into groups (account, training, exercise and listening connections, developer) so related
// settings sit together; a group whose sections are all disabled on this server isn't rendered.
function settingsGroupHTML(title, sectionsHTML) {
    return `
        <section class="settings-group">
            <h2 class="settings-group-title">${title}</h2>
            <div class="account-section-wrapper">
                ${sectionsHTML}
            </div>
        </section>
    `;
}

// accountSectionHTML renders one accordion row: a header button (title, a status slot the
// connection sections fill, and the chevron) over a collapsed body. key derives the ids the
// rest of the page uses: `${key}-section`, `${key}-wrapper`, `section-button-${key}` and
// `section-status-${key}`. bodyClass adds a layout modifier to the body (e.g. settings-body).
function accountSectionHTML(key, title, bodyHTML, bodyClass) {
    return `
        <div class="account-section" id="${key}-section">
            <button type="button" class="account-section-tab" aria-expanded="false" aria-controls="${key}-wrapper" onclick="toggleSection('${key}-wrapper', 'section-button-${key}')">
                <span class="account-section-title">${title}<span id="section-status-${key}"></span></span>
                <img id="section-button-${key}" src="assets/chevron-right.svg" class="u-m-2" alt="">
            </button>
            <div id="${key}-wrapper" class="account-section-body ${bodyClass || ""} minimized">
                ${bodyHTML}
            </div>
        </div>
    `;
}

function accountSettingsBodyHTML() {
    return `
        <form action="" onsubmit="event.preventDefault(); updateAccount('${user_id}');">

            <div class="field">
                <label for="email" class="field-label">Replace email</label>
                <input type="email" name="email" id="email" placeholder="Email" required/>
            </div>

            <div class="field">
                <label for="new_profile_image" class="field-label">Replace profile image</label>
                <input type="file" name="new_profile_image" id="new_profile_image" accept="image/png, image/jpeg" />
            </div>

            <div class="field-check">
                <input onclick="togglePasswordFields();" type="checkbox" id="password-toggle" name="confirm" value="confirm">
                <label for="password-toggle">Change my password.</label>
            </div>

            <div id="change-password-box" class="settings-stack" style="display:none;">

                <div class="field">
                    <label for="password" class="field-label">New password</label>
                    <input type="password" name="password" id="password" placeholder="New password" />
                </div>

                <div class="field">
                    <label for="password_repeat" class="field-label">Repeat password</label>
                    <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" />
                </div>

            </div>

            <div class="field-check">
                <input type="checkbox" id="share_activities" name="share_activities" value="share_activities">
                <label for="share_activities">Share my activities on the activity feed. Visible to everyone you have shared a season with, including past seasons.</label>
            </div>

            <div class="field-check">
                <input type="checkbox" id="share_statistics" name="share_statistics" value="share_statistics">
                <label for="share_statistics">Share my statistics on my page.</label>
            </div>

            <div class="field">
                <label for="password_old" class="field-label">Current password</label>
                <input type="password" name="password_old" id="password_old" placeholder="To save your changes, type your current password." required />
            </div>

            <div class="btn-group">
                <button class="btn btn--primary" id="update-button" type="submit">Update account</button>
            </div>

        </form>
    `;
}

// notificationsBodyHTML holds both channels: push on this device (chosen per device, saved by
// the button) and e-mail (saved per account as soon as it is ticked).
function notificationsBodyHTML(vapidPublicKey) {
    return `
        <h3 class="settings-subtitle">This device</h3>
        <p class="settings-text">Pick what this device alerts you about, then press the button. Press it again after changing these.</p>

        <div class="field-check-group">
            <div class="field-check">
                <input type="checkbox" id="notification-reminder-toggle" name="notification-reminder-toggle">
                <label for="notification-reminder-toggle">Logging reminders</label>
            </div>
            <div class="field-check">
                <input type="checkbox" id="notification-achievement-toggle" name="notification-achievement-toggle">
                <label for="notification-achievement-toggle">Achievements</label>
            </div>
            <div class="field-check">
                <input type="checkbox" id="notification-news-toggle" name="notification-news-toggle">
                <label for="notification-news-toggle">News</label>
            </div>
            <div class="field-check">
                <input type="checkbox" id="notification-account-toggle" name="notification-account-toggle" checked>
                <label for="notification-account-toggle">Account updates</label>
            </div>
        </div>

        <div class="btn-group">
            <button type="button" class="btn btn--primary" onclick="create_push('${vapidPublicKey}'); return false;">Notify me on this device</button>
        </div>

        <h3 class="settings-subtitle">E-mail</h3>

        <div class="field-check">
            <input type="checkbox" id="sunday_alert" name="sunday_alert" onchange="updateAccountValue('sunday_alert');">
            <label for="sunday_alert">Send me e-mail logging reminders on Sundays.</label>
        </div>
    `;
}

// trainingBodyHTML holds the settings that shape how workouts are read: the heart-rate zone
// anchors (saved on their own, no password) and, when an import integration is on, which
// activity types count toward the weekly goal (filled by renderGoalCountingSection).
function trainingBodyHTML(goalCountingEnabled) {
    var goalCountingHTML = "";
    if(goalCountingEnabled) {
        goalCountingHTML = `
            <h3 class="settings-subtitle" id="goal-counting-title">What counts toward your goal</h3>
            <div id="goal-counting-body">
                <p class="settings-text">Loading activity types…</p>
            </div>
        `;
    }

    return `
        <h3 class="settings-subtitle">Heart rate and age</h3>
        <p class="settings-text">These set the heart-rate zones on your activities.</p>

        <form action="" onsubmit="event.preventDefault(); saveTrainingProfile();">

            <div class="field">
                <label for="birth_date" class="field-label">Birth date</label>
                <input type="date" name="birth_date" id="birth_date" />
            </div>

            <div class="field-row">
                <div class="field">
                    <label for="max_heartrate" class="field-label">Max heart rate</label>
                    <input type="number" name="max_heartrate" id="max_heartrate" min="100" max="240" placeholder="Automatic" oninput="updateMaxHRHint()" />
                    <span class="field-hint">Optional. Leave it on automatic and {{.appName}} uses the highest heart rate seen in your activities, or estimates from your age.</span>
                    <span class="field-hint" id="max_heartrate_status"></span>
                </div>
                <div class="field">
                    <label for="resting_heartrate" class="field-label">Resting heart rate</label>
                    <input type="number" name="resting_heartrate" id="resting_heartrate" min="25" max="120" placeholder="e.g. 50" />
                    <span class="field-hint">Optional. When set, zones switch to heart-rate reserve (Karvonen) instead of a plain percentage of your max.</span>
                </div>
            </div>

            <div class="btn-group">
                <button class="btn btn--primary" id="training-save-button" type="submit">Save heart rate and age</button>
            </div>

        </form>

        ${goalCountingHTML}
    `;
}

function togglePasswordFields() {
    var passwordBox = document.getElementById("change-password-box");
    passwordBox.style.display = document.getElementById("password-toggle").checked ? "flex" : "none";
}

// updateAccount sends the password-gated account form: e-mail, password, profile image and
// sharing. The training profile (birth date, heart rate) is saved separately by
// saveTrainingProfile.
function updateAccount(userID) {
    var password = "";
    var passwordRepeat = "";
    if(document.getElementById("password-toggle").checked) {
        password = document.getElementById("password").value;
        passwordRepeat = document.getElementById("password_repeat").value;
    }

    var formObject = {
        "email": document.getElementById("email").value,
        "password": password,
        "password_repeat": passwordRepeat,
        "profile_image": "",
        "password_old": document.getElementById("password_old").value,
        "share_activities": document.getElementById("share_activities").checked,
        "share_statistics": document.getElementById("share_statistics").checked
    };

    var newProfileImage = document.getElementById('new_profile_image').files[0];
    if(!newProfileImage) {
        submitAccountUpdate(JSON.stringify(formObject), userID);
        return;
    }

    if(newProfileImage.size > 10000000) {
        error("Image exceeds 10MB size limit.");
        return;
    } else if(newProfileImage.size < 10000) {
        error("Image smaller than 0.01MB size requirement.");
        document.getElementById("password_old").value = "";
        return;
    }

    get_base64(newProfileImage).then(function(result) {
        formObject.profile_image = result;
        document.getElementById("user-active-profile-photo-img").src = 'assets/images/barbell.gif';
        submitAccountUpdate(JSON.stringify(formObject), userID);
    });
}

function submitAccountUpdate(formData, userID) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
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
                return;
            }

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
        } else {
            info("Updating account...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(formData);
    return false;
}

// saveTrainingProfile replaces the birth date and heart-rate anchors. A blank field is sent as
// null, which clears it (back to automatic).
function saveTrainingProfile() {
    var birthDate = document.getElementById('birth_date').value;
    var maxHeartrate = document.getElementById('max_heartrate').value;
    var restingHeartrate = document.getElementById('resting_heartrate').value;

    var formObject = {
        "birth_date": birthDate === "" ? null : new Date(birthDate).toISOString(),
        "max_heartrate": maxHeartrate === "" ? null : parseInt(maxHeartrate, 10),
        "resting_heartrate": restingHeartrate === "" ? null : parseInt(restingHeartrate, 10)
    };

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

            success(result.message);
            hrAgeEstimate = ageBasedMaxHR(formObject.birth_date);
            updateMaxHRHint();
        } else {
            info("Saving training profile...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/users/" + user_id + "/training-profile");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(JSON.stringify(formObject));
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

function getUserData(userID, stravaOauth, stravaEnabled, hevyEnabled) {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            var result;
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
                placeUserData(result.user, stravaOauth, stravaEnabled, hevyEnabled);
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/users/" + userID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

// What "automatic" would resolve to, captured from the loaded user so the anchor status
// can name the alternative (observed max from activities, else the age-based estimate).
var hrObservedMax = null;
var hrAgeEstimate = null;

// ageBasedMaxHR estimates 220 − age from an ISO birth date, or null when unknown. This is
// only illustrative for the account hint; the actual zones compute age per activity date.
function ageBasedMaxHR(birthDateISO) {
    if (!birthDateISO) return null;
    var b = new Date(birthDateISO);
    if (isNaN(b.getTime())) return null;
    var now = new Date();
    var age = now.getFullYear() - b.getFullYear();
    if (now.getMonth() < b.getMonth() || (now.getMonth() === b.getMonth() && now.getDate() < b.getDate())) age--;
    if (age <= 0 || age > 120) return null;
    return 220 - age;
}

// autoMaxHRDescription names the value zones fall back to when no explicit max is set.
function autoMaxHRDescription() {
    if (hrObservedMax) return "the highest we've seen in your activities, " + hrObservedMax + " bpm";
    if (hrAgeEstimate) return "an age-based estimate of about " + hrAgeEstimate + " bpm";
    return "each activity's own peak heart rate";
}

// updateMaxHRHint reflects the current anchor state: when a max is entered it's the anchor
// (with a link to switch back to automatic and what that would be); when blank, zones are
// automatic (with a one-click pin of the observed value).
function updateMaxHRHint() {
    var status = document.getElementById("max_heartrate_status");
    if (!status) return;
    var value = document.getElementById("max_heartrate").value.trim();
    var auto = autoMaxHRDescription();
    if (value !== "") {
        status.innerHTML = "Zones are anchored to " + escapeHTML(value) + " bpm. " +
            '<a href="#" class="user-link" onclick="clearMaxHRAnchor(); return false;">Use automatic instead</a> (' + auto + ").";
    } else {
        status.innerHTML = "Automatic — zones use " + auto + "." +
            (hrObservedMax ? ' <a href="#" class="user-link" onclick="useObservedMaxHeartrate(' + hrObservedMax + '); return false;">Pin ' + hrObservedMax + " bpm</a>." : "");
    }
}

// useObservedMaxHeartrate pins the field to the observed value; clearMaxHRAnchor removes the
// anchor so zones go back to automatic. Both need a Save to persist.
function useObservedMaxHeartrate(value) {
    document.getElementById("max_heartrate").value = value;
    updateMaxHRHint();
}
function clearMaxHRAnchor() {
    document.getElementById("max_heartrate").value = "";
    updateMaxHRHint();
}

function placeUserData(userObject, stravaOauth, stravaEnabled, hevyEnabled) {
    document.getElementById("user_name").innerHTML = userObject.first_name + " " + userObject.last_name;
    document.getElementById("email").value = userObject.email;

    if(userObject.birth_date != null) {
        document.getElementById("birth_date").value = GetShortDate(new Date(Date.parse(userObject.birth_date)));
    }
    if(userObject.max_heartrate != null) {
        document.getElementById("max_heartrate").value = userObject.max_heartrate;
    }
    if(userObject.resting_heartrate != null) {
        document.getElementById("resting_heartrate").value = userObject.resting_heartrate;
    }

    // Remember what "automatic" resolves to (the observed max, else an age-based estimate)
    // so the anchor status can spell out the alternative to whatever is set.
    hrObservedMax = (userObject.observed_max_heartrate != null && userObject.observed_max_heartrate > 0) ? userObject.observed_max_heartrate : null;
    hrAgeEstimate = ageBasedMaxHR(userObject.birth_date);
    updateMaxHRHint();

    var dateString = "Error";
    try {
        dateString = GetDateString(new Date(Date.parse(userObject.created_at)));
    } catch {}
    document.getElementById("join_date").innerHTML = "Joined: " + dateString;
    document.getElementById("user_admin").innerHTML = "Administrator: " + (userObject.admin ? "Yes" : "No");

    document.getElementById("sunday_alert").checked = !!userObject.sunday_alert;
    document.getElementById("share_activities").checked = !!userObject.share_activities;
    document.getElementById("share_statistics").checked = !!userObject.share_statistics;

    if(stravaEnabled) {
        // Promote to globals so the section can be re-rendered after disconnect
        // without re-fetching server config.
        stravaOauthURL = stravaOauth;
        stravaHevyEnabled = hevyEnabled;
        renderStravaSection(userObject);
    }

    if(hevyEnabled) {
        renderHevySection(userObject);
    }

    renderWheelSection(userObject);
}

function renderStravaSection(userObject) {
    var stravaOauth = stravaOauthURL;
    var hevyEnabled = stravaHevyEnabled;

    var stravaHTML = `
        <p class="settings-text">
            Strava exercises sync automatically every hour. Be careful to only log your sessions to either Strava or {{.appName}}.
        </p>

        <div class="btn-group">
            <button onclick="window.location.href='${stravaOauth}';" class="btn btn--primary integration-btn" type="button">Connect Strava</button>
        </div>
    `;

    // The Strava credential itself is no longer serialized — the API reports connection
    // state as a derived boolean instead (mirrors hevy_connected).
    var stravaHealth = integrationHealthFor(userObject, "strava");
    if(userObject.strava_connected) {
        // Only relevant when Hevy is available on this server.
        var skipHevyOption = "";
        if(hevyEnabled) {
            skipHevyOption = `
                <div class="field-check">
                    <input type="checkbox" id="strava_skip_hevy" name="strava_skip_hevy" onchange="updateAccountValue('strava_skip_hevy');" ${userObject.strava_skip_hevy ? "checked" : ""}>
                    <label for="strava_skip_hevy">Skip Strava activities already in Hevy</label>
                </div>
            `;
        }

        // A broken connection is kept (and flagged) rather than cleared, so offer the way
        // back in alongside the usual controls.
        var reconnectStravaButton = "";
        if(stravaHealth && stravaHealth.status == "auth_failed") {
            reconnectStravaButton = `<button onclick="window.location.href='${stravaOauth}';" class="btn integration-btn" type="button">Reconnect Strava</button>`;
        }

        stravaHTML = `
            ${integrationAlertHTML("Strava", stravaHealth)}

            <p class="settings-text">
                Strava is connected. Exercises sync automatically every hour. Be careful to only log your sessions to either Strava or {{.appName}}.
            </p>

            <div class="field-check-group">
                <div class="field-check">
                    <input type="checkbox" id="strava_public" name="strava_public" onchange="updateAccountValue('strava_public');" ${userObject.strava_public ? "checked" : ""}>
                    <label for="strava_public">Show my Strava on my profile</label>
                </div>
                ${skipHevyOption}
            </div>

            ${goalCountingHintHTML()}

            <div class="btn-group">
                ${reconnectStravaButton}
                <button onclick="syncStrava('${userObject.id}');" class="btn integration-btn" type="button">Sync Strava now</button>
                <button onclick="disconnectStrava('${userObject.id}');" class="btn btn--danger integration-btn" type="button">Disconnect Strava</button>
            </div>
        `;
    }

    document.getElementById("strava-wrapper").innerHTML = stravaHTML;
    setSectionStatus("strava", userObject.strava_connected, stravaHealth);
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

function renderHevySection(userObject) {
    var hevyHTML = `
        <p class="settings-text">
            Connect Hevy to sync your workouts automatically. Your Hevy API key is found under Settings in the Hevy app and requires a Hevy PRO subscription.
        </p>

        <div class="btn-group">
            <input id="hevy_api_key" type="password" placeholder="Hevy API key" autocomplete="off" aria-label="Hevy API key" class="u-w-16">
            <button onclick="setHevy('${userObject.id}');" class="btn btn--primary integration-btn" type="button">Connect Hevy</button>
        </div>
    `;

    var hevyHealth = integrationHealthFor(userObject, "hevy");
    if(userObject.hevy_connected) {
        hevyHTML = `
            ${integrationAlertHTML("Hevy", hevyHealth, "Paste a new API key below (Hevy PRO is required)")}

            <p class="settings-text">
                Hevy is connected. Workouts sync automatically. Be careful to only log your sessions to either Hevy or {{.appName}}.
            </p>

            <div class="field-check">
                <input type="checkbox" id="hevy_public" name="hevy_public" onchange="updateAccountValue('hevy_public');" ${userObject.hevy_public ? "checked" : ""}>
                <label for="hevy_public">Show my Hevy on my profile</label>
            </div>

            ${goalCountingHintHTML()}

            <div class="btn-group">
                <input id="hevy_api_key" type="password" placeholder="Replace Hevy API key" autocomplete="off" aria-label="Replace Hevy API key" class="u-w-16">
                <button onclick="setHevy('${userObject.id}');" class="btn integration-btn" type="button">Update key</button>
            </div>

            <div class="btn-group">
                <button onclick="syncHevy('${userObject.id}');" class="btn integration-btn" type="button">Sync Hevy now</button>
                <button onclick="disconnectHevy('${userObject.id}');" class="btn btn--danger integration-btn" type="button">Disconnect Hevy</button>
            </div>
        `;
    }

    document.getElementById("hevy-wrapper").innerHTML = hevyHTML;
    setSectionStatus("hevy", userObject.hevy_connected, hevyHealth);
}

// setSectionStatus puts a small status tag after a connection section's title, so the state
// reads without expanding it: connected, or a signal colour when it has stopped working.
// Nothing is shown while disconnected.
function setSectionStatus(key, connected, health) {
    var slot = document.getElementById("section-status-" + key);
    if(!slot) return;
    slot.innerHTML = connectionStatusTagHTML(connected, health);
}

// connectionStatusTagHTML maps a connection's health (see integrationAlertHTML) onto a tag.
function connectionStatusTagHTML(connected, health) {
    if(!connected) {
        return "";
    }
    if(!health || !health.status || health.status == "ok") {
        return '<span class="meta-tag meta-tag--success">Connected</span>';
    }
    if(health.status == "auth_failed" || health.status_reason == "not_allowlisted" || health.status_reason == "setup_incomplete") {
        return '<span class="meta-tag meta-tag--error">Needs attention</span>';
    }
    return '<span class="meta-tag meta-tag--warning">Not responding</span>';
}

// --- Goal counting ----------------------------------------------------------
// Everything imported counts toward the goal by default; the user picks the exceptions. The UI
// is an exclusion multi-select: a search box + dropdown to add an activity type, and the chosen
// exclusions shown as removable chips. Storage is still per-type (counts_toward_goal=false), so
// an excluded type is simply a stored `false`.

// Latest activity-type list from the API: {action_id, action_name, counts_toward_goal}.
var goalCountingSettings = [];
var goalExcludeOutsideCloseRegistered = false;

// renderGoalCountingSection fetches the per-activity-type preferences and renders the exclusion
// multi-select into the Training section. Excluded types (counts_toward_goal=false) don't count toward the weekly goal for
// new Strava/Hevy imports; existing sessions keep whatever they were.
function renderGoalCountingSection() {
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
            goalCountingSettings = result.activity_goal_settings || [];
            placeGoalCountingSection();
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/activity-goal-settings");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

function placeGoalCountingSection() {
    var html = `
        <p class="settings-text">
            Everything you import counts toward your weekly goal by default. Add any activity types below that should <strong>not</strong> count — they still show up in your history, they just don't move the needle. This only affects future Strava/Hevy imports; edit an existing session in the workout builder to change it there.
        </p>
        <div class="goal-exclude">
            <div class="goal-exclude-input-wrap">
                <input type="text" id="goal-exclude-search" autocomplete="off" aria-label="Search activity types to exclude" placeholder="Search activity types to exclude…" onkeyup="filterGoalExcludeOptions()" onfocus="showGoalExcludeDropdown(true)">
                <div id="goal-exclude-dropdown" class="goal-exclude-dropdown" style="display: none;"></div>
            </div>
            <div id="goal-exclude-chips" class="goal-exclude-chips"></div>
        </div>
    `;

    var body = document.getElementById("goal-counting-body");
    if(!body) return;
    body.innerHTML = html;
    renderGoalExcludeChips();
    renderGoalExcludeOptions("");
    registerGoalExcludeOutsideClose();
}

// renderGoalExcludeChips renders the currently-excluded types as removable chips.
function renderGoalExcludeChips() {
    var container = document.getElementById("goal-exclude-chips");
    if(!container) return;

    var excluded = goalCountingSettings.filter(function(setting) { return !setting.counts_toward_goal; });
    if(excluded.length === 0) {
        container.innerHTML = '<span class="goal-exclude-none">All activity types count toward your goal.</span>';
        return;
    }
    container.innerHTML = excluded.map(function(setting) {
        return `<span class="goal-exclude-chip">${escapeHTML(setting.action_name)}<button type="button" title="Count this type again" onclick="includeGoalActivity('${setting.action_id}')">×</button></span>`;
    }).join("");
}

// renderGoalExcludeOptions fills the dropdown with the not-yet-excluded types matching the search.
function renderGoalExcludeOptions(filterText) {
    var dropdown = document.getElementById("goal-exclude-dropdown");
    if(!dropdown) return;

    var term = (filterText || "").trim().toLowerCase();
    var available = goalCountingSettings.filter(function(setting) {
        return setting.counts_toward_goal && (term === "" || setting.action_name.toLowerCase().indexOf(term) !== -1);
    });

    if(available.length === 0) {
        dropdown.innerHTML = '<div class="goal-exclude-empty">No matching activity types.</div>';
        return;
    }
    dropdown.innerHTML = available.map(function(setting) {
        return `<div class="goal-exclude-option clickable" onclick="excludeGoalActivity('${setting.action_id}')">${escapeHTML(setting.action_name)}</div>`;
    }).join("");
}

function filterGoalExcludeOptions() {
    var input = document.getElementById("goal-exclude-search");
    showGoalExcludeDropdown(true);
    renderGoalExcludeOptions(input ? input.value : "");
}

function showGoalExcludeDropdown(show) {
    var dropdown = document.getElementById("goal-exclude-dropdown");
    if(dropdown) dropdown.style.display = show ? 'block' : 'none';
}

function excludeGoalActivity(actionID) { updateGoalCounting(actionID, false); }
function includeGoalActivity(actionID) { updateGoalCounting(actionID, true); }

// updateGoalCounting persists one type's preference and re-renders from the authoritative list
// the API returns. The search is cleared and the dropdown closed after a change.
function updateGoalCounting(actionID, countsTowardGoal) {
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
                renderGoalCountingSection();
                return;
            }
            goalCountingSettings = result.activity_goal_settings || [];
            var input = document.getElementById("goal-exclude-search");
            if(input) input.value = "";
            showGoalExcludeDropdown(false);
            renderGoalExcludeChips();
            renderGoalExcludeOptions("");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/activity-goal-settings");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(JSON.stringify({ "action_id": actionID, "counts_toward_goal": countsTowardGoal }));
}

// openGoalCounting expands the Training section (if collapsed) and scrolls to its goal-counting
// part — the target of the "manage exclusions" hints in the Strava/Hevy blocks.
function openGoalCounting() {
    var wrapper = document.getElementById('training-wrapper');
    var title = document.getElementById('goal-counting-title');
    if(!wrapper || !title) return;
    if(wrapper.classList.contains('minimized')) {
        toggleSection('training-wrapper', 'section-button-training');
    }
    title.scrollIntoView({ behavior: 'smooth', block: 'start' });
}

// goalCountingHintHTML is the shared "not everything has to count" line shown in the connected
// Strava/Hevy blocks, linking to the goal-counting part of the Training section.
function goalCountingHintHTML() {
    return `
        <p class="goal-count-hint">
            Not every activity has to count — pick which types are excluded under <a href="#" class="goal-count-hint-link" onclick="openGoalCounting(); return false;">Training</a>.
        </p>
    `;
}

// registerGoalExcludeOutsideClose closes the dropdown on any click outside the search box (added
// once for the page — the section re-renders often, but the listener must not stack).
function registerGoalExcludeOutsideClose() {
    if(goalExcludeOutsideCloseRegistered) return;
    goalExcludeOutsideCloseRegistered = true;
    document.addEventListener('click', function(e) {
        if(!e.target.closest('.goal-exclude-input-wrap')) {
            showGoalExcludeDropdown(false);
        }
    });
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
        <p class="settings-text">
            Connect Plex to overlay what you listened to onto your workouts. You will be sent to Plex to approve the connection.
        </p>

        <div class="btn-group">
            <button onclick="connectPlex();" class="btn btn--primary integration-btn" type="button">Connect Plex</button>
        </div>
    `;

    if(connection && connection.connected) {
        var serverValue = connection.server_url ? escapeHTML(connection.server_url) : "";
        var serverHint = connection.server_url
            ? "If syncing fails, your server may be behind a reverse proxy — set the URL you actually reach it on."
            : "No server auto-detected. Enter the URL you reach Plex on.";

        plexHTML = `
            ${integrationAlertHTML("Plex", connection)}

            <p class="settings-text">
                Plex is connected. Your listening history is matched onto activities by time.
            </p>

            <div class="field">
                <label for="plex_server_url" class="field-label">Server URL</label>
                <input id="plex_server_url" type="text" placeholder="https://plex.example.com" autocomplete="off" value="${serverValue}">
                <span class="field-hint">${serverHint}</span>
            </div>

            <div class="btn-group">
                <button onclick="savePlexServerURL();" class="btn integration-btn" type="button">Save server URL</button>
            </div>

            <div class="btn-group">
                <button onclick="connectPlex();" class="btn integration-btn" type="button">Reconnect Plex</button>
                <button onclick="disconnectPlex();" class="btn btn--danger integration-btn" type="button">Disconnect Plex</button>
            </div>
        `;
    }

    document.getElementById("plex-wrapper").innerHTML = plexHTML;
    setSectionStatus("plex", connection && connection.connected, connection);
}

// integrationAlertHTML renders a notice for a connected service that has stopped
// working, or nothing when it is fine. health carries status / status_reason /
// failing_since — a media connection object, or an entry of the user's
// integration_health (see docs/integration-health.md). fixHint says how to fix a
// rejected credential in this section.
function integrationAlertHTML(providerName, health, fixHint) {
    if(!health || !health.status || health.status == "ok") {
        return "";
    }

    var since = "";
    if(health.failing_since) {
        since = " since " + GetDateString(new Date(health.failing_since), false);
    }

    var message = "";
    var modifier = "";
    if(health.status_reason == "not_allowlisted") {
        message = `${providerName} doesn't allow this account yet${since}. Ask the admin to add it to the ${providerName} app — reconnecting won't help.`;
    } else if(health.status_reason == "setup_incomplete") {
        message = `${providerName} isn't fully set up${since}: no server or account was found. Enter the server URL below, or reconnect.`;
    } else if(health.status == "auth_failed") {
        message = `${providerName} stopped accepting the connection${since}. ${fixHint || "Reconnect below"}; history missed in the meantime is fetched automatically.`;
    } else {
        message = `${providerName} hasn't responded${since}. History will catch up once it's reachable again.`;
        modifier = " integration-alert--unavailable";
    }

    return `
        <p class="integration-alert${modifier}">
            ${message}
        </p>
    `;
}

// integrationHealthFor returns the user's own health entry for a provider, if any.
function integrationHealthFor(userObject, provider) {
    if(!userObject.integration_health) {
        return null;
    }
    return userObject.integration_health[provider] || null;
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
        <p class="settings-text">
            Connect Spotify to overlay what you listened to onto your workouts. Spotify only keeps the last ~24 hours of listening, so connect it before (or soon after) you train.
        </p>

        <div class="btn-group">
            <button onclick="connectSpotify();" class="btn btn--primary integration-btn" type="button">Connect Spotify</button>
        </div>
    `;

    if(connection && connection.connected) {
        spotifyHTML = `
            ${integrationAlertHTML("Spotify", connection)}

            <p class="settings-text">
                Spotify is connected. Recent listening is matched onto activities by time. Because Spotify only exposes the last ~24 hours, older workouts can't be back-filled.
            </p>

            <div class="btn-group">
                <button onclick="connectSpotify();" class="btn integration-btn" type="button">Reconnect Spotify</button>
                <button onclick="disconnectSpotify();" class="btn btn--danger integration-btn" type="button">Disconnect Spotify</button>
            </div>
        `;
    }

    document.getElementById("spotify-wrapper").innerHTML = spotifyHTML;
    setSectionStatus("spotify", connection && connection.connected, connection);
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
    var connected = connection && connection.connected;
    var serverValue = connected && connection.server_url ? escapeHTML(connection.server_url) : "";

    var introHTML = connected
        ? `${integrationAlertHTML("Audiobookshelf", connection, "Paste a fresh API token below")}
           <p class="settings-text">Audiobookshelf is connected. Your listening history is matched onto activities by time.</p>`
        : `<p class="settings-text">Connect Audiobookshelf to overlay the audiobooks and podcasts you listened to onto your workouts. Enter your server URL and an API token from your Audiobookshelf account settings.</p>`;

    var disconnectHTML = connected
        ? `<button onclick="disconnectAudiobookshelf();" class="btn btn--danger integration-btn" type="button">Disconnect</button>`
        : "";

    var absHTML = `
        ${introHTML}

        <div class="field">
            <label for="abs_server_url" class="field-label">Server URL</label>
            <input id="abs_server_url" type="text" placeholder="https://abs.example.com" autocomplete="off" value="${serverValue}">
        </div>

        <div class="field">
            <label for="abs_token" class="field-label">API token</label>
            <input id="abs_token" type="password" placeholder="${connected ? "New API token (to update)" : "API token"}" autocomplete="off" value="">
        </div>

        <div class="btn-group">
            <button onclick="connectAudiobookshelf();" class="btn ${connected ? "" : "btn--primary "}integration-btn" type="button">${connected ? "Reconnect" : "Connect"}</button>
            ${disconnectHTML}
        </div>
    `;

    document.getElementById("audiobookshelf-wrapper").innerHTML = absHTML;
    setSectionStatus("audiobookshelf", connected, connection);
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

function leaveSeason() {
    alert("Doesn't work yet :(");
}

function deleteAccount() {
    alert("Doesn't work yet :(");
}

function PlaceSubscriptionData(subscription) {

    document.getElementById("notification-reminder-toggle").checked = subscription.sunday_alert;
    document.getElementById("notification-achievement-toggle").checked = subscription.achievement_alert;
    document.getElementById("notification-news-toggle").checked = subscription.news_alert;
    document.getElementById("notification-account-toggle").checked = subscription.account_alert;

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

// toggleSection expands or collapses one accordion body and keeps its header's chevron and
// aria-expanded in step.
function toggleSection(divID, buttonID) {
    var section = document.getElementById(divID);
    var chevron = document.getElementById(buttonID);
    var expand = section.classList.contains("minimized");

    section.classList.toggle("minimized", !expand);
    section.classList.toggle("expand", expand);
    section.style.display = expand ? 'flex' : 'none';
    chevron.src = expand ? "assets/chevron-down.svg" : "assets/chevron-right.svg";
    chevron.closest(".account-section-tab").setAttribute("aria-expanded", expand ? "true" : "false");
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
                <button type="button" class="btn btn--sm" onclick="selectWheelColor('color', null)">Auto</button>
            </div>
        </div>

        <div class="wheel-option">
            <label>Border</label>
            <div id="wheel-swatches-border" class="wheel-swatches">${swatches('border', wheelState.border)}</div>
            <div class="wheel-controls">
                <input type="color" id="wheel-border-custom" value="${borderCustom}" onchange="selectWheelColor('border', this.value)">
                <button type="button" class="btn btn--sm" onclick="selectWheelColor('border', null)">None</button>
            </div>
        </div>

        <div class="wheel-option">
            <label for="wheel-emoji-input">Emoji</label>
            <div class="wheel-controls">
                <input type="text" id="wheel-emoji-input" maxlength="16" placeholder="🔥" value="${wheelState.emoji ? escapeHTML(wheelState.emoji) : ""}" oninput="previewWheelEmoji(this.value)" onchange="saveWheelEmoji(this.value)">
                <button type="button" class="btn btn--sm" onclick="document.getElementById('wheel-emoji-input').value=''; saveWheelEmoji('');">Clear</button>
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
            <div class="field-check pat-option">
                <input type="checkbox" id="pat_admin" name="pat_admin">
                <label for="pat_admin">Include admin access</label>
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

    <div id="pat-new-token" class="u-mt-1"></div>

    <div id="pat-list" class="u-mt-1 u-w-full">Loading...</div>
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
        <button class="btn u-w-8 u-mt-sm" type="button" onclick="copyPAT()">Copy</button>
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