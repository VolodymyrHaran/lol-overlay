package models

import "time"

type Room struct {
	Id          string    `json:"id"`
	Players     []Player  `json:"players"`
	LastUpdated time.Time `json:"lastUpdated"`
}
