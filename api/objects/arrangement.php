<?php
class Arrangement{
    // database connection and table name
    private $conn;
    private $table_name = "arrangementer";
    private $table_name2 = "ar_bilder";

    public $ar_id;
    public $ar_tittel;
    public $ar_tid_start;
    public $ar_tid_slutt;
    public $ar_sted;
    public $ar_bio;

    public $ar_bil_id;
    public $ar_bil_upload;
    public $ar_bil_extension;
    public $b_id;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }


    // get database connection
    function get_arrangementer(){

        // query to get films
        $query = 'SELECT DISTINCT `ar_id`, `ar_tittel`, `ar_tid_start`, `ar_tid_slutt`, `ar_sted`, `ar_bio` '
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
        $stmt->bindColumn(3, $tid_start);
        $stmt->bindColumn(4, $tid_slutt);
        $stmt->bindColumn(5, $sted);
        $stmt->bindColumn(6, $bio);

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
                'ar_tid_start' => $tid_start,
                'ar_tid_slutt' => $tid_slutt,
                'ar_sted' => $sted,
                'ar_bio' => $bio,
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
        $query = "SELECT MAX(ar_bil_id) AS ar_bil_id FROM " . $this->table_name2;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["ar_bil_id"];

    }

    function insert(){

        $query = "INSERT INTO " . $this->table_name2
                                . " (`ar_id`, `b_id`, `ar_bil_extension`) "
                 . "VALUES ('" . $this->ar_id . "', '" . $this->b_id . "', '" . $this->ar_bil_extension . "')";

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
