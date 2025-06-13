package model

import "cube/internal/util"

type Source struct {
	Id               int       `json:"rowid"`
	Name             string    `json:"name"`
	Type             string    `json:"type"` // module, controller, daemon, crontab, template, resource
	Lang             string    `json:"lang"` // typescript, html, text, vue
	Content          string    `json:"content,omitempty"`
	Compiled         string    `json:"compiled,omitempty"`
	Active           bool      `json:"active"`
	Method           string    `json:"method"`
	Url              string    `json:"url"`
	Cron             string    `json:"cron"`
	Tag              string    `json:"tag"`
	LastModifiedDate util.Time `json:"last_modified_date"`
	Status           string    `json:"status"`
}
