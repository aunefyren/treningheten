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
require(dirname(__FILE__) . '/objects/exercise.php');

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$user = new User($db);
$goal = new Goal($db);
$exercise = new Exercise($db);
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

} else {
    $cookie_object = json_decode($cookie_object, true);
}

$all_goals = $goal->get_goals_all();

if(!$all_goals) {

    echo json_encode(array("error" => true, "message" => "Ingen mål funnet."));
    exit(0);

}

$all_goals = json_decode($all_goals, true);

$goal_dates = $goal->get_current_season_dates();

if(!$goal_dates) {

    echo json_encode(array( "error" => true, 
                            "message" => "Fant ikke nåværende sesong."
                        ));
    exit(0);

}

$goal_dates = json_decode($goal_dates, true);
$season_start = new DateTime($goal_dates['season_start']);
$season_end = new DateTime($goal_dates['season_end']);

$goal_id_list = array();

$now = new DateTime('NOW');

for($i = 0; $i < count($all_goals); $i++) {
    $goal_start = date_create_from_format('Y-m-d H:i:s', $all_goals[$i]['goal_start']);
    $goal_end = date_create_from_format('Y-m-d H:i:s', $all_goals[$i]['goal_end']);
    $user_id = $all_goals[$i]['user_id'];
    if($goal_start >= $season_start && $goal_end <= $season_end && ($goal_start->format('Y-m-d') == $now->format('Y-m-d') || $goal_start < $now) && $user_id == $cookie_object["data"]["user_id"]) {
        array_push($goal_id_list, array('goal_id' => $all_goals[$i]['goal_id'], 'user_id' => $all_goals[$i]['user_id'], 'exercise_found' => false));
    }
}

if(count($goal_id_list) !== 1) {

    echo json_encode(array("error" => true, "message" => "Fant flere mål? Weird..."));
    exit(0);
}

$exercise->goal_id = $goal_id_list[0]['goal_id'];
$data = $exercise->get_exercises_stats();

if(!$data) {

    echo json_encode(array("error" => true, "message" => "Feilet i å finne treninger for ditt mål."));
    exit(0);

} else {
    $data = json_decode($data, true);

    $leave = 0;

    for($j = 0; $j < count($data); $j++) {
        if($data[$j]['exer_leave']) {
            $leave++;
        }
    }

}

$user->user_id = $cookie_object["data"]["user_id"];
$user->get_user_leave();

if($user->user_leave-$leave !> 0) {

    echo json_encode(array("error" => false, "message" => "Du er tom for sykedager."));
    exit(0);

}

for($j = 0; $j < count($data); $j++) {
    print_r($data[$j]);
}

echo json_encode(array("error" => false, "message" => "Fant antall sykedager i denne sesongen.", "exer_leave_sum" => $leave, "user_leave" => $user->user_leave));
exit(0);
?>