<?php
// Required headers
header("Access-Control-Allow-Origin: *");
header("Content-Type: application/json; charset=UTF-8");
header("Access-Control-Allow-Methods: POST");
header("Access-Control-Max-Age: 3600");
header("Access-Control-Allow-Headers: Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With");

// Files needed to use objects
require(dirname(__FILE__) . '/config/database.php');
require(dirname(__FILE__) . '/objects/user.php');
require(dirname(__FILE__) . '/objects/goal.php');

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$user = new User($db);
$goal = new Goal($db);
$data = json_decode(file_get_contents("php://input"));

// If POST data is empty or wrong
if(empty($data) || !isset($data->cookie)) {
    
    echo json_encode(array("error" => true, "message" => "Fikk ikke kjeks fra innlogging."));
    exit(0);
	
}

// Remove potential harmfull input
$cookie = htmlspecialchars($data->cookie);

$cookie_object = $user->validate_user_cookie($cookie);

// Check if cookie was accepted
if(!$cookie_object) {

    echo json_encode(array("error" => true, "message" => "Kjeksen ble ikke akseptert."));
    exit(0);

}

$cookie_decoded = json_decode($cookie_object, true);
$goal->user_id = $cookie_decoded['data']['user_id'];

$now = new DateTime('NOW');

if($now->format('m') == 1 || $now->format('m') == 8) {
    $can_compete = true;
} else {
    $can_compete = false;
}

$goals = $goal->get_goal();

$season_start_1 = new DateTime($now->format('Y') . '-01-01');
$season_start_2 = new DateTime($now->format('Y') . '-08-01');

$season_end_1 = new DateTime($now->format('Y') . '-06-30');
if($season_end_1->format('N') !== 7) {
    $correct_date = false;
    while(!$correct_date) {
        $season_end_1->modify('-1 days');

        if($season_end_1->format('N') == 7) {
            $correct_date = true;
        }
    }
}

$season_end_2 = new DateTime($now->format('Y') . '-12-24');
if($season_end_2->format('N') === 7) {
    $season_end_2->modify('-1 days');
}
if($season_end_2->format('N') !== 7) {
    $correct_date = false;
    while(!$correct_date) {
        $season_end_2->modify('-1 days');

        if($season_end_2->format('N') == 7) {
            $correct_date = true;
        }
    }
}

if($now > $season_start_1 && $now < $season_end_1) {
    $season = 1;
    $chosen_season_start = $season_start_1;
    $chosen_season_end = $season_end_1;
} else if($now > $season_start_2 && $now < $season_end_2) {
    $season = 2;
    $chosen_season_start = $season_start_2;
    $chosen_season_end = $season_end_2;
} else {
    return false;
}

if($goals === false) {

    echo json_encode(array( "error" => false, 
                            "message" => "Ingen mål funnet.",
                            "can_compete" => $can_compete, 
                            "season_start" => $chosen_season_start->format('d.m.Y'),
                            "season_end" => $chosen_season_end->format('d.m.Y'),
                            "goal" => false));
    exit(0);

} else {

    $goals = json_decode($goals, true);

    echo json_encode(array( "error" => false, 
                            "message" => "Fant et mål.", 
                            "can_compete" => $can_compete, 
                            "season_start" => $goals['season_start'],
                            "season_end" => $goals['season_end'],  
                            "goal" => 
                                array(
                                    "goal_id" => $goals['goal']['goal_id'],
                                    "goal_exer_week" => $goals['goal']['goal_exer_week'],
                                    "goal_start" => $goals['goal']['goal_start'],
                                    "goal_end" => $goals['goal']['goal_end'],
                                    "goal_compete" => $goals['goal']['goal_compete'],
                                    "goal_started" => $goals['goal']['goal_started']
                                )
                            ));
    exit(0);

}

?>