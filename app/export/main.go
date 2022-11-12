package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/joshwi/go-pkg/logger"
	"github.com/joshwi/go-svc/db"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

var (
	// Pull in env variables: USERNAME, PASSWORD, uri
	LOGFILE  = os.Getenv("LOGFILE")
	USERNAME = os.Getenv("NEO4J_USERNAME")
	PASSWORD = os.Getenv("NEO4J_PASSWORD")
	HOST     = os.Getenv("NEO4J_HOST")
	PORT     = os.Getenv("NEO4J_PORT")
	path     string
)

func init() {

	// Define flag arguments for the application
	flag.StringVar(&path, `filepath`, ``, `Filepath for CSV file. Default: <empty>`)
	flag.Parse()

	// Initialize logfile at user given path. Default: ./collection.log
	logger.InitLog(LOGFILE)

	logger.Logger.Info().Str("status", "start").Msg("TRANSACTIONS")

}

func main() {

	commands := []string{}

	uri := "bolt://" + HOST + ":" + PORT
	driver := db.Connect(uri, USERNAME, PASSWORD)
	sessionConfig := neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite}
	session := driver.NewSession(sessionConfig)

	res, err := db.RunCypher(session, "MATCH (m) UNWIND labels(m) AS node RETURN distinct node")
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	for _, node := range res {
		if len(node) > 0 {
			name := node[0].Value
			query := "WHERE n.year='2020'"
			command := fmt.Sprintf("MATCH (n:%v) %v WITH collect(n) AS response CALL apoc.export.csv.data(response, [], '%v/%v.csv', {}) YIELD file, source, format, nodes, relationships, properties, time, rows, batchSize, batches, done, data RETURN file, source, format, nodes, relationships, properties, time, rows, batchSize, batches, done, data", name, query, path, name)
			commands = append(commands, command)
		}
	}

	if len(commands) > 0 {
		err := db.RunTransactions(session, commands)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}

	logger.Logger.Info().Str("status", "end").Msg("TRANSACTIONS")

}