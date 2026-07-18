// OAuth 2.0 authorization consent page. Invoked by get_login(jwt): result is the
// validated session (logged in) or false (logged out).
function load_page(result) {

    showLoggedInMenu();
    document.getElementById('card-header').innerHTML = 'Authorize application';
    document.getElementById('content').innerHTML = '';
    clearResponse();

    var params = new URL(window.location.href).searchParams;
    var req = {
        client_id: params.get('client_id') || '',
        redirect_uri: params.get('redirect_uri') || '',
        response_type: params.get('response_type') || 'code',
        scope: params.get('scope') || '',
        state: params.get('state') || '',
        code_challenge: params.get('code_challenge') || '',
        code_challenge_method: params.get('code_challenge_method') || '',
        resource: params.get('resource') || ''
    };

    // Not logged in: send the user to log in, then return to this exact request.
    if(result === false) {
        var here = window.location.pathname + window.location.search;
        window.location = '/login?return=' + encodeURIComponent(here);
        return;
    }

    fetch_authorize_info(req);
}

// Validate the request and load client details for the consent screen.
function fetch_authorize_info(req) {
    var query = "client_id=" + encodeURIComponent(req.client_id)
        + "&redirect_uri=" + encodeURIComponent(req.redirect_uri)
        + "&response_type=" + encodeURIComponent(req.response_type)
        + "&scope=" + encodeURIComponent(req.scope)
        + "&code_challenge=" + encodeURIComponent(req.code_challenge)
        + "&code_challenge_method=" + encodeURIComponent(req.code_challenge_method);

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
                error(result.error_description || result.error);
                return;
            }
            render_consent(result, req);
        }
    };
    xhttp.open("get", api_url + "oauth/authorize?" + query);
    xhttp.send();
}

function render_consent(info, req) {
    var scopeList = (info.scope || "").split(" ").filter(function(s){ return s; });
    var scopeHtml = "";
    for(var i = 0; i < scopeList.length; i++) {
        scopeHtml += "<li>" + scopeList[i] + "</li>";
    }
    if(scopeHtml === "") {
        scopeHtml = "<li>Basic access</li>";
    }

    var html = `
    <div class="auth-panel">
        <div class="title">` + info.client_name + `</div>
        <div class="text-body">
            <b>` + info.client_name + `</b> wants to access your Treningheten account.
        </div>

        <div class="text-body">This will grant the following access:</div>
        <ul class="auth-scopes">` + scopeHtml + `</ul>

        <div class="action-block">
            <button class="btn btn--primary btn--block" id="approve-button" type="button">Approve</button>
            <button class="btn btn--ghost btn--block u-mt-2" id="deny-button" type="button">Deny</button>
        </div>
    </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('approve-button').onclick = function(){ submit_decision(req, true); };
    document.getElementById('deny-button').onclick = function(){ submit_decision(req, false); };
}

function submit_decision(req, approve) {
    var body = "client_id=" + encodeURIComponent(req.client_id)
        + "&redirect_uri=" + encodeURIComponent(req.redirect_uri)
        + "&scope=" + encodeURIComponent(req.scope)
        + "&state=" + encodeURIComponent(req.state)
        + "&code_challenge=" + encodeURIComponent(req.code_challenge)
        + "&code_challenge_method=" + encodeURIComponent(req.code_challenge_method)
        + "&resource=" + encodeURIComponent(req.resource)
        + "&approve=" + (approve ? "true" : "false");

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
                error(result.error_description || result.error);
                return;
            }
            if(result.redirect) {
                window.location = result.redirect;
            }
        } else {
            info("Processing...");
        }
    };
    xhttp.open("post", api_url + "oauth/authorize/decision");
    xhttp.setRequestHeader("Content-Type", "application/x-www-form-urlencoded");
    xhttp.setRequestHeader("Authorization", "Bearer " + jwt);
    xhttp.send(body);
}
