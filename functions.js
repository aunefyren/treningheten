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
    document.getElementById('log_in_tab').classList.add('disabled');
    document.getElementById('log_in_tab').classList.remove('enabled');

    document.getElementById('register_tab').classList.add('disabled');
    document.getElementById('register_tab').classList.remove('enabled');

    document.getElementById('log_out_tab').classList.add('enabled');
    document.getElementById('log_out_tab').classList.remove('disabled');

    document.getElementById('update_account').classList.add('enabled');
    document.getElementById('update_account').classList.remove('disabled');
}

function show_logged_out_menu() {
    document.getElementById('log_in_tab').classList.add('enabled');
    document.getElementById('log_in_tab').classList.remove('disabled');

    document.getElementById('register_tab').classList.add('enabled');
    document.getElementById('register_tab').classList.remove('disabled');

    document.getElementById('log_out_tab').classList.add('disabled');
    document.getElementById('log_out_tab').classList.remove('enabled');

    document.getElementById('update_account').classList.add('disabled');
    document.getElementById('update_account').classList.remove('enabled');
}

function remove_active_menu() {
  document.getElementById('log_in_tab').classList.remove('active');
  document.getElementById('register_tab').classList.remove('active');
  document.getElementById('log_out_tab').classList.remove('active');
  document.getElementById('update_account').classList.remove('active');
}

function add_active_menu(tab_id) {
  document.getElementById(tab_id).classList.add('active');
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

function alert_clear() {
  document.getElementById('response').innerHTML = '';
}

function alert_error(message) {
  document.getElementById('response').innerHTML = '<div class="response-box" style="background-color:var(--red)"><div>' + message + '</div><img onclick="alert_clear()" src="assets/close.svg" class="alert_close"></div>';
  window.scrollTo(0, 0);
}

function alert_info(message) {
  document.getElementById('response').innerHTML = '<div class="response-box" style="background-color:var(--blue)"><div>' + message + '</div><img onclick="alert_clear()" src="assets/close.svg" class="alert_close"></div>';
  window.scrollTo(0, 0);
}

function alert_success(message) {
  document.getElementById('response').innerHTML = '<div class="response-box" style="background-color:var(--green)"><div>' + message + '</div><img onclick="alert_clear()" src="assets/close.svg" class="alert_close"></div>';
  window.scrollTo(0, 0);
}

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

function getParameterByName(name, url = window.location.href) {
    name = name.replace(/[\[\]]/g, '\\$&');
    var regex = new RegExp('[?&]' + name + '(=([^&#]*)|&|#|$)'),
        results = regex.exec(url);
    if (!results) return null;
    if (!results[2]) return '';
    return decodeURIComponent(results[2].replace(/\+/g, ' '));
}

var ignoreNav = document.getElementById('nav');

document.addEventListener('click', function(event) {
  var isClickInsideElement = ignoreNav.contains(event.target);
  if (!isClickInsideElement) {
    var nav_classlist = document.getElementById('navbar').classList;
    if (nav_classlist.contains('responsive')) {
      toggle_navbar();
    }
  }
});

function trigger_fireworks(number) {
  if(number >0) {
  document.getElementById('pyro').style.display = 'block';
  setTimeout(function () {
      trigger_fireworks(number-1);
  }, 5000);
  } else {
      document.getElementById('pyro').style.display = 'none';
  }
}

function get_base64(file, onLoadCallback) {
  return new Promise(function(resolve, reject) {
      var reader = new FileReader();
      reader.onload = function() { resolve(reader.result); };
      reader.onerror = reject;
      reader.readAsDataURL(file);
  });
}