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
if(empty($data) || !isset($data->cookie) || !isset($data->goal_exer_week) || !isset($data->goal_compete)) {
    
    echo json_encode(array("error" => true, "message" => "Fikk ikke kjeks, konkuranse-valg eller treningsmål fra etterspørsel."));
    exit(0);
	
}

// Remove potential harmfull input
$cookie = htmlspecialchars($data->cookie);
$goal_exer_week = intval(htmlspecialchars($data->goal_exer_week));
$goal_compete = htmlspecialchars($data->goal_compete);

$cookie_object = $user->validate_user_cookie($cookie);

// Check if cookie was accepted
if(!$cookie_object) {

    echo json_encode(array("error" => true, "message" => "Kjeksen ble ikke akseptert."));
    exit(0);

}

$cookie_decoded = json_decode($cookie_object, true);

// Check if exercise goal was accepted
if($goal_exer_week < 1 || $goal_exer_week > 7) {

    echo json_encode(array("error" => true, "message" => "Treningsfrekvensen ble ikke akseptert."));
    exit(0);

}

$now = new DateTime('NOW');

if(($now->format('m') == 1 || $now->format('m') == 8) && $goal_compete) {
    $compete = '1';
} else {
    $compete = '0';
}

$goal_dates = $goal->get_current_season_dates();

if(!$goal_dates) {

    echo json_encode(array("error" => true, "message" => "Du kan ikke sette mål utenfor sesongene."));
    exit(0);

}

$goal_dates = json_decode($goal_dates, true);
$season_start = new DateTime($goal_dates['season_start']);
$season_end = new DateTime($goal_dates['season_end']);
$season_name = $goal_dates['season_name'];

$goal_start = new DateTime('NOW');
if($goal_start->format('N') != 1) {
    $correct_date = false;
    while(!$correct_date) {
        $goal_start->modify('+1 days');

        if($goal_start->format('N') == 1) {
            $correct_date = true;
        }
    }
}

$goal->user_id = $cookie_decoded['data']['user_id'];
$goals = $goal->get_goals();

$goal_index = false;
if($goals !== false) {
    $goals = json_decode($goals, true);

    for($i = 0; $i < count($goals); $i++) {
        $goal_start = date_create_from_format('Y-m-d H:i:s', $goals[$i]['goal_start']);
        $goal_end = date_create_from_format('Y-m-d H:i:s', $goals[$i]['goal_end']);

        if($season_end >= $goal_end && $season_start <= $goal_start) { 

            $goal_index = $i;
            break;

        }
    }
}

if($goal_index !== false) {

    echo json_encode(array("error" => true, "message" => "Du har allerede et aktivt mål."));
    exit(0);

} else {

    $goal->goal_exer_week = $goal_exer_week;

    $goal->goal_compete = $compete;

    $goal->goal_start = $goal_start;

    $goal->goal_end = $season_end;


    if($goal->create_goal()) {

        echo json_encode(array("error" => false, "message" => "Nytt mål satt."));
        exit(0);

    } else {

        echo json_encode(array("error" => true, "message" => "Klarte ikke sette nytt mål."));
        exit(0);

    }

}

?>