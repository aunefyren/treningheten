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

    function get_exercises(){

        // query to check if email exists
        $query = "SELECT * FROM " . $this->table_name . " WHERE `goal_id` = '" . $this->goal_id . "' AND `exer_enabled` = '1'";

        $stmt = $this->conn->prepare($query);

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $date);
        $stmt->bindColumn(3, $note);
        $stmt->bindColumn(4, $enabled);
        $stmt->bindColumn(5, $goal);
        $stmt->bindColumn(6, $leave);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'exer_id' => $id,
                'exer_date' => $date,
                'exer_note' => $note,
                'exer_enabled' => $enabled,
                'goal_id' => $goal,
                'exer_leave' => $leave,
                );
            }

            $json = json_encode($data);
            return $json;

        } else {

            return false;
        }
    }

    function create_exercise(){

        // insert query
        $query = "INSERT INTO " . $this->table_name .
                 " SET
                    exer_date = '" . $this->exer_date->format('Y-m-d') . "',
                    goal_id = '" . $this->goal_id . "'";

        // prepare the query
        $stmt = $this->conn->prepare($query);

        // execute the query, also check if query was successful
        if($stmt->execute()){
            return true;
        }
        
        return false;
    }

    function set_exercise($value){

        if($value) {
            $this->exer_enabled = '1';
        } else {
            $this->exer_enabled = '0';
        }

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET" .
        " exer_enabled = '" . $this->exer_enabled .
        "' WHERE exer_date = '" . $this->exer_date->format('Y-m-d') . "' AND `goal_id` = '" . $this->goal_id . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num === 1){

            return true;

        } else {

            return false;
        }
    }

    function get_exercise_date(){

        // query to check if email exists
        $query = "SELECT * FROM " . $this->table_name . " WHERE `exer_date` = '" . $this->exer_date->format('Y-m-d') . "' AND `goal_id` = '" . $this->goal_id . "'";

        $stmt = $this->conn->prepare($query);

        // execute the query
        $stmt->execute();

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num === 1){

            return true;

        } else {

            return false;
        }
    }
}