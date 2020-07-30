package main

import (
	"errors"
	"flag"
	"fmt"
	log "github.com/sirupsen/logrus"
	"html/template"
	"net/http"
	"os"
	"path"
	"regexp"
	"runtime"
	"strings"
)

var Repos *UrlRepository

var HostName string = "127.0.0.1:8080"
var Proto string = "http"
var ShortLen int = 5		// 建议至少 4 位

var RedisAddr string = ":6379"

var ListenAddr string

func init() {
	flag.StringVar(&ListenAddr, "l", ":8080", "监听端口")
	flag.Parse()
}

func init() {
	log.SetFormatter(&log.TextFormatter{
		ForceColors:     true,
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			return fmt.Sprintf("[%s]", f.Function), fmt.Sprintf("[%s:%d]", path.Base(f.File), f.Line)
		},
	})
	log.SetOutput(os.Stdout)
	log.SetLevel(log.TraceLevel)
	log.SetReportCaller(true)
}

func init() {
	Repos = NewUrlRepository(ShortLen, RedisAddr)
}

func main() {
	http.HandleFunc("/", Index)
	http.HandleFunc("/new", Store)
	http.HandleFunc("/del", Destroy)

	err := http.ListenAndServe(ListenAddr, nil)
	if err != nil {
		log.Fatal(err)
	}
}

func Index(w http.ResponseWriter, r *http.Request) {
	// 首页
	if r.URL.Path == "/" {
		t := template.Must(template.ParseFiles("view/index.html"))
		err := t.Execute(w, nil)
		if err != nil {
			log.WithField("err", err).Warn("index template execute err")
		}
		return
	}

	// 非短链
	if strings.Contains(r.URL.Path[1:], "/") {
		http.NotFound(w, r)
	}

	// 重定向
	redirect(w, r)
}

func redirect(w http.ResponseWriter, r *http.Request) {
	short := r.URL.Path[1:]

	// 获取对应长链
	long, err := Repos.Get(short)
	if err != nil {
		if err != KeyNotFound {
			log.WithFields(log.Fields{"short": short, "err": err}).Warn("repos get long url err")
		} else {
			log.Trace("short url not match: " + short)
		}
		http.NotFound(w, r)
		return
	}

	log.Traceln("short url match: " + short)

	// 重定向
	http.Redirect(w, r, long, 302)
}

func Store(w http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	long, err := parseLongUrl(r.PostFormValue("long"))
	if err != nil {
		http.Error(w, `invalid long url`, http.StatusBadRequest)
		return
	}

	short, err := Repos.Put(long)
	if err != nil {
		http.Error(w, `generate error`, http.StatusInternalServerError)
		return
	}

	log.Trace("store short url: ", short)

	fmt.Fprintf(w, "%s://%s/%s", Proto, HostName, short)
}

// 解析并验证长 url
func parseLongUrl(long string) (string, error) {
	long = strings.TrimSpace(long)

	if len(long) <= 3 {
		return "", errors.New("invalid long url")
	}

	if !strings.HasPrefix(long, "http") {
		long = "http://" + long
	}

	return long, nil
}

func Destroy(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		t := template.Must(template.ParseFiles("view/short_destroy.html"))
		t.Execute(w, nil)
	} else {
		r.ParseForm()
		short := strings.TrimSpace(r.PostFormValue("short"))
		err := validShortUrl(short)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		err = Repos.Delete(short)
		if err != nil {
			log.Warn("delete key error: ", err)
			return
		}
		fmt.Fprint(w, "ok")
	}
}

func validShortUrl(short string) error {
	reg := regexp.MustCompile(`[0-9a-zA-Z]{4,10}`)
	if !reg.MatchString(short) {
		return errors.New("invalid short url")
	}
	return nil
}