<?php
// Required headers
header("Access-Control-Allow-Origin: *");
header("Content-Type: application/json; charset=UTF-8");
header("Access-Control-Allow-Methods: POST");
header("Access-Control-Max-Age: 3600");
header("Access-Control-Allow-Headers: Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With");

// Files needed to use objects
require(dirname(__FILE__) . '/config/database.php');
require(dirname(__FILE__) . '/objects/user.php');

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$user = new User($db);
$data = json_decode(file_get_contents("php://input"));

// If POST data is empty or wrong
if(empty($data) || !isset($data->cookie) || !isset($data->user_password) || !isset($data->data)) {
    
    echo json_encode(array("error" => true, "message" => "Fikk ikke kjeks, passord eller data for innlogging."));
    exit(0);
	
}

// Remove potential harmfull input
$cookie = htmlspecialchars($data->cookie);

$cookie_object = $user->validate_user_cookie($cookie);

// Check if cookie was accepted
if(!$cookie_object) {

    echo json_encode(array("error" => true, "message" => "Kjeksen ble ikke akseptert."));
    exit(0);

}

$cookie_decoded = json_decode($cookie_object, true);

// Remove potential harmfull input
$user->user_email = htmlspecialchars($cookie_decoded['data']['user_email']);
$user->user_password = htmlspecialchars($data->user_password);

// Confirm password
if(!$user->check_password()) {

    echo json_encode(array("error" => true, "message" => "Passord og e-post kombinasjon ble ikke akseptert."));
    exit(0);

}

if($data->data->user_password !== "") {

    if(!preg_match('/^(?=.*[a-zæøå])(?=.*[A-ZÆØÅ])(?=.*\d)[a-zæøåA-ZÆØÅ\d]{8,}$/', $data->data->user_password)) {

        echo json_encode(array("message" => "Ugyldig passord. Minst 8 tegn, en stor bokstav og ett tall.", "error" => true));
        exit();
    
    }

    $user->user_id = $cookie_decoded['data']['user_id'];
    $user->user_password = $data->data->user_password;

    if(!$user->set_user_password()) {

        echo json_encode(array("message" => "Klarte ikke oppdatere passord.", "error" => true));
        exit(0);
        
    }

}

if($user->user_email !== $data->data->user_email) {

    // create the user
    if($user->check_email()) {

        // display message: unable to create user
        echo json_encode(array("message" => "E-post er i bruk.", "error" => true));
        exit(0);

    }

    $user->user_id = $cookie_decoded['data']['user_id'];
    $user->user_email = $data->data->user_email;

    if(!$user->set_user_email()) {

        echo json_encode(array("message" => "Klarte ikke oppdatere e-post.", "error" => true));
        exit(0);
        
    }

}

if($data->data->user_profile_photo !== false) {

    $photo = str_replace(' ', '+', $data->data->user_profile_photo);
    list($type, $data) = explode(';', $photo);
    list(, $data)      = explode(',', $data);

    $image = base64_decode($data);

    if(!$image) {

        echo json_encode(array("error" => true, "message" => "Bildet ble ikke godkjent."));
        exit(0);

    }

    $path = dirname(__FILE__, 2) . '/assets/profiles/' . $cookie_decoded['data']['user_id'] . '.jpg';
    $success = file_put_contents($path, $image);

    if(!$success) {
        echo json_encode(array("error" => true, "message" => "Klarte ikke oppdatere bildet."));
    exit(0);
    }

}

// Load user data for inspection
$user->get_user_data();

// Get cookie
$cookie = $user->get_user_cookie();

// Print cookie and exit
echo json_encode(array("error" => false, "message" => "Bruker ble oppdatert.", "cookie" => $cookie));
exit(0);
?>