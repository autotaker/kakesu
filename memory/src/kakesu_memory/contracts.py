"""Framework-neutral contracts used at the Memory Plane boundary."""

from dataclasses import dataclass
from typing import Any, Protocol


@dataclass(frozen=True, slots=True)
class EpisodeJobInput:
    job_id: str
    task_id: str
    input_digest: str
    schema_version: str
    max_steps: int
    payload: dict[str, Any]


@dataclass(frozen=True, slots=True)
class EpisodeJobResult:
    job_id: str
    task_id: str
    output_digest: str
    episode: dict[str, Any]


class EpisodeRunner(Protocol):
    async def run(self, job: EpisodeJobInput) -> EpisodeJobResult: ...
