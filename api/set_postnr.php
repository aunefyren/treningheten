<?php

// files needed to connect to database
include_once 'config/database.php';
include_once 'objects/postnr.php';
ini_set('max_execution_time', 300);

// get posted data
$data = json_decode(file_get_contents("json/postnr.json"));

$sum = count($data);

echo "<!doctype html><html><head><title>Legger inn postnr - (" . $sum . ")</title></head><body>";
echo "Fant " . count($data) . " postnr. ";
echo "<br>";
echo "<br>";
$postnr_count = 0;

for($i = 0; $i < count($data); $i++) {
    $database = new Database();
    $db = $database->getConnection();
    $postklasse = new Postnr($db);

    $postklasse->postnr = $data[$i]->postnr;
    $postklasse->poststed = $data[$i]->poststed;
    $postklasse->kommunenr = $data[$i]->kommunenr;
    $postklasse->kommunenavn = $data[$i]->kommunenavn;
    $postklasse->kategori = $data[$i]->kategori;

    $result = json_decode($postklasse->insert());
    if($result->error == "true") {
        echo "<p style='color:red;'>" . $data[$i]->postnr . ": " . $result->message . "</p>";
        echo "<br>";
        echo "Stopper...";
        exit();
    } else {
        $postnr_count = $postnr_count + 1;
        echo "<p style='color:green;'>" . $data[$i]->postnr . ": " . $result->message . "</p>";
    }
}

echo "<br>";
echo "Behandlet " . $postnr_count . " postnr!";
echo "<br>";
echo "<br>";
echo "Suksess!";
echo "

</body>
</html>
";
?>
