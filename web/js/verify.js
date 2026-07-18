function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);

        if(login_data.error && !login_data.error.toLowerCase().includes("you must verify your account")) {
            frontPageRedirect();
            return;
        }

    }

    var html = `
                <div class="" id="front-page">

                    <div class="module auth-panel">

                        <div class="title">
                            {{.appName}}
                        </div>

                        <div class="text-body u-text-center">
                            You must verify your account by giving us the access code we e-mailed you.
                        </div>

                        <div class="action-block">
                            <form action="" onsubmit="event.preventDefault(); verifyAccount();">
                                <input type="text" name="email_code" id="email_code" placeholder="Code" autocomplete="one-time-code" required />
                                <button class="btn btn--primary btn--block" id="verify-button" type="submit">Verify</button>
                            </form>
                        </div>

                    </div>

                    <div class="module auth-alt">
                        <a class="u-fs-sm u-pointer" onclick="newCode();">Send me a new code!</a>
                    </div>

                </div>

    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Robot or human?';

    clearResponse();
    showLoggedInMenu();
    document.getElementById('navbar').style.display = 'none';

    if(result !== false) {
        showLoggedInMenu();
    } else {
        showLoggedOutMenu();
        document.getElementById('front-page-text').innerHTML = 'Log in to use the platform.';
        document.getElementById('log-in-button').style.display = 'inline-block';
    }
}

function verifyAccount(){

    var email_code = document.getElementById("email_code").value;

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

                // store the OAuth token pair (auto-login after verification)
                if(result.data) {
                    store_tokens(result.data.access_token, result.data.refresh_token);
                }
                frontPageRedirect();

            }

        } else {
            info("Verifying account...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users/verify/" + email_code);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
    
}

function newCode(){

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

                success(result.message)

            }

        } else {
            info("Sending new code...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "open/users/verification");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
    
}

function frontPageRedirect() {

    window.location = '/'

}