package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type CusHandler struct {
}

var mux map[string]func(http.ResponseWriter, *http.Request)
var port int
var db *sql.DB

type channel struct {
	Types string `json:"type"`
	Url   string `json:"url"`
	Name  string `json:"name"`
}
type Resp struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}
type first struct {
	Name  string   `json:"name"`
	Child []second `json:"child"`
}
type second struct {
	Name  string    `json:"name"`
	Child []channel `json:"child"`
}

func (h CusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h, ok := mux[r.URL.String()]; ok {
		//Implement route forwarding with this handler, the corresponding route calls the corresponding func.
		h(w, r)
		return
	} else {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("file not found"))
		return
	}
	io.WriteString(w, "URL: "+r.URL.String())
}

func init() {
	getConfig()
	dbConf := conf.Mysql.Default
	db, _ = sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", dbConf.User, dbConf.Password, dbConf.Host, dbConf.Port, dbConf.DbName))
	db.SetConnMaxLifetime(time.Second * 60)
	db.SetMaxIdleConns(dbConf.MaxIdle)
	db.SetMaxOpenConns(dbConf.MaxOpen)
	if err := db.Ping(); err != nil {
		log.Fatalln("connect mysql error:", err)
	}
	//defer db.Close()
}
func main() {
	flag.IntVar(&port, "p", 8888, "端口号")
	flag.Parse()
	s := &http.Server{
		Addr:           fmt.Sprintf("%s:%d", "localhost", port),
		Handler:        &CusHandler{},
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	done := make(chan bool, 1)
	quit := make(chan os.Signal, 1)
	sign := []os.Signal{
		syscall.SIGABRT,
		syscall.SIGALRM,
		syscall.SIGBUS,
		syscall.SIGCHLD,
		syscall.SIGCLD,
		syscall.SIGCONT,
		syscall.SIGFPE,
		syscall.SIGHUP,
		syscall.SIGILL,
		syscall.SIGINT,
		syscall.SIGIO,
		syscall.SIGIOT,
		syscall.SIGKILL,
		syscall.SIGPIPE,
		syscall.SIGPOLL,
		syscall.SIGPROF,
		syscall.SIGPWR,
		syscall.SIGQUIT,
		syscall.SIGSEGV,
		syscall.SIGSTKFLT,
		syscall.SIGSTOP,
		syscall.SIGSYS,
		syscall.SIGTERM,
		syscall.SIGTRAP,
		syscall.SIGTSTP,
		syscall.SIGTTIN,
		syscall.SIGTTOU,
		syscall.SIGUNUSED,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGVTALRM,
		syscall.SIGWINCH,
		syscall.SIGXCPU,
		syscall.SIGXFSZ,
		os.Interrupt,
		os.Kill,
	}
	signal.Notify(quit, sign...)
	go graceStart()
	go graceShutdown(s, quit, done)
	mux = make(map[string]func(http.ResponseWriter, *http.Request))
	mux["/channel"] = load
	mux["/"] = index
	if err := s.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Panicf("Could not listen on %s:%d，err:%v", "localhost", port, err)
	}
	<-done
	log.Println("Shutdown Over,Server exiting,Bye")
}
func graceShutdown(server *http.Server, quit <-chan os.Signal, done chan<- bool) {
	sig := <-quit
	log.Println("signal", sig)
	log.Println("Now We Well Shutdown Server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	server.SetKeepAlivesEnabled(false)
	if err := server.Shutdown(ctx); err != nil {
		log.Panicf("Could not gracefully shutdown the server: %v\n", err)
	}
	close(done)
}

func graceStart() {
	for {
		time.Sleep(time.Second)
		log.Println("Checking if started...")
		resp, err := http.Get("http://localhost:" + fmt.Sprintf("%d", port))
		if err != nil {
			log.Println("Failed:", err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			log.Println("Not OK:", resp.StatusCode)
			continue
		}

		// Reached this point: server is up and running!
		break
	}
	log.Println("SERVER UP AND RUNNING!")
}
func load(w http.ResponseWriter, r *http.Request) {
	chann, e := getChannel()
	res := Resp{Code: 0, Msg: "success", Data: make(map[string]interface{}, 0)}
	statusCode := http.StatusOK
	if e != nil {
		res.Code = 500
		statusCode = http.StatusInternalServerError
		res.Msg = "获取列表失败"
	}
	res.Data = chann
	re, _ := json.Marshal(res)
	w.Header().Add("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	w.Write(re)
	return
}
func index(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("tv api index"))
}
func getChannel() ([]first, error) {
	result := make(map[string]map[string][]channel, 0)
	r, e := db.Query("select types,name,url from tv where url!='' and status=1 ")
	resp := make([]first, 0)

	if e != nil {
		log.Println("get channel list error ", e)
		return resp, e
	}
	var list []channel

	for r.Next() {
		var row channel
		e := r.Scan(&row.Types, &row.Name, &row.Url)
		if e != nil {
			log.Println("get channel list error", e)
			return resp, e
		}
		list = append(list, row)
	}
	firstKeys := make(map[string]string, 0)
	secondKeys := make(map[string]string, 0)
	for _, v := range list {
		if result[v.Types] == nil {
			result[v.Types] = make(map[string][]channel, 0)
			if result[v.Types][v.Name] == nil {
				result[v.Types][v.Name] = make([]channel, 0)
			}
		}
		if firstKeys[v.Types] == "" {
			firstKeys[v.Types] = v.Types
		}
		if secondKeys[v.Name] == "" {
			secondKeys[v.Name] = v.Name
		}
		result[v.Types][v.Name] = append(result[v.Types][v.Name], v)
	}
	firstKey := make([]string, 0)
	secondKey := make([]string, 0)
	for _, f := range firstKeys {
		firstKey = append(firstKey, f)
	}
	for _, s := range secondKeys {
		secondKey = append(secondKey, s)
	}
	sort.Strings(firstKey)
	sort.Strings(secondKey)
	for _, ff := range firstKey {
		fir := first{Name: ff}
		for _, ss := range secondKey {
			if result[ff][ss] != nil {
				sec := second{Name: ss, Child: result[ff][ss]}
				fir.Child = append(fir.Child, sec)
			}

		}
		resp = append(resp, fir)

	}
	return resp, nil
}
