<?php
// required headers
header("Access-Control-Allow-Origin: *");
header("Content-Type: application/json; charset=UTF-8");
header("Access-Control-Allow-Methods: POST");
header("Access-Control-Max-Age: 3600");
header("Access-Control-Allow-Headers: Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With");

// files needed to connect to database
include_once 'config/database.php';
include_once 'objects/user.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate user object
$brukere = new Brukere($db);

// get posted data
$data = json_decode(file_get_contents("php://input"));

// set product property values
$brukere->b_epost = $data->b_epost;
$user_exists = $brukere->getUser();

// generate json web token
include_once 'config/core.php';
include_once 'libs/php-jwt-master/src/BeforeValidException.php';
include_once 'libs/php-jwt-master/src/ExpiredException.php';
include_once 'libs/php-jwt-master/src/SignatureInvalidException.php';
include_once 'libs/php-jwt-master/src/JWT.php';
use \Firebase\JWT\JWT;

// check if email exists and if password is correct
if($user_exists && password_verify($data->b_passord, $brukere->b_passord)){

    $token = array(
        "iat" => $issued_at,
        "exp" => $expiration_time,
        "iss" => $issuer,
        "data" => array(
           "b_id" => $brukere->b_id,
           "b_fornavn" => $brukere->b_fornavn,
           "b_etternavn" => $brukere->b_etternavn,
           "b_epost" => $brukere->b_epost,
           "b_admin" => $brukere->b_admin,
           "b_update" => $brukere->b_update,
           "b_bio" => $brukere->b_bio,
           "b_active" => $brukere->b_active,
           "b_tittel" => $brukere->b_tittel,
           "b_kallenavn" => $brukere->b_kallenavn,
           "b_skapelse" => $brukere->b_skapelse,
           "postnr" => $brukere->postnr
       )
    );

    if(!$brukere->get_active_state()) {
        echo json_encode(array("message" => "Brukeren er deaktivert.", "error" => "true"));
        exit;
    }

    include_once 'objects/logg.php';
    $logg = new Logg($db);
    $logg->b_id = $brukere->b_id;
    $logg->lo_name = "login";
    $logg->insert();

    $brukere->login_activity();

    // generate jwt
    $jwt = JWT::encode($token, $key);

    $brukere->set_update_state();

    echo json_encode(
            array(
                "message" => "Bruker logget inn! Bytt side øverst.",
                "error" => "false",
                "jwt" => $jwt
            )
        );

} else {

    echo json_encode(array("message" => "Logget ikke inn. Epost eller passord er feil.", "error" => "true"));
}
?>
