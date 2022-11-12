package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"

	"github.com/joshwi/go-pkg/logger"
	"github.com/joshwi/go-pkg/utils"
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
	filepath string
)

func init() {

	// Define flag arguments for the application
	flag.StringVar(&filepath, `file`, ``, `Filename for DB transactions. Default: <empty>`)
	flag.Parse()

	// Initialize logfile at user given path. Default: ./collection.log
	logger.InitLog(LOGFILE)

	logger.Logger.Info().Str("status", "start").Msg("TRANSACTIONS")

}

func main() {

	uri := "bolt://" + HOST + ":" + PORT
	driver := db.Connect(uri, USERNAME, PASSWORD)
	sessionConfig := neo4j.SessionConfig{AccessMode: neo4j.AccessModeWrite}
	session := driver.NewSession(sessionConfig)

	fileBytes, _ := utils.Read(filepath)

	var commands []string
	json.Unmarshal(fileBytes, &commands)

	if len(commands) > 0 {
		err := db.RunTransactions(session, commands)
		if err != nil {
			log.Fatal(err)
			os.Exit(1)
		}
	}

	logger.Logger.Info().Str("status", "end").Msg("TRANSACTIONS")

}
