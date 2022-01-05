<?php
class Kommentar{
    // database connection and table name
    private $conn;
    private $table_name = "kommentarer";
    private $table_name_2 = "brukere";

    public $k_id;
    public $k_tid;
    public $k_vises;
    public $k_tekst;

    public $b_id;
    public $p_id;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }


    // get database connection
    function get_kommentarer(){

        // query to get films
        $query = 'SELECT DISTINCT `k_id`, kommentarer.`b_id`, `k_tid`, `k_tekst`, brukere.`b_kallenavn`, brukere.`b_fornavn`, brukere.`b_etternavn`, kommentarer.`p_id` '
                  . 'FROM ' . $this->table_name . ', ' . $this->table_name_2 .
                  ' WHERE `k_vises` = 1 AND brukere.`b_id` = kommentarer.`b_id` AND kommentarer.`p_id` = ' . $this->p_id . ' ORDER BY k_tid ASC';

        //Prepare the query
        $stmt = $this->conn->prepare($query);
        //echo $query;

        //return $query;

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $bid);
        $stmt->bindColumn(3, $tid);
        $stmt->bindColumn(4, $tekst);
        $stmt->bindColumn(5, $kallenavn);
        $stmt->bindColumn(6, $fornavn);
        $stmt->bindColumn(7, $etternavn);
        $stmt->bindColumn(8, $pid);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'k_id' => $id,
                'b_id' => $bid,
                'k_tid' => $tid,
                'k_tekst' => $tekst,
                'b_kallenavn' => $kallenavn,
                'b_fornavn' => $fornavn,
                'b_etternavn' => $etternavn,
                'p_id' => $pid,
                );
            }

            $json = json_encode($data);
            return $json;

        } else {

            $json = json_encode(array("message" => "Null resultater", "error" => "false"));
            return $json;
        }
    }

    function get_bilder(){

        // query to get films
        $query = 'SELECT DISTINCT `ar_bil_id` '
                  . 'FROM ' . $this->table_name2 .
                  ' WHERE ar_id = ' . $this->ar_id . ' ORDER BY ar_bil_id ASC';

        //Prepare the query
        $stmt = $this->conn->prepare($query);

        //return $query;

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'ar_bil_id' => $id,
                );
            }

            $json = json_encode($data);
            return $json;

        } else {

            $json = json_encode(array("message" => "Null resultater", "error" => "false"));
            return $json;
        }
    }

    function get_liste(){

        // query to get films
        $query = 'SELECT DISTINCT `ar_id`, `ar_tittel`, `ar_tid_start` '
                  . 'FROM ' . $this->table_name .
                  ' ORDER BY ar_tid_start DESC';

        //Prepare the query
        $stmt = $this->conn->prepare($query);

        //return $query;

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $tittel);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'ar_id' => $id,
                'ar_tittel' => $tittel,
                );
            }

            $json = json_encode($data);
            return $json;

        } else {

            $json = json_encode(array("message" => "Null resultater", "error" => "false"));
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

    function delete(){

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET `k_vises` = '" . "0" . "' WHERE `k_id` = '" . $this->k_id . "'" ;

        // prepare the query
        $stmt = $this->conn->prepare( $query );
        //echo $query;

        // execute the query
        $stmt->execute();

        return true;

    }

    function get_highest(){
        // query to check if email exists
        $query = "SELECT MAX(ar_bil_id) AS ar_bil_id FROM " . $this->table_name2;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["ar_bil_id"];

    }

    function insert(){

        $query = "INSERT INTO " . $this->table_name
                                . " (`b_id`, `k_tekst`, `p_id`) "
                 . "VALUES ('" . $this->b_id . "', '" . $this->k_tekst . "', '" . $this->p_id . "')";

        $stmt = $this->conn->prepare($query);

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }

    }

    function get_kom_score(){
        // query to check if email exists
        $query = "SELECT COUNT(k_id) AS score FROM " . $this->table_name . " WHERE b_id = " . $this->b_id . " and k_vises = 1";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["score"];

    }

}
?>
