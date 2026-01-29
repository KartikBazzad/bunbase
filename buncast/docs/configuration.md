# Buncast Configuration

## Command-line flags

| Flag    | Default           | Description                                                  |
| ------- | ----------------- | ------------------------------------------------------------ |
| -socket | /tmp/buncast.sock | Unix socket path for IPC                                     |
| -http   | :8081             | HTTP listen address (health, topics, SSE). Use "" to disable |
| -debug  | false             | Enable debug request/flow logging                            |
| -config | (empty)           | Config file path (not yet implemented)                       |

## IPC

- **Socket path**: Must be a path where the process can create a Unix socket. Existing file at that path is removed on start.
- **Max connections**: Currently unlimited (config option for future).

## HTTP

- **Listen address**: Default `:8081`. Set to `""` to run without HTTP (IPC only).
- **Read/Write timeouts**: Configured in code (e.g. 10s read); SSE connections have no write timeout so long-lived subscribe works.

## Limits

- **Topic name length**: 1024 bytes (MaxTopicLen in protocol).
- **Message/frame size**: 16 MiB (MaxFrameSize).
