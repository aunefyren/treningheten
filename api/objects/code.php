<?php
// 'user' object
class Code{

    // database connection and table name
    private $conn;
    private $table_name = "register_codes";

    // object properties
    public $code_id;
    public $code_hash;
    public $code_created;
    public $code_used;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }

    function get_code(){

        // query to check if email exists
        $query = "SELECT `code_id`, `code_used` FROM " . $this->table_name . " WHERE `code_hash` = '" . $this->code_hash . "'";

        $stmt = $this->conn->prepare($query);

        // execute the query
        $stmt->execute();

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num === 1){

            // get record details / values
            $row = $stmt->fetch(PDO::FETCH_ASSOC);

            // assign values to object properties
            $this->code_id = $row['code_id'];
            $this->code_used = $row['code_used'];

            $json = json_encode(array('code_id' => $this->code_id, 'code_used' => $this->code_used));
            return $json;

        } else {
            
            return false;
        }
    }

    function set_code_used(){
        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET code_used = '1' WHERE code_id = '" . $this->code_id . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }

    }
}
