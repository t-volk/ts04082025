package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"text/template"
	"time"
)

const (
	MaxTasks   int = 3
	MaxObjects int = 3
)

type TObject struct {
	Url    string
	Path   string
	Name   string
	Status string
}

type TTask struct {
	Objects []TObject
	ZipName string
	Number  string
}

type ServerHandler struct {
	Wg         *sync.WaitGroup
	M          *sync.Mutex
	FilesDir   string
	Tasks      []TTask
	NextTask   int
	NextObject int
}

type FileType struct {
	ContentType string
	Extension   string
}

// handle http requests
func (h *ServerHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	err := errors.New("error: Resource not found")

	path := r.URL.Path

	//for example, "/"
	if regexp.MustCompile(`^/$`).MatchString(path) {
		err = h.CheckMethod("GET", r)
	}

	//for example, "/task"
	if regexp.MustCompile(`^/task/?$`).MatchString(path) {
		err = h.CheckMethod("POST", r)
		if err == nil {
			err = h.NewTask()
		}
	}

	//for example, "/task/3"
	if regexp.MustCompile(`^/task/[0-9]+/?$`).MatchString(path) {

		err = h.CheckMethod("POST", r)
		if err == nil {
			tasknumber := regexp.MustCompile(`[0-9]+`).FindString(path)

			r.ParseForm()
			oper := r.Form.Get("oper")

			switch oper {
			case "add":
				u := r.Form.Get("url")
				err = h.NewFile(u, tasknumber)
			case "del":
				err = errors.New("error: Method with this parameter not released")
			default:
				err = errors.New("error: Incorrect use of the method")
			}
		}
	}

	//for example, "/files/file-234897234.zip"
	if regexp.MustCompile(`^/files/file-[0-9]+\.zip$`).MatchString(path) {
		//		h.FileHandler(w, r)
		return
	}

	if regexp.MustCompile(`^/stop/?$`).MatchString(path) {
		err = h.CheckMethod("POST", r)
		if err == nil {
			err = errors.New("server: The server will be stopped. Goodbye")
			h.StopServer()
		}
	}

	h.ViewUI(w, err)
}

// view html page with template
func (h *ServerHandler) ViewUI(w http.ResponseWriter, err error) {
	type ViewData struct {
		Title  string
		Status string
		Tasks  *[]TTask
	}

	data := ViewData{
		Title:  "Test server",
		Status: fmt.Sprint(err),
		Tasks:  &h.Tasks,
	}

	tmpl, err := template.ParseFiles("ui/home.html")
	if err != nil {
		log.Print(err)
		return
	}

	tmpl.Execute(w, data)
}

// stopping the server
func (h *ServerHandler) StopServer() {
	h.Wg.Done()
}

// adding a new task
func (h *ServerHandler) NewTask() error {

	if len(h.Tasks) == MaxTasks {
		return errors.New("server: Exceeding the maximum number of tasks (server is busy)")
	} else {
		var task TTask

		task.Number = fmt.Sprint(h.NextTask)
		h.M.Lock()
		h.Tasks = append(h.Tasks, task)
		h.NextTask++
		h.M.Unlock()

		return nil
	}
}

func (h *ServerHandler) NewFile(objUrl string, tasknumber string) error {

	var task *TTask
	for i := 0; i < len(h.Tasks); i++ {
		if h.Tasks[i].Number == tasknumber {
			task = &h.Tasks[i]
		}
	}

	if len(task.Objects) == MaxObjects {
		return errors.New("server: Exceeding the maximum number of objects")
	}

	u := strings.Trim(objUrl, " ")

	res, err := http.Head(u)

	if err != nil {
		log.Println(err)
		return errors.New("server: Error receiving file information")
	}

	TypeList := []FileType{
		{ContentType: "application/pdf", Extension: ".pdf"},
		{ContentType: "image/jpeg", Extension: ".jpg"},
	}

	conType := res.Header.Get("Content-Type")

	ext := ""
	for i := 0; i < len(TypeList); i++ {
		if conType == TypeList[i].ContentType {
			ext = TypeList[i].Extension
			break
		}
	}

	if ext == "" {
		return errors.New("server: Object has not good file type")
	}

	h.Wg.Add(1)
	defer h.Wg.Done()

	file, err := os.CreateTemp(h.FilesDir, "file-task-"+tasknumber+"-*"+ext)
	if err != nil {
		return errors.New("server: File creation error")
	}
	file.Close()

	var object = TObject{
		Url:  u,
		Path: file.Name(),
	}

	object.Name = path.Base(object.Path)

	f, err := os.OpenFile(object.Path, os.O_WRONLY, 0666)

	if err != nil {
		return err
	}

	defer f.Close()

	res, err = http.Get(object.Url)

	if err != nil {
		return err
	}

	defer res.Body.Close()

	_, err = io.Copy(f, res.Body)

	if err != nil {
		return err
	}

	h.M.Lock()
	task.Objects = append(task.Objects, object)
	h.M.Unlock()

	return nil
}

func (h *ServerHandler) CheckMethod(sampleMethod string, r *http.Request) error {
	if r.Method == sampleMethod {
		return nil
	} else {
		return errors.New("the " + r.Method + " method is not used")
	}
}

func main() {

	var wg sync.WaitGroup
	var m sync.Mutex

	filesDir, err := os.MkdirTemp(os.TempDir(), "temp-dir-*")
	if err != nil {
		log.Fatal(err)
	}

	var tasks []TTask

	сompoundHandler := ServerHandler{
		Wg:         &wg,
		M:          &m,
		FilesDir:   filesDir,
		Tasks:      tasks,
		NextTask:   1,
		NextObject: 1,
	}

	server := http.Server{
		Addr:              ":8080",
		ReadTimeout:       30 * time.Second,
		ReadHeaderTimeout: 30 * time.Second,
		Handler:           &сompoundHandler,
	}

	wg.Add(1)
	go func() {
		wg.Wait()
		os.RemoveAll(filesDir)
		server.Close()
	}()

	log.Println("Server will be started at 127.0.0.1:8080...")
	log.Fatal(server.ListenAndServe())
}
