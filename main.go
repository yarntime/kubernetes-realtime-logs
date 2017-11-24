package main

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/gorilla/websocket"
	"kubernetes-realtime-logs/client"
	"kubernetes-realtime-logs/logs"
	"net/http"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var logController *logs.LogController

func main() {

	k8sClient := client.NewK8sClient()

	logController = logs.NewLogController(k8sClient)

	http.HandleFunc("/websocket", handleConnections)

	fmt.Println("http server started on :8000")
	err := http.ListenAndServe("0.0.0.0:8000", nil)
	if err != nil {
		fmt.Printf("ListenAndServe: %v\n", err)
	}
}

func handleConnections(w http.ResponseWriter, r *http.Request) {

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		glog.Fatal(err)
	}

	defer ws.Close()

	logController.ServeRequest(ws)

}
