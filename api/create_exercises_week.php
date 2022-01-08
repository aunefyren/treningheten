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
require(dirname(__FILE__) . '/objects/goal.php');

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$user = new User($db);
$exercise = new Exercise($db);
$goal = new Goal($db);
$data = json_decode(file_get_contents("php://input"));

// If POST data is empty or wrong
if(empty($data) || !isset($data->cookie) || !isset($data->exercises)) {
    
    echo json_encode(array("error" => true, "message" => "Fikk ikke kjeks eller trening fra etterspørsel."));
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

$goals = $goal->get_goal();

if($goals === false) {

    echo json_encode(array( "error" => false, 
                            "message" => "Fant ikke aktivt mål."
                            ));
    exit(0);

} else {

    $now = new DateTime('NOW');

    $goals = json_decode($goals, true);
    $goal_id = $goals['goal']['goal_id'];

    $days = $data->exercises->days;

    $exercise->goal_id = $goal_id;
    $exercises = $exercise->get_exercises();
    $week_configured = array(
                                false, false,false,false,false,false,false
        
                            );
    if($exercises !== false) {
        $exercises = json_decode($exercises, true);
        for($i = 0; $i < count($exercises); $i++) {
            $exer = new DateTime($exercises[$i]['exer_date']);
            if($exer->format('W') === $now->format('W') && $exer->format('Y') === $now->format('Y')) {
                $week_configured[$exer->format('N')-1] = true;
            }
        }
    }

    for($i = 0; $i < count($days); $i++) {

        //echo $now->format('N');

        if($days[$i] !== $week_configured[$i] && ($i + 1) <= $now->format('N')) {
            
            $exer_date = new DateTime('NOW');
            if($exer_date->format('N') !== 1) {
                for($j = 0; $j < 10; $j++) {
                    if($exer_date->format('N') != 1) {
                        $exer_date->modify('-1 days');
                    } else {
                        break;
                    }
                }
            }
            
            for($j = 0; $j < 10; $j++) {
                if($exer_date->format('N') != ($i + 1)) {
                    $exer_date->modify('+1 days');
                } else {
                    break;
                }
            }

            $exercise->exer_date = $exer_date;
            if($exercise->get_exercise_date()) {
                
                if(!$exercise->set_exercise($days[$i])) {

                    echo json_encode(array("error" => true, "message" => "Trening ble ikke oppdatert."));
                    exit(0);

                }
                
            } else {
                
                if(!$exercise->create_exercise()) {

                    echo json_encode(array("error" => true, "message" => "Trening ble ikke skapt."));
                    exit(0);

                }
                
            }

        }
    }

    echo json_encode(array("error" => false, "message" => "Trening ble oppdatert."));
    exit(0);

}
?>