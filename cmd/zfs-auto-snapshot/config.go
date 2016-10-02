package main

import "time"

type seriesConfig struct {
	label    string
	interval time.Duration
	keep     int
}
