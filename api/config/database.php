<?php
// used to get mysql database connection
class Database{

    // specify your own database credentials
    private $host = "localhost";
    private $db_name = "treniufu_treningheten_db";
    private $username = "treniufu_master";
    private $password = "#tyWgx62cQ#*oF";
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
