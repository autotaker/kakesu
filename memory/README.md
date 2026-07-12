# Memory Plane scaffold

Python service using the OpenAI Agents SDK as an ephemeral tool runner.

The SDK does not own durable sessions, retries, or checkpoints. `evidence.db` will own Episode jobs, Evidence, FTS, and Memory Inbox/Outbox; the Go Core communicates through versioned messages over a Unix domain socket.
