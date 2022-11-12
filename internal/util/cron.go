package util

import "github.com/robfig/cron/v3"

func ParseCron(c string) (cron.Schedule, error) {
	return cron.ParseStandard(c)
}
