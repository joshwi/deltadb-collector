package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/joshwi/go-pkg/logger"
	"github.com/joshwi/go-pkg/parser"
	"github.com/joshwi/go-pkg/utils"
	"github.com/joshwi/go-svc/db"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

var (
	// Pull in env variables: username, password, uri
	username = os.Getenv("NEO4J_USERNAME")
	password = os.Getenv("NEO4J_PASSWORD")
	host     = os.Getenv("NEO4J_HOST")
	port     = os.Getenv("NEO4J_PORT")
	logfile  = os.Getenv("LOGFILE")

	// Init flag values
	name     string
	query    string
	filename string
)

func init() {

	// Define flag arguments for the application
	flag.StringVar(&name, `name`, ``, `Specify config. Default: <empty>`)
	flag.StringVar(&query, `query`, ``, `Run query to DB for input parameters. Default: <empty>`)
	flag.StringVar(&filename, `file`, ``, `Location of parsing config file. Default: <empty>`)
	flag.Parse()

	// Initialize logfile at user given path. Default: ./collection.log
	logger.InitLog(logfile)

	logger.Logger.Info().Str("config", name).Str("query", query).Str("status", "start").Msg("COLLECTION")
}

func AddParams(query map[string]string, urls []string) ([]string, string) {
	search := []string{}
	for _, url := range urls {
		for k, v := range query {
			re, _ := regexp.Compile(fmt.Sprintf("{%v}", k))
			url = re.ReplaceAllString(url, v)
		}
		search = append(search, url)
	}

	keys := []string{}

	for k := range query {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(a, b int) bool {
		return keys[a] < keys[b]
	})

	values := []string{}

	for _, v := range keys {
		values = append(values, query[v])
	}

	label := strings.Join(values, "_")

	return search, label
}

func main() {

	// Open file with parsing configurations
	fileBytes, err := utils.Read(filename)
	if err != nil {
		log.Println(err)
	}

	// Create application session with Neo4j
	uri := "bolt://" + host + ":" + port
	driver := db.Connect(uri, username, password)
	sessionConfig := neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite}
	session := driver.NewSession(sessionConfig)
	if err != nil {
		log.Println(err)
	}

	// Unmarshall file into []Config struct
	var configurations map[string]utils.Config
	json.Unmarshal(fileBytes, &configurations)

	config := configurations[name]

	// Get config by name
	config.Parser = parser.Compile(config.Parser)

	db_res := [][]utils.Tag{{}}

	if len(query) > 0 {
		// Grab input parameters from  Neo4j
		db_res, err = db.RunCypher(session, query)
	}

	for _, db_row := range db_res {

		// Convert params from struct [][]utils.Tag -> map[string]string
		params := map[string]string{}
		for _, item := range db_row {
			params[item.Name] = item.Value
		}

		urls, label := AddParams(params, config.Sources)

		for _, item := range urls {
			response, _ := utils.Get(item, map[string]string{})
			if response.Status == 200 {
				data := parser.Collect(response.Data, config.Parser)

				if len(data.Buckets) > 0 {
					for _, item := range data.Buckets {
						for n, index := range item.Value {
							properties := []utils.Tag{}
							properties = append(properties, db_row...)
							properties = append(properties, data.Tags...)
							properties = append(properties, index...)
							bucket := config.Id.Value
							if len(item.Name) > 0 {
								bucket = bucket + "_" + item.Name
							}
							db.PutNode(session, bucket, fmt.Sprintf("%v_%v", label, n+1), properties)
						}
					}
				} else {
					properties := []utils.Tag{}
					properties = append(properties, db_row...)
					properties = append(properties, data.Tags...)
					bucket := config.Id.Value
					db.PutNode(session, bucket, label, properties)
				}

			}
		}

	}

	logger.Logger.Info().Str("config", name).Str("query", query).Str("status", "end").Msg("COLLECTION")

}
