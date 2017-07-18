package main

import (
	"encoding/json"
	"html/template"
	"log"
	"net/http"
	"strconv"

	// Logging
	"github.com/unrolled/logger"

	// Stats/Metrics
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/exp"
	"github.com/thoas/stats"

	"github.com/GeertJohan/go.rice"
	"github.com/NYTimes/gziphandler"
	"github.com/julienschmidt/httprouter"
)

// Counters ...
type Counters struct {
	r metrics.Registry
}

func NewCounters() *Counters {
	counters := &Counters{
		r: metrics.NewRegistry(),
	}
	return counters
}

func (c *Counters) Inc(name string) {
	metrics.GetOrRegisterCounter(name, c.r).Inc(1)
}

func (c *Counters) Dec(name string) {
	metrics.GetOrRegisterCounter(name, c.r).Dec(1)
}

func (c *Counters) IncBy(name string, n int64) {
	metrics.GetOrRegisterCounter(name, c.r).Inc(n)
}

func (c *Counters) DecBy(name string, n int64) {
	metrics.GetOrRegisterCounter(name, c.r).Dec(n)
}

// Server ...
type Server struct {
	bind      string
	templates *Templates
	router    *httprouter.Router

	// Logger
	logger *logger.Logger

	// Stats/Metrics
	counters *Counters
	stats    *stats.Stats
}

func (s *Server) render(name string, w http.ResponseWriter, ctx interface{}) {
	buf, err := s.templates.Exec(name, ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	_, err = buf.WriteTo(w)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type TemplateContext struct {
	TodoList []*Todo
}

// IndexHandler ...
func (s *Server) IndexHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		s.counters.Inc("n_index")

		var todoList []*Todo
		query := db.Select().Reverse().OrderBy("Done")
		err := query.Find(&todoList)
		if err != nil && err.Error() != "not found" {
			log.Printf("error fetching todos: %s", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		ctx := &TemplateContext{
			TodoList: todoList,
		}

		s.render("index", w, ctx)
	}
}

// AddHandler ...
func (s *Server) AddHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.counters.Inc("n_add")

		todo := NewTodo(r.FormValue("title"))
		err := db.Save(todo)
		if err != nil {
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// DoneHandler ...
func (s *Server) DoneHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		s.counters.Inc("n_done")

		var id string

		id = p.ByName("id")
		if id == "" {
			id = r.FormValue("id")
		}

		if id == "" {
			log.Printf("no id specified to mark as done: %s", id)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		i, err := strconv.ParseInt(id, 10, 64)
		if err != nil {
			log.Printf("error parsing id %s: %s", id, err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		var todo Todo
		err = db.One("ID", i, &todo)
		if err != nil {
			log.Printf("error looking up todo %d: %s", i, err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		todo.ToggleDone()
		err = db.Save(&todo)
		if err != nil {
			log.Printf("error saving changes to todo %d: %s", i, err)
			http.Error(w, "Internal Error", http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusFound)
	}
}

// StatsHandler ...
func (s *Server) StatsHandler() httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		bs, err := json.Marshal(s.stats.Data())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Write(bs)
	}
}

// ListenAndServe ...
func (s *Server) ListenAndServe() {
	log.Fatal(
		http.ListenAndServe(
			s.bind,
			s.logger.Handler(
				s.stats.Handler(
					gziphandler.GzipHandler(
						s.router,
					),
				),
			),
		),
	)
}

func (s *Server) initRoutes() {
	s.router.Handler("GET", "/debug/metrics", exp.ExpHandler(s.counters.r))
	s.router.GET("/debug/stats", s.StatsHandler())

	s.router.GET("/", s.IndexHandler())
	s.router.POST("/add", s.AddHandler())
	s.router.POST("/done/:id", s.DoneHandler())
}

// NewServer ...
func NewServer(bind string) *Server {
	server := &Server{
		bind:      bind,
		router:    httprouter.New(),
		templates: NewTemplates("base"),

		// Logger
		logger: logger.New(logger.Options{
			Prefix:               "todo",
			RemoteAddressHeaders: []string{"X-Forwarded-For"},
			OutputFlags:          log.LstdFlags,
		}),

		// Stats/Metrics
		counters: NewCounters(),
		stats:    stats.New(),
	}

	// Templates
	box := rice.MustFindBox("templates")

	indexTemplate := template.New("index")
	template.Must(indexTemplate.Parse(box.MustString("index.html")))
	template.Must(indexTemplate.Parse(box.MustString("base.html")))

	server.templates.Add("index", indexTemplate)

	server.initRoutes()

	return server
}
