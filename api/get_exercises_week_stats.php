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

for($i = 0; $i < count($all_goals); $i++) {
    $goal_start = date_create_from_format('Y-m-d H:i:s', $all_goals[$i]['goal_start']);
    $goal_end = date_create_from_format('Y-m-d H:i:s', $all_goals[$i]['goal_end']);
    if($goal_start >= $season_start && $goal_end <= $season_end) {
        array_push($goal_id_list, array('goal_id' => $all_goals[$i]['goal_id'], 'user_id' => $all_goals[$i]['user_id'], 'exercise_found' => false));
    }
}

for($i = 0; $i < count($goal_id_list); $i++) {
    $exercise->goal_id = $goal_id_list[$i]['goal_id'];
    $data = $exercise->get_exercises_stats();

    if(!$data) {

        $goal_id_list[$i]['exercise_found'] = false;
        $goal_id_list[$i]['user_firstname'] = false;
        $goal_id_list[$i]['user_lastname'] = false;
        $goal_id_list[$i]['goal_start'] = false;
        $goal_id_list[$i]['goal_end'] = false;
        $goal_id_list[$i]['goal_exer_week'] = false;
        $goal_id_list[$i]['week_complete'] = false;
        $goal_id_list[$i]['week_percent'] = 0;
        $goal_id_list[$i]['goal_compete'] = false;
        $goal_id_list[$i]['streak'] = 0;

        $goal->goal_id = $goal_id_list[$i]['goal_id'];
        $goal_data = $goal->get_goal_user();

        if(!$goal_data) {

            echo json_encode(array("error" => true, "message" => "Feilet i å finne brukerdata for mål."));
            exit(0);

        }

        $goal_data = json_decode($goal_data, true);

        $goal_id_list[$i]['user_firstname'] = $goal_data[0]['user_firstname'];
        $goal_id_list[$i]['user_lastname'] = $goal_data[0]['user_lastname'];
        $goal_id_list[$i]['goal_start'] = $goal_data[0]['goal_start'];
        $goal_id_list[$i]['goal_end'] = $goal_data[0]['goal_end'];
        $goal_id_list[$i]['goal_exer_week'] = $goal_data[0]['goal_exer_week'];
        $goal_id_list[$i]['goal_compete'] = $goal_data[0]['goal_compete'];

    } else {
        $data = json_decode($data, true);

        $goal_id_list[$i]['exercise_found'] = true;
        $goal_id_list[$i]['user_firstname'] = $data[0]['user_firstname'];
        $goal_id_list[$i]['user_lastname'] = $data[0]['user_lastname'];
        $goal_id_list[$i]['goal_start'] = $data[0]['goal_start'];
        $goal_id_list[$i]['goal_end'] = $data[0]['goal_end'];
        $goal_id_list[$i]['goal_exer_week'] = $data[0]['goal_exer_week'];
        $goal_id_list[$i]['goal_compete'] = $data[0]['goal_compete'];
        $goal_id_list[$i]['week_complete'] = false;
        $goal_id_list[$i]['week_percent'] = 0;
        $goal_id_list[$i]['streak'] = 0;

        $weeks = array_fill(0, 52, 0);

        for($j = 0; $j < count($data); $j++) {
            $exer_date = new DateTime($data[$j]['exer_date']);
            $week = intval($exer_date->format('W'));
            $weeks[$week] = intval($weeks[$week]) + 1;

        }

        $streak = 0;
        
        $goal_start = date_create_from_format('Y-m-d H:i:s', $goal_id_list[$i]['goal_start']);
        $start_week = intval($goal_start->format('W'));

        $goal_end = date_create_from_format('Y-m-d H:i:s', $goal_id_list[$i]['goal_end']);
        $end_week = intval($goal_end->format('W'));

        $now = new DateTime('NOW');
        $current_week = intval($now->format('W'));

        for($j = $start_week; $j <= $current_week-1; $j++) {
            if($weeks[$j] >= $goal_id_list[$i]['goal_exer_week']) {
                $streak++;
            } else {
                $streak = 0;
            }

        }

        $goal_id_list[$i]['streak'] = $streak;

        if($weeks[$current_week] >= $goal_id_list[$i]['goal_exer_week']) {
            $goal_id_list[$i]['week_complete'] = true;
        } else {
            $goal_id_list[$i]['week_complete'] = false;
        }

        $goal_id_list[$i]['week_percent'] = $weeks[$current_week] / $goal_id_list[$i]['goal_exer_week'] * 100;
    }
}

echo json_encode(array("error" => false, "message" => "Fant trening for uken.", "users" => $goal_id_list));
exit(0);
?>