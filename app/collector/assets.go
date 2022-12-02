package main

import (
	"fmt"
	"log"
	"math"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/joshwi/go-pkg/logger"
	"github.com/joshwi/go-pkg/parser"
	"github.com/joshwi/go-pkg/utils"
	"github.com/joshwi/go-svc/db"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

type Pipeline struct {
	Urls       []string
	Label      string
	Bucket     string
	Properties map[string]string
	Collection utils.Collection
	httpError  error
	dbError    error
}

func BuildRequests(query map[string]string, urls []string) ([]string, string) {

	// Enter variables to the url templates
	req_urls := []string{}
	keys := []string{}
	for _, url := range urls {
		for k, v := range query {
			keys = append(keys, k)
			re, _ := regexp.Compile(fmt.Sprintf("{%v}", k))
			url = re.ReplaceAllString(url, v)
		}
		req_urls = append(req_urls, url)
	}

	// Sort list of keys
	sort.Slice(keys, func(a, b int) bool {
		return keys[a] < keys[b]
	})
	// Get list of values http_channel alpha order of keys
	values := []string{}
	for _, v := range keys {
		values = append(values, query[v])
	}
	// Compute label for the collection
	label := strings.Join(values, "_")

	return req_urls, label
}

func ComputeMetrics(pass int, total int) string {

	rate := "0%"

	if total > 0 {
		percent := (float64(pass) / float64(total)) * 100.0
		rate = fmt.Sprintf("%v%%", math.Round(percent*100)/100)
	}

	return rate
}

func ComputeTime(total int, start time.Time, end time.Time) (string, string) {
	elapsed := end.Sub(start)
	duration := fmt.Sprintf("%v", elapsed.Round(time.Second/1000))

	average := "0 ms"

	if total > 0 {
		average = fmt.Sprintf("%v ms", int(elapsed.Milliseconds())/total)
	}

	return duration, average
}

func HttpRequest(config utils.Config, http_channel chan Pipeline, db_channel chan Pipeline) {
	for entry := range http_channel {
		for _, item := range entry.Urls {
			response, _ := utils.Get(item, map[string]string{"User-Agent": "Mantis/1.0"})
			if response.Status == 200 {
				parsed_data := parser.Collect(response.Data, config.Parser)
				entry.Collection = parsed_data
			} else {
				logger.Logger.Error().Str("status", "Failed").Int("code", response.Status).Err(fmt.Errorf(response.Error)).Msg("Get")
				entry.httpError = fmt.Errorf("%v", response.Status)
				if response.Status == 429 {
					log.Fatal("429!!! YOU SHALL NOT PASS!!!")
				}
			}
		}
		db_channel <- entry
	}
}

func DBTransaction(driver neo4j.Driver, pipe chan Pipeline, out_channel chan Pipeline) {
	for entry := range pipe {
		if entry.httpError == nil {

			sessionConfig := neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite}
			session := driver.NewSession(sessionConfig)

			defer session.Close()

			db_response := []utils.Tag{}
			for k, v := range entry.Properties {
				db_response = append(db_response, utils.Tag{Name: k, Value: v})
			}

			data := entry.Collection
			if len(data.Buckets) > 0 {
				for _, item := range data.Buckets {
					for n, index := range item.Value {
						properties := []utils.Tag{}
						properties = append(properties, db_response...)
						properties = append(properties, data.Tags...)
						properties = append(properties, index...)
						bucket := entry.Bucket
						if len(item.Name) > 0 {
							bucket = bucket + "_" + item.Name
						}
						err := db.PutNode(session, bucket, fmt.Sprintf("%v_%v", entry.Label, n+1), properties)
						entry.dbError = err
					}
				}
			} else {
				if len(data.Tags) > 1 {
					properties := []utils.Tag{}
					properties = append(properties, db_response...)
					properties = append(properties, data.Tags...)
					err := db.PutNode(session, entry.Bucket, entry.Label, properties)
					entry.dbError = err
				}
			}
		}

		out_channel <- entry
	}
}

func QueueRequests(driver neo4j.Driver, config utils.Config, queue []Pipeline, size int, delay int) (int, int) {

	// Create channels for data flow and error reporting
	http_channel := make(chan Pipeline, size)
	db_channel := make(chan Pipeline, size)
	out_channel := make(chan Pipeline, size)
	start := time.Now()

	// Input the url search list into channel
	go func() {
		for _, entry := range queue {
			http_channel <- entry
		}
	}()

	// Run HTTP request worker to gather data
	for i := 0; i < cap(queue); i++ {
		go HttpRequest(config, http_channel, db_channel)
	}

	// Run DB transaction to POST parsed data
	for i := 0; i < cap(queue); i++ {
		go DBTransaction(driver, db_channel, out_channel)
	}

	// Count success rate of the collector
	http_pass := 0
	db_pass := 0
	for range queue {
		entry := <-out_channel
		if entry.httpError == nil {
			http_pass++
		}
		if entry.dbError == nil {
			db_pass++
		}
	}

	// Close channels when operations are complete
	close(http_channel)
	close(db_channel)
	close(out_channel)

	end := time.Now()

	duration, average := ComputeTime(len(queue), start, end)

	logger.Logger.Info().Str("duration", duration).Str("speed", fmt.Sprintf("%v/req", average)).Int("total", len(queue)).Msg("TIME STATS")

	pass_rate := ComputeMetrics(http_pass, len(queue))

	logger.Logger.Info().Str("success", pass_rate).Int("completed", http_pass).Int("total", len(queue)).Msg("HTTP REQUESTS")

	pass_rate = ComputeMetrics(db_pass, len(queue))

	logger.Logger.Info().Str("success", pass_rate).Int("completed", db_pass).Int("total", len(queue)).Msg("DB TRANSACTIONS")

	return http_pass, db_pass
}
