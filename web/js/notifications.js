function notification_button() {
    navigator.serviceWorker.ready.then(function(registration) {
        return registration.pushManager.getSubscription()
            .then(function(subscription) {
                if (subscription && PermissionStatus.state !== 'granted') {
                    document.getElementById('notification_button_div').style.display = 'none';
                }
                console.log('PermissionStatus.state: ' + PermissionStatus.state);
                console.log('Subscription: ' + subscription);
        }) 
    });
}

function register_push(jwtToken, appPubkey, sunday_alert, achievement_alert) {

    settings = {
        "sunday_alert": sunday_alert,
        "achievement_alert": achievement_alert
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
                document.getElementById('notification_button_div').style.display = 'none';
                return fetch(api_url + "auth/notification/subscribe", {
                  method: "POST",
                  headers: {"Content-Type": "application/json", "Authorization": jwtToken},
                  body: JSON.stringify({"subscription": subscription, "settings": settings})
                });
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