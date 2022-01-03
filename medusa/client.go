package medusa

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

var medusa = flag.String("medusaUrl", "http://192.168.1.216:8081/sickrage", "Medusa URL ie. <protocol>://<url>:<port>/<path>")
var token = flag.String("token", "apikey", "Medusa API Token")
var target = flag.Int("target", 6, "Target Id for status, ex: 6")
var current = flag.Int("current", 4, "Target Id for status, ex: 4")

// MedusaApi interface for interacting with medusa
type MedusaApi interface {
	getTVShowsWithDownloadedEpStatus() ([]medusaUpdateStatusShowsPayload, error)
	changeEpisodeStatus(payload medusaUpdateStatusPayload) error
	process() error
	getStatusMap() (map[string]string, error)
}

type httpClient struct {
	client http.Client
}

func newMedusaApi() MedusaApi {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := http.Client{Transport: http.DefaultTransport.(*http.Transport), Timeout: time.Second * 10}
	return &httpClient{client}
}

type medusaEpisode struct {
	season  string
	episode string
}

type medusaSeries struct {
	name string
	id   string
}

type medusaUpdateStatusPayload struct {
	Status int                              `json:"status"`
	Shows  []medusaUpdateStatusShowsPayload `json:"shows"`
}

type medusaUpdateStatusShowsPayload struct {
	Slug     string   `json:"slug"`
	Episodes []string `json:"episodes"`
}

func (med *httpClient) getTVShowsWithDownloadedEpStatus() ([]medusaUpdateStatusShowsPayload, error) {
	url := fmt.Sprintf("%s/%s%d", *medusa, "api/v2/internal/getEpisodeStatus?status=", *current)
	ids := make([]medusaUpdateStatusShowsPayload, 0)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("x-api-key", *token)
	req.Header.Add("cache-control", "no-cache")
	res, err := http.DefaultClient.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	jsonResponse := make(map[string]interface{})
	if res != nil && res.StatusCode != http.StatusOK {
		log.Print(res.StatusCode)
	}
	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return nil, err
	}
	for _, v := range jsonResponse {
		for k, v := range v.(map[string]interface{}) {
			value := v.(map[string]interface{})
			name := value["name"].(string)
			episodes := value["episodes"].([]interface{})
			downloadStatusEps := make([]string, 0)
			log.Println(name)
			for _, ep := range episodes {
				epMap := ep.(map[string]interface{})
				slug := epMap["slug"].(string)
				downloadStatusEps = append(downloadStatusEps, slug)
				log.Println(slug)

			}
			show := &medusaUpdateStatusShowsPayload{
				Slug:     k,
				Episodes: downloadStatusEps,
			}
			ids = append(ids, *show)
		}
	}
	return ids, nil
}

func (med *httpClient) getStatusMap() (map[string]string, error) {
	url := fmt.Sprintf("%s/%s", *medusa, "api/v2/config")
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("x-api-key", *token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	jsonResponse := make(map[string]interface{})

	err = json.Unmarshal(body, &jsonResponse)
	if err != nil {
		return nil, err
	}

	mConst := jsonResponse["consts"]
	statuses := mConst.(map[string]interface{})["statuses"]
	statusResponse := make(map[string]string)
	for _, item := range statuses.([]interface{}) {
		mItem := item.(map[string]interface{})
		id := fmt.Sprintf("%d", int(mItem["value"].(float64)))
		statusResponse[id] = fmt.Sprintf("%v", mItem["name"])
		fmt.Println(fmt.Sprintf("|%s|%s|", id, mItem["name"]))
	}

	return statusResponse, nil

}

func (med *httpClient) changeEpisodeStatus(payload medusaUpdateStatusPayload) error {
	url := fmt.Sprintf("%s/api/v2/internal/updateEpisodeStatus", *medusa)
	out, err := json.Marshal(payload)
	stringPayload := strings.NewReader(string(out))

	req, err := http.NewRequest("POST", url, stringPayload)
	if err != nil {
		return err
	}

	req.Header.Add("content-type", "application/json")
	req.Header.Add("x-api-key", *token)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer res.Body.Close()
	if res != nil && res.StatusCode != 200 {
		return fmt.Errorf("Server responded with status code: %d", res.StatusCode)
	}
	return nil
}

func (med *httpClient) process() error {
	medusaShows, err := med.getTVShowsWithDownloadedEpStatus()
	if err != nil {
		return err
	}
	updatePayload := medusaUpdateStatusPayload{
		Status: *target,
		Shows:  medusaShows,
	}
	err = med.changeEpisodeStatus(updatePayload)
	if err != nil {
		return err
	}
	return nil
}

// StartUpdate will execute the search for downloaded status episodes and convert them to archived
func StartUpdate() error {
	mClient := newMedusaApi()
	err := mClient.process()
	//_, err = mClient.getStatusMap()
	return err
}
