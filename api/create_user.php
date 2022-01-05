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
$brukere = new Brukere($db);

// get posted data
$data = json_decode(file_get_contents("php://input"));

$brukere->b_hash = $data->b_hash;
if(!$brukere->val_hash()) {
    echo json_encode(
        array(
            "message" => "Ugydlig registeringslenke.",
            "error" => "true"
        )
    );
    exit;
}

// set product property values
$brukere->b_fornavn = $data->b_fornavn;
$brukere->b_etternavn = $data->b_etternavn;
$brukere->b_epost = $data->b_epost;
$brukere->b_passord = $data->b_passord;
$brukere->postnr = $data->postnr;

// create the user
if($brukere->getUser()) {

    // display message: unable to create user
    echo json_encode(array("message" => "Epost er i bruk.", "error" => "true"));
    exit;
}

if(
    !empty($brukere->b_fornavn) &&
    !empty($brukere->b_epost) &&
    !empty($brukere->b_passord) &&
    !empty($brukere->postnr) &&
    $brukere->create()
){

    // display message: user was created
    echo json_encode(array("message" => "Bruker ble skapt.", "error" => "false"));
}

// message if unable to create user
else{

    // display message: unable to create user
    echo json_encode(array("message" => "Bruker ble ikke skapt.", "error" => "true"));
}
?>
