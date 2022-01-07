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
$goals = $goal->get_goals();

$now = new DateTime('NOW');

if($now->format('m') == 1 || $now->format('m') == 8) {
    $can_compete = true;
} else {
    $can_compete = false;
}

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
    echo json_encode(array("error" => true, "message" => "Du kan ikke sette mål utenfor sesongene."));
    exit(0);
}

if(!$goals) {

    echo json_encode(array("error" => false, "message" => "Ingen mål funnet.", "can_compete" => $can_compete, "season" => $season, "goal" => false));
    exit(0);

}

$goals = json_decode($goals, true);
$goal_index = false;
for($i = 0; $i < count($goals); $i++) {
    $goal_start = date_create_from_format('Y-m-d H:i:s', $goals[$i]['goal_start']);
    $goal_end = date_create_from_format('Y-m-d H:i:s', $goals[$i]['goal_end']);

    if($chosen_season_start < $goal_end && $chosen_season_end > $goal_start) {

        $goal_index = $i;
        break;
    }
}

if($goal_index === false) {

    echo json_encode(array("error" => false, "message" => "Ingen mål funnet.", "can_compete" => $can_compete, "season" => $season, "goal" => false));
    exit(0);

} else {

    $goal_started = false;

    $goal_start_chosen = new DateTime($goals[$goal_index]['goal_start']);
    if($now > $goal_start_chosen) {
        $goal_started = true;
    }

    echo json_encode(array("error" => false, "message" => "Fant et mål.", "can_compete" => $can_compete, "season" => $season, "goal" => 
                                                                                    array(
                                                                                        "goal_id" => $goals[$goal_index]['goal_id'],
                                                                                        "goal_exer_week" => $goals[$goal_index]['goal_exer_week'],
                                                                                        "goal_start" => $goals[$goal_index]['goal_start'],
                                                                                        "goal_end" => $goals[$goal_index]['goal_end'],
                                                                                        "goal_compete" => $goals[$goal_index]['goal_compete'],
                                                                                        "goal_started" => $goal_started
                                                                                    )));
    exit(0);

}

?>