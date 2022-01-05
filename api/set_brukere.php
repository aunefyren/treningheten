<?php
// required headers
header("Access-Control-Allow-Origin: *");
header("Content-Type: application/json; charset=UTF-8");
header("Access-Control-Allow-Methods: POST");
header("Access-Control-Max-Age: 3600");
header("Access-Control-Allow-Headers: Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With");

// required to encode json web token
include_once 'config/core.php';
include_once 'libs/php-jwt-master/src/BeforeValidException.php';
include_once 'libs/php-jwt-master/src/ExpiredException.php';
include_once 'libs/php-jwt-master/src/SignatureInvalidException.php';
include_once 'libs/php-jwt-master/src/JWT.php';
use \Firebase\JWT\JWT;

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

$data->jwt = stripcslashes($data->jwt);
$jwt = isset($data->jwt) ? $data->jwt : "";

try {$decoded = JWT::decode($jwt, $key, array('HS256'));}
catch (Exception $e){
    http_response_code(401);
    echo json_encode(
        array(
            "message" => "Ugyldig innlogging.",
            "error" => true
        )
    );
    exit();
}

$bruker->b_id = stripcslashes($decoded->data->b_id);

if(!$bruker->get_admin()){
    echo json_encode(
        array(
            "message" => "Ikke admin.",
            "error" => true
        )
    );
    exit();
}

// set product property values
$bruker->b_epost = stripcslashes($data->b_epost);
$bruker->b_update = stripcslashes($data->b_update);
$bruker->b_active = stripcslashes($data->b_active);
$bruker->b_tittel = stripcslashes($data->b_tittel);
$bruker->b_id = stripcslashes($data->b_id);

if($data->b_hash) {
    if(
        !isset($bruker->b_id) ||
        !$bruker->ny_hash()
    ) {
        echo json_encode(array("message" => "Ny hash kunne ikke settes", "error" => true));
        exit();
    }
}

if(!empty($data->b_passord)) {
    $bruker->b_passord = stripcslashes($data->b_passord);
    if(
        !isset($bruker->b_passord) ||
        !$bruker->set_passord()
    ) {
        echo json_encode(array("message" => "Passord kunne ikke settes", "error" => true));
        exit();
    }
}

// create the user
if(
    isset($bruker->b_epost) &&
    isset($bruker->b_update) &&
    isset($bruker->b_active) &&
    isset($bruker->b_tittel) &&
    isset($bruker->b_id) &&
    $bruker->set_brukere()
){
    include_once 'objects/logg.php';
    $logg = new Logg($db);
    $logg->b_id = $decoded->data->b_id;
    $logg->lo_name = "set_brukere";
    $logg->insert();

    echo json_encode(array("message" => "Bruker ble endret!", "error" => false));
} else {

    // display message: unable to create user
    echo json_encode(array("message" => "Bruker ble ikke endret.", "error" => true));
}
?>
