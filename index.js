function load_page() {
    show_logged_out_menu();
    remove_active_menu();
    alert_info('Laster inn...');

    var activate_email = getParameterByName('activate_email');
    var activate_hash = getParameterByName('activate_hash');
    if(activate_email !== null & activate_hash !== null && !stop_url_react) {
        load_page_verify(activate_email, activate_hash);
        return;
    }

    cookie = get_cookie('treningheten-bruker');

    if(cookie) {
        validate_user_cookie(cookie);
    }

    load_page_home();
}

function validate_user_cookie(cookie) {
    var json_cookie = JSON.stringify({"cookie": cookie});
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4 && this.status == 200) {

            try {
                var result= JSON.parse(this.responseText);
            } catch(error) {
                alert_error("API response kunne ikke tolkes.");
                console.log(error);
                console.log(this.responseText);
            }
            
            if(result.error) {
                set_cookie("treningheten-bruker", "", 1);
                logged_out = false;
                load_page_home();
            } else {
                logged_in = true;
                login_data = result.cookie;
                show_logged_in_menu();
                load_page_home();
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/validate_login_cookie.php");
    xhttp.send(json_cookie);
    return;
}

function load_page_verify(activate_email, activate_hash) {
    var html = `
        <div>
            <h1>Aktiverer brukeren din</h1>
            <p>:)</p>
        </div>
    `;

    document.getElementById('content-box').innerHTML = html;
    verify_user(activate_email, activate_hash);
}

function verify_user(activate_email, activate_hash) {

    user_create_form = {
                            "user_email" : activate_email, 
                            "user_hash" : activate_hash
                        };

    var user_create_data = JSON.stringify(user_create_form);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

        alert_info('Laster inn...');

        if (this.readyState == 4 && this.status == 200) {
            try {
                var result= JSON.parse(this.responseText);
            } catch(error) {
                alert_error('Klarte ikke tolke API respons.');
                console.log('Failed to parse API response. Response: ' + this.responseText);
            }

            stop_url_react = true;
            
            if(result.error) {
                alert_error(result.message);
            } else {
                alert_success(result.message);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/activate_user.php");
    xhttp.send(user_create_data);
    return;
}

// ( Home page
function load_page_home() {
    alert_clear();
    remove_active_menu();

    if(logged_in) {
        var html = `
            <div>
                <h1>Tren da, ` + login_data.user_firstname + `</h1>
                <p>Siden er fortsatt under konstruksjon.</p>

                <div id="exercises" class="exercises">
                    <div class="week_info" id="week_info">
                        Laster inn...
                    </div>

                    <div class="week_days">
                        <div class="form-group" style="border: solid 1px var(--blue); width: 5em; padding: 0.5em;" id="day_1_group">
                            <label for="day_1" title="Har du trent?">Mandag</label>
                            <input type="checkbox" class="form-control" id="day_1">
                        </div>

                        <div class="form-group" style="border: solid 1px var(--blue); width: 5em; padding: 0.5em;" id="day_2_group">
                            <label for="day_2" title="Har du trent?">Tirsdag</label>
                            <input type="checkbox" class="form-control" id="day_2">
                        </div>

                        <div class="form-group" style="border: solid 1px var(--blue); width: 5em; padding: 0.5em;" id="day_3_group">
                            <label for="day_3" title="Har du trent?">Onsdag</label>
                            <input type="checkbox" class="form-control" id="day_3">
                        </div>

                        <div class="form-group" style="border: solid 1px var(--blue); width: 5em; padding: 0.5em;" id="day_4_group">
                            <label for="day_4" title="Har du trent?">Torsdag</label>
                            <input type="checkbox" class="form-control" id="day_4">
                        </div>

                        <div class="form-group" style="border: solid 1px var(--blue); width: 5em; padding: 0.5em;" id="day_5_group">
                            <label for="day_5" title="Har du trent?">Fredag</label>
                            <input type="checkbox" class="form-control" id="day_5">
                        </div>

                        <div class="form-group" style="border: solid 1px var(--blue); width: 5em; padding: 0.5em;" id="day_6_group">
                            <label for="day_6" title="Har du trent?">Lørdag</label>
                            <input type="checkbox" class="form-control" id="day_6">
                        </div>

                        <div class="form-group" style="border: solid 1px var(--blue); width: 15em; padding: 0.5em;" id="day_7_group">
                            <label for="day_7" title="Har du trent?">Søndag</label>
                            <input type="checkbox" class="form-control" id="day_7">
                        </div>

                    </div>

                    <div class="form-group" style="background-color: var(--white); width: 100%; border-radius: 0 0 0.5em 0.5em; border: solid 1px var(--blue);">
                        <button type="submit" onclick="update_exercises();" class="form-input btn" id="goal_amount_button" style="width: auto;"><img src="assets/done.svg" class="btn_logo"><p2>Lagre</p2></button>
                    </div>

                </div>

            </div>
        `;

        load_home_goal();
    } else {
        var html = `
            <div>
                <h1>Tren da</h1>
                <p>Siden er under konstruksjon.</p>
            </div>
        `;
    }

    document.getElementById('content-box').innerHTML = html;
}

function load_home_goal() {

    user_goal_get_form = {
                            "cookie" : cookie
                        };

    var user_goal_get_data = JSON.stringify(user_goal_get_form);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

        alert_info('Laster inn...');

        if (this.readyState == 4 && this.status == 200) {
            try {
                var result= JSON.parse(this.responseText);
            } catch(error) {
                alert_error('Klarte ikke tolke API respons.');
                console.log('Failed to parse API response. Response: ' + this.responseText);
            }
            
            if(result.error) {
                alert_error(result.message);
            } else {
                alert_clear();
                
                if(!result.goal) {
                    var html = '';
                    html += '<div class="form-group" style="border: solid 1px var(--blue); border-radius: 0.5em; background-color: lightblue; display: block;">';
                    html += 'Du har ikke noe mål for øyeblikket. Det er aldri for sent å sette igang!';
                    html += '<br>';
                    html += '<br>';
                    html += 'Dette gjelder nåværende sesong, som er ' + result.season_start + ' til ' + result.season_end + '.';
                    html += '<br>';
                    html += '<br>';
                    html += '<form id="goal_amount_form" onsubmit="set_home_goal();return false;">';

                    html += '<div class="form-group">';
                    html += '<label for="goal_exer_week" title="Dette blir målet du må nå i 6 måneder.">Hvor mange ganger i uka vil du trene?</label>';
                    html += '<input type="number" inputmode="numeric" name="goal_exer_week" id="goal_exer_week" class="form-input" min="1" max="7" value="" autocomplete="on" required />';
                    html += '</div>';

                    html += '<div class="form-group newline">';
                    html += '</div>';
                    if(result.can_compete) {
                        html += '<div class="form-group">';
                        html += '<label for="goal_compete" title="Du kan sloss mot dine søsken.">Ja, jeg er med på utfordringen med de andre og forstår konsekvensene dersom jeg ikke når målet mitt.</label>';
                        html += '<input type="checkbox" class="form-control" id="goal_compete">';
                        html += '</div>';

                        html += '<div class="form-group newline">';
                        html += '</div>';
                    } else {
                        html += '<div class="form-group">';
                        html += 'Du kan bare konkurrere ved å melde deg på i januar og august.';
                        html += '<input type="hidden" class="form-control" id="goal_compete">';
                        html += '</div>';

                        html += '<div class="form-group newline">';
                        html += '</div>';
                    }
                    html += '<div class="form-group">';
                    html += 'Logging av trening starter førstkommende mandag.';
                    html += '<br>';
                    html += '(Ja, <b>i dag</b> hvis det er mandag)';
                    html += '<br>';
                    html += '<br>';
                    html += 'Du har to sykmeldinger per sesong.';
                    html += '<br>';
                    html += '<br>';
                    html += 'Du kan bare logge i uken du er i, og dager som er eller har vært.';
                    html += '</div>';

                    html += '<div class="form-group newline">';
                    html += '</div>';

                    html += '<div class="form-group">';
                    html += '<button type="submit" class="form-input btn" id="goal_amount_button"><img src="assets/done.svg" class="btn_logo"><p2>Start!</p2></button>';
                    html += '</div>';
                    html += '</div>';
                }  else {
                    var html = '';
                    html += '<div class="form-group newline" style="border: solid 1px var(--blue); border-radius: 0.5em; background-color: lightblue;">';
                    html += 'Du har satt et mål! Du skal trene <b>' + result.goal.goal_exer_week + '</b> ganger i uka.';
                    html += '<br>';
                    html += '<br>';
                    if(!result.goal.goal_started) {
                    html += 'Dette målet starter ' + result.goal.goal_start + '.';
                    html += '<br>';
                    }
                    html += 'Dette målet varer til og med ' + result.goal.goal_end + '.';
                    if(result.goal.goal_compete === '1') {
                        html += '<br>';
                        html += '<br>';
                        html += 'Du er med på utfordringen med de andre.';
                    }
                    html += '</div>';
                    if(result.goal.goal_started) {
                        load_home_exercises(result.goal.goal_id);
                    }
                }

                document.getElementById('goal_div').innerHTML = html;
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/get_goal.php");
    xhttp.send(user_goal_get_data);
    return;
}

function set_home_goal() {

    var goal_exer_week = document.getElementById('goal_exer_week').value;
    var goal_compete = document.getElementById('goal_compete').checked;

    if(!confirm('Er du helt sikkert på at du vil sette ' + goal_exer_week + ' som treningsmål? Dette kan ikke endres denne sesongen.')) {
        return;
    }

    user_goal_get_form = {
                            "cookie" : cookie,
                            "goal_exer_week" : goal_exer_week,
                            "goal_compete" : goal_compete
                        };

    var user_goal_get_data = JSON.stringify(user_goal_get_form);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

        alert_info('Laster inn...');

        if (this.readyState == 4 && this.status == 200) {
            try {
                var result= JSON.parse(this.responseText);
            } catch(error) {
                alert_error('Klarte ikke tolke API respons.');
                console.log('Failed to parse API response. Response: ' + this.responseText);
            }
            
            if(result.error) {
                alert_error(result.message);
            } else {
                location.reload();
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/set_goal.php");
    xhttp.send(user_goal_get_data);
    return;
}

function load_home_exercises(goal_id) {

    user_goal_get_form = {
                            "cookie" : cookie,
                            "goal_id" : goal_id
                        };

    var user_goal_get_data = JSON.stringify(user_goal_get_form);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

        alert_info('Laster inn...');

        if (this.readyState == 4 && this.status == 200) {
            try {
                var result= JSON.parse(this.responseText);
            } catch(error) {
                alert_error('Klarte ikke tolke API respons.');
                console.log('Failed to parse API response. Response: ' + this.responseText);
            }
            
            if(result.error) {
                alert_error(result.message);
            } else {
                alert_clear();
                document.getElementById('exercises').style.display = 'flex';
                document.getElementById('week_info').innerHTML = 'Uke ' + result.week_number;

                for(var i = 1; i <= result.exercises.days.length; i++) {

                    if(result.exercises.days[i-1]) {
                        document.getElementById('day_' + i).checked = true;
                        exercise_this_week += 1;
                    }

                    if(i > parseInt(result.week_day)) {
                        document.getElementById('day_' + i).disabled = true;
                    }

                    if(i == parseInt(result.week_day)) {
                        document.getElementById('day_' + i + '_group').style.backgroundColor = 'lightblue';
                    }

                }
                
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/get_exercises_week.php");
    xhttp.send(user_goal_get_data);
    return;
}

function update_exercises() {

    var exercise_this_week_new = 0;
    for(var i = 1; i < 8; i++) {
        if(document.getElementById('day_' + i).checked) {
            exercise_this_week_new += 1;
        }
    }
    
    exercises_get_form = {
        "cookie" : cookie,
        "exercises" : {
            "days": [
                document.getElementById('day_1').checked,
                document.getElementById('day_2').checked,
                document.getElementById('day_3').checked,
                document.getElementById('day_4').checked,
                document.getElementById('day_5').checked,
                document.getElementById('day_6').checked,
                document.getElementById('day_7').checked
            ]
        }

    };

    var exercises_get_data = JSON.stringify(exercises_get_form);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

    alert_info('Laster inn...');

    if (this.readyState == 4 && this.status == 200) {
        try {
            var result= JSON.parse(this.responseText);
        } catch(error) {
            alert_error('Klarte ikke tolke API respons.');
            console.log('Failed to parse API response. Response: ' + this.responseText);
        }

        if(result.error) {
            alert_error(result.message);
        } else {
            alert_success(result.message);

            if(exercise_this_week < exercise_this_week_new) {
                trigger_fireworks(1);
            }

            exercise_this_week = exercise_this_week_new;
        }

    }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/create_exercises_week.php");
    xhttp.send(exercises_get_data);
    return;
}
// Home page )

// ( Create user
function load_page_register() {

    alert_clear();
    remove_active_menu();
    add_active_menu('register_tab');
    toggle_navbar();

    var html = `
        <div>
            <h1>Registrer en bruker</h1>
            <p>Vi trenger noe info for å lage brukeren din.</p>
        </div>

        <div>
            <form id='password_login_form' onsubmit='register_user();return false;'>

            <div class='form-group'>
                <label for="user_email" title="Eposten vi kan kontakte deg på.">E-post:</label>
                <input type="email" name="user_email" id="user_email" class="form-input" value="" autocomplete="on" required />
            </div>

            <div class='form-group'>
                <label for="code_hash" title="Hva er invitasjonskoden din?">Invitasjonskode:</label>
                <input type="text" name="code_hash" id="code_hash" class="form-input" value="" autocomplete="off" required />
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <label for="user_firstname" title="Hva heter du?">Fornavn:</label>
                <input type="text" name="user_firstname" id="user_firstname" class="form-input" value="" autocomplete="on" required />
            </div>

            <div class='form-group'>
                <label for="user_lastname" title="Hva heter familien din?">Etternavn:</label>
                <input type="text" name="user_lastname" id="user_lastname" class="form-input" value="" autocomplete="on" required />
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <label for="user_password" title="Ditt hemmelige passord.">Passord:</label>
                <input type="password" name="user_password" id="user_password" class="form-input" value="" autocomplete="off" required />
            </div>

            <div class='form-group'>
                <label for="user_password_confirm" title="Gjenta ditt hemmelige passord.">Gjenta passord:</label>
                <input type="password" name="user_password_confirm" id="user_password_confirm" class="form-input" value="" autocomplete="off" required />
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <label for="accept_terms_check" title="Godkjenn vilkårene for å lage brukeren din.">Jeg godtar at denne siden lagrer data for eget bruk og at jeg er over 18 år:</label>
                <input type="checkbox" class="form-control" id="accept_terms_check" required>
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <button type="submit" class="form-input btn" id="register_user_button"><img src="assets/done.svg" class="btn_logo"><p2>Registrer</p2></button>
            </div>

            </form>
        </div>
    `;

    document.getElementById('content-box').innerHTML = html;
}

function register_user() {
    // Disable button
    document.getElementById("register_user_button").disabled = true;
    document.getElementById("register_user_button").style.opacity = '0.5';

    // Check password match
    if(document.getElementById('user_password').value != document.getElementById('user_password_confirm').value) {
        alert("The passwords must match.");
        document.getElementById('user_password').value = "";
        document.getElementById('user_password_confirm').value = "";
        document.getElementById('user_password').focus();

        document.getElementById("register_user_button").disabled = false;
        document.getElementById("register_user_button").style.opacity = '1';
        return false;
    } else {
        user_password = document.getElementById('user_password').value;
        user_email = document.getElementById('user_email').value;
        user_firstname = document.getElementById('user_firstname').value;
        user_lastname = document.getElementById('user_lastname').value;
        code_hash = document.getElementById('code_hash').value;
    }

    user_create_form = {
                            "user_password" : user_password, 
                            "user_email" : user_email,
                            "user_firstname" : user_firstname,
                            "user_lastname" : user_lastname,
                            "code_hash" : code_hash
                        };

    var user_create_data = JSON.stringify(user_create_form);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

        alert_info('Laster inn...');

        if (this.readyState == 4 && this.status == 200) {
            try {
                var result= JSON.parse(this.responseText);
            } catch(error) {
                alert_error('Klarte ikke tolke API respons.');
                console.log('Failed to parse API response. Response: ' + this.responseText);
                document.getElementById("register_user_button").disabled = false;
                document.getElementById("register_user_button").style.opacity = '1';
                document.getElementById('user_password').value = "";
                document.getElementById('user_password_confirm').value = "";
            }
            
            if(result.error) {
                alert_error(result.message);
                document.getElementById("register_user_button").disabled = false;
                document.getElementById("register_user_button").style.opacity = '1';
                document.getElementById('user_password').value = "";
                document.getElementById('user_password_confirm').value = "";
            } else {
                alert_success(result.message);
                document.getElementById('user_password').value = "";
                document.getElementById('user_password_confirm').value = "";
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/create_user.php");
    xhttp.send(user_create_data);
    return;
}
// Create user )

// ( login user
function load_page_login() {

    alert_clear();
    remove_active_menu();
    add_active_menu('log_in_tab');
    toggle_navbar();

    var html = `
        <div>
            <h1>Logg inn</h1>
            <p>Prøv å huske passordet da, Kristine.</p>
        </div>

        <div>
            <form id='password_login_form' onsubmit='login_user();return false;'>

            <div class='form-group'>
                <label for="user_email" title="Eposten du registrerte med.">E-post:</label>
                <input type="email" name="user_email" id="user_email" class="form-input" value="" autocomplete="on" required />
            </div>

            <div class='form-group'>
                <label for="user_password" title="Ditt hemmelige passord.">Passord:</label>
                <input type="password" name="user_password" id="user_password" class="form-input" value="" autocomplete="off" required />
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <button type="submit" class="form-input btn" id="login_user_button"><img src="assets/done.svg" class="btn_logo"><p2>Logg inn</p2></button>
            </div>

            </form>
        </div>
    `;

    document.getElementById('content-box').innerHTML = html;
}

function login_user() {
    // Disable button
    document.getElementById("login_user_button").disabled = true;
    document.getElementById("login_user_button").style.opacity = '0.5';

    user_password = document.getElementById('user_password').value;
    user_email = document.getElementById('user_email').value;


    user_login_form = {
                            "user_password" : user_password, 
                            "user_email" : user_email
                        };

    var user_login_data = JSON.stringify(user_login_form);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

        alert_info('Laster inn...');

        if (this.readyState == 4 && this.status == 200) {
            try {
                var result= JSON.parse(this.responseText);
            } catch(error) {
                alert_error('Klarte ikke tolke API respons.');
                console.log('Failed to parse API response. Response: ' + this.responseText)
                document.getElementById('user_password').value = '';
                document.getElementById("login_user_button").disabled = false;
                document.getElementById("login_user_button").style.opacity = '1';
            }
            
            if(result.error) {
                alert_error(result.message);
                document.getElementById('user_password').value = '';
                document.getElementById("login_user_button").disabled = false;
                document.getElementById("login_user_button").style.opacity = '1';
            } else {
                document.getElementById('user_password').value = '';
                set_cookie("treningheten-bruker", result.cookie, 1);
                window.location = window.location.pathname;
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/get_login_cookie.php");
    xhttp.send(user_login_data);
    return;
}
// Create user )

function log_out() {
    show_logged_out_menu();
    set_cookie("treningheten-bruker", "", 1);
    logged_in = false;
    document.getElementById('goal_div').innerHTML = '';
    load_page_home();
    alert_info('Du har blitt logget ut.');
    toggle_navbar();
}

// ( Update user
function load_page_user() {

    alert_clear();
    remove_active_menu();
    add_active_menu('update_account');
    toggle_navbar();
    document.getElementById('goal_div').innerHTML = '';

    var html = `
        <div>
            <h1>Brukeren din</h1>
            <p>Her kan du endre noen detaljer om deg selv.</p>
        </div>

        <div>
            <form id='password_login_form' onsubmit='update_user();return false;'>

            <div class='form-group'>
                <label for="user_email" title="Eposten du registrerte med.">E-post:</label>
                <input type="email" name="user_email" id="user_email" class="form-input" value="` + login_data.user_email + `" autocomplete="on" required />
            </div>

            <div class='form-group'>
                <label for="user_creation" title="Når du skapte brukeren.">Bruker siden:</label>
                <input type="text" name="user_creation" id="user_creation" class="form-input" value="` + login_data.user_creation + `" autocomplete="on" readonly />
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <label for="user_firstname" title="Det du heter.">Fornavn:</label>
                <input type="text" name="user_firstname" id="user_firstname" class="form-input" value="` + login_data.user_firstname + `" autocomplete="on" readonly />
            </div>

            <div class='form-group'>
                <label for="user_lastname" title="Det familien din heter.">Etternavn:</label>
                <input type="text" name="user_lastname" id="user_lastname" class="form-input" value="` + login_data.user_lastname + `" autocomplete="on" readonly />
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <label for="user_password_new" title="Ditt hemmelige passord.">Nytt passord:</label>
                <input type="password" name="user_password_new" id="user_password_new" class="form-input" value="" autocomplete="off" />
            </div>

            <div class='form-group'>
                <label for="user_password_new_repeat" title="Ditt hemmelige passord.">Gjenta ditt nye passord:</label>
                <input type="password" name="user_password_new_repeat" id="user_password_new_repeat" class="form-input" value="" autocomplete="off" />
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <label for="user_password" title="Ditt hemmelige passord.">Passord:</label>
                <input type="password" name="user_password" id="user_password" class="form-input" value="" autocomplete="off" required />
            </div>

            <div class='form-group newline'>
            </div>

            <div class='form-group'>
                <button type="submit" class="form-input btn" id="login_user_button"><img src="assets/done.svg" class="btn_logo"><p2>Lagre endringer</p2></button>
            </div>

            </form>
        </div>
    `;

    document.getElementById('content-box').innerHTML = html;
    alert_info('Denne siden fungerer ikke enda.');
}

function update_user() {
    // Disable button
    document.getElementById("login_user_button").disabled = true;
    document.getElementById("login_user_button").style.opacity = '0.5';

    user_password = document.getElementById('user_password').value;
    user_email = document.getElementById('user_email').value;


    user_login_form = {
                            "user_password" : user_password, 
                            "user_email" : user_email
                        };

    var user_login_data = JSON.stringify(user_login_form);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {

        alert_info('Laster inn...');

        if (this.readyState == 4 && this.status == 200) {
            try {
                var result= JSON.parse(this.responseText);
            } catch(error) {
                alert_error('Klarte ikke tolke API respons.');
                console.log('Failed to parse API response. Response: ' + this.responseText)
                document.getElementById('user_password').value = '';
                document.getElementById("login_user_button").disabled = false;
                document.getElementById("login_user_button").style.opacity = '1';
            }
            
            if(result.error) {
                alert_error(result.message);
                document.getElementById('user_password').value = '';
                document.getElementById("login_user_button").disabled = false;
                document.getElementById("login_user_button").style.opacity = '1';
            } else {
                document.getElementById('user_password').value = '';
                set_cookie("treningheten-bruker", result.cookie, 1);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/update_user.php");
    xhttp.send(user_login_data);
    return;
}
// Update user )