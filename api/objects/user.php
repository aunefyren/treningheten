<?php
// 'user' object
class User{

    // database connection and table name
    private $conn;
    private $table_name = "users";
    const METHOD = 'aes-256-ctr';
    private $token_encrypter = 'this_token_encrypts_trening_420_69';

    // object properties
    public $user_id;
    public $user_email;
    public $user_password;
    public $user_firstname;
    public $user_lastname;
    public $user_leave;
    public $user_hash;
    public $user_active;
    public $user_disabled;
    public $user_admin;
    public $user_creation;
    public $user_lastactivity;
    public $code_id;

    // constructor
    public function __construct($db){
        $this->conn = $db;
    }

    // create new user record
    function create_user(){

        $this->user_hash = md5(rand(0,1000));

        // hash the password before saving to database
        $password_hash = password_hash($this->user_password, PASSWORD_BCRYPT);

        // insert query
        $query = "INSERT INTO " . $this->table_name .
                 " SET
                    user_firstname = '" . $this->user_firstname . "',
                    user_lastname = '" . $this->user_lastname . "',
                    user_email = '" . $this->user_email . "',
                    user_hash = '" . $this->user_hash . "',
                    code_id = '" . $this->code_id . "',
                    user_password = '" . $password_hash . "'";

        // prepare the query
        $stmt = $this->conn->prepare($query);

        // execute the query, also check if query was successful
        if($stmt->execute()){
            return true;
        }
        
        return false;
    }

    function check_email(){

        // query to check if email exists
        $query = "SELECT `user_email` FROM " . $this->table_name . " WHERE user_email = '" . $this->user_email . "'";
        
        // prepare the query
        $stmt = $this->conn->prepare( $query );

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

    function verification_email(){
        
        $this->get_user_data();

        $to      = $this->user_email; // Send email to our user
        $subject = 'Aktiver brukeren din!'; // Give the email a subject
        $message = '

        Takk for registreringen!
        Brukeren din er skapt, men er ikke aktivert. Følg lenken under for å aktivere brukeren din.

        Lenke:
        https://treningheten.no?activate_email=' . urlencode($this->user_email) . '&activate_hash=' . urlencode($this->user_hash) . '

        '; // Our message above including the link

        $headers = 'From:noreply@treningheten.no' . "\r\n" . "Content-Type: text/html; charset=UTF-8"; // Set from headers
        if(@mail($to, $subject, $message, $headers)) {
            return true;
        }

        return false;
    }

    function get_user_data(){

        // query to check if email exists
        $query = "SELECT * FROM " . $this->table_name . " WHERE `user_email` = '" . $this->user_email . "'";

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
            $this->user_id = $row['user_id'];
            $this->user_firstname = $row['user_firstname'];
            $this->user_lastname = $row['user_lastname'];
            $this->user_leave = $row['user_leave'];
            $this->user_hash = $row['user_hash'];
            $this->user_active = $row['user_active'];
            $this->user_disabled = $row['user_disabled'];
            $this->user_admin = $row['user_admin'];
            $this->user_creation = $row['user_creation'];
            $this->user_lastactivity = $row['user_lastactivity'];
            $this->code_id = $row['code_id'];

            return true;

        } else {
            
            return false;
        }
    }

    function get_user_cookie() {

        $this->get_user_data();

        // Get the current date
        $now = new DateTime('NOW');

        $token_data = array(
            "issued" => $now->format('Y-m-d H:i:s'),
            "data" => array(
               "user_id" => $this->user_id,
               "user_email" => $this->user_email,
               "user_firstname" => $this->user_firstname,
               "user_lastname" => $this->user_lastname,
               "user_leave" => $this->user_leave,
               "user_hash" => $this->user_hash,
               "user_active" => $this->user_active,
               "user_disabled" => $this->user_disabled,
               "user_admin" => $this->user_admin,
               "user_creation" => $this->user_creation,
               "user_lastactivity" => $this->user_lastactivity,
               "code_id" => $this->code_id
           )
        );
  
        $nonceSize = openssl_cipher_iv_length(self::METHOD);
        $nonce = openssl_random_pseudo_bytes($nonceSize);

        $token = openssl_encrypt(
            json_encode($token_data),
            self::METHOD,
            $this->token_encrypter,
            OPENSSL_RAW_DATA,
            $nonce
        );

        return base64_encode($nonce.$token);
        
    }

    function validate_user_cookie($cookie) {

        // Debase cookie
        $cookie_debased = base64_decode($cookie, true);
        if ($cookie_debased === false) {
            return false;
        }

        // Assign variables
        $nonceSize = openssl_cipher_iv_length(self::METHOD);
        $nonce = mb_substr($cookie_debased, 0, $nonceSize, '8bit');
        $ciphertext = mb_substr($cookie_debased, $nonceSize, null, '8bit');
        
        // Decrypt cookie
        $cookie_data = openssl_decrypt(
            $ciphertext,
            self::METHOD,
            $this->token_encrypter,
            OPENSSL_RAW_DATA,
            $nonce
        );

        // Decode the JSON response
        $cookie_result = json_decode($cookie_data, true);

        if(!isset($cookie_result['issued'])) {
            return false;
        }

        $now = new DateTime('NOW');
        $then = date_create_from_format('Y-m-d H:i:s', $cookie_result['issued']);
        $diff = (array) date_diff($now, $then);

        if($diff['days'] > 7) {
            return false;
        }
        
        // Return Plex token
        return $cookie_data;
        
    }

    function refresh_hash(){
        $hash = md5( rand(0,1000) );

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET user_hash = '" . $hash . "' WHERE user_email = '" . $this->user_email . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        if($stmt->execute()) {
            return true;
        }

        return false;

    }

    function set_user_active(){
        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET user_active = '1' WHERE user_email = '" . $this->user_email . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }

    }

    function check_password(){

        // query to check if email exists
        $query = "SELECT `user_password` FROM " . $this->table_name . " WHERE `user_email` = '" . $this->user_email . "'";
        
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
            $password_hash = $row['user_password'];
            
            if(password_verify($this->user_password, $password_hash)) {
                return true;
            }

            return false;

        } else {
            
            return false;
        }
    }

    function set_user_email(){

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET" .
        " user_email = '" . $this->user_email .
        "' WHERE user_id = '" . $this->user_id . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }
    }

    function set_user_password(){

        // hash the password before saving to database
        $password_hash = password_hash($this->user_password, PASSWORD_BCRYPT);

        // query to check if email exists
        $query = "UPDATE " . $this->table_name . " SET" .
        " user_password = '" . $password_hash .
        "' WHERE user_id = '" . $this->user_id . "'";

        // prepare the query
        $stmt = $this->conn->prepare( $query );

        // execute the query
        if($stmt->execute()) {
            return true;
        } else {
            return false;
        }
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
