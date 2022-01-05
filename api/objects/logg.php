<?php
class Logg{
    // database connection and table name
    private $conn;
    private $table_name = "logg";
    private $table_name2 = "brukere";

    public $lo_id;
    public $b_id;
    public $lo_time;
    public $lo_name;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }

    function get_logg(){

        // query to get films
        $query = 'SELECT `lo_id`, logg.`b_id`, `lo_time`, `lo_name`, brukere.`b_fornavn`, brukere.`b_etternavn` '
                  . 'FROM ' . $this->table_name . ', ' . $this->table_name2 .
                  ' WHERE brukere.`b_id` = logg.`b_id` ORDER BY lo_time DESC';

        //Prepare the query
        $stmt = $this->conn->prepare($query);

        //return $query;

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $bruker_id);
        $stmt->bindColumn(3, $time);
        $stmt->bindColumn(4, $name);
        $stmt->bindColumn(5, $fornavn);
        $stmt->bindColumn(6, $etternavn);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'lo_id' => $id,
                'b_id' => $bruker_id,
                'lo_time' => $time,
                'lo_name' => $name,
                'b_fornavn' => $fornavn,
                'b_etternavn' => $etternavn,
                );
            }

            $json = json_encode(array("error" => "false", "message" => "Logg ble levert", "logg" => $data));
            return $json;

        } else {

            $json = json_encode(array("message" => "Null resultater", "error" => "false"));
            return $json;
        }
    }

    function insert(){

        $query = "INSERT INTO " . $this->table_name
                                . " (`b_id`, `lo_name`) "
                                . "VALUES "
                                . "('"
                                . $this->b_id . "','"
                                . $this->lo_name . ""
                                . "')";

        $stmt = $this->conn->prepare($query);

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }

    }

}
?>
