package medusa

import (
	"fmt"
	"github.com/anaskhan96/soup"
	"net/http"
	"encoding/json"
	"strconv"
	"crypto/tls"
	"log"
)

const medusaRoot = "http://localhost:8081"
const medusaToken = "<INSERT TOKEN>"
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
	season string
	episode string
}

type Series struct {
	name string
	id string
}

func(med *Client) getTVShowsWithDownloadedEpStatus() ([]Series, error) {
	soupHtml, err  := soup.GetWithClient(fmt.Sprintf("%s/%s%s", medusaRoot,"manage/episodeStatuses?whichStatus=", downloadedStatus), &med.client)
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

func(med *Client) getEpisodesWithDownloadedStatus(tvid string) ([]Episode, error){
	url := fmt.Sprintf("%s/%s%s%s%s", medusaRoot,"api/",medusaToken,"/?cmd=show.seasons&tvdbid=", tvid)
	soupResponse, err  := soup.GetWithClient(url, &med.client)
	if err != nil {
		return nil, err
	}
	jsonResponse := make(map[string]interface{})

	json.Unmarshal([]byte(soupResponse), &jsonResponse)
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
	apiUrl := fmt.Sprintf("home/setStatus?indexername=tvdb&seriesid=%s&eps=s%se%s&status=%s",  tvid, season, id, archivedStatus)
	url := fmt.Sprintf("%s/%s", medusaRoot, apiUrl)
	_, err  := soup.GetWithClient(url, &med.client)
	if err != nil {
		return err
	}
	return nil
}

func StartUpdate () {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := http.Client{Transport:http.DefaultTransport.(*http.Transport)}
	mClient := Client{client:client}
	tvShowIds, _ := mClient.getTVShowsWithDownloadedEpStatus()

	for _, show := range tvShowIds {
		episodeIds, _ := mClient.getEpisodesWithDownloadedStatus(show.id)
		for _, episode := range episodeIds {
			log.Printf("Show: %s, Season: %s, Episode: %s", show.name, episode.season, episode.episode)
			mClient.changeEpisodeStatus(show.id, episode.season, episode.episode)
		}
	}
}