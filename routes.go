package main

/*

ideas from:

https://github.com/matryer/goblueprints

https://semaphoreci.com/community/tutorials/building-and-testing-a-rest-api-in-go-with-gorilla-mux-and-postgresql

*/

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	//AppVersion is the app version number
	AppVersion = "1.3"
	CT_JSON    = "application/json"
)

// Myserver is a MS server
type Myserver struct {
	db     *sql.DB
	router *http.ServeMux
	//email  EmailSender
}

// NewHTTPServer makes a new  HTTP service server.
func NewHTTPServer(ctx context.Context, dbPath string) *Myserver {

	s := Myserver{router: http.NewServeMux()}
	s.initDb(dbPath)
	// init routes
	s.routes()

	// This will serve files under http://localhost:5000/static/<filename>
	fileDir := "./static"
	fs := http.FileServer(http.Dir(fileDir))
	s.router.Handle("/static/", http.StripPrefix("/static/", fs))
	return &s
}

func (s *Myserver) handler() http.Handler {
	return s.router
}

func (s *Myserver) initDb(dbPath string) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("[WARN] opened DB: ", dbPath)
	s.db = db
}

func (s *Myserver) terminate() {
	log.Println("[WARN] closing DB")
	err := s.db.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func (s *Myserver) routes() {
	s.router.HandleFunc("/", s.handleIndex())
	s.router.HandleFunc("/about", s.handleAbout(AppVersion))
	s.router.HandleFunc("/health", s.handleHealth())
	s.router.HandleFunc("/api/cli", s.handleAPI1())
	s.router.HandleFunc("/api/bonus/", s.handleBonus())
	s.router.HandleFunc("/form1", s.handleForm1())
	s.router.HandleFunc("/page1", s.handlePage1())
	s.router.HandleFunc("/chart1", s.handleChart1())
	///s.router.HandleFunc("/admin", s.adminOnly(s.handleAdminIndex()))
}

func (s *Myserver) handleAbout(version string) http.HandlerFunc {
	///thing := prepareThing()
	return func(w http.ResponseWriter, r *http.Request) {
		// use thing
		fmt.Fprintf(w, "About MS-TAC2 app version %s\r\n", version)
		fmt.Fprintf(w, "GO version %s %s/%s\r\n", runtime.Version(), runtime.GOOS, runtime.GOARCH)
		var ver, dt string
		s.db.QueryRow("select sqlite_version() as v, datetime('now','localtime') as dt").Scan(&ver, &dt)
		fmt.Fprintf(w, "SQLITE Version: %s; Now(): %s\r\n", ver, dt)
	}
}

func (s *Myserver) handleIndex() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Main Index page: %v\r\n", time.Now())
	}
}

func (s *Myserver) handleHealth() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", CT_JSON)
		s := `{"status": "UP","now": "` + time.Now().Format("2006-01-02T15:04:05.999Z") + `"}`
		w.Write([]byte(s))
	}
}

// Logger middlerware that logs time taken to process each request
//  use:  s.router.HandleFunc("/xpto", s.Logger(someHandler))
func (s *Myserver) Logger(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		h.ServeHTTP(w, r)
		endTime := time.Since(startTime)
		log.Printf("%s %d %v", r.URL, r.Method, endTime)
	})
}

func (s *Myserver) handleBonus() http.HandlerFunc {
	myDb := s.db
	type Bonus struct {
		Ename string `json:"ename"`
		Job   string `json:"job"`
		Sal   int    `json:"sal"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", CT_JSON)
		id := ""
		ar := strings.SplitN(r.URL.Path, "/", 4) // /api/bonus/{id}, or pId := r.URL.Query().Get("id")
		if len(ar) > 2 {
			id = ar[3]
			log.Printf("[INFO] id: %s ", id)
		}

		if id != "" && id[0] != '?' {
			var tempBn Bonus
			err := myDb.QueryRow("SELECT ename, job, sal FROM bonus where ename = ?", id).Scan(&tempBn.Ename, &tempBn.Job, &tempBn.Sal)
			if err != nil {
				if err == sql.ErrNoRows {
					w.WriteHeader(http.StatusNotFound)
					response, _ := json.Marshal(map[string]string{"error": "Product not found"})
					w.Write(response)
					return
				}
				log.Printf("[WARN] Query row error: %v", err)
				w.Write([]byte("{}"))
				return
			} else {
				log.Printf("[INFO] Ename:%s, Job:%s, Sal:%d\n", tempBn.Ename, tempBn.Job, tempBn.Sal)
			}
			json.NewEncoder(w).Encode(tempBn)
			return
		}

		rows, err := myDb.Query("SELECT ename, job, sal FROM bonus")
		myBonus := []Bonus{}

		for rows.Next() {
			var tempBn Bonus
			rows.Scan(&tempBn.Ename, &tempBn.Job, &tempBn.Sal)
			log.Printf("[INFO] Ename:%s, Job:%s, Sal:%d\n", tempBn.Ename, tempBn.Job, tempBn.Sal)
			myBonus = append(myBonus, tempBn)
		}
		rows.Close()

		if err = rows.Err(); err != nil {
			log.Printf("[WARN] Query rows error: %v", err)
		}

		json.NewEncoder(w).Encode(myBonus)
	}
}

func (s *Myserver) handleAPI1() http.HandlerFunc {
	type myRequest struct {
		Name string
	}
	type myResponse struct {
		ID       int    `json:"id"`
		Greeting string `json:"greeting"`
	}

	return func(w http.ResponseWriter, r *http.Request) {
		res := myResponse{
			Greeting: "Mat Ryer",
			ID:       int(time.Now().Unix()), // secs form Epoch

		}
		w.Header().Set("Content-Type", CT_JSON)
		json.NewEncoder(w).Encode(res)
	}
}

/*
// middleware funcs

func (s *Myserver) adminOnly(h http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        if !currentUser(r).IsAdmin {
            http.NotFound(w, r)
            return
        }
        h(w, r)
    }
}

// notFound is handled by setting the status code in the reply to StatusNotFound. return badRequest{err}; return notFound{}
type notFound struct{ error }

// errorHandler wraps a function returning an error by handling the error and returning a http.Handler.
// If the error is of the one of the types defined above, it is handled as described for every type.
// If the error is of another type, it is considered as an internal error and its message is logged.
func errorHandler(f func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		err := f(w, r)
		if err == nil {
			return
		}
		switch err.(type) {
		case badRequest:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case notFound:
			http.Error(w, "task not found", http.StatusNotFound)
		default:
			log.Println(err)
			http.Error(w, "oops", http.StatusInternalServerError)
		}
	}
}
*/

func (s *Myserver) showTemplate(w http.ResponseWriter, req *http.Request, fp string, tdata interface{}) {

	lp := filepath.Join("templates", "layout.html")

	// Return a 404 if the template doesn't exist
	info, err := os.Stat(fp)
	if err != nil {
		if os.IsNotExist(err) {
			http.NotFound(w, req)
			return
		}
	}

	// Return a 404 if the request is for a directory
	if info.IsDir() {
		http.NotFound(w, req)
		return
	}

	tmpl, err := template.ParseFiles(lp, fp)
	if err != nil {
		// Log the detailed error
		log.Println("[ERROR]", err.Error())
		// Return a generic "Internal Server Error" message
		http.Error(w, http.StatusText(501), 501)
		return
	}

	if err := tmpl.ExecuteTemplate(w, "layout", tdata); err != nil {
		log.Println("[ERROR]", err.Error())
		http.Error(w, http.StatusText(500), 500)
	}
}

func (s *Myserver) handleForm1() http.HandlerFunc {
	type person struct {
		Name  string
		Phone string
		Age   int
	}

	return func(w http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {

			fp := filepath.Join("templates", "form1.html")
			log.Printf("[DEBUG] Serving %s\n", fp)

			p1 := person{Name: "John", Phone: "123-456", Age: 33}
			s.showTemplate(w, req, fp, p1)
			return
		}

		err := req.ParseForm()
		if err != nil {
			log.Fatal(err)
		}
		age, err := strconv.Atoi(req.FormValue("age"))
		if err != nil {
			log.Println("[WARN] bad value for Age in Form,", err)
		}

		person1 := person{
			Name:  req.FormValue("name"),
			Phone: req.FormValue("phone"),
			Age:   age,
		}

		log.Printf("[INFO] person=%v", person1)

		w.Header().Set("Content-Type", CT_JSON)
		json.NewEncoder(w).Encode(person1)
	}
}

func (s *Myserver) handlePage1() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		file1 := "page1.html"
		fp := filepath.Join("templates", file1)
		log.Printf("[DEBUG] Serving %s\n", fp)
		s.showTemplate(w, req, fp, nil)
	}
}

func (s *Myserver) handleChart1() http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		type tVars struct {
			Label1 string
			Vals1  []int
			Vals2  []int
		}
		myVars := tVars{
			Label1: "Receitas",
			Vals1:  []int{10, 9, 8, 7, 6, 4, 7, 8},
			Vals2:  []int{1, 3, 6, 5, 4, 2, 3, 5},
		}

		file1 := "chart1.html"
		fp := filepath.Join("templates", file1)
		log.Printf("[DEBUG] Serving %s\n", fp)
		log.Printf("[DEBUG] With %v\n", myVars)
		s.showTemplate(w, req, fp, myVars)
	}
}
