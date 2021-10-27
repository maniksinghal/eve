package stats

import (
	"database/sql"
	"fmt"
	"strconv"

	_ "github.com/go-sql-driver/mysql"
)

type MySql_handle struct {
	db         *sql.DB
	db_name    string
	stat_table string
}

func (handle *MySql_handle) Initialize() error {

	handle.db_name = "eve_stats_db"
	handle.stat_table = "queries"

	db, err := sql.Open("mysql", "root:cisco@123@tcp(127.0.0.1:3306)/")
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
	table_str += " category VARCHAR(256) NOT NULL,"
	table_str += " numResponses INT NOT NULL,"
	table_str += " fullResponse VARCHAR(2048),"
	table_str += " PRIMARY KEY (id)"
	table_str += ")"
	fmt.Printf("Creating table: %s\n", table_str)
	_, err = db.Exec(table_str)
	if err != nil {
		panic(err)
	}

	return nil
}

func (handle *MySql_handle) GetLastNstats(last_n int) ([]Stat_data, error) {
	var stat_data []Stat_data

	table_str := "SELECT timestamp,id,query,category,numResponses from " +
		handle.db_name + "." + handle.stat_table + " order by timestamp DESC limit " +
		strconv.Itoa(last_n)
	fmt.Printf("Going to query: %s\n", table_str)
	result, err := handle.db.Query(table_str)
	if err != nil {
		fmt.Println("Failed to query database: " + err.Error())
		return nil, err
	}

	for result.Next() {
		stat_entry := new(Stat_data)
		err = result.Scan(&stat_entry.Timestamp, &stat_entry.Id, &stat_entry.Query,
			&stat_entry.Category, &stat_entry.NumResponses)
		if err != nil {
			fmt.Printf("Error scanning response: %s\n", err.Error())
			return nil, err
		}

		stat_data = append(stat_data, *stat_entry)
	}

	fmt.Printf("Got %d responses from query\n", len(stat_data))
	return stat_data, nil
}

func (handle *MySql_handle) GetResponseById(Id int) (string, error) {
	table_str := "SELECT fullResponse from " + handle.stat_table +
		" where id=" + strconv.Itoa(Id)
	fmt.Printf("Going to query: %s\n", table_str)
	result, err := handle.db.Query(table_str)
	if err != nil {
		fmt.Println("Failed to query database: " + err.Error())
		return "", err
	}

	var full_result string

	result.Next()
	err = result.Scan(&full_result)
	if err != nil {
		fmt.Println("Failed to scan result: " + err.Error())
		return "", err
	}

	fmt.Printf("Got response for id:%d - %s\n", Id, full_result)
	return full_result, nil
}

func (handle *MySql_handle) Updatestat(query string, category string,
	num_responses int, full_response string) error {

	table_str := "INSERT into " + handle.stat_table
	table_str += " (query, category, numResponses, fullResponse) VALUES ("
	table_str += "\"" + query + "\", \"" + category + "\", \"" +
		strconv.Itoa(num_responses) + "\", \"" + full_response +
		"\")"
	fmt.Printf("Executing query %s\n", table_str)

	_, err := handle.db.Exec(table_str)
	if err != nil {
		fmt.Println(err.Error())
	}

	return err
}
