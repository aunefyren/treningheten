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
if(empty($data) || !isset($data->user_email) || !isset($data->user_password)) {
    
    echo json_encode(array("error" => true, "message" => "Fikk ikke passord eller e-post for innlogging."));
    exit(0);
	
}

// Remove potential harmfull input
$user->user_email = htmlspecialchars($data->user_email);
$user->user_password = htmlspecialchars($data->user_password);

// Confirm password
if(!$user->check_password()) {

    echo json_encode(array("error" => true, "message" => "Passord og e-post kombinasjon ble ikke akseptert."));
    exit(0);

}

// Load user data for inspection
$user->get_user_data();

// Check if user is disabled
if($user->user_disabled !== '0') {

    echo json_encode(array("error" => true, "message" => "Denne brukeren har blitt deaktivert."));
    exit(0);

}

// Check if user is active
if($user->user_active !== '1') {

    echo json_encode(array("error" => true, "message" => "Denne brukeren er ikke aktivert enda. Sjekk e-posten din for lenke, eventuelt spam-mappen."));
    exit(0);

}

// Get cookie
$cookie = $user->get_user_cookie();

// Check profile photo
$filename = dirname(__FILE__, 2) . '/assets/profiles/' . $user->user_id . '.jpg';
$default = dirname(__FILE__, 2) . '/assets/default.jpg';

if (!file_exists($filename)) {
    
    copy($default, $filename);

}

// Print cookie and exit
echo json_encode(array("error" => false, "message" => "Innlogging suksessfull!", "cookie" => $cookie));
exit(0);
?>