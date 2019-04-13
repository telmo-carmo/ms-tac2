/*

A simple Micro Service App
---

https://github.com/PacktPublishing/Go-Programming-Blueprints

see:
  https://github.com/matryer/goblueprints/blob/master/chapter10/vault/cmd/vaultd/main.go

  https://github.com/PacktPublishing/Go-Programming-Blueprints/blob/master/Chapter11/vault/cmd/vaultd/main.go

---

also runs on IBM Cloud with GO runtime

https://ms-tac2.eu-gb.mybluemix.net/api/bonus/
https://ms-tac2.eu-gb.mybluemix.net/about

*/
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

const (
	// SvDefaultPort is default app Port
	SvDefaultPort = "5000"
)

func getDbPath() string {
	dbPath := os.Getenv("LOCAL_DB_PATH")
	if dbPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		dbPath = filepath.Join(cwd, "sqlite_scott.db")
	}
	return dbPath
}

func main() {
	var (
		port     string
		httpAddr = flag.String("addr", "", "http listen address :nnnn")
		//gRPCAddr = flag.String("grpc", ":5001", "gRPC listen address")
	)

	//IBM Cloud and GAE sets env vars like: PORT 	The port that receives HTTP requests.
	if port = os.Getenv("PORT"); len(port) == 0 {
		port = SvDefaultPort
	}
	addr := ":" + port

	flag.Parse()
	if *httpAddr != "" {
		addr = *httpAddr
	}

	log.SetFlags(log.Ltime | log.Lshortfile)

	svr := NewHTTPServer(context.Background(), getDbPath())
	//defer svr.terminate()

	errChan := make(chan error)
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		errChan <- fmt.Errorf("SIGNAL: %s", <-c)
		svr.terminate()
	}()

	// HTTP transport serve
	go func() {
		log.Printf("[INFO] Listening at %s\n", addr)
		errChan <- http.ListenAndServe(addr, svr.handler())
	}()

	log.Fatalln(<-errChan)

}
