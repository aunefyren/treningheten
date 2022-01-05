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
include_once 'objects/post.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$poster = new Post($db);

$data_jwt = $_POST['jwt'];
$data_id = $_POST['p_id'];

$jwt = isset($data_jwt) ? $data_jwt : "";

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

$poster->p_id = $data_id;

if(
    !empty($poster->p_id) &&
    $poster->delete()
){
    include_once 'objects/logg.php';
    $logg = new Logg($db);
    $logg->b_id = $decoded->data->b_id;
    $logg->lo_name = "set_post_del";
    $logg->insert();

    header('Location: ../innlegg.html?del=true&type=post');
    exit;
}

// message if unable to create user
else{

    // display message: unable to create user
    header('Location: ../innlegg.html?del=false&type=post');
    exit;
}

?>
