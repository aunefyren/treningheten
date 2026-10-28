// The season and goal the page shows, kept so saved workouts can be turned on and off and
// the week re-read without reloading the page.
var currentSeason = null;
var currentGoal = null;

function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        if(login_data.error === "You must verify your account.") {
            load_verify_account();
            return;
        }

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
        var admin = false;
    }

    console.log(user_id)

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module hero" id="top-module">

                        <!-- Weekly-goal ring: the app's core loop as the hero. The ring fills
                             blue with this week's completion %, snaps green at 100%, and rests
                             (no arc, "Sick leave") when the viewed season is on sick leave this
                             week. Saved workouts that make up the week show as a second, paler
                             segment after the arc. Hidden until a season/week populates it
                             (placeWeekProgress). -->
                        <div class="hero-ring" id="hero-ring" style="display: none;">
                            <div class="hero-ring-figures">
                                <span class="hero-ring-count hero-ring-week"><a id="workout_this_week">0</a> / <a id="goal_this_week">0</a></span>
                                <span class="hero-ring-caption hero-ring-week" id="hero-ring-caption">this week</span>
                                <span class="hero-ring-rest hero-ring-resting">Sick leave</span>
                                <span class="hero-ring-caption hero-ring-resting">Streak kept</span>
                            </div>
                        </div>

                        <p class="hero-rest-note" id="hero-rest-note" style="display: none;"></p>

                        <!-- Saved workouts (docs/flexible-workouts.md): the chip says what you hold
                             and when it runs out, and opens the explanation; the note under it says
                             what this week does with them, with Undo. Both stay hidden in a season
                             that doesn't save workouts. -->
                        <button type="button" class="saved-chip" id="saved-chip" onclick="openSavedWorkouts();" style="display: none;"></button>
                        <div class="hero-saved-note" id="hero-saved-note" style="display: none;"></div>

                        <!-- The nav already shows the app name; the ring + CTA carry the hero, so no
                             redundant title/subtitle here. #front-page-text is used for the logged-out
                             prompt only (hidden when logged in). -->
                        <div class="text-body u-text-center" id="front-page-text">

                        </div>

                        <div id="log-in-button" style="margin-top: 2em; display: none; width: 10em;">
                            <button class="btn" id="update-button" type="submit" href="#" onclick="window.location = '/login';">Log in</button>
                        </div>

                        <div class="module" id="barbell-gif" style="display: none;">
                            <img src="/assets/images/barbell.gif">
                        </div>

                        <div class="ai-button-row">
                            <div id="ai-message-inline" style="display: none;"></div>
                            <button type="submit" onclick="" id="add-exercise-button" class="btn btn--primary" style="display: none;"><img src="assets/plus.svg" class="color-invert">Start new workout</button>
                        </div>

                        <div id="integration-banner" class="integration-banner" style="display: none;"></div>

                        <div id="push-prompt" class="push-prompt" style="display: none;"></div>

                    </div>

                    <div class="module" id="ongoingseason" style="display: none;">

                        <div class="modules">

                            <div id="exercises" class="exercises">

                                <div class="week_days" id='calendar'>

                                    <div class="form-group" id="day_1_group">
                                        <div class="day-check">
                                            <label class="u-m-0" for="day_1_check" title="Have you been working out?">Monday</label>
                                            <div class="number-box" id="day_1_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_1_buttons">
                                                <img src="assets/minus.svg" class="btn btn--icon clickable" onclick="DecreaseNumberInput('day_1_check', 0, 3);">
                                                <img src="assets/plus.svg" class="btn btn--icon clickable" onclick="IncreaseNumberInput('day_1_check', 0, 3);">
                                                <img src="assets/edit-3.svg" class="btn btn--icon clickable" onclick="EditExercise(1);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_1_note" name="day_1_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_1_date">
                                        <input type="hidden" value="" id="day_1_id">
                                    </div>

                                    <hr class="u-m-1">

                                    <div class="form-group" id="day_2_group">
                                        <div class="day-check">
                                            <label class="u-m-0" for="day_2_check" title="Have you been working out?">Tuesday</label>
                                            <div class="number-box" id="day_2_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_2_buttons">
                                                <img src="assets/minus.svg" class="btn btn--icon clickable" onclick="DecreaseNumberInput('day_2_check', 0, 3);">
                                                <img src="assets/plus.svg" class="btn btn--icon clickable" onclick="IncreaseNumberInput('day_2_check', 0, 3);">
                                                <img src="assets/edit-3.svg" class="btn btn--icon clickable" onclick="EditExercise(2);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_2_note" name="day_2_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_2_date">
                                        <input type="hidden" value="" id="day_2_id">
                                    </div>

                                    <hr class="u-m-1">

                                    <div class="form-group" id="day_3_group">
                                        <div class="day-check">
                                            <label class="u-m-0" for="day_3_check" title="Have you been working out?">Wednesday</label>
                                            <div class="number-box" id="day_3_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_3_buttons">
                                                <img src="assets/minus.svg" class="btn btn--icon clickable" onclick="DecreaseNumberInput('day_3_check', 0, 3);">
                                                <img src="assets/plus.svg" class="btn btn--icon clickable" onclick="IncreaseNumberInput('day_3_check', 0, 3);">
                                                <img src="assets/edit-3.svg" class="btn btn--icon clickable" onclick="EditExercise(3);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_3_note" name="day_3_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_3_date">
                                        <input type="hidden" value="" id="day_3_id">
                                    </div>

                                    <hr class="u-m-1">

                                    <div class="form-group" id="day_4_group">
                                        <div class="day-check">
                                            <label class="u-m-0" for="day_4_check" title="Have you been working out?">Thursday</label>
                                            <div class="number-box" id="day_4_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_4_buttons">
                                                <img src="assets/minus.svg" class="btn btn--icon clickable" onclick="DecreaseNumberInput('day_4_check', 0, 3);">
                                                <img src="assets/plus.svg" class="btn btn--icon clickable" onclick="IncreaseNumberInput('day_4_check', 0, 3);">
                                                <img src="assets/edit-3.svg" class="btn btn--icon clickable" onclick="EditExercise(4);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_4_note" name="day_4_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_4_date">
                                        <input type="hidden" value="" id="day_4_id">
                                    </div>

                                    <hr class="u-m-1">

                                    <div class="form-group" id="day_5_group">
                                        <div class="day-check">
                                            <label class="u-m-0" for="day_5_check" title="Have you been working out?">Friday</label>
                                            <div class="number-box" id="day_5_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_5_buttons">
                                                <img src="assets/minus.svg" class="btn btn--icon clickable" onclick="DecreaseNumberInput('day_5_check', 0, 3);">
                                                <img src="assets/plus.svg" class="btn btn--icon clickable" onclick="IncreaseNumberInput('day_5_check', 0, 3);">
                                                <img src="assets/edit-3.svg" class="btn btn--icon clickable" onclick="EditExercise(5);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_5_note" name="day_5_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_5_date">
                                        <input type="hidden" value="" id="day_5_id">
                                    </div>

                                    <hr class="u-m-1">

                                    <div class="form-group" id="day_6_group">
                                        <div class="day-check">
                                            <label class="u-m-0" for="day_6_check" title="Have you been working out?">Saturday</label>
                                            <div class="number-box" id="day_6_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_6_buttons">
                                                <img src="assets/minus.svg" class="btn btn--icon clickable" onclick="DecreaseNumberInput('day_6_check', 0, 3);">
                                                <img src="assets/plus.svg" class="btn btn--icon clickable" onclick="IncreaseNumberInput('day_6_check', 0, 3);">
                                                <img src="assets/edit-3.svg" class="btn btn--icon clickable" onclick="EditExercise(6);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_6_note" name="day_6_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_6_date">
                                        <input type="hidden" value="" id="day_6_id">
                                    </div>

                                    <hr class="u-m-1">

                                    <div class="form-group" id="day_7_group">
                                        <div class="day-check">
                                            <label class="u-m-0" for="day_7_check" title="Have you been working out?">Sunday</label>
                                            <div class="number-box" id="day_7_check">
                                                0
                                            </div>
                                            <div class="day-buttons" id="day_7_buttons">
                                                <img src="assets/minus.svg" class="btn btn--icon clickable" onclick="DecreaseNumberInput('day_7_check', 0, 3);">
                                                <img src="assets/plus.svg" class="btn btn--icon clickable" onclick="IncreaseNumberInput('day_7_check', 0, 3);">
                                                <img src="assets/edit-3.svg" class="btn btn--icon clickable" onclick="EditExercise(7);">
                                            </div>
                                        </div>
                                        <div class="day-note">
                                            <textarea class="day-note-area" id="day_7_note" name="day_7_note" rows="3" cols="33" placeholder="Notes">
                                            </textarea>
                                        </div>
                                        <input type="hidden" value="" id="day_7_date">
                                        <input type="hidden" value="" id="day_7_id">
                                    </div>

                                    <input type="hidden" value="" id="calendar_user_id">
                                    <input type="hidden" value="" id="calendar_season_id">

                                    <button type="submit" onclick="update_exercises(false, 0);" id="goal_amount_button" class="btn btn--block" style="margin-bottom: 0em; transition: 1s;"><img src="assets/done.svg" class="color-invert">Save</button>

                                    <!-- Week actions: sick leave, and saved workouts when they would make up
                                         the week. The hint says what using them does before you choose. -->
                                    <div class="week-actions" id="week-actions" style="display: none;">
                                        <div class="btn-group">
                                            <button type="button" class="btn btn--ghost btn--sm" id="sickleave-button" onclick="useSickLeave();" style="display: none;">Use sick leave</button>
                                            <button type="button" class="btn btn--ghost btn--sm" id="saved-workouts-button" onclick="toggleSavedWorkouts();" style="display: none;">Use saved workouts</button>
                                        </div>
                                        <p class="week-actions-hint" id="saved-workouts-hint" style="display: none;"></p>
                                    </div>

                                </div>

                            </div>

                            <div class="module-two">

                                <div id="season-module" class="season">

                                    <div id="season-progress-bar-wrapper" class="season-progress-bar-wrapper">
                                        <div id="season-progress-bar" class="season-progress-bar">
                                            <div class="calender_status unselectable" id="season_status">
                                                <a id="weeks_so_far">...</a>
                                                /
                                                <a id="weeks_total">...</a>
                                                weeks
                                            </div>
                                        </div>
                                    </div>

                                    <h3 id="season_title">Loading...</h3>
                                    <p id="season_desc" class="u-text-center">...</p>

                                    <p id="season_start_title" class="u-mt-1">Season start: <a id="season_start">...</a></p>
                                    <p id="season_end_title">Season end: <a id="season_end">...</a></p>

                                    <p id="week_goal_title" style="margin: 1em 0 0 0;">Weekly goal: <b><a id="week_goal">0</a></b></p>
                                    <p id="goal_sickleave_title">Sick leave left: <b><a id="goal_sickleave">0</a></b></p>
                                    <div id="goal_saved_title" class="season-saved" style="display: none;">
                                        <p class="u-m-0">Saved workouts: <b><a id="goal_saved">0</a></b></p>
                                        <p class="season-saved-rule" id="goal_saved_rule"></p>
                                    </div>

                                    <p id="prize-title" class="u-mt-1">Potential prize:</p>
                                    <div class="prize-wrapper" id="prize-wrapper">
                                        <div id="prize-text" class="prize-text">...</div>
                                    </div>

                                    <hr id="seasonDivider" style="display:none;">

                                    <div id="potentialSeasonsWrapper" style="display:none;">
                                    </div>

                                    <div id="countdownSeasonsWrapper" style="display:none;">
                                    </div>

                                </div>

                                <div id="debt-module" class="debt-module" style="display: none;">

                                    <h3 id="debt-module-title">Prizes</h3>

                                    <div id="debt-module-notifications" class="debt-module-notifications">
                                    </div>

                                </div>

                                <div id="current-week" class="current-week">

                                    <h3 id="current-week-title">Current week</h3>

                                    <div id="current-week-users" class="current-week-users">
                                        <div class="fp-skeleton">
                                            <div class="fp-skeleton-row">
                                                <div class="fp-skeleton-user"></div>
                                                <div class="fp-skeleton-user"></div>
                                                <div class="fp-skeleton-user"></div>
                                            </div>
                                            <div class="fp-skeleton-row">
                                                <div class="fp-skeleton-user"></div>
                                                <div class="fp-skeleton-user"></div>
                                                <div class="fp-skeleton-user"></div>
                                            </div>
                                            <div class="fp-skeleton-row">
                                                <div class="fp-skeleton-user"></div>
                                                <div class="fp-skeleton-user"></div>
                                                <div class="fp-skeleton-user"></div>
                                            </div>
                                        </div>
                                    </div>


                                </div>

                            </div>

                            <div class="module-two">

                                <div id="activities" class="activities">

                                    <h3 class="u-m-2">Activity feed</h3>

                                    <div id="activities-week" class="activities-week">
                                        <p class="u-mb-2 u-text-center">No shared workouts yet this week.</p>
                                    </div>

                                </div>
                                

                                <div id="leaderboard" class="leaderboard">

                                    <h3 class="u-m-2">Previous weeks</h3>

                                    <div id="leaderboard-weeks" class="leaderboard-weeks">
                                        <div class="fp-skeleton">
                                            <div class="fp-skeleton-week-row">
                                                <div class="skeleton-block fp-skeleton-week-label"></div>
                                                <div class="fp-skeleton-week-grid">
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                </div>
                                            </div>
                                            <div class="fp-skeleton-week-row">
                                                <div class="skeleton-block fp-skeleton-week-label"></div>
                                                <div class="fp-skeleton-week-grid">
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                    <div class="fp-skeleton-week-user"></div>
                                                </div>
                                            </div>
                                        </div>
                                    </div>

                                </div>

                            </div>


                        </div>

                    </div>

                    <div class="module hero" id="unspun-wheel" style="display: none;">

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Welcome to the frontpage!';
    clearResponse();

    // Check URL for parameters
    var activeSeason = null;
    const query_string = window.location.search;
    parameters = get_url_parameters(query_string)
    console.log(parameters)
    if(parameters != false && "season" in parameters) {
        activeSeason = decodeURI(parameters.season)
    } else {
        console.log("Season not found in parameters.")
    }

    if(result !== false) {
        showLoggedInMenu();
        get_season(user_id, true, activeSeason);
        document.getElementById('front-page-text').style.display = 'none';
        if ("{{.ollamaEnabled}}" == "true") {
            load_ai_message();
        }
        loadIntegrationBanner();
        maybeShowPushPrompt();
    } else {
        showLoggedOutMenu();
        document.getElementById('front-page-text').innerHTML = 'Log in to use the platform.';
        document.getElementById('log-in-button').style.display = 'inline-block';
    }
}

// Key used to remember which integration outages the user hid from the front-page banner:
// an object of provider → failing_since. A new breakage has a new failing_since, so it shows
// again. Per device, like the push prompt.
const INTEGRATION_BANNER_HIDDEN_KEY = 'trh_integration_banner_hidden';

// loadIntegrationBanner shows a notice for each of the user's connected services that has
// stopped working (docs/integration-health.md). A rejected credential stays until the user
// reconnects; an outage of the provider can be hidden, since it clears on its own.
function loadIntegrationBanner() {
    var banner = document.getElementById('integration-banner');
    if (!banner) return;

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState != 4) return;
        try {
            var result = JSON.parse(this.responseText);
            if (this.status === 200 && result.integrations) {
                placeIntegrationBanner(result.integrations);
            }
        } catch(e) {
            console.log('Failed to load integration health. Error: ' + e);
        }
    };
    xhttp.open("GET", api_url + "auth/integrations/health", true);
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

function placeIntegrationBanner(integrations) {
    var banner = document.getElementById('integration-banner');
    var hidden = hiddenIntegrationOutages();

    var html = "";
    integrations.forEach(function(integration) {
        if (!/^[a-z]+$/.test(integration.provider)) return;
        var text = integrationNoticeText(integration.name, integration, null, null);
        if (!text) return;

        var actions = "";
        if (text.fixable) {
            actions = `<a class="btn btn--sm" href="/account#${integration.provider}">Fix in settings</a>`;
        } else if (text.severity == "warning") {
            if (hidden[integration.provider] === integration.failing_since) return;
            actions = `<button type="button" class="btn btn--ghost btn--sm" onclick="hideIntegrationOutage('${integration.provider}', '${integration.failing_since}');">Hide</button>`;
        }
        html += `<div id="integration-notice-${integration.provider}">${integrationNoticeHTML(text, actions)}</div>`;
    });

    banner.innerHTML = html;
    banner.style.display = html ? 'flex' : 'none';
}

// hideIntegrationOutage hides one provider's outage notice until it breaks again.
function hideIntegrationOutage(provider, failingSince) {
    var notice = document.getElementById('integration-notice-' + provider);
    if (notice) notice.remove();

    var banner = document.getElementById('integration-banner');
    if (banner && !banner.children.length) banner.style.display = 'none';

    var hidden = hiddenIntegrationOutages();
    hidden[provider] = failingSince;
    try {
        localStorage.setItem(INTEGRATION_BANNER_HIDDEN_KEY, JSON.stringify(hidden));
    } catch (e) {
        // localStorage unavailable; the notice simply returns next visit.
    }
}

function hiddenIntegrationOutages() {
    try {
        return JSON.parse(localStorage.getItem(INTEGRATION_BANNER_HIDDEN_KEY)) || {};
    } catch (e) {
        return {};
    }
}

// Key used to remember that the user dismissed the front-page notification prompt.
// Stored per-device (localStorage), which matches push being a per-device feature.
const PUSH_PROMPT_DISMISSED_KEY = 'trh_push_prompt_dismissed';

// maybeShowPushPrompt shows a small, dismissible card encouraging the user to enable
// push notifications, but only when it makes sense: the browser supports push, the
// user has not already decided (permission is still "default"), there is no existing
// subscription, and they have not previously dismissed the prompt.
function maybeShowPushPrompt() {

    var prompt = document.getElementById('push-prompt');
    if (!prompt) {
        return;
    }

    // Browser capability check.
    if (!('serviceWorker' in navigator) || !('PushManager' in window) || !('Notification' in window)) {
        console.log('Push prompt: hidden — browser does not support push notifications (needs a secure context: HTTPS or localhost).');
        return;
    }

    // Only "denied" is a dead end — nothing we can do without the user re-allowing in
    // browser settings. "default" (undecided) and "granted" (allowed, but maybe not
    // subscribed on THIS device) both still warrant a prompt; the subscription check
    // below decides whether there is actually anything to offer.
    if (Notification.permission === 'denied') {
        console.log('Push prompt: hidden — Notification.permission is "denied". The user must re-allow notifications in browser settings.');
        return;
    }

    // Respect a previous dismissal.
    try {
        if (localStorage.getItem(PUSH_PROMPT_DISMISSED_KEY) === 'true') {
            console.log('Push prompt: hidden — previously dismissed. Run localStorage.removeItem("' + PUSH_PROMPT_DISMISSED_KEY + '") to re-test.');
            return;
        }
    } catch (e) {
        // localStorage unavailable (e.g. private mode); just proceed to show it.
    }

    navigator.serviceWorker.ready.then(function(registration) {
        return registration.pushManager.getSubscription().then(function(subscription) {
            if (subscription) {
                // Already subscribed on this device; nothing to prompt.
                console.log('Push prompt: hidden — this device already has a push subscription.');
                return;
            }

            console.log('Push prompt: showing (permission "' + Notification.permission + '").');

            // When permission is already granted, the user allowed notifications before
            // but this device has no subscription, so frame it as re-enabling here.
            var promptMessage = (Notification.permission === 'granted')
                ? 'Notifications are off on this device. Turn them back on?'
                : 'Want reminders and updates? Enable notifications.';

            prompt.innerHTML = `
                <div class="push-prompt-text">
                    <img src="assets/bell.svg" class="push-prompt-icon">
                    <span>${promptMessage}</span>
                </div>
                <div class="push-prompt-actions">
                    <button type="submit" class="btn btn--sm" onclick="enablePushFromPrompt(); return false;">Enable</button>
                    <button type="submit" class="btn btn--ghost btn--sm" onclick="dismissPushPrompt(); return false;">Not now</button>
                </div>
            `;
            prompt.style.display = 'flex';
        });
    }).catch(function(e) {
        console.log('Failed to evaluate push prompt. Error: ' + e);
    });

}

// enablePushFromPrompt asks for notification permission (within the click gesture) and,
// if granted, subscribes with all alert types on. The user can fine-tune them in /account.
function enablePushFromPrompt() {

    var prompt = document.getElementById('push-prompt');

    Notification.requestPermission().then(function(permission) {
        if (permission === 'granted') {
            // sunday_alert, achievement_alert, news_alert, account_alert all on by default.
            register_push(jwt, '{{ .vapidPublicKey }}', true, true, true, true);
        } else {
            error('Notifications were not enabled.');
        }
        if (prompt) {
            prompt.style.display = 'none';
        }
    }).catch(function(e) {
        console.log('Failed to request notification permission. Error: ' + e);
        if (prompt) {
            prompt.style.display = 'none';
        }
    });

}

// dismissPushPrompt hides the prompt and remembers the choice so it does not return.
function dismissPushPrompt() {

    var prompt = document.getElementById('push-prompt');
    if (prompt) {
        prompt.style.display = 'none';
    }

    try {
        localStorage.setItem(PUSH_PROMPT_DISMISSED_KEY, 'true');
    } catch (e) {
        // localStorage unavailable; the prompt simply may reappear next visit.
    }

}

function load_ai_message() {
    var el = document.getElementById('ai-message-inline');
    if (!el) return;

    el.style.display = 'block';
    el.innerHTML = `
        <div class="ai-message-card">
            <span class="ai-message-label">AI</span>
            <div class="skeleton-block ai-message-skeleton"></div>
        </div>`;

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState != 4) return;
        try {
            var result = JSON.parse(this.responseText);
            if (this.status === 200 && result.data) {
                el.innerHTML = `
                    <div class="ai-message-card">
                        <span class="ai-message-label">AI</span>
                        <span class="ai-message-text">${result.data}</span>
                    </div>`;
            } else {
                el.style.display = 'none';
            }
        } catch(e) {
            el.style.display = 'none';
        }
    };
    xhttp.open("GET", "/api/auth/ai/frontpage", true);
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}

function get_season(user_id, loadingMessage, activeSeason){

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
                error_splash_image();

            } else {

                clearResponse();

                document.getElementById("ongoingseason").style.display = "flex"
                getDebtOverview();

                // The activity feed spans every season you have ever been in, so it is
                // loaded regardless of whether there is an ongoing season to show.
                getActivities();

                // If one or more seasons were found
                if(result.seasons.length > 0) {
                    var season = null;
                    var seasonAlternatives = [];
                    var activeSeasonFound = false;

                    if(activeSeason) {
                        result.seasons.forEach(currentSeason => {
                            if(currentSeason.id === activeSeason) {
                                season = currentSeason;
                                activeSeasonFound = true;
                            } else {
                                seasonAlternatives.push(currentSeason)
                            }
                        });
                    }
                    if(!activeSeasonFound) {
                        season = null;
                        seasonAlternatives = [];
                        for (let index = 0; index < result.seasons.length; index++) {
                            if(index == 0) {
                                season = result.seasons[index];
                            } else {
                                seasonAlternatives.push(result.seasons[index])
                            }
                        }
                    }

                    const url = new URL(window.location.href);
                    url.searchParams.set('season', encodeURI(season.id)); // Set or update ?name=John
                    window.history.replaceState({}, '', url);
                    
                    var goal = null;

                    var user_found = false;
                    for(var i = 0; i < season.goals.length; i++) {
                        if(season.goals[i].user.id == user_id) {
                            user_found = true
                            goal = season.goals[i]
                            break
                        }
                    }

                    for(var i = 0; i < season.goals.length; i++) {
                        userList[season.goals[i].user.id] = season.goals[i].user
                    }

                    if(user_found) {
                        get_calendar(false, user_id, loadingMessage);
                        place_season(season, user_id, seasonAlternatives);
                        get_leaderboard(season, goal, true, false);
                    }
                } else {
                    get_calendar(false, user_id, loadingMessage);
                    place_season(false, user_id, null);

                    try {
                        document.getElementById('current-week').outerHTML = ""
                        document.getElementById('leaderboard').outerHTML = ""
                    } catch (e) {
                        console.log('Removing divs threw error: ' + e)
                    }

                    document.getElementById('goal_this_week').innerHTML = "0"
                }
            }

        } else {
            if(loadingMessage) {
                info("Loading season...");
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/get-on-going");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_season(season_object, userID, alternatives) {
    if(season_object) {
        document.getElementById("calendar_season_id").value = season_object.id
        document.getElementById("season_desc").innerHTML = season_object.description
        document.getElementById("prize-text").innerHTML = season_object.prize.quantity + " " + season_object.prize.name

        document.getElementById("season_title").outerHTML = `<h3 id="season_title">${season_object.name}</h3>`;

        placeSeasonAlternatives(season_object, userID, alternatives);

        try {
            var date_start = new Date(season_object.start);
            var date_end = new Date(season_object.end);
    
            var date_start_string = GetDateString(date_start, true)
            var date_end_string = GetDateString(date_end, true)
        } catch {
            var date_start_string = "Error"
            var date_end_string = "Error"
        }
        
        document.getElementById("season_start").innerHTML = date_start_string
        document.getElementById("season_end").innerHTML = date_end_string
    
        placeSeasonProgress(date_start, date_end);
        getPotentialSeasons();
        getCountdownSeasons(userID);
    } else {
        document.getElementById("season_title").innerHTML = "No active season"
        document.getElementById("season_desc").innerHTML = "You can join or create a season to start competing."
        getPotentialSeasons();
        getCountdownSeasons(userID);

        try {
            document.getElementById("season_start_title").outerHTML = ""
            document.getElementById("season_end_title").outerHTML = ""
            document.getElementById("week_goal_title").outerHTML = ""
            document.getElementById("goal_sickleave_title").outerHTML = ""
            document.getElementById("prize-title").outerHTML = ""
            document.getElementById("prize-wrapper").outerHTML = ""
        } catch(e) {
            console.log('Removing div\'s threw error: ' + e)
        }

        var currentYear = new Date().getFullYear()  // returns the current yea
        var startOfYear = new Date(currentYear + "-01-01")
        var endOfYear = new Date(currentYear + "-12-24")

        placeSeasonProgress(startOfYear, endOfYear);
    }
}

function get_calendar(fireworks, user_id, loadingMessage){

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

                clearResponse();
                var week = result.week;

                console.log(week);

                console.log("Placing intial week: ")
                place_week(week, fireworks, user_id);

            }

        } else {
            if(loadingMessage) {
                // info("Loading week...");
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/exercises/week");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_week(week, fireworks, user_id) {

    if(fireworks) {
        console.log("Triggered fireworks.")
        trigger_fireworks(1);
    }

    // Sum of exercise to decide fireworks
    fireworks_int = 0
    if(week.days && week.days.length > 0) {
        for(var i = 0; i < week.days.length; i++) {
            if(week.days[i].exercise_interval) {
                fireworks_int += week.days[i].exercise_interval
            }
        }
    }

    document.getElementById("calendar_user_id").value = user_id

    document.getElementById("day_1_check").innerHTML = week.days[0].exercise_interval
    document.getElementById("day_1_note").value = HTMLDecode(week.days[0].note)
    document.getElementById("day_1_date").value = week.days[0].date
    document.getElementById("day_1_id").value = week.days[0].id

    document.getElementById("day_2_check").innerHTML = week.days[1].exercise_interval
    document.getElementById("day_2_note").value = HTMLDecode(week.days[1].note)
    document.getElementById("day_2_date").value = week.days[1].date
    document.getElementById("day_2_id").value = week.days[1].id

    document.getElementById("day_3_check").innerHTML = week.days[2].exercise_interval
    document.getElementById("day_3_note").value = HTMLDecode(week.days[2].note)
    document.getElementById("day_3_date").value = week.days[2].date
    document.getElementById("day_3_id").value = week.days[2].id

    document.getElementById("day_4_check").innerHTML = week.days[3].exercise_interval
    document.getElementById("day_4_note").value = HTMLDecode(week.days[3].note)
    document.getElementById("day_4_date").value = week.days[3].date
    document.getElementById("day_4_id").value = week.days[3].id

    document.getElementById("day_5_check").innerHTML = week.days[4].exercise_interval
    document.getElementById("day_5_note").value = HTMLDecode(week.days[4].note)
    document.getElementById("day_5_date").value = week.days[4].date
    document.getElementById("day_5_id").value = week.days[4].id

    document.getElementById("day_6_check").innerHTML = week.days[5].exercise_interval
    document.getElementById("day_6_note").value = HTMLDecode(week.days[5].note)
    document.getElementById("day_6_date").value = week.days[5].date
    document.getElementById("day_6_id").value = week.days[5].id

    document.getElementById("day_7_check").innerHTML = week.days[6].exercise_interval
    document.getElementById("day_7_note").value = HTMLDecode(week.days[6].note)
    document.getElementById("day_7_date").value = week.days[6].date
    document.getElementById("day_7_id").value = week.days[6].id

    // Find day int
    const now = new Date(Date.now());
    var day = now.getDay();
    if(day == 0) {
        day = 7
    }

    // Add class to current day
    document.getElementById("day_" + day + "_check").classList.add("active-day") 

    // Enable workout buttons
    // Assigned rather than added: place_week runs again after every save, and stacked
    // listeners made one click create several sessions.
    var addExerciseButton = document.getElementById("add-exercise-button")
    addExerciseButton.onclick = function() {
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
                if(result.error) {
                    error(result.error);
                } else {
                    addExercise(result.exercise.id);
                }
            }
        };
        xhttp.withCredentials = true;
        xhttp.open("get", api_url + "auth/exercise-days/week?today=true");
        xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
        xhttp.setRequestHeader("Authorization", jwt);
        xhttp.send();
    };
    addExerciseButton.style.display = 'flex'

    // Place editing icon for exercise
    for(var i = 1; i <= day; i++) {
        document.getElementById("day_" + i + "_buttons").style.display = "flex" 
        console.log("index: " + i)
        let checkObject = document.getElementById(`day_${i}_check`)
        let dayInteger = i;
        checkObject.onclick = function(){EditExercise(dayInteger)};
        checkObject.classList.add('clickable')
    }

    document.getElementById("workout_this_week").innerText  = fireworks_int

    return

}

function update_exercises() {
    var user_id = document.getElementById("calendar_user_id").value

    var day_1_check = document.getElementById("day_1_check").innerHTML
    var day_1_note = document.getElementById("day_1_note").value
    var day_1_date = document.getElementById("day_1_date").value

    var day_2_check = document.getElementById("day_2_check").innerHTML
    var day_2_note = document.getElementById("day_2_note").value
    var day_2_date = document.getElementById("day_2_date").value

    var day_3_check = document.getElementById("day_3_check").innerHTML
    var day_3_note = document.getElementById("day_3_note").value
    var day_3_date = document.getElementById("day_3_date").value

    var day_4_check = document.getElementById("day_4_check").innerHTML
    var day_4_note = document.getElementById("day_4_note").value
    var day_4_date = document.getElementById("day_4_date").value

    var day_5_check = document.getElementById("day_5_check").innerHTML
    var day_5_note = document.getElementById("day_5_note").value
    var day_5_date = document.getElementById("day_5_date").value

    var day_6_check = document.getElementById("day_6_check").innerHTML
    var day_6_note = document.getElementById("day_6_note").value
    var day_6_date = document.getElementById("day_6_date").value

    var day_7_check = document.getElementById("day_7_check").innerHTML
    var day_7_note = document.getElementById("day_7_note").value
    var day_7_date = document.getElementById("day_7_date").value

    var form_obj = {
        "days": [
            {
                "date": day_1_date,
                "note": day_1_note,
                "exercise_interval": Number(day_1_check)
            },
            {
                "date": day_2_date,
                "note": day_2_note,
                "exercise_interval": Number(day_2_check)
            },
            {
                "date": day_3_date,
                "note": day_3_note,
                "exercise_interval": Number(day_3_check)
            },
            {
                "date": day_4_date,
                "note": day_4_note,
                "exercise_interval": Number(day_4_check)
            },
            {
                "date": day_5_date,
                "note": day_5_note,
                "exercise_interval": Number(day_5_check)
            },
            {
                "date": day_6_date,
                "note": day_6_note,
                "exercise_interval": Number(day_6_check)
            },
            {
                "date": day_7_date,
                "note": day_7_note,
                "exercise_interval": Number(day_7_check)
            },
        ],
        "timezone" : Intl.DateTimeFormat().resolvedOptions().timeZone
    };

    var form_data = JSON.stringify(form_obj);

    console.log("Saving new week: ")
    console.log(form_data)

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
                blinkCalendar();

                week = result.week;

                console.log(week);

                new_fireworks_int = 0;
                new_fireworks = false
                for(var i = 0; i < week.days.length; i++) {
                    new_fireworks_int += week.days[i].exercise_interval
                }

                if (new_fireworks_int > fireworks_int) {
                    new_fireworks = true
                }

                console.log("Placing initial week: ")
                place_week(week, new_fireworks, user_id);
                get_season(user_id, false);

                //success(result.message)
            }

        } else {
            //info("Saving week...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises/week");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function get_leaderboard(season, goal, refresh, fireworks){
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
                if(result.leaderboard.length < 1) {
                    return;
                }
                
                this_week = result.leaderboard[0];

                if(result.leaderboard.length > 1) {
                    past_weeks = result.leaderboard.splice(1, result.leaderboard.length-1);
                } else {
                    past_weeks = [];
                }

                currentSeason = season;
                currentGoal = goal;

                place_current_week(this_week);
                placeSickLeaveState(isOnSickLeave(this_week), goal.sickleave_left, season.name);
                placeSavedWorkoutsState(myWeekResult(this_week), season, goal);

                if(refresh) {
                    place_season_details(goal, season);
                    place_leaderboard(past_weeks);
                }
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons/" + season.id + "/weeks");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

// userDisplayName returns the cached user record for an ID, or a safe placeholder.
// userList is built from the current season's goals, so historical week/leaderboard
// results can reference a user who has since left the season; without this guard those
// rows would throw on `.first_name`.
function userDisplayName(userID) {
    var user = userList[userID];
    if(user && user.first_name) {
        return user;
    }
    return { first_name: "Former member", last_name: "" };
}

function place_leaderboard(weeks_array) {

    var leaderboardWeeks = document.getElementById("leaderboard-weeks")

    if(weeks_array.length == 0) {
        leaderboardWeeks.innerHTML = `
            <div id="" class="leaderboard-weeks">
                <p id="" class="u-m-2 u-text-center">No past weeks.</p>
            </div>
        `;
        return;
    }

    // Build the whole board as one string and assign once, rather than reparsing the
    // DOM on every `innerHTML +=`.
    var html = "";

    for(var i = 0; i < weeks_array.length; i++) {
        var week_html = `
            <div class="leaderboard-week" id="">
                <div class="leaderboard-week-number">
                    Week ` + weeks_array[i].week_number + ` (` + weeks_array[i].week_year + `)
                </div>
                <div class="leaderboard-week-results">
        `;

        var results_html = "";

        // Sort users by how much of their weekly goal they completed (highest first),
        // matching the current-week board.
        weeks_array[i].users = weeks_array[i].users.sort((a,b) => b.week_completion - a.week_completion);

        for(var j = 0; j < weeks_array[i].users.length; j++) {
            var completion = weekStatusIcon(weeks_array[i].users[j])

            var onclick_command_str = "return;"
            var clickable_str = ""
            if(weeks_array[i].users[j].debt !== null && weeks_array[i].users[j].debt.winner !== null) {
                onclick_command_str = "location.replace('/wheel?debt_id=" + weeks_array[i].users[j].debt.id + "'); "
                clickable_str = "clickable grey-underline"
                completion += "🎡"
            }


            var result_html = `
            <div class="leaderboard-week-result" id="">
                <div class="leaderboard-week-result-user clickable" onclick="location.href='/users/${weeks_array[i].users[j].user_id}'">
                    ` + userDisplayName(weeks_array[i].users[j].user_id).first_name + `
                </div>
                <div class="leaderboard-week-result-exercise ` + clickable_str  + `" onclick="` + onclick_command_str  + `">
                    ` + completion  + `
                </div>
            </div>
            `;
            results_html += result_html;

        }

        week_html += results_html + `</div></div>`;

        html += week_html;
    }

    leaderboardWeeks.innerHTML = html;

    return
}

function place_season_details(goal, season) {
    document.getElementById("week_goal").innerHTML = goal.exercise_interval
    document.getElementById("goal_this_week").innerText  = goal.exercise_interval
    document.getElementById("goal_sickleave").innerHTML = goal.sickleave_left

    // Saved workouts: how many you hold, and the season's rule in one line. Absent in a
    // season that doesn't save workouts.
    var savedRow = document.getElementById("goal_saved_title");
    if(savedRow) {
        var rule = season ? savedWorkoutsRule(season.carry_over_max, season.carry_over_weeks) : "";
        savedRow.style.display = rule ? "block" : "none";
        document.getElementById("goal_saved_rule").textContent = rule;
        var mine = typeof this_week !== "undefined" ? myWeekResult(this_week) : null;
        document.getElementById("goal_saved").textContent = mine ? mine.carried_in : 0;
    }
}

function place_current_week(week_array) {
    var currentWeekUsers = document.getElementById("current-week-users")

    // Sort users
    week_array.users = week_array.users.sort((a,b) => b.week_completion - a.week_completion);

    document.getElementById('current-week-title').innerHTML = `Current week (${week_array.week_number})`

    // Build the board as one string and assign once. Profile images load directly via
    // <img src> against the cookie-authenticated image endpoint (browser caches/dedupes).
    var html = "";

    for(var i = 0; i < week_array.users.length; i++) {

        var completion = Math.trunc((week_array.users[i].week_completion * 100))
        var transparent = week_array.users[i].sick_leave ? "transparent" : ""
        var current_streak = week_array.users[i].current_streak + streakStatusIcon(week_array.users[i])

        // Saved workouts that make up the rest of the week: the goal counts as met.
        var savedMark = "";
        if(week_array.users[i].carry_used > 0 && !week_array.users[i].sick_leave) {
            savedMark = " " + statusIcon("💾", countLabel(week_array.users[i].carry_used, "saved workout makes", "saved workouts make") + " up the rest");
        }

        if(weekGoalMet(week_array.users[i])) {
            transparent += " bold-font "
        }

        if(week_array.users[i].user_id == user_id) {
            var savedPercentage = 0;
            if(currentGoal && currentGoal.exercise_interval > 0) {
                savedPercentage = Math.round(week_array.users[i].carry_used / currentGoal.exercise_interval * 100);
            }
            placeWeekProgress(completion, week_array.users[i].sick_leave, savedPercentage, week_array.users[i].carry_used)
        }

        var user = userDisplayName(week_array.users[i].user_id);

        html += `
            <div class="current-week-user unselectable" id="">

                <div class="">

                    <div class="" style="font-size: 0.8em;">
                        <b>${user.first_name}</b>
                    </div>

                    <div class="current-week-user-photo" title="` + user.first_name + ` ` + user.last_name + `" onclick="location.href='/users/${week_array.users[i].user_id}'">
                        <img class="current-week-user-photo-img u-fill" src="${profileImageURL(week_array.users[i].user_id, true)}" onerror="${IMAGE_FALLBACK_ONERROR}">
                    </div>
                </div>

                <div class="current-week-user-results">

                    <div class="current-week-user-completion ` + transparent + `" title="How much of the goal for this week is finished.">
                        ` + completion + `%` + savedMark + `
                    </div>

                    <div class="current-week-user-completion" title="How many weeks in a row have been at least 100%.">
                        ` + current_streak + `
                    </div>

                </div>

            </div>
        `;
    }

    currentWeekUsers.innerHTML = html;

    return
}

// isOnSickLeave reports whether the signed-in user has used sick leave for this week in the
// season the week results belong to.
function isOnSickLeave(weekResults) {
    var users = (weekResults && weekResults.users) || [];
    for(var i = 0; i < users.length; i++) {
        if(users[i].user_id == user_id) {
            return users[i].sick_leave === true;
        }
    }
    return false;
}

// placeSickLeaveState reflects the viewed season's sick-leave week on the page: the note under
// the ring, and whether sick leave can still be used. Sick leave belongs to one season's goal,
// so this is set *and* cleared every time a season is placed — switching to a season without
// sick leave lifts it again. Logging stays open on purpose: workouts are shared across seasons
// and still count in any season that isn't on sick leave (Strava imports land regardless).
function placeSickLeaveState(isSick, sickleaveLeft, seasonName) {
    var note = document.getElementById("hero-rest-note");
    if(note) {
        note.textContent = isSick
            ? `This week is sick leave in ${seasonName}. Your streak is kept and the week can't count against you.`
            : "";
        note.style.display = isSick ? "block" : "none";
    }

    var button = document.getElementById("sickleave-button");
    if(button) {
        button.dataset.seasonName = seasonName || "";
        button.dataset.left = sickleaveLeft || 0;
        button.style.display = !isSick && sickleaveLeft > 0 ? "inline-flex" : "none";
    }
    syncWeekActions();
}

// syncWeekActions hides the week-actions block when none of its buttons or its hint show, so
// it leaves no gap under the calendar's Save.
function syncWeekActions() {
    var block = document.getElementById("week-actions");
    if(!block) {
        return;
    }
    var shown = ["sickleave-button", "saved-workouts-button", "saved-workouts-hint"].some(function(id) {
        var element = document.getElementById(id);
        return element && element.style.display !== "none";
    });
    block.style.display = shown ? "flex" : "none";
}

function useSickLeave() {
    var button = document.getElementById("sickleave-button");
    var seasonName = button ? button.dataset.seasonName : "";
    var left = button ? button.dataset.left : "";

    if(!confirm(`Use sick leave for this week in ${seasonName}? Your streak is kept and the week can't count against you. You have ${left} left, and this uses one.`)) {
        return
    }

    var seasonID = document.getElementById("calendar_season_id").value;

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
                location.reload();
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/sickleave/" + seasonID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

// myWeekResult picks the signed-in user's result out of a week's results.
function myWeekResult(weekResults) {
    var users = (weekResults && weekResults.users) || [];
    for(var i = 0; i < users.length; i++) {
        if(users[i].user_id == user_id) {
            return users[i];
        }
    }
    return null;
}

// savedWorkoutsView works out everything the page says about saved workouts this week from
// the user's week result (docs/flexible-workouts.md). The server judges the week; this only
// phrases it. Returns null in a season that doesn't save workouts.
function savedWorkoutsView(result, season, goal) {
    if(!result || !season || !goal || !savedWorkoutsRule(season.carry_over_max, season.carry_over_weeks)) {
        return null;
    }

    var goalCount = goal.exercise_interval;
    var workouts = Math.round(result.week_completion * goalCount);
    var held = result.carried_in || 0;
    var expiries = (result.carry_expiries || []).map(calendarDate);

    return {
        rule: savedWorkoutsRule(season.carry_over_max, season.carry_over_weeks),
        held: held,
        expiries: expiries,
        workouts: workouts,
        goal: goalCount,
        shortfall: Math.max(0, goalCount - workouts),
        optedIn: result.carry_opt_in === true,
        used: result.carry_used || 0,
        earned: result.carry_earned || 0,
        sickLeave: result.sick_leave === true,
    };
}

// savedChipText: "2 saved workouts · use by Sun 9 Nov". When they run out on different
// days it names how many go first: "2 saved workouts · 1 to use by Sun 9 Nov".
function savedChipText(view) {
    if(view.held > 0) {
        var first = view.expiries[0];
        var goingFirst = view.expiries.filter(function(date) { return date.getTime() == first.getTime(); }).length;
        var useBy = goingFirst == view.held ? "use by " : goingFirst + " to use by ";
        return countLabel(view.held, "saved workout", "saved workouts") + " · " + useBy + shortDateString(first);
    }
    if(view.earned > 0) {
        return countLabel(view.earned, "extra workout", "extra workouts") + " will be saved this week";
    }
    return "";
}

// savedOfferText says what "Use saved workouts" will do before it is chosen.
function savedOfferText(view) {
    return "Uses " + view.shortfall + " of your " + countLabel(view.held, "saved workout", "saved workouts") +
        " if you end the week at " + view.workouts + "/" + view.goal + ".";
}

// savedOnText says what saved workouts are doing this week once they are on.
function savedOnText(view) {
    if(view.used > 0) {
        return countLabel(view.used, "saved workout makes", "saved workouts make") + " up the rest of this week.";
    }
    if(view.shortfall == 0) {
        return "You've met your goal, so your saved workouts stay saved.";
    }
    return "Saved workouts are on. Yours make up the week once you're at " + (view.goal - view.held) + "/" + view.goal + ".";
}

// placeSavedWorkoutsState shows the viewed season's saved workouts: the chip under the ring,
// the note saying what this week does with them (with Undo), and in the week actions the
// "Use saved workouts" button — offered only while the week is short and what you hold
// covers all of it, which is the only time it can help. Set and cleared on every placement.
function placeSavedWorkoutsState(result, season, goal) {
    var view = savedWorkoutsView(result, season, goal);

    var chip = document.getElementById("saved-chip");
    var note = document.getElementById("hero-saved-note");
    var button = document.getElementById("saved-workouts-button");
    var hint = document.getElementById("saved-workouts-hint");
    if(!chip || !note || !button || !hint) {
        return;
    }

    chip.style.display = "none";
    note.style.display = "none";
    button.style.display = "none";
    hint.style.display = "none";
    showSavedWorkoutsState(view, chip, note, button, hint);
    syncWeekActions();
}

function showSavedWorkoutsState(view, chip, note, button, hint) {
    if(!view) {
        return;
    }

    var chipText = savedChipText(view);
    if(chipText) {
        chip.innerHTML = `<span class="saved-chip-dot" aria-hidden="true"></span>${chipText}`;
        chip.title = "What are saved workouts?";
        chip.style.display = "inline-flex";
    }

    // A sick-leave week neither uses nor earns saved workouts; the chip still shows what you hold.
    if(view.sickLeave) {
        return;
    }

    if(view.optedIn) {
        note.innerHTML = `<span>${savedOnText(view)}</span><button type="button" class="btn btn--ghost btn--sm" onclick="toggleSavedWorkouts();">Undo</button>`;
        note.style.display = "flex";
        button.textContent = "Stop using saved workouts";
        button.style.display = "inline-flex";
        return;
    }

    if(view.shortfall > 0 && view.held >= view.shortfall) {
        button.textContent = "Use saved workouts";
        button.style.display = "inline-flex";
        hint.textContent = savedOfferText(view);
        hint.style.display = "block";
    } else if(view.shortfall > 0 && view.held > 0) {
        // Explain the missing button: what you hold can't cover the gap yet.
        hint.textContent = "Your " + countLabel(view.held, "saved workout", "saved workouts") +
            " can make up the week once you're at " + (view.goal - view.held) + "/" + view.goal + ".";
        hint.style.display = "block";
    }
}

// openSavedWorkouts explains saved workouts in plain words, lists what you hold and when each
// runs out, and offers the same choice as the week actions.
function openSavedWorkouts() {
    var view = currentSeason && currentGoal && typeof this_week !== "undefined"
        ? savedWorkoutsView(myWeekResult(this_week), currentSeason, currentGoal) : null;
    if(!view) {
        return;
    }

    var held = "<p>You have no saved workouts right now.</p>";
    if(view.held > 0) {
        var rows = view.expiries.map(function(date) {
            return `<li class="saved-list-row"><span class="saved-chip-dot" aria-hidden="true"></span>1 saved workout<span class="saved-list-date">use by ${shortDateString(date)}</span></li>`;
        }).join("");
        held = `<ul class="saved-list">${rows}</ul>`;
    }

    var thisWeek = "";
    if(view.sickLeave) {
        thisWeek = "<p>This week is sick leave, so it doesn't use or save any.</p>";
    } else if(view.optedIn) {
        thisWeek = `<p>${savedOnText(view)}</p>`;
    } else if(view.shortfall > 0 && view.held >= view.shortfall) {
        thisWeek = `<p>${savedOfferText(view)}</p>`;
    }
    if(!view.sickLeave && view.earned > 0) {
        thisWeek += `<p>You're beyond your goal: ${countLabel(view.earned, "extra workout", "extra workouts")} will be saved when the week ends.</p>`;
    }

    var action = "";
    if(!view.sickLeave && view.optedIn) {
        action = `<div class="btn-group"><button type="button" class="btn btn--ghost" onclick="toggleSavedWorkouts();">Stop using saved workouts</button></div>`;
    } else if(!view.sickLeave && view.shortfall > 0 && view.held >= view.shortfall) {
        action = `<div class="btn-group"><button type="button" class="btn btn--primary" onclick="toggleSavedWorkouts();">Use saved workouts</button></div>`;
    }

    TRModal.open({
        eyebrow: currentSeason.name,
        title: "Saved workouts",
        body: `
            <div class="saved-explainer">
                <p>${view.rule} Do more workouts than your weekly goal and the extras are saved.</p>
                <p>If a later week falls short, choose <b>Use saved workouts</b> and they make up the difference when the week ends, but only if they cover all of it. You can change your mind until Sunday.</p>
                <hr class="trm-divider">
                ${held}
                ${thisWeek}
                ${action}
            </div>
        `,
    });
}

// toggleSavedWorkouts turns "Use saved workouts" on or off for this week, then re-reads the
// week so the ring, the chip and the buttons follow.
function toggleSavedWorkouts() {
    if(!currentSeason || !currentGoal) {
        return;
    }
    var view = typeof this_week !== "undefined" ? savedWorkoutsView(myWeekResult(this_week), currentSeason, currentGoal) : null;
    var method = view && view.optedIn ? "delete" : "post";

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
                if(typeof TRModal !== "undefined" && TRModal.isOpen()) {
                    TRModal.setBody(`<p>${result.error}</p>`);
                } else {
                    error(result.error);
                }
            } else {
                if(typeof TRModal !== "undefined" && TRModal.isOpen()) {
                    TRModal.close();
                }
                get_leaderboard(currentSeason, currentGoal, false, false);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open(method, api_url + "auth/carryover/" + currentSeason.id);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeDebtOverview(overviewArray) {

    var html = "";

    document.getElementById("debt-module").style.display = "flex";

    for(var i = 0; i < overviewArray.debt_unviewed.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_unviewed[i].debt.date);
            var date_week = date.getWeek(1);
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        html += `
            <div class="debt-module-notification-view" id="">
                ${overviewArray.debt_unviewed[i].debt.loser.first_name} ${overviewArray.debt_unviewed[i].debt.loser.last_name} spun the wheel for week ${date_str}.<br>See if you won!<br>
                <img src="assets/arrow-right.svg" class="btn btn--icon clickable" onclick="location.replace('/wheel?debt_id=${overviewArray.debt_unviewed[i].debt.id}'); ">
            </div>
        `;
    }

    for(var i = 0; i < overviewArray.debt_won.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_won[i].date);
            var date_week = date.getWeek(1);
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        console.log(overviewArray.debt_won)

        html += `
            <div class="debt-module-notification-prize" id="">
                ${overviewArray.debt_won[i].loser.first_name} ${overviewArray.debt_won[i].loser.last_name} spun the wheel for week ${date_str} and you won <b>${overviewArray.debt_won[i].season.prize.quantity} ${overviewArray.debt_won[i].season.prize.name}</b>!<br>Have you received it?<br>
                <img src="assets/done.svg" class="btn btn--icon clickable" onclick="setPrizeReceived('${overviewArray.debt_won[i].id}');">
            </div>
        `;
    }

    for(var i = 0; i < overviewArray.debt_unpaid.length; i++) {

        var date_str = ""
        try {
            var date = new Date(overviewArray.debt_unpaid[i].date);
            var date_week = date.getWeek(1);
            var date_year = date.getFullYear();
            var date_str = date_week + " (" + date_year + ")"
        } catch {
            date_str = "Error"
        }

        console.log(overviewArray.debt_unpaid)

        html += `
            <div class="debt-module-notification-debt" id="">
                You spun the wheel for week ${date_str} and ${overviewArray.debt_unpaid[i].winner.first_name} ${overviewArray.debt_unpaid[i].winner.last_name} won ${overviewArray.debt_unpaid[i].season.prize.quantity} ${overviewArray.debt_unpaid[i].season.prize.name}!<br>Provide the prize as soon as possible!<br>
            </div>
            `;
    }

    document.getElementById("debt-module-notifications").innerHTML = html;

}

function placeDebtSpin(overview) {
    
    var date_str = ""
    try {
        var date = new Date(overview.debt_lost[0].date);
        var date_week = date.getWeek(1);
        var date_year = date.getFullYear();
        var date_str = date_week + " (" + date_year + ")"
    } catch {
        date_str = "Error"
    }   
    
    try {
        document.getElementById("ongoingseason").style.display = "none";
    } catch (e) {
        console.log("Failed to hide ongoing season. Error: " + e)
    }  

    try {
        document.getElementById("top-module").style.display = "none";
    } catch (e) {
        console.log("Failed to hide top module. Error: " + e)
    }  
    
    document.getElementById("unspun-wheel").style.display = "flex";
    document.getElementById("unspun-wheel").innerHTML = `
        You failed to reach your goal for week ${date_str} and must spin the wheel.
        <div id="canvas-buttons" class="canvas-buttons">
            <button class="btn" id="go-to-wheel" onclick="location.replace('/wheel?debt_id=${overview.debt_lost[0].id}');">Take me there</button>
        </div>
    `;
    return;
}

function EditExercise(weekdayInt) {

    var existingID = document.getElementById("day_" + weekdayInt + "_id").value;
    if(existingID && existingID !== "00000000-0000-0000-0000-000000000000") {
        GoToExercise(existingID);
        return;
    }

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
            if(result.error) {
                error(result.error);
            } else {
                GoToExercise(result.exercise.id);
            }
        }
    };
    xhttp.withCredentials = true;
    // The front page numbers days Monday=1..Sunday=7, but the API expects Go's
    // time.Weekday (Sunday=0..Saturday=6). 7 % 7 maps Sunday to 0; 1-6 are unchanged.
    xhttp.open("get", api_url + "auth/exercise-days/week?weekDay=" + (weekdayInt % 7));
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();

}

function GoToExercise(exerciseID) {

    window.location = '/exercises/' + exerciseID

}

function placeSeasonProgress(seasonStartObject, seasonEndObject) {

    const weekSum = weeksBetween(seasonStartObject, seasonEndObject)

    const now = new Date();
    const weekAmount = weeksBetween(seasonStartObject, now)

    document.getElementById("weeks_so_far").innerHTML = weekAmount
    document.getElementById("weeks_total").innerHTML = weekSum

    var ach_percentage = Math.floor((weekAmount / weekSum) * 100)
    
    if(ach_percentage > 100) {
        ach_percentage = 100;
    }

    document.getElementById("season-progress-bar").style.width  = ach_percentage + "%"

    if(ach_percentage > 99) {
        document.getElementById("season-progress-bar-wrapper").classList.remove('transparent');
        setTimeout(function() {
            document.getElementById("season-progress-bar").classList.add("blink")
        }, 1500);
        setTimeout(function() {
            document.getElementById("season-progress-bar").classList.remove('blink');
        }, 2500);
    }
}

// placeWeekProgress fills the hero ring: the arc is the week's workouts, and savedPercentage
// (saved workouts making up the rest, docs/flexible-workouts.md) is a second, paler segment
// right after it. The ring turns green once the two together reach the goal.
function placeWeekProgress(percentage, sickLeave, savedPercentage, savedCount) {

    console.log("This weeks progress: " + percentage)

    if(percentage > 100) {
        percentage = 100;
    }
    var saved = Math.max(0, Math.min(savedPercentage || 0, 100 - percentage));
    if(saved > 0 && percentage + saved > 99) {
        saved = 100 - percentage; // rounding never leaves a sliver of track
    }

    var ring = document.getElementById("hero-ring");
    if(!ring) {
        return;
    }

    // Reveal the ring (hidden until a season populates it).
    ring.style.display = "flex";

    var caption = document.getElementById("hero-ring-caption");
    if(caption) {
        caption.textContent = saved > 0 && sickLeave !== true ? "+ " + countLabel(savedCount, "saved workout", "saved workouts") : "this week";
    }
    ring.classList.toggle("has-saved", saved > 0 && sickLeave !== true);

    // A sick-leave week is paused: no arc, no celebration — the ring says so instead.
    ring.classList.toggle("is-resting", sickLeave === true);
    if(sickLeave === true) {
        ring.classList.remove("complete");
        ring.style.setProperty("--pct", 0);
        ring.style.setProperty("--saved-pct", 0);
        return;
    }

    // Fill the arc to the completion %, and the saved segment after it.
    ring.style.setProperty("--pct", percentage);
    ring.style.setProperty("--saved-pct", saved);

    if(percentage + saved > 99) {
        // Complete: snap the ring green and give it a brief celebratory pulse.
        ring.classList.add("complete");
        setTimeout(function() { ring.classList.add("blink"); }, 1500);
        setTimeout(function() { ring.classList.remove("blink"); }, 2500);
    } else {
        ring.classList.remove("complete");
    }

}

function verifyPageRedirect() {

    window.location = '/verify'

}


function registerGoalRedirect() {

    window.location = '/registergoal'
    
}

function exerciseRedirect(exerciseDayID) {
    window.location = '/exercises/' + exerciseDayID
}

function addExercise(exerciseDayID) {
    var formData = JSON.stringify({
        "exercise_day_id": exerciseDayID,
        "is_on": true,
        "note": "",
        "duration": null
    });

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
                exerciseRedirect(exerciseDayID)
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/exercises");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(formData);
    return false;
}

function getPotentialSeasons(user_id){

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
                return;
            } else {
                placePotentialSeasons(result.seasons);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons?potential=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function placePotentialSeasons(seasonsArray) {
    try {
        if(!seasonsArray || seasonsArray.length == 0) {
            return;
        }

        var html = "Seasons you can join:<br><div class='potentialSeasonList'><hr>";
        for(var i = 0; i < seasonsArray.length; i++) {
            html += `
                <button class="btn btn--block" onclick="document.location.href = '/registergoal?season_id=${seasonsArray[i].id}'">
                    ${seasonsArray[i].name}
                </button>
            `
        }
        html += '</div>'

        potentialSeasonsWrapper = document.getElementById("potentialSeasonsWrapper");
        potentialSeasonsWrapper.innerHTML = html;
        potentialSeasonsWrapper.style.display = "flex";
        document.getElementById("seasonDivider").style.display = "flex";
    } catch (e) {
        console.log("Failed to place potential seasons. Error: " + e)
        error("Failed to place potential seasons.")
        return;
    }
}

function getCountdownSeasons(userID){

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
                return;
            } else {
                placeCountdownSeasons(result.seasons, userID);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/seasons?countdown=true");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function placeCountdownSeasons(seasonsArray, userID) {
    console.log(seasonsArray)

    try {
        if(!seasonsArray || seasonsArray.length == 0) {
            return;
        }

        var html = "Seasons you are waiting for:<br><div class='countdownSeasonList'><hr>";
        for(var i = 0; i < seasonsArray.length; i++) {
            var goalID = seasonsArray[i].user_goal_id || '';

            var date_start = new Date(seasonsArray[i].start);
            var date_end = new Date(seasonsArray[i].end);

            var joinText = "..."
            if(seasonsArray[i].join_anytime) {
                joinText = "<b>You can join at any point in the season.</b>"
            } else {
                joinText = "<b>You must join before the start date.</b>"
            }

            var participantCount = seasonsArray[i].participant_count;
            var partici_string = "participants"
            if(participantCount == 1) {
                partici_string = "participant"
            }

            html += `
                <div class="countdownSeason">
                    <h3 id="countdown_season_title" style="margin: 0 0 0.5em 0;">${seasonsArray[i].name}</h3>
                    <p id="countdown_season_start">${GetDateString(date_start, true)}</p>
                    <p id="countdown_season_end">${GetDateString(date_end, true)}</p>

                    <p id="countdown_title" style="margin-top: 0.25em;">${participantCount + " " + partici_string}. Starting in:</p>

                    <p style="font-size: 2em; text-align: center;" id="countdown_number_${seasonsArray[i].id}" class="countdown_number">00d 00h 00m 00s</p>

                    <a class="clickable hover" style="margin: 1em 0 0 0; font-size:0.75em;" onclick="deleteGoal('${goalID}');">I changed my mind!</i></a>
                    <hr>
                </div>
            `
        }
        html += '</div>'

        countdownSeasonsWrapper = document.getElementById("countdownSeasonsWrapper");
        countdownSeasonsWrapper.innerHTML = html;
        countdownSeasonsWrapper.style.display = "flex";

        for(var i = 0; i < seasonsArray.length; i++) {
            var date_start = new Date(seasonsArray[i].start);
            activateCountdown(date_start, seasonsArray[i].id);
        }

        document.getElementById("seasonDivider").style.display = "flex";
    } catch (e) {
        console.log("Failed to place countdown seasons. Error: " + e)
        error("Failed to place countdown seasons.")
        return;
    }
}

function activateCountdown(countdownDate, seasonID){

    // Set the date we're counting down to
    var countDownDate = countdownDate.getTime();

    // Update the count down every 1 second
    var x = setInterval(function() {

        // Get today's date
        var now = new Date();

        // Find the distance between now and the count down date
        var distance = Math.floor(countDownDate - now.getTime());
    
        // Time calculations for days, hours, minutes and seconds
        var days = Math.floor(distance / (1000 * 60 * 60 * 24));
        var hours = Math.floor((distance % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
        var minutes = Math.floor((distance % (1000 * 60 * 60)) / (1000 * 60));
        var seconds = Math.floor((distance % (1000 * 60)) / 1000);

        if (distance > 0) {
            // Display the result in the element with id="demo"
            document.getElementById("countdown_number_" + seasonID).innerHTML = padNumber(days, 2) + "d " + padNumber(hours, 2) + "h "
            + padNumber(minutes, 2) + "m " + padNumber(seconds, 2) + "s ";
        
            // If the count down is finished, write some text
        } else {
            clearInterval(x);
            document.getElementById("countdown_number_" + seasonID).innerHTML = "...";

            setTimeout(() => {
                frontPageRedirect(true);
            }, 5000);
              
        }
        
    }, 1000);
}

function deleteGoal(goalID) {
    if(!confirm("Are you sure you want to delete your goal? You are free to create another one afterward as long as the season has not started.")) {
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
            
            if(result.error) {
                error(result.error);
            } else {
                frontPageRedirect(true);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/goals/" + goalID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function blinkCalendar() {
    try {
        document.getElementById("goal_amount_button").classList.add("blink")
        setTimeout(function() {
            document.getElementById("goal_amount_button").classList.remove('blink');
        }, 1500);
    } catch (e) {
        console.log("Failed to blink the calendar. Error: " + e)
    }
}

// getActivities loads this week's activities from everyone the user has ever shared a
// season with (see docs/activity-feed.md). It is deliberately not season-scoped, so the
// module keeps working between seasons and for users who haven't joined one yet.
function getActivities(){
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
                placeActivities(result.activities);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/activities/shared");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function placeActivities(activitiesArray) {
    if(activitiesArray.length < 1) {
        return
    }

    var activitiesHTML = `
        <div class="activityWrapper">
    `;

    activitiesArray.forEach(activity => {
        activitiesHTML += generateActivityHTML(activity);
    });

    activitiesHTML += `
        </div>
    `;

    try {
        document.getElementById("activities-week").innerHTML = activitiesHTML
    } catch (error) {
        console.log(error)
    }
}

// generateActivityHTML renders one front-page feed entry. The phrase ("did some strength
// training") and the logos come from the server's session headline, which weighs the
// session's activity categories, so a Hevy push day reads as strength training with one logo
// instead of "a range of workouts".
function generateActivityHTML(activity) {
    const headline = activity.headline || {};
    const activitySentence = headline.past_tense_verb || "worked out";

    var activityLogoBarHTML = "";
    (headline.logos || []).forEach(logo => {
        activityLogoBarHTML += `
            <div class="activity-logo">
                <img class="u-w-full" src="assets/actions/${logo}.svg" title="${logo}"></img>
            </div>
        `;
    });

    var activityStravaHTML = "";
    if(activity.strava_ids && activity.strava_ids.length > 0) {
        activity.strava_ids.forEach(stravaID => {
            activityStravaHTML += `
                <img class="strava-logo-img clickable" src="/assets/strava-logo.svg" onclick="window.open('https://www.strava.com/activities/${stravaID}', '_blank')">
                </img>
            `;
        });
    }

    // Hevy has no public per-workout URL, so the logo just marks the source.
    if(activity.hevy_workout_id) {
        activityStravaHTML += `
            <img class="hevy-logo-img" src="/assets/hevy.png" title="Imported from Hevy">
            </img>
        `;
    }

    

    var dateObject = new Date(activity.time)

    activityHTML = `
        <div class="activity">

            <div class="activity-date">
                ${GetDateString(dateObject, true)}
            </div>

            <div class="activity-sections">
                <div class="activity-photo-wrapper">
                    <div class="activity-user-photo" title="` + activity.user.first_name + ` ` + activity.user.last_name + `" onclick="location.href='/users/${activity.user.id}'">
                        <img class="activity-user-photo-img" src="${profileImageURL(activity.user.id, true)}" onerror="${IMAGE_FALLBACK_ONERROR}">
                    </div>
                </div>

                <div class="activity-details">
                    <div>
                        ${activity.user.first_name} ${activitySentence}!
                    </div>

                    <div class="activity-logo-wrapper">
                        ${activityLogoBarHTML}
                    </div>

                </div>

                <div class="activity-strava-wrapper">
                    ${activityStravaHTML}
                </div>

            </div>
        </div>
    `;

    return activityHTML;
}

function placeSeasonAlternatives(seasonObject, userID, alternatives) {
    if(alternatives == null || alternatives.length < 1) {
        return
    }

    titleObject = document.getElementById('season_title');
    
    var html = `
        <select name="selectedSeason" id="season_title" style="height: 2rem; font-weight: bold;" onchange="switchSeason('${userID}');">
            <option value="${seasonObject.id}">${seasonObject.name}</option>
    `;

    alternatives.forEach(seasonAlternative => {
        html += `<option value="${seasonAlternative.id}">${seasonAlternative.name}</option>`
    });

    html += `</select>`;

    titleObject.outerHTML = html;
}

function switchSeason(userID) {
    console.log("Changing season.")
    var seasonID = document.getElementById('season_title').value;
    get_season(userID, true, seasonID)
}