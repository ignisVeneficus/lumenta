package db

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/ignisVeneficus/lumenta/config"

	_ "github.com/go-sql-driver/mysql"
	"github.com/rs/zerolog/log"
)

var (
	database *sql.DB
	once     sync.Once
)

//var connector driver.Connector

func connectToDatabase(config config.DatabaseConfig, multipleQuery bool) *sql.DB {
	/*
		cfg := mysql.Config{
			User:                 config.User,
			Passwd:               config.Pass,
			Net:                  "tcp",
			Addr:                 config.Url,
			DBName:               config.Database,
			AllowNativePasswords: true,
			Params:               map[string]string{"parseTime": "true"},
		}
	*/
	// Get a driver-specific connector.
	//connector, err := mysql.NewConnector(&cfg)
	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s)/%s?parseTime=true&multiStatements=%t", config.User, config.Password, config.Host, config.Name, multipleQuery))
	if err != nil {
		log.Logger.Fatal().Err(err).Msg("Connect to database")
		panic(err)
	}
	/*
		// Get a database handle.
		db := sql.OpenDB(connector)
		// Confirm a successful connection.
	*/
	if err := db.Ping(); err != nil {
		log.Logger.Fatal().Err(err).Msg("Connection check")
		panic(err)
	}
	return db
}
func GetDatabase() *sql.DB {
	once.Do(func() {
		database = connectToDatabase(config.Global().Database, false)
	})
	return database
}
func GetDatabaseMulti() *sql.DB {
	return connectToDatabase(config.Global().Database, true)

}
