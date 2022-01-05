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
include_once 'objects/bilde.php';

// get database connection
$database = new Database();
$db = $database->getConnection();

// instantiate product object
$brukere = new Brukere($db);
$bilder = new Bilde($db);

$data_jwt = $_POST['jwt'];

$jwt = isset($data_jwt) ? $data_jwt : "";

try {$decoded = JWT::decode($jwt, $key, array('HS256'));}
catch (Exception $e){
    http_response_code(401);
    header('Location: ../bilder.html?reg=false');
    exit;
}

$bilder->b_id = $decoded->data->b_id;
$target_dir = "../assets/bilder/";

$fileCount = 0;
foreach ($_FILES["bi_img"]["name"] as $upload){
    $fileCount ++;
}

for ($i = 0; $i < $fileCount; $i++) {
    if($_FILES["bi_img"]["type"][$i] == "video/mp4"){
        $file = ".mp4";
        $bilder->bi_extension = "mp4";
    } else {
        $file = ".jpg";
        $bilder->bi_extension = "jpg";
    }

    if(
        !empty($bilder->b_id) &&
        $bilder->insert()
    ){
        $bi_bilder = $bilder->get_highest();
        $target_file = $target_dir . $bi_bilder . $file;

        if (move_uploaded_file($_FILES["bi_img"]["tmp_name"][$i], $target_file)) {

            echo "Bildet ble registrert og bildet ". basename( $_FILES["bi_img"]["name"][$i]). " ble lastet opp.";

        } else {

            echo "Bildet registrert, men bildet ble ikke lastet opp. " . $data_ar_id . " " . $target_dir . " " . $target_file . " " . $fileCount;
            print_r($_FILES["bi_img"]["error"]);
            header('Location: ../bilder.html?reg=false');
            exit;
        }

    } else {

        echo "Error, ingenting ble registrert";
        header('Location: ../bilder.html?reg=false');
        exit;
    }
}

echo "All good";

include_once 'objects/logg.php';
$logg = new Logg($db);
$logg->b_id = $decoded->data->b_id;
$logg->lo_name = "upload_bilde";
$logg->insert();

header('Location: ../bilder.html?reg=true');
exit;

?>
