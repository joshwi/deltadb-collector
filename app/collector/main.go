package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math"
	"os"
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

var (
	// Pull http_channel env variables: username, password, uri
	USERNAME  = os.Getenv("NEO4J_USERNAME")
	PASSWORD  = os.Getenv("NEO4J_PASSWORD")
	HOST      = os.Getenv("NEO4J_HOST")
	PORT      = os.Getenv("NEO4J_PORT")
	BASE_PATH = os.Getenv("BASE_PATH")

	filename string = fmt.Sprintf("%v/config/collector/nfl.json", BASE_PATH)
	name     string
	query    string
)

func init() {

	// Define flag arguments for the application
	flag.StringVar(&name, `name`, ``, `Specify name of the parsing config. Default: <empty>`)
	flag.StringVar(&query, `query`, ``, `Run query to DB for input parameters. Default: <empty>`)
	flag.Parse()

	// Initialize logfile at user given path.
	logfile := fmt.Sprintf("%v/run.log", BASE_PATH)
	logger.InitLog(logfile)

	logger.Logger.Info().Str("status", "start").Str("name", name).Str("query", query).Msg("COLLECTION")
}

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
			}
		}
		db_channel <- entry
	}
}

func DBTransaction(session neo4j.Session, mantis chan Pipeline, out_channel chan Pipeline) {
	for entry := range mantis {
		if entry.httpError == nil {

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
						err := db.PutNode(session, entry.Bucket, fmt.Sprintf("%v_%v", entry.Label, n+1), properties)
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

func main() {

	// Open file with parsing configurations
	fileBytes, err := utils.Read(filename)
	if err != nil {
		log.Fatal("No such file or directory!")
	}
	// Unmarshall file into []Config struct
	var configurations map[string]utils.Config
	json.Unmarshal(fileBytes, &configurations)
	// Get config by name and compile regex
	config := configurations[name]
	config.Parser = parser.Compile(config.Parser)

	if len(config.Sources) == 0 {
		logger.Logger.Error().Str("name", name).Err(fmt.Errorf("Invalid config name")).Msg("Config")
		log.Fatalf("Invalid config name: %v", name)
	}

	// Create application session with Neo4j
	uri := "bolt://" + HOST + ":" + PORT
	driver := db.Connect(uri, USERNAME, PASSWORD)
	sessionConfig := neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite}
	session := driver.NewSession(sessionConfig)

	// Query DB to get audit search list
	nodes := []map[string]string{{}}
	if len(query) > 0 {
		nodes, _ = db.Query(session, query)
	}

	// Build queue for http requests and db transaction
	queue := []Pipeline{}
	for _, n := range nodes {
		urls, label := BuildRequests(n, config.Sources)
		new_mantis := Pipeline{Label: label, Bucket: config.Id.Value, Properties: n, Urls: urls}
		queue = append(queue, new_mantis)
	}

	// Create channels for data flow and error reporting
	http_channel := make(chan Pipeline, 10)
	db_channel := make(chan Pipeline, 10)
	out_channel := make(chan Pipeline, 10)
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
		go DBTransaction(session, db_channel, out_channel)
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

	session.Close()

	end := time.Now()

	duration, average := ComputeTime(len(queue), start, end)

	logger.Logger.Info().Str("duration", duration).Str("speed", fmt.Sprintf("%v/req", average)).Int("total", len(nodes)).Msg("TIME STATS")

	pass_rate := ComputeMetrics(http_pass, len(queue))

	logger.Logger.Info().Str("success", pass_rate).Int("completed", http_pass).Int("total", len(nodes)).Msg("HTTP REQUESTS")

	pass_rate = ComputeMetrics(db_pass, len(queue))

	logger.Logger.Info().Str("success", pass_rate).Int("completed", db_pass).Int("total", len(nodes)).Msg("DB TRANSACTIONS")

	logger.Logger.Info().Str("status", "end").Str("name", name).Str("query", query).Msg("COLLECTION")
}
