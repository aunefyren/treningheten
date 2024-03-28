function load_page(result) {
    var code = "";
    var user_id = "";

    if(result !== false) {
        var login_data = JSON.parse(result);
        user_id = login_data.data.id;
        const queryString = window.location.search;
        const urlParams = new URLSearchParams(queryString);
        code = urlParams.get('code')
    }

    var html = `
        <div class="" id="front-page">    
            <div class="module" id="barbell-gif" style="">
                <img src="/assets/images/barbell.gif">
            </div>
        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Authorizing...';

    clearResponse();
    showLoggedInMenu();
    document.getElementById('navbar').style.display = 'none';

    if(result !== false) {
        showLoggedInMenu();
        setNewStravaCode(user_id, code);
    } else {
        showLoggedOutMenu();
    }
}

function setNewStravaCode(user_id, code){
    var form_obj = { 
        "strava_code" : code,
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
                return;
            }
            
            if(result.error) {
                error(result.error);
            } else {
                frontPageRedirect();
            }

        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/users/" + user_id + "/strava");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
    
}

function frontPageRedirect() {
    window.location = '/'
}