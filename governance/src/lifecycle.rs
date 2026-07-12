use tokio::signal;

#[derive(Debug, Clone, Copy, PartialEq, Eq)]
pub enum ServiceState {
    Starting,
    Ready,
    Draining,
    Stopped,
}

#[derive(Debug, Default)]
pub struct GovernanceService;

impl GovernanceService {
    pub fn new() -> Self {
        Self
    }

    pub async fn run_until_shutdown(&self) -> Result<(), std::io::Error> {
        signal::ctrl_c().await
    }
}
