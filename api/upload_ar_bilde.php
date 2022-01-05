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
include_once 'objects/arrangement.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$brukere = new Brukere($db);
$arrangementer = new Arrangement($db);

$data_jwt = $_POST['jwt'];
$data_ar_id = $_POST['ar_id'];

$jwt = isset($data_jwt) ? $data_jwt : "";

try {$decoded = JWT::decode($jwt, $key, array('HS256'));}
catch (Exception $e){
    http_response_code(401);
    header('Location: ../arrangementer.html?reg=false');
    exit;
}

$arrangementer->ar_id = $data_ar_id;
$arrangementer->b_id = $decoded->data->b_id;
$target_dir = "../assets/ar_bilder/" . $data_ar_id . "/";

if(!is_dir($target_dir)) {
    mkdir($target_dir);
    mkdir($target_dir . "thumbs");
}

$fileCount = 0;
foreach ($_FILES["ar_img"]["name"] as $upload){
    $fileCount ++;
}

for ($i = 0; $i < $fileCount; $i++) {
    if($_FILES["ar_img"]["type"][$i] == "video/mp4"){
        $file = ".mp4";
        $arrangementer->ar_bil_extension = "mp4";
    } else {
        $file = ".jpg";
        $arrangementer->ar_bil_extension = "jpg";
    }

    if(
        !empty($arrangementer->ar_id) &&
        !empty($arrangementer->b_id) &&
        $arrangementer->insert()
    ){
        $ar_bilder = $arrangementer->get_highest();
        $target_file = $target_dir . $ar_bilder . $file;

        if (move_uploaded_file($_FILES["ar_img"]["tmp_name"][$i], $target_file)) {

            echo "Bildet ble registrert og bildet ". basename( $_FILES["ar_img"]["name"][$i]). " ble lastet opp.";

        } else {

            echo "Bildet registrert, men bildet ble ikke lastet opp. " . $data_ar_id . " " . $target_dir . " " . $target_file . " " . $fileCount;
            print_r($_FILES["ar_img"]["error"]);
            header('Location: ../arrangementer.html?reg=false');
            exit;
        }

    } else {

        echo "Error, ingenting ble registrert";
        header('Location: ../arrangementer.html?reg=false');
        exit;
    }
}

echo "All good";

include_once 'objects/logg.php';
$logg = new Logg($db);
$logg->b_id = $decoded->data->b_id;
$logg->lo_name = "upload_ar_bilde";
$logg->insert();

header('Location: ../arrangementer.html?reg=true');
exit;

?>
