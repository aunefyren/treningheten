function set_cookie(cname, cvalue, exdays) {
    var d = new Date();
    d.setTime(d.getTime() + (exdays*24*60*60*1000));
    var expires = "expires="+ d.toUTCString();
    document.cookie = cname + "=" + cvalue + ";" + expires + ";path=/;SameSite=None;secure";
}

function get_cookie(cname) {
    var name = cname + "=";
    var decodedCookie = decodeURIComponent(document.cookie);
    var ca = decodedCookie.split(';');
    for(var i = 0; i <ca.length; i++) {
        var c = ca[i];
        while (c.charAt(0) == ' '){
            c = c.substring(1);
        }

        if (c.indexOf(name) == 0) {
            return c.substring(name.length, c.length);
        }
    }
    return "";
}

function makeRequest (method, url, data) {
    return new Promise(function (resolve, reject) {
    var xhr = new XMLHttpRequest();
    xhr.open(method, url);
    xhr.onload = function () {
      if (this.status >= 200 && this.status < 300) {
        resolve(xhr.response);
      } else {
        reject({
          status: this.status,
          statusText: xhr.statusText
        });
      }
    };
    xhr.onerror = function () {
      reject({
        status: this.status,
        statusText: xhr.statusText
      });
    };
    if(method=="POST" && data){
        xhr.send(data);
    }else{
        xhr.send();
    }
    });
}

function show_logged_in_menu() {
    // hide login and sign up from navbar & show logout button
    document.getElementById('logg_inn').classList.add('disabled');
    document.getElementById('logg_inn').classList.remove('enabled');

    document.getElementById('registrer').classList.add('disabled');
    document.getElementById('registrer').classList.remove('enabled');

    document.getElementById('logg_ut').classList.add('enabled');
    document.getElementById('logg_ut').classList.remove('disabled');

    document.getElementById('update_account').classList.add('enabled');
    document.getElementById('update_account').classList.remove('disabled');
}

function show_logged_out_menu() {
    document.getElementById('logg_inn').classList.add('enabled');
    document.getElementById('logg_inn').classList.remove('disabled');

    document.getElementById('registrer').classList.add('enabled');
    document.getElementById('registrer').classList.remove('disabled');

    document.getElementById('logg_ut').classList.add('disabled');
    document.getElementById('logg_ut').classList.remove('enabled');

    document.getElementById('update_account').classList.add('disabled');
    document.getElementById('update_account').classList.remove('enabled');
}

function toggle_navbar() {
    var x = document.getElementById("navbar");
    var y = document.getElementById("nav-logo");
    if (x.className === "navbar") {
      x.className += " responsive";
      y.className += " responsive";
    } else {
      x.className = "navbar";
      y.className = "nav-logo";
    }
  }