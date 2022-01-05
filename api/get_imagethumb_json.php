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
$data_jwt = $data->jwt;
$jwt = isset($data_jwt) ? $data_jwt : "";

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

    $error = false;
    if(@$id[$x]->bi_extension == "mp4" || @$id[$x]->ar_bil_extension == "mp4") {
        $file = False;
    } else{
        $file = ".jpg";
    }

    if($type == "autist") {
        $folder = "autister/";
        $path = "";
        $path2 = $id[$x];

    } else if($type == "ar") {
        $folder = "ar_bilder/";
        $path = $ar;
        $path2 = $id[$x]->ar_bil_id;


    } else if($type == "bilde") {
        $folder = "bilder/";
        $path = "";
        $path2 = $id[$x]->bi_id;

    } else if($type == "medlem") {
        $folder = "medlemmer/";
        $path = "";
        $path2 = $id[$x]->b_id;

    } else {
        $content = error();
        $error = true;
    }

    if($error) {
        $base64 = 'data:image/' . $file . ';base64,' . base64_encode($content);
        $array = array("i" => $i, "j" => $x, "base64" => $base64, "logo" => "true");
        array_push($pictures, $array);
        //echo "!5!";

    } else if(!$file) {
        $content = video();
        $base64 = 'data:image/' . $file . ';base64,' . base64_encode($content);
        $array = array("i" => $i, "j" => $x, "base64" => $base64, "logo" => "false");
        array_push($pictures, $array);

    } else {
        @$img_size = filesize('../assets/' . $folder . $path . '/' . $path2 . $file);
        //echo '../assets/' . $folder . $path . "/thumbs/" . $path2 . '_thumb' . $file;
        if($img_size > 1000000) {
            if(!@$content = file_get_contents('../assets/' . $folder . $path . "/thumbs/" . $path2 . '_thumb' . $file)) {
                $content = create_thumb($pictures, $i, $x, $id, $ar, $type);
                $base64 = 'data:image/' . $file . ';base64,' . base64_encode($content);

                if($error == true) {
                    $array = array("i" => $i, "j" => $x, "base64" => $base64, "logo" => "true");
                } else {
                    $array = array("i" => $i, "j" => $x, "base64" => $base64, "logo" => "false");
                }

                array_push($pictures, $array);

                logg($decoded->data->b_id, $db);

                //echo "!1!";

            } else {
                $base64 = 'data:image/' . $file . ';base64,' . base64_encode($content);
                $array = array("i" => $i, "j" => $x, "base64" => $base64, "logo" => "false");
                array_push($pictures, $array);

                logg($decoded->data->b_id, $db);

                //echo "!2!";

            }

        } else {
            if(!@$content = file_get_contents('../assets/' . $folder . $path . '/' . $path2 . $file)) {
                $content = error();
                $base64 = 'data:image/' . $file . ';base64,' . base64_encode($content);
                $array = array("i" => $i, "j" => $x, "base64" => $base64, "logo" => "true");
                array_push($pictures, $array);

                logg($decoded->data->b_id, $db);

                //echo "!4!";

            } else {
                $base64 = 'data:image/' . $file . ';base64,' . base64_encode($content);
                $array = array("i" => $i, "j" => $x, "base64" => $base64, "logo" => "false");
                array_push($pictures, $array);

                logg($decoded->data->b_id, $db);

                //echo "!3! - " . '../assets/' . $folder . $path . $path2 . $file;

            }
        }
    }
}

echo json_encode(array("pictures" => $pictures, "error" => "false"));

function error_return($i, $j, $pictures) {
    $content = file_get_contents('../assets/logo.png');
    $base64 = 'data:image/' . '.png' . ';base64,' . base64_encode($content);
    $array = array("i" => $i, "j" => $j, "base64" => $base64, "logo" => "true");
    array_push($pictures, $array);
    return $content;
}

function error() {
    $content = file_get_contents('../assets/logo/logo.png');
    $error = true;
    return $content;
}

function video() {
    $content = file_get_contents('../assets/play.png');
    return $content;
}

function logg($b_id, $db) {
    include_once 'objects/logg.php';
    $logg = new Logg($db);
    $logg->b_id = $b_id;
    $logg->lo_name = "get_imagethumb_json";
    $logg->insert();
}

function create_thumb($pictures, $i, $x, $id, $ar, $type) {
    $file = ".jpg";
    if($type == "autist") {
        $folder = "autister/";
        $path = "";
        $path2 = $id[$x];

    } else if($type == "ar") {
        $folder = "ar_bilder/";
        $path = $ar;
        $path2 = $id[$x]->ar_bil_id;

    } else if($type == "bilde") {
        $folder = "bilder/";
        $path = "";
        $path2 = $id[$x]->bi_id;

    } else if($type == "medlem") {
        $folder = "medlemmer/";
        $path = "";
        $path2 = $id[$x];

    } else {
        return $pictures = error($i, $x, $pictures);

    }

    if(!@$content = file_get_contents('../assets/' . $folder . $path . '/' . $path2 . $file)) {
        return error();

    } else {
        $size = getimagesize('../assets/' . $folder . $path . '/' . $path2 . $file);
        $resize = imagecreatetruecolor($size[0]/4, $size[1]/4);
        $ori = imagecreatefromstring($content);

        imagecopyresampled($resize, $ori, 0 , 0 , 0 , 0 , $size[0]/4, $size[1]/4, $size[0] , $size[1]);
        //imagestring ( $resize , int $font , int $x , int $y , string $string , int $color )
        ob_start(); // Let's start output buffering.
        imagejpeg($resize); //This will normally output the image, but because of ob_start(), it won't.
        $content2 = ob_get_contents(); //Instead, output above is saved to $contents
        $content3 = ob_get_contents();
        ob_end_clean();

        file_put_contents('../assets/' . $folder . $path . '/thumbs/' . $path2 . '_thumb' . $file, $content3);

        return $content2;
    }
}
?>