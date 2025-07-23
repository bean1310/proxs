use anyhow::Result;
use clap::{Parser, Subcommand};
// We will spawn the daemon executable rather than import its library
use hyper::{Body, Client, Method, Request};
use hyperlocal::{UnixConnector, Uri};
use serde_json::json;
use std::process::{Command, Stdio};

/// Proxy Switcher daemon CLI
#[derive(Parser, Debug)]
#[command(name = "pros", version, about, long_about = None)]
struct Cli {
    #[command(subcommand)]
    command: Commands,
}

#[derive(Subcommand, Debug)]
enum Commands {
    /// Run as a background macOS daemon
    Daemon,
    /// Show the list of attached SOCKS proxies
    Status,
    /// Manage proxy rules
    Rule {
        #[command(subcommand)]
        action: RuleAction,
    },
}

#[derive(Subcommand, Debug)]
enum RuleAction {
    /// Add a SOCKS proxy rule
    Add {
        /// Name of the server
        name: String,
        /// IP address of the server
        ipaddr: String,
        /// Sites to associate with the server
        sites: Vec<String>,
    },
    /// Delete a SOCKS proxy rule
    Del {
        /// Name of the server to delete
        name: String,
    },
}

#[tokio::main]
async fn main() -> Result<()> {
    let cli = Cli::parse();
    let client = Client::builder().build::<_, Body>(UnixConnector);

    match cli.command {
        Commands::Daemon => {
            // Spawn the daemon executable in the background
            let mut child = Command::new("proxsd")
                .stdout(Stdio::inherit())
                .stderr(Stdio::inherit())
                .spawn()?;
            println!("Daemon started (pid {})", child.id());
            return Ok(());
        }
        Commands::Status => {
            let uri: hyper::Uri = Uri::new("/tmp/pros.sock", "/status").into();
            let res = client
                .request(
                    Request::builder()
                        .method(Method::GET)
                        .uri(uri)
                        .body(Body::empty())?,
                )
                .await?;
            let body = hyper::body::to_bytes(res.into_body()).await?;
            println!("{}", String::from_utf8_lossy(&body));
        }
        Commands::Rule { action } => match action {
            RuleAction::Add {
                name,
                ipaddr,
                sites,
            } => {
                let uri: hyper::Uri = Uri::new("/tmp/pros.sock", "/rule").into();
                let json = json!({ "name": name, "ipaddr": ipaddr, "sites": sites });
                let req = Request::builder()
                    .method(Method::POST)
                    .uri(uri)
                    .header("content-type", "application/json")
                    .body(Body::from(json.to_string()))?;
                let res = client.request(req).await?;
                let body = hyper::body::to_bytes(res.into_body()).await?;
                println!("{}", String::from_utf8_lossy(&body));
            }
            RuleAction::Del { name } => {
                let path = format!("/rule/{}", name);
                let uri: hyper::Uri = Uri::new("/tmp/pros.sock", &path).into();
                let res = client
                    .request(
                        Request::builder()
                            .method(Method::DELETE)
                            .uri(uri)
                            .body(Body::empty())?,
                    )
                    .await?;
                let body = hyper::body::to_bytes(res.into_body()).await?;
                println!("{}", String::from_utf8_lossy(&body));
            }
        },
    }
    Ok(())
}
