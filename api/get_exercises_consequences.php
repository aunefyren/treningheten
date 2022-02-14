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
require(dirname(__FILE__) . '/objects/spin.php');

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

$earliest_date = new DateTime('NOW');
$now = new DateTime('NOW'); 

for($i = 0; $i < count($all_goals); $i++) {
    $goal_start = date_create_from_format('Y-m-d H:i:s', $all_goals[$i]['goal_start']);
    if($goal_start <= $earliest_date) {
        $earliest_date = date_create_from_format('Y-m-d H:i:s', $all_goals[$i]['goal_start']);
    }
}

$weeks = array();

while($earliest_date->format('W') < $now->format('W') || $earliest_date->format('Y') != $now->format('Y')) {

    $week_contestants = array();

    for($i = 0; $i < count($all_goals); $i++) {
        $goal_start = date_create_from_format('Y-m-d H:i:s', $all_goals[$i]['goal_start']);
        $goal_end = date_create_from_format('Y-m-d H:i:s', $all_goals[$i]['goal_end']);
        $goal_compete = $all_goals[$i]['goal_compete'];

        if(intval($goal_start->format('W')) <= intval($earliest_date->format('W')) && intval($earliest_date->format('W')) <= intval($goal_end->format('W')) && intval($earliest_date->format('Y')) == intval($goal_start->format('Y')) && $goal_compete) {
            array_push($week_contestants, array('goal_id' => $all_goals[$i]['goal_id'], 'user_id' => $all_goals[$i]['user_id'], 'goal_exer_week' => $all_goals[$i]['goal_exer_week'], 'workouts' => 0));
        }
    }

    if(count($week_contestants) > 0) {
        array_push($weeks, array('date' => $earliest_date->format('Y-m-d'), 'week' => $earliest_date->format('W'), 'year' => $earliest_date->format('Y'), 'contestants' => $week_contestants));
    }

    $earliest_date->modify('+1 week');

}

$all_exercises = $exercise->get_exercises_all();

if(!$all_goals) {

    echo json_encode(array("error" => false, "message" => "Uke stats funnet.", "weeks" => $weeks));
    exit(0);

}

$all_exercises = json_decode($all_exercises, true);

for($j = 0; $j < count($weeks); $j++) {

    $date = new DateTime($weeks[$j]['date']);
    
    for($i = 0; $i < count($weeks[$j]['contestants']); $i++) {

        $goal_id = $weeks[$j]['contestants'][$i]['goal_id'];
        
        for($l = 0; $l < count($all_exercises); $l++) {

            $workout_date = new DateTime($all_exercises[$l]['exer_date']);
            $workout_goal_id = $all_exercises[$l]['goal_id'];

            if($date->format('W') == $workout_date->format('W') && $date->format('Y') == $workout_date->format('Y') && $goal_id == $workout_goal_id) {
                $weeks[$j]['contestants'][$i]['workouts']++;
            }
        
        }
    
    }

}

$date = array_column($weeks, 'date');
array_multisort($date, SORT_DESC, $weeks);

// Set week to fail/success
for($j = 0; $j < count($weeks); $j++) {

    for($i = 0; $i < count($weeks[$j]['contestants']); $i++) {

        if($weeks[$j]['contestants'][$i]['goal_exer_week'] <= $weeks[$j]['contestants'][$i]['workouts']) {
            
            $weeks[$j]['contestants'][$i]['week_contest'] = true;
            
        } else {

            $weeks[$j]['contestants'][$i]['week_contest'] = false;

        }

    }

}

$prices_to_register = array();

// Detirmine if someone won the week
for($j = 0; $j < count($weeks); $j++) {

    $winners = array();
    $losers = array();

    for($i = 0; $i < count($weeks[$j]['contestants']); $i++) {

        if($weeks[$j]['contestants'][$i]['week_contest']) {
            
            array_push($winners, $weeks[$j]['contestants'][$i]);
            
        } else {

            array_push($losers, $weeks[$j]['contestants'][$i]);

        }

    }

    if(count($losers) > 0 && count($winners) > 0) {

        $weeks[$j]['winners'] = $winners;
        $weeks[$j]['losers'] = $losers;

        array_push($prices_to_register, $weeks[$j]);

    }

}

$spin = new Spin($db);

// Check or register in DB
for($j = 0; $j < count($prices_to_register); $j++) {

    for($i = 0; $i < count($prices_to_register[$j]['losers']); $i++) {

        $spin->goal_id = $prices_to_register[$j]['losers'][$i]['goal_id'];
        $spin->spin_week = $prices_to_register[$j]['week'];
        $spin->spin_year = $prices_to_register[$j]['year'];

        if(!$spin->get_spin_date()) {
            
            $winner_str = '';

            for($b = 0; $b < count($prices_to_register[$j]['winners']); $b++) {
                if($b !== 0) {
                    $winner_str .= ',';
                }
                $winner_str .= $prices_to_register[$j]['winners'][$b]['goal_id'];
            }

            $spin->create_spin($winner_str);
            
        }

    }

}

echo json_encode(array("error" => false, "message" => "Everything is up to date."));
exit(0);
?>