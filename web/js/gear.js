// Dedicated gear management page (/gear). The gear list, render helpers and CRUD are shared
// with the exercise-page "Manage gear" modal — see web/js/gear-shared.js. This file only builds
// the page shell and points onGearChanged at the on-page list.

function load_page(result) {

    if (result !== false) {
        var login_data = JSON.parse(result);
        user_id = login_data.data.id;

        try {
            admin = login_data.data.admin;
        } catch {
            admin = false;
        }

        showAdminMenu(admin);
    } else {
        user_id = 0;
        admin = false;
    }

    var html = `
        <div id="front-page">

            <div class="module">
                <div class="text-body u-text-center">
                    Manage your shoes, bikes and other equipment. Assign gear to a session from an
                    exercise and its distance tallies up here.
                </div>
            </div>

            <div class="gear-panel">
                <div class="gear-loading" id="gear-loading"><div class="trh-spinner"></div></div>
                <div class="gear-list" id="gear-list"></div>
                ${gearAddFormHTML()}
            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Shoes, wheels and the miles on them.';
    clearResponse();

    // After each load/change, render the list into the page and hide the spinner.
    onGearChanged = function() {
        var listEl = document.getElementById("gear-list");
        if (listEl) {
            listEl.innerHTML = gearListHTML();
        }
        var loading = document.getElementById("gear-loading");
        if (loading) {
            loading.style.display = "none";
        }
    };

    if (result !== false) {
        showLoggedInMenu();
        getGear();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

// escapeHTML makes a value safe inside HTML text and attributes (gear names/brands are
// user-provided). Used by the shared render helpers.
function escapeHTML(value) {
    return String(value == null ? "" : value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}
