# WebSocket Chat Server

A Golang-based WebSocket server implementation that supports multiple client connections with messaging capabilities.


## Prerequisites

- Go 1.20 or higher
- [gorilla/websocket](https://github.com/gorilla/websocket) package

## Dependencies

```bash
go get github.com/gorilla/websocket
```

## Project Structure

```
.
├── Dockerfile
├── README.md
├── cmd
│   └── server
│       └── main.go
├── go.mod
├── go.sum
├── internal
│   └── logger
│       └── logger.go
├── server
└── test_websocket.py
```

## Installation

- git clone `https://github.com/purav10/Golang-Server`
- cd `Golang-Server`
- go mod tidy

## Building the Server

```bash
go build -o server cmd/server/main.go
```

## Running the Server

```go run cmd/server/main.go```  OR  ```./server```

## Message Format       

```json
{
    "id": "recipient-id",
    "message": "your message"
}
```

## Docker Commands

```bash
# Build the Docker image
docker build -t websocket-chat-server .

# Run the container
docker run -p 8080:8080 websocket-chat-server
``` 

## Test Client

```python
python3 test_websocket.py
```