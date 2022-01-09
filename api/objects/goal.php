<?php
// 'goal' object
class Goal{

    // database connection and table name
    private $conn;
    private $table_name = "goals";

    // object properties
    public $goal_id;
    public $goal_exer_week;
    public $goal_start;
    public $goal_end;
    public $user_id;
    public $goal_enabled;
    public $goal_compete;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }

    function get_goals(){

        // query to check if email exists
        $query = "SELECT * FROM " . $this->table_name . " WHERE `user_id` = '" . $this->user_id . "' AND `goal_enabled` = '1'";

        $stmt = $this->conn->prepare($query);

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $exer_week);
        $stmt->bindColumn(3, $start);
        $stmt->bindColumn(4, $end);
        $stmt->bindColumn(5, $user);
        $stmt->bindColumn(6, $enabled);
        $stmt->bindColumn(7, $compete);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'goal_id' => $id,
                'goal_exer_week' => $exer_week,
                'goal_start' => $start,
                'goal_end' => $end,
                'user_id' => $user,
                'goal_enabled' => $enabled,
                'goal_compete' => $compete,
                );
            }

            $json = json_encode($data);
            return $json;

        } else {

            return false;
        }
    }

    function get_goal() {
        $goals = $this->get_goals();

        $now = new DateTime('NOW');

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
            return false;
        }

        if(!$goals) {

            return false;

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

            return false;

        } else {

            $goal_started = false;

            $goal_start_chosen = new DateTime($goals[$goal_index]['goal_start']);
            if($now > $goal_start_chosen) {
                $goal_started = true;
            }

            $goal_start = new DateTime($goals[$goal_index]['goal_start']);
            $goal_end = new DateTime($goals[$goal_index]['goal_end']);

            return json_encode(array(  "season_start" => $chosen_season_start->format('d.n.Y'), 
                                "season_end" => $chosen_season_end->format('d.n.Y'), 
                                "goal" => array(
                                    "goal_id" => $goals[$goal_index]['goal_id'],
                                    "goal_exer_week" => $goals[$goal_index]['goal_exer_week'],
                                    "goal_start" => $goal_start->format('d.n.Y'),
                                    "goal_end" => $goal_end->format('d.n.Y'),
                                    "goal_compete" => $goals[$goal_index]['goal_compete'],
                                    "goal_started" => $goal_started
                                    )
                                ));
        }
    }

    function create_goal(){

        // insert query
        $query = "INSERT INTO " . $this->table_name .
                 " SET
                    goal_exer_week = '" . $this->goal_exer_week . "',
                    goal_start = '" . $this->goal_start->format('Y-m-d H:i:s') . "',
                    goal_end = '" . $this->goal_end->format('Y-m-d H:i:s') . "',
                    goal_compete = '" . $this->goal_compete . "',
                    user_id = '" . $this->user_id . "'";

        // prepare the query
        $stmt = $this->conn->prepare($query);

        // execute the query, also check if query was successful
        if($stmt->execute()){
            return true;
        }
        
        return false;
    }
}
