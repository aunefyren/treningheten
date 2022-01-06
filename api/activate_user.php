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
if(empty($data) || !isset($data->user_email) || !isset($data->user_hash)) {
    
    echo json_encode(array("error" => true, "message" => "Fikk ikke hash eller e-post for aktivering."));
    exit(0);
	
}

// Remove potential harmfull input
$user->user_email = htmlspecialchars($data->user_email);

// Load user data for inspection
$user->get_user_data();

// Check if user is disabled
if($user->user_hash !== $data->user_hash) {

    echo json_encode(array("error" => true, "message" => "Ugyldig aktiveringslenke."));
    exit(0);

}

$user->set_user_active();
$user->refresh_hash();

// Print cookie and exit
echo json_encode(array("error" => false, "message" => "Brukeren ble aktivert!"));
exit(0);
?>