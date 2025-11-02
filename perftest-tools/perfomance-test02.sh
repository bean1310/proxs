#!/usr/bin/env bash


function do_test() {
    size=$1 # 100k, 1m, 10m
    echo -n '[Proxs Socks] '; curl --http2 --socks5-hostname 127.0.0.1:8080 -k -o /dev/null -sS -w '%{speed_download} http_version=%{http_version}\n' https://dev-instance-1.local/${size}.bin
    echo -n '[Native Socks] '; curl --http2 --socks5-hostname 127.0.0.1:9191 -k -o /dev/null -sS -w '%{speed_download} http_version=%{http_version}\n' https://dev-instance-1.local/${size}.bin
    echo -n '[Direct] '; curl --http2 -k -o /dev/null -sS -w '%{speed_download} http_version=%{http_version}\n' https://10.20.0.11/${size}.bin
}

echo "Testing with SOCKS enabled:"
do_test 10k 
do_test 100k
do_test 1m 
do_test 10m
do_test 100m 
do_test 1g
