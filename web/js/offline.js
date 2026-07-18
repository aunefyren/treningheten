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
            <div class="module">
                You are offline :/
            </div>
            <div class="module">
                <button type="button" onclick="frontPageRedirect();" class="btn btn--primary"><img src="assets/refresh-cw.svg" class="color-invert">Reload</button>
            </div>
        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'No cell reception up here?';
    clearResponse();

    checkOnlineAndRedirect();
}

function checkOnlineAndRedirect(retryDelayMs = 10000) {
    fetch('/manifest.json', { method: 'GET', cache: 'no-store' })
        .then(response => {
            if (response.ok) {
                frontPageRedirect();
            } else {
                setTimeout(() => checkOnlineAndRedirect(retryDelayMs), retryDelayMs);
            }
        })
        .catch(() => {
            setTimeout(() => checkOnlineAndRedirect(retryDelayMs), retryDelayMs);
        });
}