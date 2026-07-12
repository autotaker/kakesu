use kakesu_governance::lifecycle::GovernanceService;

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    let service = GovernanceService::new();
    service.run_until_shutdown().await?;
    Ok(())
}
