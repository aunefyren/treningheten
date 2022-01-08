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

if($goals === false) {

    echo json_encode(array( "error" => false, 
                            "message" => "Ingen mål funnet.",
                            "can_compete" => $can_compete, 
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