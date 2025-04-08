package roomhandler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	aireview "github.com/Xavanion/Hack-KU-2025/backend/ai-review"
	codehandler "github.com/Xavanion/Hack-KU-2025/backend/code-handling"
	"github.com/gorilla/websocket"
)

type Room struct {
	ID                string
	activeConnections map[*websocket.Conn]bool
	con_mu            sync.Mutex
	mainText          []byte
	text_mu           sync.Mutex
}
type RoomManager struct {
	Rooms map[string]*Room
	mu    sync.RWMutex
}
type ApiRequest struct {
	Event    string `json:"event"`
	Language string `json:"language"`
	Room     string `json:"room"`
}

type sendUpdateJson struct {
	Event  string      `json:"event"`
	Update interface{} `json:"update"`
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		Rooms: make(map[string]*Room),
	}
}

func (manager *RoomManager) CreateRoom(roomid string) *Room {
	manager.mu.Lock()
	defer manager.mu.Unlock()
	manager.Rooms[roomid] = &Room{
		ID:                roomid,
		activeConnections: make(map[*websocket.Conn]bool),
		mainText:          make([]byte, 0),
	}
	return manager.Rooms[roomid]
}

func (manager *RoomManager) GetRoom(id string) (*Room, bool) {
	manager.mu.RLock()
	defer manager.mu.RUnlock()
	if manager.Rooms[id] == nil {
		return nil, false
	} else {
		room := manager.Rooms[id]
		return room, true
	}
}

func (room *Room) FileApiRequest(requestData ApiRequest, c *gin.Context) {
	switch requestData.Event {
	case "run_code":
		//codehandler.Run_file("1", "Python", "main", "print(\"Hello World\")\n")
		/*codehandler.Run_file("1", "C", "main", `
		#include <stdio.h>

		int main(){
		    printf("Hello World");
		    return 0;
		}`)*/
		out := codehandler.Run_file("one", string(requestData.Language), "main-", string(room.mainText))
		room.broadcastUpdate(nil, "output_update", out, false)
		fmt.Printf("Output: %s\n", out)
		c.JSON(http.StatusOK, gin.H{"message": "Data processed successfully"})
	case "code_save":
	case "code_review":
		response, err := aireview.Gemini_Request(string(room.mainText))
		if err != nil {
			fmt.Println(err)
			err_out := "internal server error"
			c.JSON(http.StatusInternalServerError, gin.H{"review": err_out})
		}
		c.JSON(http.StatusOK, gin.H{"review": response})
	}
}

func (room *Room) broadcastUpdate(startconn *websocket.Conn, event string, message string, isParsed bool) {
	room.con_mu.Lock()
	defer room.con_mu.Unlock()
	//fmt.Println(room.activeConnections)
	for conn := range room.activeConnections {
		//fmt.Println("Sending message to:", conn.RemoteAddr())
		if conn == startconn {
			continue
		}
		var jsonData []byte
		var err error
		if isParsed {
			var parsed map[string]interface{}
			json.Unmarshal([]byte(message), &parsed)

			msg := sendUpdateJson{
				Event:  event,
				Update: parsed,
			}
			jsonData, err = json.Marshal(msg)
		} else {
			msg := sendUpdateJson{
				Event:  event,
				Update: message,
			}
			jsonData, err = json.Marshal(msg)
		}

		if err != nil {
			log.Println("Failed to marshall update message json: ", err)
		}

		if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
			log.Println("Failed to send message ", err)
			conn.Close()                         // Close connection if it fails to send a message
			delete(room.activeConnections, conn) // Remove broken connection
		}
	}
}

func (room *Room) handleMessages(message string, conn *websocket.Conn) {
	// Turn the raw text back into a usable type
	var json_mess map[string]any
	json.Unmarshal([]byte(message), &json_mess)
	//fmt.Println(json_mess)
	//fmt.Println(message)
	switch json_mess["event"] {
	case "text_update":
		if json_mess["type"] == "insert" {
			//fmt.Println(json_mess["value"].(string))
			position := int(json_mess["pos"].(float64))
			if position > len(room.mainText) {
				position -= 1
			}
			room.insertBytes(position, []byte(json_mess["value"].(string)))
		} else if json_mess["type"] == "delete" {
			from := int(json_mess["from"].(float64))
			to := int(json_mess["to"].(float64))
			room.deleteByte(from, to-from)
		}
		room.broadcastUpdate(conn, "input_update", message, true)
	default:
		log.Print("Invalid json event")
	}
	fmt.Printf("Body:%s\n", string(room.mainText))
}

func (room *Room) NewConnection(conn *websocket.Conn) {
	// update our activeConnections so we can message persistently
	room.con_mu.Lock()
	room.activeConnections[conn] = true
	room.con_mu.Unlock()
	defer conn.Close()

	time.Sleep(time.Duration(500) * time.Millisecond)
	msg := sendUpdateJson{
		Event:  "connection_update",
		Update: string(room.mainText),
	}
	jsonData, err := json.Marshal(msg)

	if err != nil {
		log.Println("Failed to marshall update message json: ", err)
	}

	// Catch the new connection up with what's going on

	if err := conn.WriteMessage(websocket.TextMessage, jsonData); err != nil {
		conn.Close()                         // Close connection if it fails to send a message
		delete(room.activeConnections, conn) // Remove broken connection
	}

	// Listen for incoming messages from this specific connection
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Println("Error reading message:", err)
			break
		}
		room.handleMessages(string(message), conn)
	}

	// Once the connection is closed, remove it from the active connections map
	room.con_mu.Lock()
	delete(room.activeConnections, conn)
	room.con_mu.Unlock()
}

func (room *Room) insertBytes(index int, value []byte) {
	slice := room.mainText
	room.text_mu.Lock()
	defer room.text_mu.Unlock()
	// Ensure index is valid
	if index < 0 || index > len(slice) {
		log.Println("Index out of range")
		return
	}

	// Insert the byte at the given index
	room.mainText = append(slice[:index], append(value, slice[index:]...)...)
}

func (room *Room) deleteByte(index int, num_chars int) {
	slice := room.mainText
	room.text_mu.Lock()
	defer room.text_mu.Unlock()

	// Ensure index is valid
	if index < 0 || index > len(slice) {
		fmt.Println("Index out of range")
		return
	}

	// Remove the byte at the given index
	room.mainText = append(slice[:index], slice[index+num_chars:]...)
}
