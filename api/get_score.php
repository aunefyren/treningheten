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
include_once 'objects/autist.php';
include_once 'objects/bilde.php';
include_once 'objects/kommentar.php';
include_once 'objects/post.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$bruker = new Brukere($db);
$autist = new Autist($db);
$bilde = new Bilde($db);
$kommentar = new Kommentar($db);
$post = new Post($db);

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

try{
    $autist->b_id = $data->b_id;
    $bilde->b_id = $data->b_id;
    $post->b_id = $data->b_id;
    $kommentar->b_id = $data->b_id;

    $autist_score = $autist->get_autist_score();
    $ar_score = $bilde->get_ar_score();
    $bil_score = $bilde->get_bil_score();
    $post_score = $post->get_post_score();
    $kom_score = $kommentar->get_kom_score();
    echo json_encode(array( "error" => "false",
                            "message" => "No error.",
                            "b_id" => $data->b_id,
                            "autist_score" => $autist_score*10,
                            "ar_score" => $ar_score*5,
                            "bil_score" => $bil_score*5,
                            "post_score" => $post_score*5,
                            "kom_score" => $kom_score*2
                           ));
} catch(Exception $e) {

    // set response code
    http_response_code(400);

    // display message: unable to create user
    echo json_encode(array("error" => "true", "message" => "Nådde ikke databasen."));
}
?>
