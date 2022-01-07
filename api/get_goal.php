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

if(!$goals) {

    echo json_encode(array("error" => false, "message" => "Ingen mål funnet.", "goal" => false));
    exit(0);

}

$goals = json_decode($goals, true);
$goal_index = false;
$now = new DateTime('NOW');
for($i = 0; $i < count($goals); $i++) {
    $goal_start = date_create_from_format('Y-m-d H:i:s', $goals[$i]['goal_start']);
    $goal_end = date_create_from_format('Y-m-d H:i:s', $goals[$i]['goal_end']);

    if($now < $goal_end && $now > $goal_start) {

        $goal_index = $i;
        break;
    }
}

if($goal_index === false) {

    echo json_encode(array("error" => false, "message" => "Ingen mål funnet.", "goal" => false));
    exit(0);

} else {

    echo json_encode(array("error" => false, "message" => "Fant et mål.", "goal" => 
                                                                                    array(
                                                                                        "goal_id" => $goals[$goal_index]['goal_id'],
                                                                                        "goal_exer_week" => $goals[$goal_index]['goal_exer_week'],
                                                                                        "goal_start" => $goals[$goal_index]['goal_start'],
                                                                                        "goal_end" => $goals[$goal_index]['goal_end']
                                                                                    )));
    exit(0);

}

?>