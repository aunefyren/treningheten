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
include_once 'objects/postnr.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$postnr = new Postnr($db);

// get posted data
$data = json_decode(file_get_contents("php://input"));

$jwt = isset($data->jwt) ? $data->jwt : "";

try {$decoded = JWT::decode($jwt, $key, array('HS256'));}
catch (Exception $e){
    http_response_code(401);
    echo json_encode(
        array(
            "message" => "Ugyldig innlogging.",
            "error" => "true"
        )
    );
    exit();
}

$postnr->postnr = $data->postnr;

if($poststed = $postnr->get_poststed()) {
    echo json_encode(array("error" => "false", "poststed" => $poststed));
    exit;
} else {
    echo json_encode(array("error" => "true", "message" => "Ugyldig postnr."));
    exit;
}
?>
