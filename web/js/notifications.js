function register_push(jwtToken, appPubkey, sunday_alert, achievement_alert, news_alert) {

    settings = {
        "sunday_alert": sunday_alert,
        "achievement_alert": achievement_alert,
        "news_alert": news_alert
    }

    console.log("VAPID public key: " + appPubkey)

    navigator.serviceWorker.ready.then(function(registration) {

        return registration.pushManager.getSubscription()
            .then(function(subscription) {
                if (subscription) {
                    return subscription;
                }

                return registration.pushManager.subscribe({
                    userVisibleOnly: true,
                    applicationServerKey: urlBase64ToUint8Array(appPubkey)
                });
            }) 
            .then(function(subscription) {
                //console.log(JSON.stringify({ subscription: subscription }));
                //document.getElementById('notification_button_div').style.display = 'none';
                return fetch(api_url + "auth/notification/subscribe", {
                  method: "POST",
                  headers: {"Content-Type": "application/json", "Authorization": jwtToken},
                  body: JSON.stringify({"subscription": subscription, "settings": settings})
                });
            })
            .then(function() {
                success("Subscription created.")
            });
    });
}

function urlBase64ToUint8Array(base64String) {
    var padding = '='.repeat((4 - base64String.length % 4) % 4);
    var base64 = (base64String + padding)
        .replace(/\-/g, '+')
        .replace(/_/g, '/');

    var rawData = window.atob(base64);
    var outputArray = new Uint8Array(rawData.length);

    for (var i = 0; i < rawData.length; ++i)  {
        outputArray[i] = rawData.charCodeAt(i);
    }

    return outputArray;
}

function create_push(vapid_public_key) {

    navigator.serviceWorker.ready.then(function(registration) {
        return registration.pushManager.getSubscription()
            .then(function(subscription) {
                if (subscription && PermissionStatus.state !== 'granted') {
                    update_subscription(vapid_public_key, subscription);
                } else {
                    create_new_subscription(vapid_public_key);
                }
            }
        ) 
    });

}

function create_new_subscription(vapid_public_key) {

    var sunday_alert = document.getElementById("notification-reminder-toggle").checked;
    var achievement_alert = document.getElementById("notification-achievement-toggle").checked;
    var news_alert = document.getElementById("notification-news-toggle").checked;

    register_push(jwt, vapid_public_key, sunday_alert, achievement_alert, news_alert);

}

function update_subscription(vapid_public_key, subscription) {

    var sunday_alert = document.getElementById("notification-reminder-toggle").checked;
    var achievement_alert = document.getElementById("notification-achievement-toggle").checked;
    var news_alert = document.getElementById("notification-news-toggle").checked;
    
    var form_obj = { 
        "endpoint" : subscription.endpoint,
        "sunday_alert": sunday_alert,
        "achievement_alert": achievement_alert,
        "news_alert": news_alert
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
                
                success(result.message);
                
            }

        } else {
            // info("");
        }
    };
    
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/notification/subscription/update");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;
    
}

function CheckForSubscription() {

    navigator.serviceWorker.ready.then(function(registration) {
        return registration.pushManager.getSubscription()
            .then(function(subscription) {
                if (subscription && PermissionStatus.state !== 'granted') {
                    GetSubscriptionSettings(subscription.endpoint);
                }
            }
        ) 
    });

}

function GetSubscriptionSettings(enpoint) {

    var form_obj = { 
        "endpoint" : enpoint
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
            
            if(result.error == "No subscription found.") {

                DeregisterPush()
                
            } else if(result.error) {

                error(result.error);

            } else {
                
                PlaceSubscriptionData(result.subscription);
                
            }

        } else {
            // info("");
        }
    };

    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/notification/subscription/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send(form_data);
    return false;

}

function DeregisterPush() {
    navigator.serviceWorker.ready.then(function(registration) {
        return registration.pushManager.getSubscription()
            .then(function(subscription) {
                if (subscription) {
                    console.log("De-registering subscription.")
                    registration.unregister();
                }
            }) 
    });
}