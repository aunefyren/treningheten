<?php
class Postnr{
    // database connection and table name
    private $conn;
    private $table_name = "postnr";

    public $postnr;
    public $poststed;
    public $kommunenr;
    public $kommunenavn;
    public $kategori;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }

    function insert(){

        $query = "INSERT INTO " . $this->table_name
                                . " (`postnr`, `poststed`, `kommunenr`, `kommunenavn`, `kategori`) "
                 . "VALUES ('" . $this->postnr . "', '" . $this->poststed . "', '" . $this->kommunenr . "', '" . $this->kommunenavn . "', '"
                                . $this->kategori . "')";
        //echo "<br>" . $query . "<br>";
        $stmt = $this->conn->prepare($query);

        // execute the query
        if($stmt->execute()) {
            return json_encode(array("message" => "Ble lagt inn", "error" => "false"));;
        } else {
            return json_encode(array("message" => "Ble ikke lagt inn", "error" => "true"));
        }

    }

    function get_poststed(){

        // query to check if email exists
        $query = "SELECT poststed FROM " . $this->table_name . " WHERE postnr = " . $this->postnr;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return$row["poststed"];
    }

}
