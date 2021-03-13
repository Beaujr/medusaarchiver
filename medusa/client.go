package medusa

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/anaskhan96/soup"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var medusa = flag.String("medusaUrl", "http://192.168.1.112:8081/sickrage", "Medusa URL ie. <protocol>://<url>:<port>/<path>")
var token = flag.String("token", "apikey", "Medusa API Token")
var target = flag.String("target", "6", "Target Id for status, ex: 6")
var current = flag.String("current", "4", "Target Id for status, ex: 4")

// MedusaApi interface for interacting with medusa
type MedusaApi interface {
	getTVShowsWithDownloadedEpStatus() ([]medusaSeries, error)
	changeEpisodeStatus(tvid string, season string, id string) error
	getEpisodesWithDownloadedStatus(tvid string) ([]medusaEpisode, error)
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

func (med *httpClient) getTVShowsWithDownloadedEpStatus() ([]medusaSeries, error) {
	soupHtml, err := soup.GetWithClient(fmt.Sprintf("%s/%s%s", *medusa, "manage/episodeStatuses?whichStatus=", *current), &med.client)
	if err != nil {
		return nil, err
	}
	rootHtml := soup.HTMLParse(soupHtml)
	roots := rootHtml.FindAll("input", "class", "pull-right")
	ids := make([]medusaSeries, 0)
	for _, root := range roots {
		if val, ok := root.Attrs()["data-series-id"]; ok {
			ids = append(ids, medusaSeries{id: val, name: root.FindPrevElementSibling().Text()})
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

func (med *httpClient) getEpisodesWithDownloadedStatus(tvid string) ([]medusaEpisode, error) {
	statuses, err := med.getStatusMap()
	if err != nil {
		return nil, err
	}
	url := fmt.Sprintf("%s/%s%s%s%s", *medusa, "api/", *token, "/?cmd=show.seasons&tvdbid=", tvid)
	soupResponse, err := soup.GetWithClient(url, &med.client)
	if err != nil {
		return nil, err
	}
	jsonResponse := make(map[string]interface{})

	err = json.Unmarshal([]byte(soupResponse), &jsonResponse)
	if err != nil {
		return nil, err
	}
	episodeIds := make([]medusaEpisode, 0)

	if success, ok := jsonResponse["result"]; ok && success.(string) == "success" {
		if data, ok := jsonResponse["data"]; ok {
			seasonData := data.(map[string]interface{})
			seasonNumber := 0
			for {
				seasonNumber++
				seasonString := strconv.Itoa(seasonNumber)
				if episodes, ok := seasonData[seasonString]; ok {
					episodesData := episodes.(map[string]interface{})
					episodeNumber := 0
					for {
						episodeNumber++
						episodeString := strconv.Itoa(episodeNumber)
						if episode, ok := episodesData[episodeString]; ok {

							if status, ok := episode.(map[string]interface{})["status"]; ok && status == statuses[*current] {
								episodeIds = append(episodeIds, medusaEpisode{season: seasonString, episode: episodeString})
							}
						} else {
							break
						}
					}

				} else {
					break
				}
			}

		}
	}
	return episodeIds, nil
}

func (med *httpClient) changeEpisodeStatus(tvid string, season string, id string) error {
	url := fmt.Sprintf("%s/api/v2/series/tvdb%s/episodes", *medusa, tvid)

	payload := strings.NewReader(fmt.Sprintf("{\"s%se%s\":{\"status\":%s}}", season, id, *target))

	req, err := http.NewRequest("PATCH", url, payload)
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
	return err
}

func (med *httpClient) process() error {
	tvShowIds, err := med.getTVShowsWithDownloadedEpStatus()
	if err != nil {
		return err
	}

	for _, show := range tvShowIds {
		episodeIds, _ := med.getEpisodesWithDownloadedStatus(show.id)
		for _, episode := range episodeIds {
			log.Printf("Show: %s, Season: %s, medusaEpisode: %s", show.name, episode.season, episode.episode)
			err := med.changeEpisodeStatus(show.id, episode.season, episode.episode)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// StartUpdate will execute the search for downloaded status episodes and convert them to archived
func StartUpdate() error {
	mClient := newMedusaApi()
	//err := mClient.process()
	_, err := mClient.getStatusMap()
	return err
}
