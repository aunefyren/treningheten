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

$data = json_decode(file_get_contents("php://input"));

$jwt = $_POST['jwt'];
$kommentar->p_id = $_POST['p_id'];
$kommentar->k_tekst = $_POST['k_tekst'];

try {$decoded = JWT::decode($jwt, $key, array('HS256'));}
catch (Exception $e){
    http_response_code(401);
    header('Location: ../innlegg.html?reg=false&type=kom');
    exit();
}

$kommentar->b_id = $decoded->data->b_id;

if(
    !empty($kommentar->b_id) &&
    !empty($kommentar->k_tekst) &&
    !empty($kommentar->p_id) &&
    $kommentar->insert()
){
    include_once 'objects/logg.php';
    $logg = new Logg($db);
    $logg->b_id = $decoded->data->b_id;
    $logg->lo_name = "create_kommentar";
    $logg->insert();

    header('Location: ../innlegg.html?reg=true&type=kom');
    exit;
}

// message if unable to create user
else{

    header('Location: ../innlegg.html?reg=false&type=kom');
    exit;
}

?>
