"""OpenAI Agents SDK adapter.

The SDK run is intentionally ephemeral. Job lease/retry and all durable state
belong to the Memory Plane store rather than an SDK session or trace.
"""

import hashlib
import json
from collections.abc import Callable
from typing import Any

from .contracts import EpisodeJobInput, EpisodeJobResult


class OpenAIEpisodeRunner:
    def __init__(
        self,
        *,
        instructions: str,
        query_evidence: Callable[..., Any],
        output_type: type[Any],
    ) -> None:
        self._instructions = instructions
        self._query_evidence = query_evidence
        self._output_type = output_type

    async def run(self, job: EpisodeJobInput) -> EpisodeJobResult:
        from agents import Agent, Runner, function_tool, set_tracing_disabled

        set_tracing_disabled(True)
        evidence_tool = function_tool(self._query_evidence)
        agent = Agent(
            name="episode_compiler",
            instructions=self._instructions,
            tools=[evidence_tool],
            output_type=self._output_type,
        )
        result = await Runner.run(
            agent,
            input=json.dumps(job.payload, ensure_ascii=False, sort_keys=True),
            max_turns=job.max_steps,
        )
        episode = self._normalize_output(result.final_output)
        encoded = json.dumps(episode, ensure_ascii=False, sort_keys=True).encode()
        return EpisodeJobResult(
            job_id=job.job_id,
            task_id=job.task_id,
            output_digest=f"sha256:{hashlib.sha256(encoded).hexdigest()}",
            episode=episode,
        )

    @staticmethod
    def _normalize_output(output: Any) -> dict[str, Any]:
        if hasattr(output, "model_dump"):
            return dict(output.model_dump(mode="json"))
        if isinstance(output, dict):
            return output
        raise TypeError("episode output must be a mapping or Pydantic model")
