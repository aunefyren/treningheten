<?php
// required headers
header("Access-Control-Allow-Origin: *");
header("Content-Type: application/json; charset=UTF-8");
header("Access-Control-Allow-Methods: POST");
header("Access-Control-Max-Age: 3600");
header("Access-Control-Allow-Headers: Content-Type, Access-Control-Allow-Headers, Authorization, X-Requested-With");

// Files needed to use objects
require(dirname(__FILE__) . '/config/database.php');
require(dirname(__FILE__) . '/objects/user.php');
require(dirname(__FILE__) . '/objects/code.php');

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$user = new User($db);
$code = new Code($db);

// get posted data
$data = json_decode(file_get_contents("php://input"));

// Validate creation hash/invite
$code->code_hash = $data->code_hash;
$code_object = $code->get_code();
if(!$code_object) {
    echo json_encode(
        array(
            "message" => "Ugydlig invitasjonskode.",
            "error" => true
        )
    );
    exit;
}
$code_object = json_decode($code_object);
if($code_object->code_used !== '0') {
    echo json_encode(
        array(
            "message" => "Invitasjonskoden er allerede brukt.",
            "error" => true
        )
    );
    exit;
}
$code_id = $code_object->code_id;

if(!preg_match('/^(?=.*[a-zæøå])(?=.*[A-ZÆØÅ])(?=.*\d)[a-zæøåA-ZÆØÅ\d]{8,}$/', $data->user_password)) {

    echo json_encode(array("message" => "Ugyldig passord. Minst 8 tegn, en stor bokstav og ett tall.", "error" => true));
    exit();

}

if(!filter_var($data->user_email, FILTER_VALIDATE_EMAIL)) {
    
    echo json_encode(array("message" => "Ugyldig e-post.", "error" => true));
    exit();

}

// set product property values
$user->user_firstname = htmlspecialchars(strip_tags($data->user_firstname));
$user->user_lastname = htmlspecialchars(strip_tags($data->user_lastname));
$user->user_email = $data->user_email;
$user->user_password = $data->user_password;
$user->code_id = htmlspecialchars(strip_tags($code_id));

// create the user
if($user->check_email()) {
    // display message: unable to create user
    echo json_encode(array("message" => "Epost er i bruk.", "error" => true));
    exit;
}

if(
    !empty($user->user_firstname) &&
    !empty($user->user_lastname) &&
    !empty($user->user_email) &&
    !empty($user->user_password) &&
    !empty($user->code_id) &&
    $user->create_user()
){

    // Mark code as used
    $code->set_code_used();

    // Mark code as used
    $email = $user->verification_email();
    if(!$email) {
        echo json_encode(array("message" => "Bruker ble skapt, men aktiverings-epost ble ikke sendt.", "error" => false));
        exit();
    }

    // display message: user was created
    echo json_encode(array("message" => "Bruker ble skapt. Sjekk e-post for aktiverings-lenke.", "error" => false));
}

// message if unable to create user
else{

    // display message: unable to create user
    echo json_encode(array("message" => "Bruker ble ikke skapt.", "error" => true));
}
?>
