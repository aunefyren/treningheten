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
// get posted data
$data = json_decode(file_get_contents("php://input"));

$location = "../vault/config.json";
$config = json_decode(file_get_contents($location));

$config->minecraft_ip = $data->minecraft_ip;
$config->minecraft_port = $data->minecraft_port;

if(
    file_put_contents($location, json_encode($config))
) {
    $result = array("error" => false, "message" => "Minecraft IP ble endret.");
} else {
    $result = array("error" => true, "message" => "Minecraft IP ble ikke endret.");
}

include_once 'objects/logg.php';
$logg = new Logg($db);
$logg->b_id = $decoded->data->b_id;
$logg->lo_name = "set_minecraft";
$logg->insert();

echo json_encode($result);

?>
