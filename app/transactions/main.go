package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

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
	BASE_PATH = os.Getenv("BASE_PATH")
	file      string
)

func init() {

	// Define flag arguments for the application
	flag.StringVar(&file, `file`, ``, `Filename for DB transactions. Default: <empty>`)
	flag.Parse()

	// Initialize logfile at user given path.
	logfile := fmt.Sprintf("%v/run.log", BASE_PATH)
	logger.InitLog(logfile)

	logger.Logger.Info().Str("status", "start").Msg("TRANSACTIONS")

}

func main() {

	filepath := fmt.Sprintf("%v/%v", BASE_PATH, file)
	fileBytes, err := utils.Read(filepath)
	if err != nil {
		log.Fatal("No such file or directory!")
	}

	var commands []string
	json.Unmarshal(fileBytes, &commands)

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
