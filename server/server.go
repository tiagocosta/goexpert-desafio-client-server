package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Cotacao struct {
	USDBRL struct {
		Code       string `json:"code"`
		Codein     string `json:"codein"`
		Name       string `json:"name"`
		High       string `json:"high"`
		Low        string `json:"low"`
		VarBid     string `json:"varBid"`
		PctChange  string `json:"pctChange"`
		Bid        string `json:"bid"`
		Ask        string `json:"ask"`
		Timestamp  string `json:"timestamp"`
		CreateDate string `json:"create_date"`
	}
}

const (
	DATABASE_TYPE string = "sqlite3"
	DATABASE_NAME string = "cotacao.db"
)

func main() {
	db, err := sql.Open(DATABASE_TYPE, DATABASE_NAME)

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	criaBanco(db)

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", CotacaoHandler)
	http.ListenAndServe(":8080", mux)

	// var version string
	// err = db.QueryRow("SELECT SQLITE_VERSION()").Scan(&version)

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// fmt.Println(version)

	// rows, err := db.Query("SELECT * FROM cars")

	// if err != nil {
	// 	log.Fatal(err)
	// }

	// defer rows.Close()

	// for rows.Next() {

	// 	var id int
	// 	var name string
	// 	var price int

	// 	err = rows.Scan(&id, &name, &price)

	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}

	// 	fmt.Printf("%d %s %d\n", id, name, price)
	// }
}

func CotacaoHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/cotacao" {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	ctxReq, cancelReq := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancelReq()
	err := monitoraTimeoutRequest(ctxReq)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	cotacao, err := BuscaCotacao()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	ctxDB, cancelDB := context.WithTimeout(r.Context(), 10*time.Millisecond)
	defer cancelDB()
	err = monitoraTimeoutDB(ctxDB)
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}

	bid, err := strconv.ParseFloat(cotacao.USDBRL.Bid, 64)
	if err != nil {
		w.Write([]byte("invalid value for bid"))
		return
	}

	db, err := sql.Open(DATABASE_TYPE, DATABASE_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	insertCotacao(db, bid, cotacao.USDBRL.CreateDate)
	selectAll(db)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(bid)
}

func BuscaCotacao() (*Cotacao, error) {
	resp, err := http.Get("https://economia.awesomeapi.com.br/json/last/USD-BRL")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var cotacao Cotacao
	err = json.Unmarshal(body, &cotacao)
	if err != nil {
		return nil, err
	}

	return &cotacao, nil
}

func monitoraTimeoutRequest(ctx context.Context) error {
	log.Println("request started")
	defer log.Println("request ended")
	select {
	case <-ctx.Done():
		log.Println("request timeout")
		return fmt.Errorf("request timeout")
	case <-time.After(200 * time.Millisecond):
		log.Println("request ok")
		return nil
	}
}

func monitoraTimeoutDB(ctx context.Context) error {
	log.Println("request started")
	defer log.Println("request ended")
	select {
	case <-ctx.Done():
		log.Println("db timeout")
		return fmt.Errorf("db timeout")
	case <-time.After(10 * time.Millisecond):
		log.Println("db operation ok")
		return nil
	}
}

func criaBanco(db *sql.DB) error {
	stmt := `
		DROP TABLE IF EXISTS cotacao;
		CREATE TABLE cotacao(id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT, bid REAL, created_at TEXT);
		`
	_, err := db.Exec(stmt)

	if err != nil {
		return err
	}

	return nil
}

func insertCotacao(db *sql.DB, bid float64, createdAt string) error {
	stmt, err := db.Prepare("insert into cotacao(name, bid, created_at) values(?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()
	_, err = stmt.Exec("USD-BRL", bid, createdAt)
	if err != nil {
		return err
	}
	return nil
}

func selectAll(db *sql.DB) {
	rows, err := db.Query("SELECT * FROM cotacao")

	if err != nil {
		log.Fatal(err)
	}

	defer rows.Close()

	for rows.Next() {

		var id int
		var name string
		var bid float64
		var createdAt string

		err = rows.Scan(&id, &name, &bid, &createdAt)

		if err != nil {
			log.Fatal(err)
		}

		fmt.Printf("%d %s %.4f %s\n", id, name, bid, createdAt)
	}
}
