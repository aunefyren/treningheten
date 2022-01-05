<?php
class Bilde{
    // database connection and table name
    private $conn;
    private $table_name = "bilder";
    private $table_name2 = "ar_bilder";

    public $bi_id;
    public $bi_vises;
    public $bi_extension;

    public $ar_id;

    public $b_id;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }


    // get database connection
    function get_bilder(){

        // query to get films
        $query = 'SELECT DISTINCT `bi_id`, `bi_upload`, `b_id`, `bi_vises`, `bi_extension` '
                  . 'FROM ' . $this->table_name .
                  ' WHERE bi_vises = 1 ORDER BY bi_upload DESC';

        //Prepare the query
        $stmt = $this->conn->prepare($query);

        //return $query;

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $upload);
        $stmt->bindColumn(3, $bruker);
        $stmt->bindColumn(4, $vises);
        $stmt->bindColumn(5, $extension);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'bi_id' => $id,
                'bi_upload' => $upload,
                'b_id' => $bruker,
                'bi_vises' => $vises,
                'bi_extension' => $extension,
                );
            }

            $json = json_encode($data);
            return $json;

        } else {

            $json = json_encode(array("message" => "Null resultater", "error" => "false"));
            return $json;
        }
    }

    function get_ar_bilder(){

        // query to get films
        $query = 'SELECT DISTINCT `ar_bil_id`, `ar_bil_extension` '
                  . 'FROM ' . $this->table_name2 .
                  ' WHERE ar_id = ' . $this->ar_id . ' AND ar_bil_vises = 1 ORDER BY ar_bil_id ASC';

        //Prepare the query
        $stmt = $this->conn->prepare($query);

        //return $query;

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $extension);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'ar_bil_id' => $id,
                'ar_bil_extension' => $extension,
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

    function get_highest(){
        // query to check if email exists
        $query = "SELECT MAX(bi_id) AS bi_id FROM " . $this->table_name;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["bi_id"];

    }

    function insert(){

        $query = "INSERT INTO " . $this->table_name
                                . " (`b_id`, `bi_extension`) "
                 . "VALUES ('" . $this->b_id . "', '" . $this->bi_extension . "')";

        $stmt = $this->conn->prepare($query);

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }

    }

    function get_ar_score(){
        // query to check if email exists
        $query = "SELECT COUNT(ar_bil_id) AS score FROM " . $this->table_name2 . " WHERE b_id = " . $this->b_id . " AND ar_bil_vises = 1";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["score"];

    }

    function get_bil_score(){
        // query to check if email exists
        $query = "SELECT COUNT(bi_id) AS score FROM " . $this->table_name . " WHERE b_id = " . $this->b_id . " AND bi_vises = 1";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["score"];

    }

}
?>
