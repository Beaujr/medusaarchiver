package medusa

import (
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/anaskhan96/soup"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var medusa = flag.String("medusaUrl", "http://192.168.1.112:8081/sickrage", "Medusa URL ie. <protocol>://<url>:<port>/<path>")
var token = flag.String("token", "apikey", "Medusa API Token")

const archivedStatus = "6"
const downloadedStatus = "4"
const searchStatus = "Downloaded"

type Medusa interface {
	getTVShowsWithDownloadedEpStatus() ([]Series, error)
	changeEpisodeStatus(tvid string, season string, id string) error
	getEpisodesWithDownloadedStatus(tvid string) ([]Episode, error)
}

type Client struct {
	client http.Client
}

type Episode struct {
	season  string
	episode string
}

type Series struct {
	name string
	id   string
}

func (med *Client) getTVShowsWithDownloadedEpStatus() ([]Series, error) {
	soupHtml, err := soup.GetWithClient(fmt.Sprintf("%s/%s%s", *medusa, "manage/episodeStatuses?whichStatus=", downloadedStatus), &med.client)
	if err != nil {
		return nil, err
	}
	rootHtml := soup.HTMLParse(soupHtml)
	roots := rootHtml.FindAll("input", "class", "pull-right")
	ids := make([]Series, 0)
	for _, root := range roots {
		if val, ok := root.Attrs()["data-series-id"]; ok {
			ids = append(ids, Series{id: val, name: root.FindPrevElementSibling().Text()})
		}
	}

	return ids, nil
}

func (med *Client) getEpisodesWithDownloadedStatus(tvid string) ([]Episode, error) {
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
	episodeIds := make([]Episode, 0)

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

							if status, ok := episode.(map[string]interface{})["status"]; ok && status == searchStatus {
								episodeIds = append(episodeIds, Episode{season: seasonString, episode: episodeString})
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

func (med *Client) changeEpisodeStatus(tvid string, season string, id string) error {
	url := fmt.Sprintf("%s/api/v2/series/tvdb%s/episodes", *medusa, tvid)

	payload := strings.NewReader(fmt.Sprintf("{\"s%se%s\":{\"status\":%s}}", season, id, archivedStatus))

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

func StartUpdate() error {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := http.Client{Transport: http.DefaultTransport.(*http.Transport), Timeout: time.Second * 10}
	mClient := Client{client: client}
	tvShowIds, err := mClient.getTVShowsWithDownloadedEpStatus()
	if err != nil {
		return err
	}

	for _, show := range tvShowIds {
		episodeIds, _ := mClient.getEpisodesWithDownloadedStatus(show.id)
		for _, episode := range episodeIds {
			log.Printf("Show: %s, Season: %s, Episode: %s", show.name, episode.season, episode.episode)
			err := mClient.changeEpisodeStatus(show.id, episode.season, episode.episode)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
