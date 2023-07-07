package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)
	if err != nil {
		panic(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Println("time limit exceeded")
		return
	}
	defer res.Body.Close()

	b, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatalln(err)
	}

	f, err := os.Create("cotacao.txt")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	replacer := strings.NewReplacer("\n", "")
	bid, err := strconv.ParseFloat(replacer.Replace(string(b)), 64)
	if err != nil {
		panic(err)
	}

	_, err = f.Write([]byte("Dólar: {" + fmt.Sprintf("%f", bid) + "}"))
	if err != nil {
		panic(err)
	}
}
