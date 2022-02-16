<?php
// used to get mysql database connection
class Database{

    // specify your own database credentials
    private $host = "trening_db";
    private $db_name = "trening_db";
    private $username = "root";
    private $password = "ocs#bKX2r#qBM^&q74d!PU";
    public $conn;
    private $utf = "utf8mb4";

    // get the database connection
    public function getConnection(){

        $this->conn = null;

        try{
            $this->conn = new PDO("mysql:host=" . $this->host . ";dbname=" . $this->db_name . ";charset=". $this->utf, $this->username, $this->password);
        }catch(PDOException $exception){
            echo "Connection error: " . $exception->getMessage();
        }

        return $this->conn;
    }
}
?>
