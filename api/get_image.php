<?php
ob_start() ;

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

// get database connection
$database = new Database();
$db = $database->getConnection();

if(!$data_jwt = $_GET['jwt']) {
    error();
    exit();
}

$jwt = isset($data_jwt) ? $data_jwt : "";

try {$decoded = JWT::decode($jwt, $key, array('HS256'));}
catch (Exception $e){
    error();
    exit();
}

$id = $_GET['id'];
$type = $_GET['type'];
$ar = $_GET['ar'];


$file = ".jpg";
$file2 = ".mp4";

if($type == "autist") {
    $path = "autister/" . $id;

} else if($type == "ar") {
    $path = "ar_bilder/" . $ar . "/" . $id;

} else if($type == "bilde") {
    $path = "bilder/" . $id;

} else if($type == "medlem") {
    $path = "medlemmer/" . $id;

} else {
    error();
}

$image = True;
if(!$content = file_get_contents('../assets/' . $path . $file)) {
    if(!$content = file_get_contents('../assets/' . $path . $file2)) {
        error();
    }

    $image = False;
}

include_once 'objects/logg.php';
$logg = new Logg($db);
$logg->b_id = $decoded->data->b_id;
$logg->lo_name = "get_image";
$logg->insert();

ob_end_clean();

if($image) {
header('Content-Type: image/jpeg');
} else {
header('Content-Type: video/mp4');
}
echo $content;

function error() {
    $content = file_get_contents('../assets/logo/logo.png');
    ob_end_clean();
    header('Content-Type: image/jpeg');
    echo $content;
    exit();
}
?>