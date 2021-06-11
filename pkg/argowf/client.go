package argowf

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/sktelecom/tks-contract/pkg/log"
)

// Client is
type Client struct {
	client *http.Client
	url    string
}

// New
func New(host string, port int, ssl bool, token string) (*Client, error) {
	var baseUrl string
	if ssl {
		if token == "" {
			return nil, fmt.Errorf("argo ssl enabled but token is empty.")
		}
		baseUrl = fmt.Sprintf("https://%s:%d", host, port)
	} else {
		baseUrl = fmt.Sprintf("http://%s:%d", host, port)
	}
	return &Client{
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns: 10,
			},
		},
		url: baseUrl,
	}, nil
}

func (c Client) GetWorkflowTemplates(namespace string) (*GetWorkflowTemplatesResponse, error) {
	res, err := http.Get(fmt.Sprintf("%s/api/v1/workflow-templates/%s", c.url, namespace))
	if err != nil && res.StatusCode != 200 {
		log.Fatal("error from get workflow-templats return code: ", res.StatusCode)
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Error("error closing http body")
		}
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	wftplRes := GetWorkflowTemplatesResponse{}
	if err := json.Unmarshal(body, &wftplRes); err != nil {
		log.Error("an error was unexpected while parsing response from api /workflow template.")
		return nil, err
	}
	return &wftplRes, nil
}

func (c Client) SumbitWorkflowFromWftpl(wftplName, targetNamespace string,
	opts SubmitOptions, params []string) (*SubmitWorkflowResponse, error) {
	reqBody := submitWorkflowRequestBody{
		Namespace:     targetNamespace,
		ResourceKind:  "WorkflowTemplate",
		ResourceName:  wftplName,
		Parameters:    params,
		SubmitOptions: opts,
	}
	reqBodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil,
			fmt.Errorf("an error was unexpected while marshaling request body")
	}
	buff := bytes.NewBuffer(reqBodyBytes)
	res, err := http.Post(fmt.Sprintf("%s/api/v1/workflows/%s/submit", c.url, targetNamespace),
		"application/json", buff)

	if err != nil || res.StatusCode != 200 {
		log.Fatal("error from post workflow. return code: ", res.StatusCode)
		log.Fatal("error message ", err.Error())
		return nil, err
	}
	defer func() {
		if err := res.Body.Close(); err != nil {
			log.Error("error closing http body")
		}
	}()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	submitRes := SubmitWorkflowResponse{}
	if err := json.Unmarshal(body, &submitRes); err != nil {
		log.Error("an error was unexpected while parsing response from api /submit.")
		return nil, err
	}
	return &submitRes, nil
}
