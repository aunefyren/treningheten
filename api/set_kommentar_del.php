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
include_once 'objects/kommentar.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$kommentar = new Kommentar($db);

$jwt = $_POST['jwt'];

$jwt = isset($jwt) ? $jwt : "";

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

$kommentar->k_id = $_POST['k_id'];

if(
    !empty($kommentar->k_id) &&
    $kommentar->delete()
){
    include_once 'objects/logg.php';
    $logg = new Logg($db);
    $logg->b_id = $decoded->data->b_id;
    $logg->lo_name = "set_kommentar_del";
    $logg->insert();

    header('Location: ../innlegg.html?del=true&type=kom');
    exit;
}

// message if unable to create user
else{

    // display message: unable to create user
    header('Location: ../innlegg.html?del=false&type=kom');
    exit;
}

?>
