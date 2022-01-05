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
include_once 'objects/autist.php';
include_once 'objects/user.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$autist = new Autist($db);
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
            "error" => "true"
        )
    );
    exit();
}

if($data->a_fornavn != "") {
    $data->a_fornavn = htmlspecialchars(strip_tags($data->a_fornavn));
    $autist->a_fornavn = "`a_fornavn` LIKE '%" . $data->a_fornavn . "%'";
}

if($data->a_etternavn != "") {
    $data->a_etternavn = htmlspecialchars(strip_tags($data->a_etternavn));
    $autist->a_etternavn = "`a_etternavn` LIKE '%" . $data->a_etternavn . "%'";
}

if($data->a_bio != "") {
    $data->a_bio = htmlspecialchars(strip_tags($data->a_bio));
    $autist->a_bio = "`a_bio` LIKE '%" . $data->a_bio . "%'";
}

if($data->postnr != "") {
    $data->postnr = htmlspecialchars(strip_tags($data->postnr));
    $autist->postnr = "postnr.`postnr` LIKE '%" . $data->postnr . "%'";
}

if($data->a_grad != "") {
    $data->a_grad = htmlspecialchars(strip_tags($data->a_grad));
    $autist->a_grad = "`a_grad` LIKE '%" . $data->a_grad . "%'";
}

if($data->a_risiko != "") {
    $data->a_risiko = htmlspecialchars(strip_tags($data->a_risiko));
    $autist->a_risiko = "`a_risiko` LIKE '%" . $data->a_risiko . "%'";
}

if($data->poststed != "") {
    $data->poststed = htmlspecialchars(strip_tags($data->poststed));
    $autist->poststed = "postnr.`poststed` LIKE '%" . $data->poststed . "%'";
}

$autist->b_id = $decoded->data->b_id;

$result = $autist->search();

include_once 'objects/logg.php';
$logg = new Logg($db);
$logg->b_id = $decoded->data->b_id;
$logg->lo_name = "get_autisme";
$logg->insert();

echo $result;
?>
