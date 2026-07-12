import asyncio

from kakesu_memory import EpisodeJobInput, EpisodeJobResult, MemoryService


class FakeRunner:
    async def run(self, job: EpisodeJobInput) -> EpisodeJobResult:
        return EpisodeJobResult(
            job_id=job.job_id,
            task_id=job.task_id,
            output_digest="sha256:result",
            episode={"task_id": job.task_id},
        )


def test_compile_episode_delegates_to_ephemeral_runner() -> None:
    service = MemoryService(FakeRunner())
    job = EpisodeJobInput(
        job_id="job-1",
        task_id="task-1",
        input_digest="sha256:input",
        schema_version="draft-v0",
        max_steps=8,
        payload={"task_id": "task-1"},
    )

    result = asyncio.run(service.compile_episode(job))

    assert result.task_id == "task-1"
    assert result.episode == {"task_id": "task-1"}


def test_compile_episode_rejects_invalid_budget() -> None:
    service = MemoryService(FakeRunner())
    job = EpisodeJobInput(
        job_id="job-1",
        task_id="task-1",
        input_digest="sha256:input",
        schema_version="draft-v0",
        max_steps=0,
        payload={},
    )

    try:
        asyncio.run(service.compile_episode(job))
    except ValueError as error:
        assert str(error) == "max_steps must be positive"
    else:
        raise AssertionError("expected invalid max_steps to fail")
