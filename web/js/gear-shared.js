// Shared gear management — used by BOTH the /gear page (gear.js) and the exercise-page
// "Manage gear" modal (exercise.js). One render + one set of CRUD calls + one `.gear-*` class
// set, so the two views can't drift (they previously duplicated all of this).
//
// Each page assigns `onGearChanged` to re-render its own view after a mutation. `escapeHTML`,
// `api_url`, `jwt`, `error`, `success` come from the page/functions.js (both callers define them).

var gearList = [];

// onGearChanged runs after the gear list is (re)loaded — each page renders its own view.
var onGearChanged = function() {};

function gearTypeGlyph(type) {
    switch (type) {
        case "bike":  return "🚲";
        case "other": return "🎒";
        default:      return "👟";
    }
}

function gearTypeOptions(selected) {
    return ["shoe", "bike", "other"].map(function(t) {
        return `<option value="${t}" ${t === selected ? "selected" : ""}>${t.charAt(0).toUpperCase() + t.slice(1)}</option>`;
    }).join("");
}

// gearItemHTML renders one gear card — identical on the page and in the modal. Strava-synced
// gear owns its identity (name/type/brand), so those fields are disabled; only primary/retired/
// delete stay editable.
function gearItemHTML(gear) {
    var isStrava = !!gear.strava_gear_id;
    var readonly = isStrava ? "disabled" : "";
    var stravaTag = isStrava ? `<span class="gear-badge">Strava</span>` : "";
    var dist = gear.distance ? gear.distance.toFixed(1) : "0.0";
    var brand = gear.brand || "";
    var retiredClass = gear.retired ? " gear-item-retired" : "";

    return `
        <div class="gear-item${retiredClass}" id="gear-item-${gear.id}">
            <div class="gear-glyph">${gearTypeGlyph(gear.type)}</div>
            <div class="gear-main">
                <div class="gear-row-top">
                    <input class="gear-input gear-name" type="text" value="${escapeHTML(gear.name)}" ${readonly} onchange="updateGearField('${gear.id}', 'name', this.value)" title="Name">
                    <select class="gear-select" ${readonly} onchange="updateGearField('${gear.id}', 'type', this.value)" title="Type">${gearTypeOptions(gear.type)}</select>
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
}

// gearListHTML renders the whole list, or the empty state.
function gearListHTML() {
    if (!gearList.length) {
        return `<div class="gear-empty">No gear yet. Add a pair of shoes or a bike below.</div>`;
    }
    return gearList.map(gearItemHTML).join("");
}

// gearAddFormHTML renders the "Add gear" block (shared by page + modal).
function gearAddFormHTML() {
    return `
        <div class="gear-add">
            <div class="gear-add-title">Add gear</div>
            <div class="gear-add-fields">
                <input class="gear-input" type="text" id="new-gear-name" placeholder="e.g. Nike Pegasus">
                <select class="gear-select" id="new-gear-type">${gearTypeOptions("shoe")}</select>
                <input class="gear-input" type="text" id="new-gear-brand" placeholder="Brand (optional)">
                <button class="btn btn--primary" type="submit" onclick="createGear();"><img src="/assets/plus.svg" class="color-invert">Add gear</button>
            </div>
        </div>
    `;
}

// getGear loads the list, then hands off to the page's onGearChanged to render its view.
function getGear() {
    var xhttp = new XMLHttpRequest();
    xhttp.onreadystatechange = function() {
        if (this.readyState != 4) {
            return;
        }
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
        onGearChanged();
    };
    xhttp.withCredentials = true;
    xhttp.open("get", api_url + "auth/gear");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
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
        if (this.readyState != 4) {
            return;
        }
        try {
            result = JSON.parse(this.responseText);
        } catch(e) {
            error("Could not reach API.");
            return;
        }
        if (result.error) {
            error(result.error);
            return;
        }
        success("Gear added.");
        // Clear the add fields (the page keeps its static add form; the modal re-renders it anyway).
        try {
            document.getElementById("new-gear-name").value = "";
            document.getElementById("new-gear-brand").value = "";
        } catch (e) { /* fields already gone if the view re-rendered */ }
        getGear();
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
        if (this.readyState != 4) {
            return;
        }
        try {
            result = JSON.parse(this.responseText);
        } catch(e) {
            error("Could not reach API.");
            return;
        }
        if (result.error) {
            error(result.error);
            return;
        }
        success("Gear updated.");
        // Setting one gear primary demotes the others, so reload the whole list.
        getGear();
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
        if (this.readyState != 4) {
            return;
        }
        try {
            result = JSON.parse(this.responseText);
        } catch(e) {
            error("Could not reach API.");
            return;
        }
        if (result.error) {
            error(result.error);
            return;
        }
        success("Gear deleted.");
        getGear();
    };
    xhttp.withCredentials = true;
    xhttp.open("delete", api_url + "auth/gear/" + gearID);
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
}
