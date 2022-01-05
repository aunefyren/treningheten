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
$bruker->b_id = $data->b_id;
$bruker->b_epost = $data->b_epost;

// create the user
if(
    !empty($bruker->b_id) &&
    !empty($bruker->b_epost) &&
    $bruker->ver_email()
){
    echo json_encode(array("message" => "Ny hash er satt", "error" => "false"));
}

// message if unable to create user
else{

    // set response code
    http_response_code(400);

    // display message: unable to create user
    echo json_encode(array("message" => "Nådde ikke databasen."));
}
?>
