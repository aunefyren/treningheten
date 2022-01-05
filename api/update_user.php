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

// instantiate user object
$brukere = new Brukere($db);

// get posted data
$data = json_decode(file_get_contents("php://input"));

// get jwt
$jwt=isset($data->jwt) ? $data->jwt : "";

// if jwt is not empty
if($jwt){

    // if decode succeed, show user details
    try {

        // decode jwt
        $decoded = JWT::decode($jwt, $key, array('HS256'));

        $brukere->b_fornavn = $decoded->data->b_fornavn;
        $brukere->b_etternavn = $decoded->data->b_etternavn;
        $brukere->b_tittel = $decoded->data->b_tittel;
        $brukere->b_id = $decoded->data->b_id;
        $gammel_epost = $decoded->data->b_epost;
        $gammel_passord = $data->g_passord;

        $brukere->b_epost = $data->b_epost;
        $brukere->b_passord = $data->b_passord;
        $brukere->postnr = $data->postnr;
        $brukere->b_bio = $data->b_bio;
        $brukere->b_kallenavn = $data->b_kallenavn;

        $json_data = array("b_epost" => $gammel_epost, "b_passord" => $gammel_passord);
        $json = json_encode($json_data);
        $log_in = json_decode(sendPostData('https://krenkelsesarmeen.no/api/login.php', $json));
        if($log_in->error == "true") {
            echo json_encode(array("message" => "Gammelt passord er feil.", "error" => "true"));
            exit;
        }

        //if email is in use
        if($gammel_epost != $brukere->b_epost && $brukere->sjekk_epost()) {
          echo json_encode(array("message" => "Epost er i bruk.", "error" => "true"));
          exit();

        }

        // update the user record
        if($brukere->update()){
            // we need to re-generate jwt because user details might be different
            $brukere->getUser();

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

            $jwt = JWT::encode($token, $key);

            include_once 'objects/logg.php';
            $logg = new Logg($db);
            $logg->b_id = $decoded->data->b_id;
            $logg->lo_name = "update_user";
            $logg->insert();

            // set response code
            http_response_code(200);

            // response in json format
            echo json_encode(
                  array(
                      "message" => "Bruker ble oppdatert.",
                      "jwt" => $jwt,
                      "error" => false
                  )
            );
        }

            // message if unable to update user
            else{
                // set response code
                http_response_code(401);

                // show error message
                echo json_encode(array("message" => "Bruker ble ikke oppdatert.", "error" => true));
            }
    }

    // if decode fails, it means jwt is invalid
    catch (Exception $e){

        // set response code
        http_response_code(401);

        // show error message
        echo json_encode(array(
            "message" => "Tilgang nektet.",
            "error" => true
        ));
    }
}

// show error message if jwt is empty
else{

    // set response code
    http_response_code(401);

    // tell the user access denied
    echo json_encode(array("message" => "Access denied."));
}

function sendPostData($url, $post){
    $ch = curl_init($url);

    // Attach encoded JSON string to the POST fields
    curl_setopt($ch, CURLOPT_POSTFIELDS, $post);

    // Set the content type to application/json
    curl_setopt($ch, CURLOPT_HTTPHEADER, array('Content-Type:application/json'));

    // Return response instead of outputting
    curl_setopt($ch, CURLOPT_RETURNTRANSFER, true);

    // Execute the POST request
    $result = curl_exec($ch);

    // Close cURL resource
    curl_close($ch);

    return $result;
}

?>
