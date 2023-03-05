function load_page(result) {

    if(result !== false) {
        var login_data = JSON.parse(result);
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
        admin = false;
    }

    var html = `
                <div class="" id="front-page">
                    
                    <div class="module">
                    
                        <div class="text-body" style="text-align: center;">
                            The following posts are created by the website administrator.
                        </div>

                    </div>

                    <div class="module">

                        <div id="news-title" class="title" style="display: none;">
                            News:
                        </div>

                        <div id="news-box" class="news">
                        </div>
                        
                    </div>

                </div>
    `;

    document.getElementById('content').innerHTML = html;
    document.getElementById('card-header').innerHTML = 'NRK 2.0...';
    clearResponse();

    if(result !== false) {
        showLoggedInMenu();
        
        get_news(login_data.admin);
    } else {
        showLoggedOutMenu();
        invalid_session();
    }
}

function get_news(admin){

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

                clearResponse();
                news = result.news;

                console.log(news);

                console.log("Placing intial news: ")
                place_news(news, admin);

            }

        } else {
            info("Loading news...");
        }
    };
    xhttp.withCredentials = true;
    xhttp.open("post", api_url + "auth/news/get");
    xhttp.setRequestHeader("Content-Type", "application/json;charset=UTF-8");
    xhttp.setRequestHeader("Authorization", jwt);
    xhttp.send();
    return false;

}

function place_news(news_array, admin) {

    if(news_array.length == 0) {
        return;
    } else {
        document.getElementById("news-title").style.display = "inline-block"
    }

    var html = ''

    for(var i = 0; i < news_array.length; i++) {

        // parse date object
        try {
            var date = new Date(Date.parse(news_array[i].date));
            var date_string = date.toLocaleDateString();
        } catch {
            var date_string = "Error"
        }

        html += '<div class="news-post">'
        
        html += '<div id="news-title" class="title">';
        html += news_array[i].title
        html += '</div>';

        html += '<div id="news-body" class="text-body">';
        html += news_array[i].body
        html += '</div>';

        html += '<div id="news-body" class="text-date">';
        html += date_string
        html += '</div>';

        html += '</div>'

    }

    news_object = document.getElementById("news-box")
    news_object.innerHTML = html

}