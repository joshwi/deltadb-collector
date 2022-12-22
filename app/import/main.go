package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/joshwi/go-pkg/logger"
	"github.com/joshwi/go-pkg/utils"
	"github.com/joshwi/go-svc/db"
	"github.com/neo4j/neo4j-go-driver/v4/neo4j"
)

var (
	// Pull in env variables: USERNAME, PASSWORD, uri
	USERNAME  = os.Getenv("NEO4J_USERNAME")
	PASSWORD  = os.Getenv("NEO4J_PASSWORD")
	HOST      = os.Getenv("NEO4J_HOST")
	PORT      = os.Getenv("NEO4J_PORT")
	REPO_PATH = os.Getenv("REPO_PATH")
	LOG_FILE  = os.Getenv("LOG_FILE")
	path      string
)

func init() {

	// Define flag arguments for the application
	flag.StringVar(&path, `filepath`, ``, `Filepath for CSV file. Default: <empty>`)
	flag.Parse()

	// Initialize logfile at user given path.
	logger.InitLog(LOG_FILE)

	logger.Logger.Info().Str("status", "start").Msg("TRANSACTIONS")

}

func main() {

	commands := []string{}

	newpath := fmt.Sprintf("%v/%v", REPO_PATH, path)
	files, err := utils.Scan(newpath)
	if err != nil {
		log.Println(err)
	}

	for _, entry := range files {
		base := filepath.Base(entry)
		name := strings.TrimSuffix(base, filepath.Ext(base))
		if len(name) > 0 {
			command := fmt.Sprintf("LOAD CSV WITH HEADERS FROM 'file:////%v/%v.csv' as row MERGE (n:%v {label: row.label}) SET n += row", path, name, name)
			commands = append(commands, command)
		}
	}

	uri := "bolt://" + HOST + ":" + PORT
	driver := db.Connect(uri, USERNAME, PASSWORD)
	sessionConfig := neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite}
	session := driver.NewSession(sessionConfig)

	if len(commands) > 0 {
		err := db.RunTransactions(session, commands)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}

	logger.Logger.Info().Str("status", "end").Msg("TRANSACTIONS")

}
