package main

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

const (
	host     = "localhost"
	port     = 5432
	user     = "clemens"
	password = "test"
	dbname   = "osm_germany_orginal"
)

func main() {
	fmt.Println("Connecting to Database " + dbname)

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable", host, port, user, password, dbname)

	db, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	err = db.Ping()
	if err != nil {
		panic(err)
	}

	fmt.Println("Successfully connected!")

	fmt.Println("")

	tables := getTables(db)

	fmt.Println("")

	for _, table := range tables {
		fmt.Println("Cachen von " + table + " vorbereiten. Zum Fortfahren Enter drücken...")

	
		fmt.Scanln()

		fmt.Println("PG stat_statements reset...")
		rows, err := db.Query("SELECT pg_stat_statements_reset()")
		if err != nil {
			log.Fatal(err)
		}
		defer rows.Close()

		fmt.Println("Nach erfolgreichem Cachen bitte Enter Taste drücken...")

		fmt.Scanln()

		singletable := make([]string, 0)
		singletable = append(singletable, table)

		execute(db, singletable)
	}

	fmt.Println("Cachen von allen Tabellen vorbereiten. Zum Fortfahren Enter drücken...")
	fmt.Scanln()
	fmt.Println("PG stat_statements reset...")
	rows, err := db.Query("SELECT pg_stat_statements_reset()")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()
	fmt.Println("Nach erfolgreichem Cachen bitte Enter Taste drücken...")
	fmt.Scanln()
	execute(db, tables)

}

func execute(db *sql.DB, tableNames []string) {
	sqlQuery := "SELECT query, calls, total_time, mean_time, max_time, min_time, rows FROM pg_stat_statements ORDER BY total_time DESC;"

	rows, err := db.Query(sqlQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	var averageTime float64 = 0
	var totalRows int64 = 0
	var totalCalls int64 = 0
	var maximalTime float64 = 0
	var minimalTime float64 = 10000000000
	var overallTotalTime float64 = 0
	queryCount := 0

	for rows.Next() {
		var query string
		var calls int64
		var totalTime float64
		var meanTime float64
		var maxTime float64
		var minTime float64
		var returnedRows int64

		if err := rows.Scan(&query, &calls, &totalTime, &meanTime, &maxTime, &minTime, &returnedRows); err != nil {
			log.Fatal(err)
		}

		contains := false
		for _, table := range tableNames {
			if strings.Contains(query, table) {
				contains = true
				break
			}
		}

		if contains {
			queryCount++
			averageTime += meanTime
			totalRows += returnedRows
			totalCalls += calls
			if maxTime > maximalTime {
				maximalTime = maxTime
			}

			if minTime < minimalTime {
				minimalTime = minTime
			}

			overallTotalTime += totalTime

			fmt.Println("Query "+strconv.Itoa(queryCount)+":", query)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("")
	fmt.Println("Totale Zeit:", overallTotalTime, "ms")
	fmt.Println("Durchschnittliche Zeit:", averageTime/float64(queryCount), "ms")
	fmt.Println("Maximale Zeit:", maximalTime, "ms")
	fmt.Println("Minimale Zeit:", minimalTime, "ms")
	fmt.Println("Zeilen:", totalRows)
	fmt.Println("Anfragen gesamt:", totalCalls)
}

func getTables(db *sql.DB) []string {
	sqlQuery := "select table_schema, table_name from information_schema.tables"

	rows, err := db.Query(sqlQuery)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	tables := make([]string, 0)

	for rows.Next() {
		var tableSchema string
		var tableName string

		if err := rows.Scan(&tableSchema, &tableName); err != nil {
			log.Fatal(err)
		}

		if tableSchema == "import" {
			tables = append(tables, tableName)
		}
	}

	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Tables:", tables)
	return tables
}
