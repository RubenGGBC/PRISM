package api

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/ruffini/prism/graph"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for dev
	},
}

// WSServer manages WebSocket connections
type WSServer struct {
	clients   map[*websocket.Conn]bool
	broadcast chan map[string]interface{}
	mutex     sync.Mutex
	graph     *graph.CodeGraph
}

// NewWSServer creates a new WebSocket server
func NewWSServer(g *graph.CodeGraph) *WSServer {
	return &WSServer{
		clients:   make(map[*websocket.Conn]bool),
		broadcast: make(chan map[string]interface{}),
		graph:     g,
	}
}

// HandleWSConnection handles a new WebSocket connection
func (ws *WSServer) HandleWSConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	ws.mutex.Lock()
	ws.clients[conn] = true
	ws.mutex.Unlock()

	for {
		var msg map[string]interface{}
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				fmt.Printf("WebSocket error: %v\n", err)
			}
			ws.mutex.Lock()
			delete(ws.clients, conn)
			ws.mutex.Unlock()
			break
		}

		// Process message (e.g., file selection, search query)
		response := ws.handleMessage(msg)
		conn.WriteJSON(response)
	}
}

// handleMessage processes incoming WebSocket messages
func (ws *WSServer) handleMessage(msg map[string]interface{}) map[string]interface{} {
	action, _ := msg["action"].(string)

	switch action {
	case "get_file":
		file, _ := msg["file"].(string)
		nodes, err := ws.graph.GetNodesByFile(file)
		if err != nil {
			return map[string]interface{}{
				"type": "error",
				"msg":  err.Error(),
			}
		}
		return map[string]interface{}{
			"type": "file_data",
			"data": nodes,
		}
	case "search":
		query, _ := msg["query"].(string)
		nodes, err := ws.graph.SearchByName(query)
		if err != nil {
			return map[string]interface{}{
				"type": "error",
				"msg":  err.Error(),
			}
		}
		return map[string]interface{}{
			"type": "search_results",
			"data": nodes,
		}
	default:
		return map[string]interface{}{
			"type": "error",
			"msg":  "unknown action",
		}
	}
}

// Broadcast sends message to all connected clients
func (ws *WSServer) Broadcast(msg map[string]interface{}) {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()

	for conn := range ws.clients {
		err := conn.WriteJSON(msg)
		if err != nil {
			delete(ws.clients, conn)
		}
	}
}
