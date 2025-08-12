# Proxs

## Configuration
Proxs uses a configuration file named `config.toml` located in the user's configuration directory.
The default location is `~/Library/Application Support/proxs/config.toml` on macOS.

## Diagram
```mermaid
sequenceDiagram
    autonumber
    participant C as Client
    participant PS as Proxs
    participant TS as Socks Server
    participant T as Target Server (example.com:443)

    %% --------- SOCKS5 Handshake (plaintext) ---------
    rect rgb(255,250,205)
    Note over C,PS: SOCKS5 ハンドシェイク（平文）

    C->>PS: Greeting<br/>(VER=5, NMETHODS, METHODS…)
    PS-->>C: Method Selection<br/>(VER=5, METHOD)

    %% opt 認証が必要な場合
    %%   C->>S: 認証要求（例: Username/Password）
    %%   S-->>C: 認証応答（成功/失敗）
    %% end

    C->>PS: Request CONNECT<br/>(VER=5, CMD=1, RSV=0,<br/>ATYP=DOMAIN/IP, DST.ADDR, DST.PORT)

    rect rgb(255,120,0)
    PS->>TS: Greeting<br/>(VER=5, NMETHODS, METHODS…)
    TS-->>PS: Method Selection<br/>(VER=5, METHOD)

    PS->>TS: Forward Request CONNECT<br/>(VER=5, CMD=1, RSV=0,<br/>ATYP=DOMAIN/IP, DST.ADDR, DST.PORT)
    TS-->>PS: Reply<br/>(VER=5, REP=0=成功, BND.ADDR, BND.PORT)

    PS-->>C: Reply<br/>(VER=5, REP=0=成功, BND.ADDR, BND.PORT)
    
    end %% True Socks connection end

    end %% Root socks connetction end

    %% --------- Server connects to Target ---------
    TS->>T: TCP 接続 (DST.ADDR:DST.PORT)
    T-->>TS: 接続確立 (SYN/ACK)

    %% --------- TLS/HTTPS payload (encrypted) ---------
    rect rgb(220,235,255)
    Note over C,T: ここから上位プロトコル（TLS/HTTPS）のデータ転送（暗号化）

    C-)PS: TLS ClientHello<br/>(SNI が平文の場合あり / ECH 有効なら不可視)
    PS-)T: 転送
    T-)PS: TLS ServerHello ほか
    PS-)C: 転送

    C-->>PS: HTTPS Application Data（暗号化）
    PS-->>T: 転送
    T-->>PS: HTTPS Application Data（暗号化）
    PS-->>C: 転送
    end
```