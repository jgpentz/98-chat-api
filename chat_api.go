package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"
    "fmt"

	_ "github.com/lib/pq"
)

type Message struct {
    User string
    Msg string
    ChatRoom string
    Timestamp time.Time
}

func enableCors(w *http.ResponseWriter, req *http.Request) {
    (*w).Header().Set("Access-Control-Allow-Origin", "*")
    (*w).Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
    (*w).Header().Set("Access-Control-Allow-Headers", "Content-Type")
    if req.Method == "OPTIONS" {
        (*w).WriteHeader(http.StatusOK)
        return
    }
}

func writeMsgHandler (w http.ResponseWriter, req *http.Request) {
    enableCors(&w, req)

    // Decode the json message from the http body
    var message Message
    err := json.NewDecoder(req.Body).Decode(&message)
    if err != nil {
        http.Error(w, "Failed to decode JSON: "+err.Error(), http.StatusBadRequest)
        return
    }
    message.Timestamp = time.Now()
    fmt.Println(message)

    // Open a connection to the database
    connStr := "user=jpentz dbname=chat_api sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    err = db.Ping()
    if err != nil {
        log.Fatal(err)
    }

    // Insert message into the database
    _, err = db.Exec(
        "INSERT INTO messages (username, message, chat_room, timestamp) VALUES ($1, $2, $3, $4)", 
        message.User, 
        message.Msg, 
        message.ChatRoom,
        message.Timestamp,
    )
    if err != nil {
        fmt.Println(err)
        http.Error(w, "Database insert error", http.StatusInternalServerError)
        return
    }

    // Respond with success message
    w.WriteHeader(http.StatusCreated)
    w.Write([]byte("Message added to chat log"))
}


func readChatHandler (w http.ResponseWriter, req *http.Request) {
    var messages []Message

    enableCors(&w, req)

    // Parse query parameters from the request URL
    url_query := req.URL.Query()
    chatRoom := url_query.Get("chat_room")

    if chatRoom == "" {
        http.Error(w, "Chat room parameter missing", http.StatusBadRequest)
        return
    }

    // Open a connection to the db
    connStr := "user=jpentz dbname=chat_api sslmode=disable"
    db, err := sql.Open("postgres", connStr)
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    query := `
        SELECT username, message, timestamp
        FROM messages
        WHERE chat_room = $1
        ORDER BY timestamp DESC
        LIMIT 50;
    `

    rows, err := db.Query(query, chatRoom)
    if err != nil {
        log.Println(err)
    }
    defer rows.Close()

    for rows.Next() {
        var user, message string
        var timestamp time.Time
        if err := rows.Scan(&user, &message, &timestamp); err != nil {
            log.Fatal(err)
        }
        msg := Message{
            User: user,
            Msg: message,
            Timestamp: timestamp,
        }
        messages = append(messages, msg)
    }

    // Encode messages array as JSON and send in response
    w.Header().Set("Content-Type", "application/json")
    if err := json.NewEncoder(w).Encode(messages); err != nil {
        http.Error(w, "Error encoding JSON response", http.StatusInternalServerError)
        return
    }

    fmt.Println(messages)
}


func main() {
    http.HandleFunc("/write_msg", writeMsgHandler)
    http.HandleFunc("/read_chat", readChatHandler)
    log.Fatal(http.ListenAndServe(":8080", nil))
}
