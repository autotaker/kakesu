"""Memory Service orchestration without framework-owned durable sessions."""

from .contracts import EpisodeJobInput, EpisodeJobResult, EpisodeRunner


class MemoryService:
    def __init__(self, runner: EpisodeRunner) -> None:
        self._runner = runner

    async def compile_episode(self, job: EpisodeJobInput) -> EpisodeJobResult:
        if not job.job_id or not job.task_id or not job.input_digest:
            raise ValueError("job identity and input digest are required")
        if job.max_steps < 1:
            raise ValueError("max_steps must be positive")
        return await self._runner.run(job)
