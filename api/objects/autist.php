<?php
class Autist{
    // database connection and table name
    private $conn;
    private $table_name = "autister";
    private $table_name2 = "postnr";

    public $a_id = "`a_id` IS NOT NULL";
    public $a_fornavn = "`a_fornavn` IS NOT NULL";
    public $a_etternavn = "`a_etternavn` IS NOT NULL";
    public $a_bio = "`a_bio` IS NOT NULL";
    public $postnr = "postnr.`postnr` IS NOT NULL";
    public $a_grad = "`a_grad` IS NOT NULL";
    public $a_risiko = "`a_risiko` IS NOT NULL";
    public $poststed = "postnr.`poststed` IS NOT NULL";

    public $b_id;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }


    // get database connection
    function search(){

        // query to get films
        $query = 'SELECT `a_id`, `a_fornavn`, `a_etternavn`, `a_bio`, postnr.`postnr`, `a_grad`, `a_risiko`, postnr.`poststed` '
                  . 'FROM ' . $this->table_name . ', ' . $this->table_name2 .
                  ' WHERE (autister.postnr = postnr.postnr) AND
                  (' . $this->a_id . ') AND
                  (' . $this->a_fornavn . ') AND
                  (' . $this->a_etternavn . ') AND
                  (' . $this->a_bio . ') AND
                  (' . $this->postnr . ') AND
                  (' . $this->a_grad . ') AND
                  (' . $this->a_risiko . ') AND
                  (' . $this->poststed . ') AND a_vises = 1 ORDER BY a_etternavn ASC';

        //Prepare the query
        $stmt = $this->conn->prepare($query);

        //return $query;

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $fornavn);
        $stmt->bindColumn(3, $etternavn);
        $stmt->bindColumn(4, $bio);
        $stmt->bindColumn(5, $temp_nr);
        $stmt->bindColumn(6, $grad);
        $stmt->bindColumn(7, $risiko);
        $stmt->bindColumn(8, $temp_sted);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'a_id' => $id,
                'a_fornavn' => $fornavn,
                'a_etternavn' => $etternavn,
                'a_bio' => $bio,
                'postnr' => $temp_nr,
                'a_grad' => $grad,
                'a_risiko' => $risiko,
                'poststed' => $temp_sted,
                );
            }

            $json = json_encode($data);
            $this->set_logg($num, $json);
            return $json;

        } else {

            $json = json_encode(array("message" => "Null resultater", "error" => "false"));
            $this->set_logg($num, $json);
            return $json;
        }
    }

    function set_logg($num, $json){

        $s_results = $json;

        $s_results_num = $num;

        // query to check if email exists
        $query = "INSERT INTO search_logg SET b_id = '" . $this->b_id . "', s_results = '" . $s_results . "', s_results_num = '" . $s_results_num . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        return true;

    }

    function get_newest(){
        // query to check if email exists
        $query = "SELECT MAX(a_id) AS a_id FROM " . $this->table_name;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["a_id"];

    }

    function insert(){

        $query = "INSERT INTO " . $this->table_name
                                . " (`a_fornavn`, `a_etternavn`, `a_bio`, `postnr`, `a_grad`, `a_risiko`, `b_id`) "
                 . "VALUES ('" . $this->a_fornavn   . "', '"
                               . $this->a_etternavn . "', '"
                               . $this->a_bio       . "', '"
                               . $this->postnr      . "', '"
                               . $this->a_grad      . "', '"
                               . $this->a_risiko    . "', '"
                               . $this->b_id        . "')";

        $stmt = $this->conn->prepare($query);

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }

    }

    function get_autist_score(){
        // query to check if email exists
        $query = "SELECT COUNT(a_id) AS score FROM " . $this->table_name . " WHERE b_id = " . $this->b_id . " AND a_vises = 1";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["score"];

    }

}
?>
