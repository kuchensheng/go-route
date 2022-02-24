package tracer

import (
	"bytes"
	"encoding/json"
	"github.com/rs/zerolog/log"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type jsonValue [2]string

type jsonStream struct {
	Stream map[string]string `json:"stream"`
	Values []jsonValue       `json:"values"`
}

type jsonMessage struct {
	Streams []jsonStream `json:"streams"`
}

type LokiClient struct {
	url            string
	endpoints      Endpoints
	currentMessage jsonMessage
	streams        chan *jsonStream
	quit           chan struct{}
	batchCounter   int
	maxBatch       int
	maxWaitTime    time.Duration
	wait           sync.WaitGroup
}

type Message struct {
	Message string
	Time    string
}

type Endpoints struct {
	push  string
	query string
	ready string
}

func (client *LokiClient) initEndpoints() {
	client.endpoints.push = "/loki/api/v1/push"
	client.endpoints.query = "/loki/api/v1/query"
	client.endpoints.ready = "/ready"
}

//IsReady Checks if the loki is ready
func (client *LokiClient) IsReady() bool {
	response, err := http.Get(client.url + client.endpoints.ready)
	return err == nil && response.StatusCode == 200
}

var httpClient http.Client

//CreateClient Creates a new loki client
// The client runs in a goroutine and sends the data either
// once it reaches the maxBatch or when it waited for maxWaitTime
//
// the batch counter is incremented every time add is called
// maxWaitTime uses nanoseconds
func CreateClient(url string, maxBatch int, maxWaitTime time.Duration) (*LokiClient, error) {
	client := LokiClient{
		url:         url,
		maxBatch:    maxBatch,
		maxWaitTime: maxWaitTime,
		quit:        make(chan struct{}),
		streams:     make(chan *jsonStream),
	}
	client.initEndpoints()
	//if !client.IsReady() {
	//	return &client, errors.New("The server on: " + url + "isn't ready.")
	//}

	client.wait.Add(1)
	hc := http.Client{
		Timeout: 1 * time.Second,
	}
	httpClient = hc
	go client.run()

	return &client, nil
}

func (client *LokiClient) Shutdown() {
	close(client.quit)
	client.wait.Wait()
}

func (client *LokiClient) run() {
	batchCounter := 0
	maxWait := time.NewTimer(client.maxWaitTime)

	defer func() {
		if batchCounter > 0 {
			client.send()
		}
		client.wait.Done()
	}()

	for {
		select {
		case <-client.quit:
			return
		case stream := <-client.streams:
			client.currentMessage.Streams =
				append(client.currentMessage.Streams, *stream)
			batchCounter++
			if batchCounter == client.maxBatch {
				client.send()
				batchCounter = 0
				client.currentMessage.Streams = []jsonStream{}
				maxWait.Reset(client.maxWaitTime)
			}
		case <-maxWait.C:
			if batchCounter > 0 {
				client.send()
				client.currentMessage.Streams = []jsonStream{}
				batchCounter = 0
			}
			maxWait.Reset(client.maxWaitTime)
		}
	}
}

// The template for the message sent to Loki is:
//{
//  "streams": [
//    {
//      "stream": {
//        "label": "value"
//      },
//      "values": [
//          [ "<unix epoch in nanoseconds>", "<log line>" ],
//          [ "<unix epoch in nanoseconds>", "<log line>" ]
//      ]
//    }
//  ]
//}

//AddStream Adds another stream to be sent with the next batch
func (client *LokiClient) AddStream(labels map[string]string, messages []Message) {
	var vals []jsonValue
	for i := range messages {
		var val jsonValue
		val[0] = messages[i].Time
		val[1] = messages[i].Message
		vals = append(vals, val)
	}
	stream := &jsonStream{
		Stream: labels,
		Values: vals,
	}
	client.streams <- stream
}

//send Encodes the messages and sends them to loki
func (client *LokiClient) send() error {
	//log.Info().Msgf("执行发送任务")
	length := len(client.currentMessage.Streams)
	if length == 0 {
		return nil
	}
	defer func() {
		client.currentMessage.Streams = []jsonStream{}
	}()
	str, err := json.Marshal(client.currentMessage)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", client.url+client.endpoints.push, bytes.NewReader(str))
	if err != nil {
		return err
	}
	//req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Connection", "keep-alive")
	response, err := httpClient.Do(req)
	if err != nil {
		return err
	} else if response != nil && response.StatusCode != 204 {
		log.Warn().Msgf("上传条数[%d],请求头信息：%v", req)
		return err
	}
	return nil
}

type returnedJSON struct {
	Status interface{}
	Data   struct {
		ResultType string
		Result     []struct {
			Stream interface{}
			Values [][]string
		}
		Stats interface{}
	}
}

//Query Queries the server. The queryString is expected to be in the
// LogQL format described here:
// https://github.com/grafana/loki/blob/master/docs/logql.md
func (client *LokiClient) Query(queryString string) ([]Message, error) {
	response, err := http.Get(client.url + client.endpoints.query + "?query=" + queryString)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return []Message{}, err
	}
	var answer returnedJSON
	json.Unmarshal(body, &answer)
	var values []Message
	for i := range answer.Data.Result {
		for j := range answer.Data.Result[i].Values {
			msg := Message{
				Time:    answer.Data.Result[i].Values[j][0],
				Message: answer.Data.Result[i].Values[j][1],
			}
			values = append(values, msg)
		}
	}
	return values, nil
}
