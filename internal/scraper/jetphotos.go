package scraper

import (
	"fmt"
	"strings"
	"sync"
)

type jetPhotosInfo struct {
	Reg    string         `json:"Reg"`
	Images []imagesStruct `json:"Images"`
}

type imagesStruct struct {
	Image        string          `json:"Image"`
	Link         string          `json:"Link"`
	Thumbnail    string          `json:"Thumbnail"`
	DateTaken    string          `json:"DateTaken"`
	DateUploaded string          `json:"DateUploaded"`
	Location     string          `json:"Location"`
	Photographer string          `json:"Photographer"`
	Aircraft     *aircraftStruct `json:"Aircraft"`
}

type aircraftStruct struct {
	Aircraft string `json:"Aircraft"`
	Serial   string `json:"Serial"`
	Airline  string `json:"Airline"`
}

type jetPhotosRes struct {
	Res *jetPhotosInfo
	Err error
}

const jpHomeURL = "https://www.jetphotos.com"

func getJetPhotosStruct(q *Queries, done chan jetPhotosRes) {
	if q.Photos == 0 {
		result := jetPhotosRes{Res: &jetPhotosInfo{Reg: strings.ToUpper(q.Reg)}}
		done <- result
		return
	}

	URL := fmt.Sprintf("%s/photo/keyword/%s", jpHomeURL, q.Reg)
	b, err := fetchHTML(URL)
	if err != nil {
		result := jetPhotosRes{Res: nil, Err: err}
		done <- result
		return
	}

	s := newScraper(b)
	pageLinks := []string{}
	thumbnails := []string{}
	atLeastOne := false
	for i := 0; i < q.Photos; i++ {
		pageLink, err1 := s.fetchLinks("a", "result__photoLink", 1)
		thumbnail, err2 := s.fetchLinks("img", "result__photo", 1)
		if err1 != nil || err2 != nil {
			if atLeastOne {
				break
			}
			result := jetPhotosRes{Res: nil, Err: err}
			done <- result
			return
		}
		pageLinks = append(pageLinks, pageLink[0])
		thumbnails = append(thumbnails, thumbnail[0])
		atLeastOne = true
	}
	s.close()

	imgs := len(pageLinks)

	var registration string
	images := make([]imagesStruct, imgs)

	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, link := range pageLinks {
		wg.Add(1)
		go func(i int, link string) {
			defer wg.Done()

			photoURL := fmt.Sprintf("%s%s", jpHomeURL, link)
			img := imagesStruct{
				Link:      photoURL,
				Thumbnail: "https:" + thumbnails[i],
			}

			b, err := fetchHTML(photoURL)
			if err != nil {
				fmt.Printf("⚠️ Failed to fetch photo page for %s: %v\n", link, err)
				mu.Lock()
				images[i] = img
				mu.Unlock()
				return
			}

			s := newScraper(b)

			if photoLinkArr, err := s.fetchLinks("img", "large-photo__img", 1); err == nil && len(photoLinkArr) > 0 {
				img.Image = photoLinkArr[0]
			}

			if regText, err := s.fetchText("h4", "headerText4 color-shark", 3); err == nil {
				if len(regText) > 0 {
					mu.Lock()
					registration = regText[0]
					mu.Unlock()
				}
				if len(regText) > 1 {
					img.DateTaken = regText[1]
				}
				if len(regText) > 2 {
					img.DateUploaded = regText[2]
				}
			}

			s.advance("h2", "header-reset", 1)
			aircraft := &aircraftStruct{}
			if aircraftText, err := s.fetchText("a", "link", 3); err == nil {
				if len(aircraftText) > 0 {
					aircraft.Aircraft = aircraftText[0]
				}
				if len(aircraftText) > 1 {
					aircraft.Airline = aircraftText[1]
				}
				if len(aircraftText) > 2 {
					aircraft.Serial = strings.TrimSpace(aircraftText[2])
				}
				img.Aircraft = aircraft
			}

			s.advance("h5", "header-reset", 1)
			if location, err := s.fetchText("a", "link", 1); err == nil && len(location) > 0 {
				img.Location = location[0]
			}

			if photographer, err := s.fetchText("h6", "header-reset", 1); err == nil && len(photographer) > 0 {
				img.Photographer = photographer[0]
			}

			s.close()

			mu.Lock()
			images[i] = img
			mu.Unlock()
		}(i, link)
	}
	wg.Wait()

	j := &jetPhotosInfo{
		Images: images,
		Reg:    registration,
	}

	result := jetPhotosRes{Res: j, Err: nil}
	done <- result
}
