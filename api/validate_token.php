<?php
// required headers
header("Access-Control-Allow-Origin: *");
header("Content-Type: application/json; charset=UTF-8");
header("Access-Control-Allow-Methods: POST");
header("Access-Control-Max-Age: 3600");
header("Access-Control-Allow-Headers: Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With");

// required to decode jwt
include_once 'config/core.php';
include_once 'libs/php-jwt-master/src/BeforeValidException.php';
include_once 'libs/php-jwt-master/src/ExpiredException.php';
include_once 'libs/php-jwt-master/src/SignatureInvalidException.php';
include_once 'libs/php-jwt-master/src/JWT.php';
use \Firebase\JWT\JWT;

// get posted data
$data = json_decode(file_get_contents("php://input"));

// get jwt
$jwt=isset($data->jwt) ? $data->jwt : "";

if($jwt){

    // if decode succeed, show user details
    try {
        // decode jwt
        $decoded = JWT::decode($jwt, $key, array('HS256'));

        //Check if user needs to relog
        include_once 'objects/user.php';
        include_once 'config/database.php';
        $database = new Database();
        $db = $database->getConnection();
        $user = new Brukere($db);
        $user->b_id = $decoded->data->b_id;

        $update = $user->get_update_state();
        if($update) {
            echo json_encode(array("message" => "Du må logge inn på nytt.", "error" => "true"));
            exit;
        }

        $active = $user->get_active_state();
        if(!$active) {
            echo json_encode(array("message" => "Brukeren er deaktivert.", "error" => "true"));
            exit;
        }

        // set response code
        http_response_code(200);

        // show user details
        echo json_encode(array(
            "message" => "Tilgang avgitt.",
            "error" => "false",
            "data" => $decoded->data
        ));

    }

    // if decode fails, it means jwt is invalid
    catch (Exception $e){

        // set response code
        http_response_code(401);

        // tell the user access denied  & show error message
        echo json_encode(array(
            "message" => "Logg inn for tilgang.",
            "error" => $e->getMessage()
        ));
    }
}

// show error message if jwt is empty
else{

    // set response code
    http_response_code(401);

    // tell the user access denied
    echo json_encode(array("message" => "Tilgang nektet.", "error" => "false"));
}

?>
