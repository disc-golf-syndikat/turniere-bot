package turniere

import (
	"io"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

type Fields int

const (
	Title Fields = iota + 1
	Location
	TurnamentDate
	RegistrationStartDate
	Changed
	Series
)

const timezone = "Europe/Berlin"
const dateFormat = "02.01.2006"
const dateTimeFormat = dateFormat + " 15:04"

var location time.Location

func Parse(reader io.Reader) []Turnament {
	l, _ := time.LoadLocation(timezone)
	location = *l

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Fatal(err)
	}

	var keys [10]Fields

	table := doc.Find("#list_tournaments")
	table.Find("thead tr").Each(func(i int, s *goquery.Selection) {
		s.Find("th").Each(func(j int, t *goquery.Selection) {
			title := t.Contents().Nodes[0].Data
			switch title {
			case "Turnier":
				keys[j] = Title
			case "Austragungsort":
				keys[j] = Location
			case "Erster Turniertag":
				keys[j] = TurnamentDate
			case "Anmeldung öffnet":
				keys[j] = RegistrationStartDate
			case "Letzte Änderung":
				keys[j] = Changed
			case "Serie/n":
				keys[j] = Series
			}
		})
	})

	var result []Turnament
	table.Find("tbody tr").Each(func(i int, s *goquery.Selection) {
		r := &Turnament{}
		cols := s.Find("td")
		cols.Each(func(j int, t *goquery.Selection) {
			switch keys[j] {
			case Title:
				r.Title = extractTitle(t)
				r.Link = extractLink(t)
			case RegistrationStartDate:
				r.RegistrationStartDate = extractDate(t, dateTimeFormat)
			case TurnamentDate:
				r.TurnamentDate = extractDate(t, dateFormat)
			case Location:
				r.Location = extractText(t)
			case Changed:
				a := extractDate(t, dateTimeFormat)
				if a != nil {
					r.Changed = *a
				}
			case Series:
				r.Series = extractSeries(t)
			}
		})
		u, _ := url.Parse(r.Link)
		v := u.Query()
		r.Id = v["id"][0]
		result = append(result, *r)
	})
	return result
}

func extractTitle(td *goquery.Selection) string {
	anker := td.Find("a").First()
	return strings.TrimSpace(anker.Text())
}

func extractText(td *goquery.Selection) string {
	return strings.TrimSpace(td.Text())
}

func extractLink(td *goquery.Selection) string {
	anker := td.Find("a").First()
	return anker.AttrOr("href", "NO HREF")
}

func extractDate(td *goquery.Selection, format string) *time.Time {
	raw := strings.TrimSpace(td.Text())
	if raw == "" {
		return nil
	}
	t, err := time.ParseInLocation(format, raw, &location)
	if err != nil {
		log.Print(err)
		return nil
	}
	return &t
}

func extractSeries(td *goquery.Selection) []string {
	result := []string{}
	td.Find("span").Each(func(i int, s *goquery.Selection) {
		t := extractText(s)
		if t != "Info" {
			result = append(result, t)
		}
	})
	return result
}

func parseDateTime(i string) time.Time {
	raw := strings.TrimSpace(i)
	t, _ := time.ParseInLocation(dateTimeFormat, raw, &location)
	return t
}

func ParsePhases(reader io.Reader) []Phase {
	phases := []Phase{}

	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		log.Fatal(err)
	}

	card := doc.Find(".card").FilterFunction(func(i int, s *goquery.Selection) bool {
		t := s.Find(".card-title")
		return t != nil && t.Text() == "Anmeldephasen"
	})

	card.Find(".row").First().Find(".card-header h5").Each(func(i int, s *goquery.Selection) {
		title := s.Text()
		dr := s.Parent().Find("small").Text()
		dates := strings.Split(dr, "-")

		phases = append(phases, Phase{Title: title, RegistrationStartDate: parseDateTime(dates[0])})
	})

	return phases
}
