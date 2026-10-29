function load_page(result) {

    // Reset session tokens
    clear_tokens();

    var html = `
                <div class="" id="forside">

                    <div class="module auth-panel">

                        <div class="title">
                            Register
                        </div>

                        <div class="text-body">
                            Did you get an invitation?
                        </div>

                        <div class="action-block">
                            <form action="" onsubmit="event.preventDefault(); send_registration();">

                                <input type="email" name="email" id="email" placeholder="Email" autocomplete="email" required/>

                                <input type="text" name="first_name" id="first_name" placeholder="First name" required/>

                                <input type="text" name="last_name" id="last_name" placeholder="Last name" required/>

                                <input type="password" name="password" id="password" placeholder="Password" autocomplete="new-password" required/>

                                <input type="password" name="password_repeat" id="password_repeat" placeholder="Repeat the password" autocomplete="new-password" required/>

                                <input type="text" name="invitation_code" id="invitation_code" placeholder="Invitation code" autocomplete="one-time-code" required/>

                                <div class="auth-consent">
                                    <input class="clickable" type="checkbox" id="confirm" name="confirm" value="confirm" required>
                                    <label for="confirm" class="clickable">I confirm that {{.appName}} can store relevant information about me and that I am at least thirteen years of age.</label>
                                </div>

                                <button class="btn btn--primary btn--block" id="register-button" type="submit">Register</button>

                            </form>
                        </div>

                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Tell me more about yourself.';
    clearResponse();
    showLoggedOutMenu();
    applyInviteFromURL();
}

// applyInviteFromURL fills in the invitation code from an invite link
// (/register?invite=CODE, copied from /admin) so the new user only types their own details.
// The field stays editable in case the link was mangled.
function applyInviteFromURL() {
    var code = new URLSearchParams(window.location.search).get("invite");
    if(!code || code.trim() === "") {
        return;
    }
    document.getElementById("invitation_code").value = code.trim();
    document.querySelector("#forside .text-body").textContent = "You've been invited. Fill in your details to join.";
    document.getElementById("email").focus();
}

function send_registration(){

    var user_email = document.getElementById("email").value;
    var user_password = document.getElementById("password").value;
    var user_password_repeat = document.getElementById("password_repeat").value;
    var user_first_name = document.getElementById("first_name").value;
    var user_last_name = document.getElementById("last_name").value;
    var invitation_code = document.getElementById("invitation_code").value;

    var form_obj = { 
        "email" : user_email,
        "password" : user_password,
        "password_repeat": user_password_repeat,
        "first_name": user_first_name,
        "last_name": user_last_name,
        "invite_code": invitation_code
    };

    var form_data = JSON.stringify(form_obj);

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                console.log(e +' - Response: ' + this.responseText);
                error("Could not reach API.");
                clear_data();
                return;
            }
            
            if(result.error) {

                error(result.error);
                clear_data();

            } else {

                success(result.message);
                clear_data();
                disable_register_button();

            }

        } else {
            info("Registering...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.send(form_data);
    return false;
}

function clear_data() {
    document.getElementById("password").value = "";
    document.getElementById("password_repeat").value = ""
}

function disable_register_button() {
    document.getElementById("register-button").disabled = true;
}


// Page bootstrap (formerly an inline <script> in web/html/register.html).
var jwt;

document.addEventListener('DOMContentLoaded', function() {
    info('Loading...');
    showLoggedOutMenu();
    jwt = get_cookie('treningheten');
    get_login(jwt);
});
