<?php
// required headers
header("Access-Control-Allow-Origin: *");
header("Content-Type: application/json; charset=UTF-8");
header("Access-Control-Allow-Methods: POST");
header("Access-Control-Max-Age: 3600");
header("Access-Control-Allow-Headers: Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With");

// files needed to connect to database
include_once 'config/database.php';
include_once 'objects/user.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$bruker = new Brukere($db);

// get posted data
$data = json_decode(file_get_contents("php://input"));

// set product property values
$bruker->b_epost = $data->b_epost;
$bruker->b_hash = $data->b_hash;

// create the user
if(
    !empty($bruker->b_epost) &&
    !empty($bruker->b_hash) &&
    $bruker->set_account()
){
    echo json_encode(array("message" => "Bruker ble aktivert!", "error" => "false"));
} else {

    // display message: unable to create user
    echo json_encode(array("message" => "Bruker ble ikke aktivert.", "error" => "true"));
}
?>
