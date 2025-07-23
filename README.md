## Cargo Workspace

This repository is a Cargo workspace with two crates:

- `cli`: the `pros` command-line interface
- `daemon`: the library implementing daemon and config logic

Build and run the CLI:
```bash
cargo build -p proxs
cargo run -p proxs -- daemon
```

## Proxy Switcher
macOSのdaemonとして動き、ブラウザからのリクエストを解析し適切なSocks Proxyを動的に貼り管理するソフトウェア


## User Interfaces

デーモンを起動してバックグラウンドでプロキシを管理する
```shell-session
$ pros daemon
```

貼っているSocks Proxyの一覧を表示する
```shell-session
$ pros status
- lb3 established
- lb4 error
- lb5 timeout
```

貼るSocks Proxyを追加する
```shell-session
$ pros rule add <NAME> <IPADDR> [SITES...]
```

貼るSocks Proxyを削除する
```shell-session
$ pros rule del <NAME>
```


## Config

`~/.config/proxs/config.toml` に toml形式で設定を記述する。

```toml
[global]
start_port=1080
ports=10
identity_file='.ssh/id_rsa'

[servers]
[servers.lb3]
ipaddr=10.0.0.1
sites=["google.com", "apple.com"]
[servers.lb4]
ipaddr=10.0.1.1
sites=["yahoo.co.jp", "x.com"]
```
