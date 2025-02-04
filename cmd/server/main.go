package main

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "sync"
    "time"

    "github.com/google/uuid"
    "github.com/gorilla/websocket"
)

const (
    writeWait = 10 * time.Second
    
    pongWait = 60 * time.Second
    
    pingPeriod = (pongWait * 9) / 10
    
    maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool { return true },
}

type Client struct {
    ID       string
    conn     *websocket.Conn
    hub      *Hub
    send     chan []byte
    mu       sync.Mutex
}

type Message struct {
    ID      string `json:"id"`
    Message string `json:"message"`
}

type Hub struct {
    clients    map[string]*Client
    register   chan *Client
    unregister chan *Client
    mu         sync.RWMutex
}

func NewHub() *Hub {
    return &Hub{
        clients:    make(map[string]*Client),
        register:   make(chan *Client),
        unregister: make(chan *Client),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.clients[client.ID] = client
            
            welcomeMsg := fmt.Sprintf("Welcome! Your ID is: %s", client.ID)
            client.send <- []byte(welcomeMsg)
            
            var connectedClients []string
            for id := range h.clients {
                if id != client.ID {
                    connectedClients = append(connectedClients, id)
                }
            }
            
            if len(connectedClients) > 0 {
                clientList := fmt.Sprintf("Connected clients: %v", connectedClients)
                client.send <- []byte(clientList)
            }
            h.mu.Unlock()

        case client := <-h.unregister:
            h.mu.Lock()
            if _, ok := h.clients[client.ID]; ok {
                delete(h.clients, client.ID)
                close(client.send)
            }
            h.mu.Unlock()
        }
    }
}

func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        log.Println("Error during connection upgrade:", err)
        return
    }

    clientID := uuid.New().String()
    log.Printf("New client connected. ID: %s", clientID)

    client := &Client{
        ID:   clientID,
        hub:  hub,
        conn: conn,
        send: make(chan []byte, 256),
    }
    client.hub.register <- client

    // Start the read and write pumps
    go client.writePump()
    go client.readPump()
}

func (c *Client) readPump() {
    defer func() {
        log.Printf("Client disconnected. ID: %s", c.ID)
        c.hub.unregister <- c
        c.conn.Close()
    }()

    c.conn.SetReadLimit(maxMessageSize)
    c.conn.SetReadDeadline(time.Now().Add(pongWait))
    c.conn.SetPongHandler(func(string) error {
        log.Printf("Received pong from client %s", c.ID)
        c.conn.SetReadDeadline(time.Now().Add(pongWait))
        return nil
    })

    for {
        _, message, err := c.conn.ReadMessage()
        if err != nil {
            if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
                log.Printf("Error reading message from client %s: %v", c.ID, err)
            }
            break
        }

        var msg Message
        if err := json.Unmarshal(message, &msg); err != nil {
            log.Printf("Error parsing message from client %s: %v", c.ID, err)
            continue
        }

        log.Printf("Received message from client %s to client %s: %s", c.ID, msg.ID, msg.Message)

        c.hub.mu.RLock()
        if targetClient, ok := c.hub.clients[msg.ID]; ok {
            select {
            case targetClient.send <- message:
                log.Printf("Message forwarded to client %s", msg.ID)
            default:
                log.Printf("Failed to forward message to client %s", msg.ID)
                close(targetClient.send)
                delete(c.hub.clients, targetClient.ID)
            }
        } else {
            log.Printf("Target client %s not found", msg.ID)
        }
        c.hub.mu.RUnlock()
    }
}

func (c *Client) writePump() {
    ticker := time.NewTicker(pingPeriod)
    defer func() {
        ticker.Stop()
        c.conn.Close()
    }()

    for {
        select {
        case message, ok := <-c.send:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            if !ok {
                log.Printf("Client %s send channel closed", c.ID)
                c.conn.WriteMessage(websocket.CloseMessage, []byte{})
                return
            }

            w, err := c.conn.NextWriter(websocket.TextMessage)
            if err != nil {
                log.Printf("Error getting writer for client %s: %v", c.ID, err)
                return
            }
            w.Write(message)

            if err := w.Close(); err != nil {
                log.Printf("Error closing writer for client %s: %v", c.ID, err)
                return
            }
        case <-ticker.C:
            c.conn.SetWriteDeadline(time.Now().Add(writeWait))
            log.Printf("Sending ping to client %s", c.ID)
            if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
                log.Printf("Error sending ping to client %s: %v", c.ID, err)
                return
            }
        }
    }
}

func main() {
    log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
    log.Println("Initializing WebSocket server...")
    
    hub := NewHub()
    go hub.Run()

    http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
        log.Printf("Received connection request from %s", r.RemoteAddr)
        serveWs(hub, w, r)
    })

    log.Println("Server starting on :8080")
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        log.Fatal("ListenAndServe: ", err)
    }
}