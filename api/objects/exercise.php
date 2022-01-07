<?php
// 'exercise' object
class Exercise{

    // database connection and table name
    private $conn;
    private $table_name = "exercises";

    // object properties
    public $exer_id;
    public $exer_date;
    public $exer_note;
    public $exer_enabled;
    public $goal_id;
    public $exer_leave;

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
                );
            }

            $json = json_encode($data);
            return $json;

        } else {

            return false;
        }
    }

    function create_goal(){

        // insert query
        $query = "INSERT INTO " . $this->table_name .
                 " SET
                    goal_exer_week = '" . $this->goal_exer_week . "',
                    goal_end = '" . $this->goal_end->format('Y-m-d H:i:s') . "',
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
