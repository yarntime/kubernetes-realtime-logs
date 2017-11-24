package logs

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	ws "github.com/gorilla/websocket"
	"k8s.io/client-go/pkg/api/v1"
	"kubernetes-realtime-logs/client"
)

type LogController struct {
	k8sClient *client.K8sClient
}

type LogMeta struct {
	Namespace     string `json:"namespace"`
	PodName       string `json:"pod"`
	ContainerName string `json:"container"`
	Timestamps    bool   `json:"timestamps"`
	SinceSeconds  *int64 `json:"since"`
	TailLines     *int64 `json:"tail"`
}

func NewLogController(c *client.K8sClient) *LogController {
	return &LogController{
		k8sClient: c,
	}
}

func (lc *LogController) ServeRequest(ws *ws.Conn) error {

	var logMeta LogMeta

	err := ws.ReadJSON(&logMeta)
	if err != nil {
		fmt.Printf("Failed to parse request message.%v\n", err)
		return errors.New("Failed to parse request message.")
	}

	if logMeta.SinceSeconds == nil {
		logMeta.SinceSeconds = newInt64(120)
	}

	if logMeta.TailLines == nil {
		logMeta.TailLines = newInt64(100)
	}

	req := lc.k8sClient.WatchLogs(logMeta.Namespace, logMeta.PodName, &v1.PodLogOptions{
		Follow:       true,
		Timestamps:   logMeta.Timestamps,
		Container:    logMeta.ContainerName,
		SinceSeconds: logMeta.SinceSeconds,
		TailLines:    logMeta.TailLines,
	})

	stream, err := req.Stream()
	if err != nil {
		fmt.Printf("Error opening stream to %s/%s: %s\n", logMeta.Namespace, logMeta.PodName, logMeta.ContainerName)
		return errors.New(fmt.Sprintf("Error opening stream to %s/%s: %s\n", logMeta.Namespace, logMeta.PodName, logMeta.ContainerName))
	}
	defer stream.Close()

	closeChan := make(chan int)
	defer close(closeChan)

	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				fmt.Println("Conn is closed by client.")
				stream.Close()
				return
			}
		}
	}()

	reader := bufio.NewReader(stream)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			ws.Close()
			return nil
		}
		err = ws.WriteMessage(websocket.TextMessage, line)
		if err != nil {
			return nil
		}
	}
	return nil
}

func newInt64(val int64) *int64 {
	p := new(int64)
	*p = val
	return p
}
