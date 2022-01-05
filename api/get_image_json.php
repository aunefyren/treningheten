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

// get database connection
$database = new Database();
$db = $database->getConnection();

$data = json_decode(file_get_contents("php://input"));
//$data_jwt = $data->jwt;

$jwt = isset($data->jwt) ? $data->jwt : "";

try {$decoded = JWT::decode($jwt, $key, array('HS256'));}
catch (Exception $e){
    http_response_code(401);
    echo json_encode(
        array(
            "message" => "Ugyldig innlogging.",
            "error" => "true"
        )
    );
    exit();
}

$pictures = array();

$id = $data->id;
$type = $data->type;
$ar = $data->ar;
$i = $data->i;
$j = $data->j;
$size;

for($x = 0; $x < $j; $x++) {
    $file = ".jpg";
    if($type == "autist") {
        $path = "autister/" . $id[$x];

    } else if($type == "ar") {
        //print_r($id);
        $path = "ar_bilder/" . $ar . "/" . $id[$x]->ar_bil_id;


    } else if($type == "bilde") {
        $path = "bilder/" . $id[$x];

    } else if($type == "medlem") {
        $path = "medlemmer/" . $id[$x];

    } else {
        $pictures = error($i, $x, $pictures);

    }

    if(!@$content = file_get_contents('../assets/' . $path . $file)) {
        $pictures = error($i, $x, $pictures);

    } else {
        $size = getimagesize('../assets/' . $path . $file);
        if(($size[0]*$size[1]) > 1000000) {
            $resize = imagecreatetruecolor($size[0]/4, $size[1]/4);
            $ori = imagecreatefromstring($content);

            imagecopyresampled($resize, $ori, 0 , 0 , 0 , 0 , $size[0]/4, $size[1]/4, $size[0] , $size[1]);
            //imagestring ( $resize , int $font , int $x , int $y , string $string , int $color )
            ob_start(); // Let's start output buffering.
            imagejpeg($resize); //This will normally output the image, but because of ob_start(), it won't.
            $content2 = ob_get_contents(); //Instead, output above is saved to $contents
            ob_end_clean();
            $base64 = 'data:image/' . $file . ';base64,' . base64_encode($content2);
            $array = array("i" => $i, "j" => $x, "base64" => $base64);
            array_push($pictures, $array);

        } else {
            $base64 = 'data:image/' . $file . ';base64,' . base64_encode($content);
            $array = array("i" => $i, "j" => $x, "base64" => $base64);
            array_push($pictures, $array);
        }
    }
}

echo json_encode(array("pictures" => $pictures, "error" => "false"));

function error($i, $j, $pictures) {
    $content = file_get_contents('../assets/logo/logo.png');
    $base64 = 'data:image/' . '.png' . ';base64,' . base64_encode($content);
    $array = array("i" => $i, "j" => $j, "base64" => $base64);
    array_push($pictures, $array);
    return $pictures;
}
?>