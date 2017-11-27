package logs

import (
	"bufio"
	"errors"
	"fmt"
	ws "github.com/gorilla/websocket"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/pkg/api/v1"
	"kubernetes-realtime-logs/client"
)

type LogController struct {
	k8sClient *client.K8sClient
}

type LogMeta struct {
	Namespace     string            `json:"namespace"`
	Timestamps    bool              `json:"timestamps"`
	SinceSeconds  *int64            `json:"since"`
	TailLines     *int64            `json:"tail"`
	Selector      map[string]string `json:"selector,omitempty"`
	podName       string
	containerName string
}

type NotifyInfo struct {
	PodName       string `json:"pod"`
	ContainerName string `json:"container"`
	Log           string `json:"log"`
}

func NewLogController(c *client.K8sClient) *LogController {
	return &LogController{
		k8sClient: c,
	}
}

func (lc *LogController) ServeRequest(ws *ws.Conn) error {

	logMeta := LogMeta{
		SinceSeconds: newInt64(120),
		TailLines:    newInt64(100),
	}

	err := ws.ReadJSON(&logMeta)
	if err != nil {
		return errors.New("Failed to parse request message.")
	}

	listOptions := meta_v1.ListOptions{
		LabelSelector: labels.FormatLabels(logMeta.Selector),
	}

	pods, err := lc.k8sClient.ListPods(logMeta.Namespace, listOptions)
	if err != nil {
		return errors.New("Failed to list pods.")
	}

	if len(pods.Items) == 0 {
		return errors.New("Failed to find related pods.")
	}

	notifyChan := make(chan NotifyInfo)
	closeChanList := make([]chan bool, 0)
	for i := 0; i < len(pods.Items); i++ {
		pod := pods.Items[i]
		for j := 0; j < len(pod.Spec.Containers); j++ {
			logMeta.podName = pod.Name
			logMeta.containerName = pod.Spec.Containers[j].Name
			closeChan := make(chan bool)
			closeChanList = append(closeChanList, closeChan)
			go lc.listenToOnePod(logMeta, notifyChan, closeChan)
		}
	}

	go func() {
		for {
			_, _, err := ws.ReadMessage()
			if err != nil {
				fmt.Println("Conn is closed by client.")
				for i := 0; i < len(closeChanList); i++ {
					closeChanList[i] <- true
				}
				return
			}
		}
	}()

	for {
		select {
		case n := <-notifyChan:
			ws.WriteJSON(n)
		}
	}

	return nil
}

func (lc *LogController) listenToOnePod(logMeta LogMeta, notifyChan chan NotifyInfo, closeChan chan bool) {
	req := lc.k8sClient.WatchLogs(logMeta.Namespace, logMeta.podName, &v1.PodLogOptions{
		Follow:       true,
		Timestamps:   logMeta.Timestamps,
		Container:    logMeta.containerName,
		SinceSeconds: logMeta.SinceSeconds,
		TailLines:    logMeta.TailLines,
	})

	stream, err := req.Stream()
	if err != nil {
		fmt.Printf("Error opening stream to %s/%s: %s\n", logMeta.Namespace, logMeta.podName, logMeta.containerName)
		return
	}
	defer stream.Close()

	fmt.Printf("listening to %s/%s: %s\n", logMeta.Namespace, logMeta.podName, logMeta.containerName)
	go func() {
		finish := <-closeChan
		if finish {
			stream.Close()
		}
	}()

	reader := bufio.NewReader(stream)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Printf("return from %s/%s: %s\n", logMeta.Namespace, logMeta.podName, logMeta.containerName)
			return
		}
		notifyChan <- NotifyInfo{
			PodName:       logMeta.podName,
			ContainerName: logMeta.containerName,
			Log:           string(line),
		}
	}
}

func newInt64(val int64) *int64 {
	p := new(int64)
	*p = val
	return p
}
