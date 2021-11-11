package stats

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type MySql_handle struct {
	db         *sql.DB
	db_name    string
	stat_table string
}

func (handle *MySql_handle) Initialize(user string, password string,
	host string, port int) error {

	Stats = handle

	handle.db_name = "eve_stats_db"
	handle.stat_table = "queries"

	db_access := fmt.Sprintf("%s:%s@tcp(%s:%d)/", user, password, host, port)
	log.Printf("Connecting to mysql database with access: %s\n", db_access)
	db, err := sql.Open("mysql", db_access)
	if err != nil {
		panic(err)
	}

	handle.db = db

	_, err = db.Exec("CREATE DATABASE IF NOT EXISTS " + handle.db_name)
	if err != nil {
		panic(err)
	}

	_, err = db.Exec("USE " + handle.db_name)
	if err != nil {
		panic(err)
	}

	table_str := "CREATE TABLE IF NOT EXISTS " + handle.stat_table + " ("
	table_str += " timestamp TIMESTAMP(0) DEFAULT CURRENT_TIMESTAMP,"
	table_str += " id INT NOT NULL AUTO_INCREMENT,"
	table_str += " query VARCHAR(256) NOT NULL,"
	table_str += " requestor VARCHAR(64) NOT NULL,"
	table_str += " category VARCHAR(256) NOT NULL,"
	table_str += " numResponses INT NOT NULL,"
	table_str += " fullResponse VARCHAR(2048),"
	table_str += " PRIMARY KEY (id)"
	table_str += ")"
	log.Printf("Creating table: %s\n", table_str)
	_, err = db.Exec(table_str)
	if err != nil {
		panic(err)
	}

	return nil
}

func (handle *MySql_handle) GetLastNstats(last_n int) ([]Stat_data, error) {
	var stat_data []Stat_data

	table_str := "SELECT timestamp,id,query,requestor,category,numResponses from " +
		handle.db_name + "." + handle.stat_table + " order by timestamp DESC limit " +
		strconv.Itoa(last_n)
	log.Printf("Going to query: %s\n", table_str)
	result, err := handle.db.Query(table_str)
	if err != nil {
		log.Println("Failed to query database: " + err.Error())
		return nil, err
	}

	for result.Next() {
		stat_entry := new(Stat_data)
		err = result.Scan(&stat_entry.Timestamp, &stat_entry.Id, &stat_entry.Query,
			&stat_entry.Requestor, &stat_entry.Category, &stat_entry.NumResponses)
		if err != nil {
			log.Printf("Error scanning response: %s\n", err.Error())
			return nil, err
		}

		stat_data = append(stat_data, *stat_entry)
	}

	log.Printf("Got %d responses from query\n", len(stat_data))
	return stat_data, nil
}

func (handle *MySql_handle) GetResponseById(Id int) (string, error) {
	table_str := "SELECT fullResponse from " + handle.stat_table +
		" where id=" + strconv.Itoa(Id)
	log.Printf("Going to query: %s\n", table_str)
	result, err := handle.db.Query(table_str)
	if err != nil {
		log.Println("Failed to query database: " + err.Error())
		return "", err
	}

	var full_result string

	result.Next()
	err = result.Scan(&full_result)
	if err != nil {
		log.Println("Failed to scan result: " + err.Error())
		return "", err
	}

	log.Printf("Got response for id:%d - %s\n", Id, full_result)
	return full_result, nil
}

func (handle *MySql_handle) Updatestat(query string, requestor string, category string,
	num_responses int, full_response string) error {

	table_str := "INSERT into " + handle.stat_table
	table_str += " (query, requestor, category, numResponses, fullResponse) VALUES ("
	table_str += "\"" + query + "\", \"" + requestor + "\", \"" + category + "\", \"" +
		strconv.Itoa(num_responses) + "\", \"" + full_response +
		"\")"
	log.Printf("Executing query %s\n", table_str)

	_, err := handle.db.Exec(table_str)
	if err != nil {
		log.Println(err.Error())
	}

	return err
}
