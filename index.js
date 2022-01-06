function load_page() {
    show_logged_out_menu();

    var activate_email = getParameterByName('activate_email');
    var activate_hash = getParameterByName('activate_hash');
    if(activate_email !== null & activate_hash !== null) {
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
            }
            
            if(result.error) {
                set_cookie("treningheten-bruker", "", 1);
                load_page_home();
            } else {
                load_page_home();
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/validate_user_cookie.php");
    xhttp.send(json_cookie);
    return;
}

function load_page_verify(activate_email, activate_hash) {
    var html = `
        <div>
            <h1>Aktiverer brukeren din</h1>
            <p>Vær tålmodig.</p>
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

function load_page_home() {
    var html = `
        <div>
            <h1>Tren da</h1>
            <p>Siden er under konstruksjon.</p>
        </div>
    `;

    document.getElementById('content-box').innerHTML = html;
}

// ( Create user
function load_page_register() {

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

            <div class='form-group'>
                <label for="code_hash" title="Hva er invitasjonskoden din?">Invitasjonskode:</label>
                <input type="text" name="code_hash" id="code_hash" class="form-input" value="" autocomplete="off" required />
            </div>

            <div class='form-group newline'>
                <label for="accept_terms_check" title="Godkjenn vilkårene for å lage brukeren din.">Jeg godtar at denne siden lagrer data for eget bruk og at jeg er over 18 år:</label>
                <input type="checkbox" class="form-control" id="accept_terms_check" required>
            </div>


            <div class='form-group newline'>
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
            }
            
            if(result.error) {
                alert_error(result.message);
                document.getElementById("register_user_button").disabled = false;
                document.getElementById("register_user_button").style.opacity = '1';
            } else {
                alert_success(result.message);
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
                alert_success(result.message);
                set_cookie("treningheten-bruker", result.cookie, 1);
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", "api/get_login_cookie.php");
    xhttp.send(user_login_data);
    return;
}
// Create user )