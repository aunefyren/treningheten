// Dedicated gear management page (/gear). Reuses the same backend as the exercise-page
// gear modal (GET/POST /auth/gear, PUT/DELETE /auth/gear/:id) — see controllers/gear.go
// and docs/gear.md — but renders the list inline as a dark instrument panel instead of in
// a pop-up. Gear is assigned to a session from an exercise; distances tally here.

var gearList = [];

// load_page is the entry point invoked by get_login (see gear.html).
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
                <div class="text-body" style="text-align: center;">
                    Manage your shoes, bikes and other equipment. Assign gear to a session from an
                    exercise and its distance tallies up here.
                </div>
            </div>

            <div class="gear-panel">

                <div class="gear-loading" id="gear-loading"><div class="trh-spinner"></div></div>

                <div class="gear-list" id="gear-list"></div>

                <div class="gear-add">
                    <div class="gear-add-title">Add gear</div>
                    <div class="gear-add-fields">
                        <input class="gear-input" type="text" id="new-gear-name" placeholder="e.g. Nike Pegasus">
                        <select class="gear-select" id="new-gear-type">
                            <option value="shoe">Shoe</option>
                            <option value="bike">Bike</option>
                            <option value="other">Other</option>
                        </select>
                        <input class="gear-input" type="text" id="new-gear-brand" placeholder="Brand (optional)">
                        <button class="gear-btn" type="submit" onclick="createGear();"><img src="/assets/plus.svg">Add gear</button>
                    </div>
                </div>

            </div>

        </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'Shoes, wheels and the miles on them.';
    clearResponse();

    if (result !== false) {
        showLoggedInMenu();
        getGear();
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

// escapeHTML makes a value safe inside HTML text and attributes (gear names/brands are
// user-provided). Matches the helper used on the exercise page.
function escapeHTML(value) {
    return String(value == null ? "" : value)
        .replace(/&/g, "&amp;")
        .replace(/</g, "&lt;")
        .replace(/>/g, "&gt;")
        .replace(/"/g, "&quot;")
        .replace(/'/g, "&#39;");
}

// gearTypeGlyph is the little emoji shown in a gear card's tile, by type.
function gearTypeGlyph(type) {
    switch (type) {
        case "bike":  return "🚲";
        case "other": return "🎒";
        default:      return "👟";
    }
}

function getGear() {
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
                return;
            }
            gearList = result.gear || [];
            renderGearList();
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/gear");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;
}

function renderGearList() {
    var loading = document.getElementById("gear-loading");
    if (loading) {
        loading.style.display = "none";
    }
    var listEl = document.getElementById("gear-list");
    if (!listEl) {
        return;
    }

    if (!gearList.length) {
        listEl.innerHTML = `<div class="gear-empty">No gear yet. Add a pair of shoes or a bike below.</div>`;
        return;
    }

    var typeOptions = function(selected) {
        return ["shoe", "bike", "other"].map(function(t) {
            return `<option value="${t}" ${t === selected ? "selected" : ""}>${t.charAt(0).toUpperCase() + t.slice(1)}</option>`;
        }).join("");
    };

    var html = "";
    gearList.forEach(function(gear) {
        // Strava-synced gear's identity (name/type/brand) is owned by Strava, so those
        // fields are read-only here — only primary/retired/delete stay editable.
        var isStrava = !!gear.strava_gear_id;
        var readonly = isStrava ? "disabled" : "";
        var stravaTag = isStrava ? `<span class="gear-badge">Strava</span>` : "";
        var dist = gear.distance ? gear.distance.toFixed(1) : "0.0";
        var brand = gear.brand || "";
        var retiredClass = gear.retired ? " gear-item-retired" : "";

        html += `
            <div class="gear-item${retiredClass}">
                <div class="gear-glyph">${gearTypeGlyph(gear.type)}</div>
                <div class="gear-main">
                    <div class="gear-row-top">
                        <input class="gear-input gear-name" type="text" value="${escapeHTML(gear.name)}" ${readonly} onchange="updateGearField('${gear.id}', 'name', this.value)" title="Name">
                        <select class="gear-select" ${readonly} onchange="updateGearField('${gear.id}', 'type', this.value)" title="Type">${typeOptions(gear.type)}</select>
                        ${stravaTag}
                    </div>
                    <div class="gear-row-mid">
                        <input class="gear-input" type="text" value="${escapeHTML(brand)}" ${readonly} placeholder="Brand" onchange="updateGearField('${gear.id}', 'brand', this.value)">
                    </div>
                    <div class="gear-row-toggles">
                        <label class="gear-toggle"><input type="checkbox" ${gear.is_primary ? "checked" : ""} onchange="updateGearField('${gear.id}', 'is_primary', this.checked)"> Primary</label>
                        <label class="gear-toggle"><input type="checkbox" ${gear.retired ? "checked" : ""} onchange="updateGearField('${gear.id}', 'retired', this.checked)"> Retired</label>
                        <img src="/assets/trash-2.svg" class="gear-del clickable" title="Delete gear" onclick="deleteGear('${gear.id}')">
                    </div>
                </div>
                <div class="gear-distance" title="Total logged distance">
                    <span class="gear-distance-value">${dist}</span>
                    <span class="gear-distance-unit">km</span>
                </div>
            </div>
        `;
    });

    listEl.innerHTML = html;
}

function createGear() {
    var name = document.getElementById("new-gear-name").value.trim();
    if (name === "") {
        error("Gear must have a name.");
        return;
    }
    var brand = document.getElementById("new-gear-brand").value.trim();
    var form_obj = {
        "name": name,
        "type": document.getElementById("new-gear-type").value,
        "brand": brand !== "" ? brand : null
    };

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                error("Could not reach API.");
                return;
            }
            if (result.error) {
                error(result.error);
            } else {
                success("Gear added.");
                document.getElementById("new-gear-name").value = "";
                document.getElementById("new-gear-brand").value = "";
                getGear();
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/gear");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(JSON.stringify(form_obj));
}

function updateGearField(gearID, field, value) {
    var form_obj = {};
    form_obj[field] = value;

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                error("Could not reach API.");
                return;
            }
            if (result.error) {
                error(result.error);
            } else {
                success("Gear updated.");
                // Setting one gear primary demotes the others, so reload the whole list.
                getGear();
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("put", api_url + "auth/gear/" + gearID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(JSON.stringify(form_obj));
}

function deleteGear(gearID) {
    if (!confirm("Delete this gear? Its logged distance history stays on the sessions it was assigned to.")) {
        return;
    }

    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState == 4) {
            try {
                result = JSON.parse(this.responseText);
            } catch(e) {
                error("Could not reach API.");
                return;
            }
            if (result.error) {
                error(result.error);
            } else {
                success("Gear deleted.");
                getGear();
            }
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/gear/" + gearID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}
