use anyhow::Result;
use daemon::run_daemon;

fn main() -> Result<()> {
    // Start the background daemon (HTTP server over UDS)
    println!("Starting Proxy Switcher daemon...");
    run_daemon()
}
