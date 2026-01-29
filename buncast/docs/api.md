# Buncast API

## IPC (Unix socket)

Framing: 4-byte little-endian length prefix, then body.

### Request

- **Body**: RequestID (8 bytes) + Command (1 byte) + PayloadLen (4 bytes) + Payload.

| Command     | Code | Payload                                 | Description        |
| ----------- | ---- | --------------------------------------- | ------------------ |
| CreateTopic | 1    | TopicLen(2)+Topic                       | Create topic       |
| DeleteTopic | 2    | TopicLen(2)+Topic                       | Delete topic       |
| ListTopics  | 3    | (empty)                                 | List topic names   |
| Publish     | 4    | TopicLen(2)+Topic+PayloadLen(4)+Payload | Publish message    |
| Subscribe   | 5    | TopicLen(2)+Topic                       | Subscribe (stream) |

### Response (for commands 1–4)

- **Body**: RequestID (8) + Status (1) + PayloadLen (4) + Payload.
- **Status**: 0 = OK, 1 = Error. Error payload is JSON `{"error":"..."}`.
- **ListTopics** success payload: JSON array of strings.

### Subscribe (command 5)

After the OK response, the server streams **message frames** on the same connection. Each frame:

- Length (4 bytes) + TopicLen (2) + Topic + PayloadLen (4) + Payload.

Client closes the connection to unsubscribe.

## Go client

```go
import "github.com/kartikbazzad/bunbase/buncast/pkg/client"

c := client.New("/tmp/buncast.sock")
defer c.Close()

c.CreateTopic("my-topic")
c.Publish("my-topic", []byte(`{"event":"deploy"}`))
topics, _ := c.ListTopics()

c.Subscribe("my-topic", func(msg *client.Message) error {
    // handle msg.Topic, msg.Payload
    return nil
})
```

## HTTP

| Method | Path                  | Description                      |
| ------ | --------------------- | -------------------------------- |
| GET    | /health               | Returns `{"status":"ok"}`        |
| GET    | /topics               | Returns JSON array of topics     |
| GET    | /subscribe?topic=NAME | SSE stream of messages for topic |

### SSE (/subscribe)

- Query parameter **topic** (required): topic name.
- Response: `Content-Type: text/event-stream`. Each message is sent as an SSE event: `data: <payload>\n\n`.
- Client disconnect unsubscribes.
