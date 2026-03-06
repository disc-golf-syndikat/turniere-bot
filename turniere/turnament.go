package turniere

import "time"

type Turnament struct {
	Link                  string
	Id                    string
	Title                 string
	Series                []string
	Location              string
	TurnamentDate         *time.Time
	RegistrationStartDate *time.Time
	Phases                []Phase
	Changed               time.Time
}

type Phase struct {
	Title                 string
	RegistrationStartDate time.Time
	Requirements          map[string]string
	Notified              bool
}
