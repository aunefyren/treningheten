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

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$brukere = new Brukere($db);
$autist = new Autist($db);

$data_jwt = $_POST['jwt'];
$data_fornavn = $_POST['a_fornavn'];
$data_etternavn = $_POST['a_etternavn'];
$data_postnr = $_POST['postnr'];
$data_grad = $_POST['a_grad'];
$data_risiko = $_POST['a_risiko'];
$data_bio = $_POST['a_bio'];

$jwt = isset($data_jwt) ? $data_jwt : "";

try {$decoded = JWT::decode($jwt, $key, array('HS256'));}
catch (Exception $e){
    http_response_code(401);
    header('Location: ../autister.html?error=true&message=Autist%20ble%20ikke%20registrert.');
    exit;
}

$autist->a_fornavn = $data_fornavn;
$autist->a_etternavn = $data_etternavn;
$autist->a_bio = $data_bio;
$autist->postnr = $data_postnr;
$autist->a_grad = $data_grad;
$autist->a_risiko = $data_risiko;
$autist->b_id = $decoded->data->b_id;

include_once 'objects/postnr.php';
$postnr = new Postnr($db);
$postnr->postnr = $data_postnr;
$poststed = $postnr->get_poststed();
if($poststed == "") {
    header('Location: ../autister.html?error=true&message=Postnummeret%20er%20ikke%20gyldig.');
    exit;
}

if(
    !empty($autist->a_fornavn) &&
    !empty($autist->a_etternavn) &&
    !empty($autist->a_bio) &&
    !empty($autist->postnr) &&
    !empty($autist->a_grad) &&
    !empty($autist->a_risiko) &&
    !empty($autist->b_id) &&
    $autist->insert()
){
    $a_id = $autist->get_newest();
    $target_dir = "../assets/autister/";
    $target_file = $target_dir . $a_id . ".jpg";

    if (move_uploaded_file($_FILES["a_img"]["tmp_name"], $target_file)) {
        echo "Autisten ble registrert og bildet ". basename( $_FILES["a_img"]["name"]). " ble lastet opp.";

        include_once 'objects/logg.php';
        $logg = new Logg($db);
        $logg->b_id = $decoded->data->b_id;
        $logg->lo_name = "upload_autist";
        $logg->insert();

        header('Location: ../autister.html?error=false&message=Autist%20ble%20registrert.');
        exit;
    } else {
        echo "Autist registrert, men bildet ble ikke lastet opp. " . $a_id . " " . $target_dir . " " . $target_file;
        header('Location: ../autister.html?error=true&message=Autist%20ble%20ikke%20registrert.');
        exit;
    }
}

// message if unable to create user
else{

    // display message: unable to create user
    echo "Error, ingenting ble registrert";
    header('Location: ../autister.html?error=true&message=Autist%20ble%20ikke%20registrert.');
    exit;
}

?>
