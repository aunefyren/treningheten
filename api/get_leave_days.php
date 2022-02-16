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

$goal_dates = $goal->get_current_season_dates();

if(!$goal_dates) {

    echo json_encode(array( "error" => false, 
                            "message" => "Ingen mål funnet. Ingen sesong funnet.",
                            "can_compete" => $can_compete, 
                            "season_start" => false,
                            "season_end" => false,
                            "season_name" => false,
                            "goal" => false));
    exit(0);

}

$goal_dates = json_decode($goal_dates, true);
$season_start = new DateTime($goal_dates['season_start']);
$season_end = new DateTime($goal_dates['season_end']);
$season_name = $goal_dates['season_name'];

$goals = $goal->get_goal($season_start, $season_end);

if($goals === false) {

    echo json_encode(array( "error" => false, 
                            "message" => "Ingen mål funnet.",
                            "can_compete" => $can_compete, 
                            "season_start" => $season_start->format('d.m.Y'),
                            "season_end" => $season_end->format('d.m.Y'),
                            "season_name" => $season_name,
                            "goal" => false));
    exit(0);

} else {

    $goals = json_decode($goals, true);

    echo json_encode(array( "error" => false, 
                            "message" => "Fant et mål.", 
                            "can_compete" => $can_compete, 
                            "season_start" => $season_start->format('d.m.Y'),
                            "season_end" => $season_end->format('d.m.Y'),
                            "season_name" => $season_name,
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