use anyhow::{Context, Result};
use daemonize::Daemonize;
use serde::{Deserialize, Serialize};
use serde_json;
use std::{collections::HashMap, env, fs, path::PathBuf};

/// Global configuration
#[derive(Debug, Deserialize, Serialize)]
pub struct Global {
    pub start_port: u16,
    pub ports: u16,
    pub identity_file: String,
}

/// Server entry in configuration
#[derive(Debug, Deserialize, Serialize)]
pub struct Server {
    pub ipaddr: String,
    pub sites: Vec<String>,
}

/// Top-level configuration
#[derive(Debug, Deserialize, Serialize)]
pub struct Config {
    pub global: Global,
    pub servers: HashMap<String, Server>,
}

/// Return the path to the pros config.toml file.
pub fn get_config_path() -> PathBuf {
    let config_dir = env::var("XDG_CONFIG_HOME")
        .map(PathBuf::from)
        .unwrap_or_else(|_| {
            let home = env::var("HOME").expect("HOME environment variable not set");
            PathBuf::from(home).join(".config")
        });
    config_dir.join("proxs").join("config.toml")
}

/// Load configuration from disk
pub fn load_config() -> Result<Config> {
    let path = get_config_path();
    let content = fs::read_to_string(&path)
        .with_context(|| format!("failed to read config file {:?}", &path))?;
    let cfg: Config = toml::from_str(&content)
        .with_context(|| format!("failed to parse TOML from {:?}", &path))?;
    Ok(cfg)
}

/// Daemon main loop stub
pub fn run_daemon() -> Result<()> {
    let stdout = fs::File::create("/tmp/pros.out")?;
    let stderr = fs::File::create("/tmp/pros.err")?;
    Daemonize::new().stdout(stdout).stderr(stderr).start()?;

    // Start HTTP server over UNIX socket
    println!("Starting Proxy Switcher daemon...");
    let rt = tokio::runtime::Builder::new_current_thread()
        .enable_all()
        .build()?;
    rt.block_on(async move { serve().await.map_err(|e| anyhow::anyhow!(e)) })?;
    Ok(())
}

use hyper::service::{make_service_fn, service_fn};
use hyper::{Body, Method, Request, Response, Server as HyperServer, StatusCode};
use hyperlocal::UnixServerExt;
use std::convert::Infallible;

async fn serve() -> Result<()> {
    let socket = "/tmp/pros.sock";
    let _ = fs::remove_file(socket);
    let make_svc = make_service_fn(|_| async { Ok::<_, Infallible>(service_fn(handle)) });
    HyperServer::bind_unix(socket)?.serve(make_svc).await?;
    Ok(())
}

async fn handle(req: Request<Body>) -> Result<Response<Body>, Infallible> {
    let resp = match (req.method(), req.uri().path()) {
        (&Method::GET, "/status") => {
            println!("Received status request");
            let cfg = load_config().unwrap();
            let list: Vec<&String> = cfg.servers.keys().collect();
            let body = serde_json::to_string(&list).unwrap();
            Response::new(Body::from(body))
        }
        (&Method::POST, "/rule") => {
            let bytes = hyper::body::to_bytes(req.into_body()).await.unwrap();
            let v: serde_json::Value = serde_json::from_slice(&bytes).unwrap();
            let name = v["name"].as_str().unwrap();
            let ipaddr = v["ipaddr"].as_str().unwrap();
            let sites = v["sites"]
                .as_array()
                .unwrap()
                .iter()
                .map(|v| v.as_str().unwrap().to_string())
                .collect();
            let mut cfg = load_config().unwrap();
            cfg.servers.insert(
                name.to_string(),
                Server {
                    ipaddr: ipaddr.to_string(),
                    sites,
                },
            );
            let toml = toml::to_string_pretty(&cfg).unwrap();
            fs::write(get_config_path(), toml).unwrap();
            Response::new(Body::from(format!("Added rule {}", name)))
        }
        (&Method::DELETE, path) if path.starts_with("/rule/") => {
            let name = &path[6..];
            let mut cfg = load_config().unwrap();
            if cfg.servers.remove(name).is_some() {
                let toml = toml::to_string_pretty(&cfg).unwrap();
                fs::write(get_config_path(), toml).unwrap();
                Response::new(Body::from(format!("Deleted rule {}", name)))
            } else {
                Response::builder()
                    .status(StatusCode::NOT_FOUND)
                    .body(Body::from(format!("No such rule '{}'", name)))
                    .unwrap()
            }
        }
        (&Method::GET, "/daemon") => Response::new(Body::from("OK")),
        _ => Response::builder()
            .status(StatusCode::NOT_FOUND)
            .body(Body::from("Not foundppp"))
            .unwrap(),
    };
    Ok(resp)
}
