#!/usr/bin/env bash


function do_test() {
    size=$1 # 100k, 1m, 10m
    enable_socks=$2 # true or false
    if [ "$enable_socks" = true ]; then
        curl --http2 --socks5-hostname 127.0.0.1:8080 -k -o /dev/null -sS -w 'sz=%{size_download}B\tspd=%{speed_download}B/s t_conn=%{time_connect}s t_tls=%{time_appconnect}s t_ttfb=%{time_starttransfer}s t_tot=%{time_total}s http_version=%{http_version}\n' https://dev-instance-1.local/${size}.bin
    else
        curl --http2 -k -o /dev/null -sS -w 'sz=%{size_download}B\tspd=%{speed_download}B/s t_conn=%{time_connect}s t_tls=%{time_appconnect}s t_ttfb=%{time_starttransfer}s t_tot=%{time_total}s http_version=%{http_version}\n' https://10.20.0.11/${size}.bin
    fi
}

echo "Testing with SOCKS enabled:"
do_test 10k true
do_test 100k true
do_test 1m true
do_test 10m true
do_test 100m true
do_test 1g true
echo "Testing with SOCKS disabled:"
do_test 10k false
do_test 100k false
do_test 1m false
do_test 10m false
do_test 100m false
do_test 1g false
