<?php
// 'user' object
class Brukere{

    // database connection and table name
    private $conn;
    private $table_name = "brukere";
    private $table_name2 = "postnr";

    // object properties
    public $b_id;
    public $b_fornavn;
    public $b_etternavn;
    public $b_epost;
    public $b_passord;
    public $b_admin;
    public $b_bio;
    public $b_update;
    public $b_skapelse;
    public $b_active;
    public $b_tittel;
    public $b_kallenavn;
    public $b_hash;
    public $postnr;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }

    // create new user record
    function create(){

        $hash = md5( rand(0,1000) );

        // sanitize
        $this->b_fornavn=htmlspecialchars(strip_tags($this->b_fornavn));
        $this->b_etternavn=htmlspecialchars(strip_tags($this->b_etternavn));
        $this->b_epost=htmlspecialchars(strip_tags($this->b_epost));
        $this->postnr=htmlspecialchars(strip_tags($this->postnr));

        // hash the password before saving to database
        $password_hash = password_hash($this->b_passord, PASSWORD_BCRYPT);

        // insert query
        $query = "INSERT INTO " . $this->table_name .
                 " SET
                    b_fornavn = '" . $this->b_fornavn . "',
                    b_etternavn = '" . $this->b_etternavn . "',
                    b_epost = '" . $this->b_epost . "',
                    b_hash = '" . $hash . "',
                    postnr = '" . $this->postnr . "',
                    b_passord = '" . $password_hash . "'";

        // prepare the query
        $stmt = $this->conn->prepare($query);

        // execute the query, also check if query was successful
        if($stmt->execute()){
            //$this->ver_email();

            return true;
        }
        echo $query;
        return false;
    }

    // check if given email exist in the database
    function getUser(){

        // query to check if email exists
        $query = "SELECT b_id, b_fornavn, b_etternavn, b_passord, b_admin, b_tittel, b_kallenavn, b_update, b_bio, b_active, b_skapelse, postnr
                FROM " . $this->table_name . "
                WHERE b_epost = ?
                LIMIT 0,1";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // sanitize
        $this->b_epost=htmlspecialchars(strip_tags($this->b_epost));

        // bind given email value
        $stmt->bindParam(1, $this->b_epost);

        // execute the query
        $stmt->execute();

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $row = $stmt->fetch(PDO::FETCH_ASSOC);

            // assign values to object properties
            $this->b_id = $row['b_id'];
            $this->b_fornavn = $row['b_fornavn'];
            $this->b_etternavn = $row['b_etternavn'];
            $this->b_passord = $row['b_passord'];
            $this->b_kallenavn = $row['b_kallenavn'];
            $this->b_tittel = $row['b_tittel'];
            $this->b_admin = $row['b_admin'];
            $this->b_update = $row['b_update'];
            $this->b_bio = $row['b_bio'];
            $this->b_active = $row['b_active'];
            $this->b_skapelse = $row['b_skapelse'];
            $this->postnr = $row['postnr'];

            return true;
        }

        // return false if email does not exist in the database
        return false;
    }

    function sjekk_epost(){

        // query to check if email exists
        $query = "SELECT b_epost FROM " . $this->table_name . " WHERE b_epost = '" . $this->b_epost . "' LIMIT 0,1";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // sanitize
        $this->b_epost=htmlspecialchars(strip_tags($this->b_epost));

        // bind given email value
        $stmt->bindParam(1, $this->b_epost);

        // execute the query
        $stmt->execute();

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){
            return true;
        }

        // return false if email does not exist in the database
        return false;
    }

    function ver_email(){

        $hash    = $this->sjekk_hash();
        $to      = $this->b_epost; // Send email to our user
        $subject = 'Registrering | Validering'; // Give the email a subject
        $message = '

        Takk for registreringen!
        Brukeren din er skapt, men er ikke aktivert. Følg lenken under for å aktivere brukeren din.

        Lenke:
        http://www.localhost/planteverden/verify.html?email='.$this->b_epost.'&hash='.$hash.'

        '; // Our message above including the link

        $headers = 'From:oystein.sverre@gmail.com' . "\r\n"; // Set from headers
        if(mail($to, $subject, $message, $headers)) {
            return true;
        }

        return false;
    }

    function set_account(){
        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET b_active = '1' WHERE b_hash = '" . $this->b_hash . "' AND b_epost = '" . $this->b_epost . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }

    }

    function set_passord(){
            $passord = password_hash($this->b_passord, PASSWORD_BCRYPT);

            // query to check if email exists
            $query = "UPDATE " . $this->table_name . " SET b_passord = '" . $passord . "' WHERE b_id = '" . $this->b_id . "'";

            // prepare the query
            $stmt = $this->conn->prepare( $query );

            // execute the query
            if($stmt->execute()) {
                return true;
            } else {
                return false;
            }

        }

    function ny_hash(){
        $hash = md5( rand(0,1000) );

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET b_hash = '" . $hash . "' WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        if($stmt->execute()) {
            return true;
        }

        return false;

    }

    function sjekk_hash(){

        // query to check if email exists
        $query = "SELECT b_hash FROM " . $this->table_name . " WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        return $row["b_hash"];

    }

    function val_hash(){

        // query to check if email exists
        $query = "SELECT b_id FROM " . $this->table_name . " WHERE b_hash = '" . $this->b_hash . "'";

        $stmt = $this->conn->prepare( $query );

        // bind given email value
        $stmt->bindParam(1, $id);

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);
        $this->b_id = $row["b_id"];
        $num = $stmt->rowCount();

        if($num>0){

            if($this->ny_hash()) {
                return true;
            }

            return false;
        }

        return false;
    }

    function get_medlemmer(){

        // query to check if email exists
        $query = "SELECT `b_id`, `b_fornavn`, `b_etternavn`, `b_bio`, `b_tittel`, postnr.`postnr`, postnr.`poststed` FROM " . $this->table_name . ", " . $this->table_name2 . " WHERE brukere.`postnr` = postnr.`postnr` ORDER BY b_etternavn ASC";

        $stmt = $this->conn->prepare($query);

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $fornavn);
        $stmt->bindColumn(3, $etternavn);
        $stmt->bindColumn(4, $bio);
        $stmt->bindColumn(5, $tittel);
        $stmt->bindColumn(6, $temp_nr);
        $stmt->bindColumn(7, $temp_sted);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'b_id' => $id,
                'b_fornavn' => $fornavn,
                'b_etternavn' => $etternavn,
                'b_bio' => $bio,
                'b_tittel' => $tittel,
                'postnr' => $temp_nr,
                'poststed' => $temp_sted,
                );
            }

            $json = json_encode($data);
            return $json;

        } else {

            $json = json_encode(array("message" => "Null resultater", "error" => "false"));
            return $json;
        }

    }

    function get_brukere(){

        // query to check if email exists
        $query = "SELECT `b_id`, `b_fornavn`, `b_etternavn`, `b_epost`, `b_tittel`, `b_active`, `b_update`, `b_hash`, `b_last_login` FROM " . $this->table_name . ", " . $this->table_name2 . " WHERE brukere.`postnr` = postnr.`postnr` ORDER BY b_etternavn ASC";

        $stmt = $this->conn->prepare($query);

        // execute the query
        $stmt->execute();

        //Bind by column number
        $stmt->bindColumn(1, $id);
        $stmt->bindColumn(2, $fornavn);
        $stmt->bindColumn(3, $etternavn);
        $stmt->bindColumn(4, $epost);
        $stmt->bindColumn(5, $tittel);
        $stmt->bindColumn(6, $active);
        $stmt->bindColumn(7, $update);
        $stmt->bindColumn(8, $hash);
        $stmt->bindColumn(9, $login);

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $data = array();

            while($stmt->fetch()){
                $data[] = array(
                'b_id' => $id,
                'b_fornavn' => $fornavn,
                'b_etternavn' => $etternavn,
                'b_epost' => $epost,
                'b_tittel' => $tittel,
                'b_active' => $active,
                'b_update' => $update,
                'b_hash' => $hash,
                'b_last_login' => $login,
                );
            }

            $data = array("error" => false, "message" => "Alle brukere lastet inn", "brukere" => $data);
            $json = json_encode($data);
            return $json;

        } else {

            $json = json_encode(array("message" => "Null resultater", "error" => "false"));
            return $json;
        }

    }

    function set_brukere(){

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET" .
        " b_update = '" . $this->b_update .
        "', b_epost = '" . $this->b_epost .
        "', b_active = '" . $this->b_active .
        "', b_tittel = '" . $this->b_tittel .
        "' WHERE b_id = '" . $this->b_id . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        return true;
    }

    function sjekk_refresh(){

        // query to check if email exists
        $query = "SELECT b_update FROM " . $this->table_name . " WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        if($row["b_update"] == 1 ) {
            return true;
        } else {
            return false;
        }
    }

    function sjekk_active(){

        // query to check if email exists
        $query = "SELECT b_active FROM " . $this->table_name . " WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        if($row["b_active"] == 1 ) {
            return true;
        } else {
            return false;
        }
    }

    function get_update_state(){

        // query to check if email exists
        $query = "SELECT b_update FROM " . $this->table_name . " WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        if($row["b_update"] == 1 ) {
            return true;
        } else {
            return false;
        }
    }

    function set_update_state(){

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET b_update = 0 WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        return true;
    }

    function get_active_state(){

        // query to check if email exists
        $query = "SELECT b_active FROM " . $this->table_name . " WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        $row = $stmt->fetch(PDO::FETCH_ASSOC);

        if($row["b_active"] == 1 ) {
            return true;
        } else {
            return false;
        }
    }

    function get_admin(){

        // query to check if email exists
        $query = "SELECT b_admin FROM " . $this->table_name . " WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        // get number of rows
        $num = $stmt->rowCount();

        // if email exists, assign values to object properties for easy access and use for php sessions
        if($num>0){

            // get record details / values
            $row = $stmt->fetch(PDO::FETCH_ASSOC);

            if($row["b_admin"] == 1 ) {
                return true;
            }
        }
        return false;
    }

    function login_activity(){

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET b_last_login = NOW() WHERE b_id = " . $this->b_id;

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        $stmt->execute();

        return true;
    }


    // update a user record
    public function update(){

        // if password needs to be updated
        $password_set=!empty($this->b_passord) ? ", b_passord = :b_passord" : "";

        // if no posted password, do not update the password
        $query = "UPDATE " . $this->table_name . "
                SET
                    b_kallenavn = :b_kallenavn,
                    b_bio = :b_bio,
                    postnr = :postnr,
                    b_epost = :b_epost
                    {$password_set}
                WHERE b_id = :b_id";

        // prepare the query
        $stmt = $this->conn->prepare($query);

        // sanitize
        $this->b_kallenavn=htmlspecialchars(strip_tags($this->b_kallenavn));
        $this->b_bio=htmlspecialchars(strip_tags($this->b_bio));
        $this->b_epost=htmlspecialchars(strip_tags($this->b_epost));
        $this->postnr=htmlspecialchars(strip_tags($this->postnr));

        // bind the values from the form
        $stmt->bindParam(':b_kallenavn', $this->b_kallenavn);
        $stmt->bindParam(':b_bio', $this->b_bio);
        $stmt->bindParam(':postnr', $this->postnr);
        $stmt->bindParam(':b_epost', $this->b_epost);

        // hash the password before saving to database
        if(!empty($this->b_passord)){
            $this->b_passord=htmlspecialchars(strip_tags($this->b_passord));
            $password_hash = password_hash($this->b_passord, PASSWORD_BCRYPT);
            $stmt->bindParam(':b_passord', $password_hash);
        }

        // unique ID of record to be edited
        $stmt->bindParam(':b_id', $this->b_id);

        // execute the query
        if($stmt->execute()){

            return true;
        }

        return false;
    }
}
