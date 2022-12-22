package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
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
	REPO_PATH = os.Getenv("REPO_PATH")
	LOG_FILE  = os.Getenv("LOG_FILE")

	filename string = fmt.Sprintf("%v/deltadb-assets/collector/nfl.json", REPO_PATH)
	name     string
	query    string
	size     int
	delay    int
)

func init() {

	// Define flag arguments for the application
	flag.StringVar(&name, `name`, ``, `Specify name of the parsing config. Default: <empty>`)
	flag.StringVar(&query, `query`, ``, `Run query to DB for input parameters. Default: <empty>`)
	flag.IntVar(&size, `size`, 10, `Specify size of request pool. Default: 10`)
	flag.IntVar(&delay, `delay`, 60, `Specify delay between collection pools in seconds. Default: 120 sec`)
	flag.Parse()

	// Initialize logfile at user given path.
	logger.InitLog(LOG_FILE)

	logger.Logger.Info().Str("status", "start").Str("name", name).Str("query", query).Msg("COLLECTION")
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

	session.Close()

	// Build queue for http requests and db transaction
	queue := []Pipeline{}
	for _, n := range nodes {
		urls, label := BuildRequests(n, config.Sources)
		new_pipe := Pipeline{Label: label, Bucket: config.Id.Value, Properties: n, Urls: urls}
		queue = append(queue, new_pipe)
	}

	sleep_time := time.Duration(int(time.Second) * delay)
	temp := []Pipeline{}

	for i, entry := range queue {
		if i%size == 0 && i > 0 {
			QueueRequests(driver, config, temp, size, delay)
			time.Sleep(sleep_time)
			temp = []Pipeline{entry}
		} else {
			temp = append(temp, entry)
		}
	}
	if len(temp) > 0 {
		QueueRequests(driver, config, temp, size, delay)
	}

	logger.Logger.Info().Str("status", "end").Str("name", name).Str("query", query).Msg("COLLECTION")
}
