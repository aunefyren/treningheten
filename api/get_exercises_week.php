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
require(dirname(__FILE__) . '/objects/exercise.php');

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$user = new User($db);
$exercise = new Exercise($db);
$data = json_decode(file_get_contents("php://input"));

// If POST data is empty or wrong
if(empty($data) || !isset($data->cookie) || !isset($data->goal_id)) {
    
    echo json_encode(array("error" => true, "message" => "Fikk ikke kjeks eller mål-id fra innlogging."));
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
$exercise->goal_id = $data->goal_id;
$exercises = $exercise->get_exercises();

$now = new DateTime('NOW');

if(!$exercises) {

    echo json_encode(array("error" => false, "message" => "Ingen trening funnet.", "exercises" => array(), "week_number" => $now->format('W')));
    exit(0);

}

$week = array(
                'days' => array(
                    1 => false,
                    2 => false,
                    3 => false,
                    4 => false,
                    5 => false,
                    6 => false,
                    7 => false
                ), 
                "week_number" => $now->format('W')
            );

$exercises = json_decode($exercises, true);
for($i = 0; $i < count($exercises); $i++) {
    $exer = new DateTime($exercises[$i]['exer_date']);
    if($exer->format('W') === $now->format('W') && $exer->format('Y') === $now->format('Y')) {
        $week['days'][$exer->format('N')] = true;
    }
}

echo json_encode(array("error" => false, "message" => "Fant trening for uken.", "exercises" => $week));
exit(0);
?>