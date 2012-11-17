/*
	The web package contains all the code to provide Inbucket's web GUI
*/
package web

import (
	"fmt"
	"github.com/goods/httpbuf"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jhillyerd/inbucket/config"
	"github.com/jhillyerd/inbucket/log"
	"net"
	"net/http"
	"time"
)

type handler func(http.ResponseWriter, *http.Request, *Context) error

var Router *mux.Router
var listener net.Listener
var sessionStore sessions.Store
var shutdown bool

func setupRoutes(cfg config.WebConfig) {
	log.Info("Theme templates mapped to '%v'", cfg.TemplateDir)
	log.Info("Theme static content mapped to '%v'", cfg.PublicDir)

	r := mux.NewRouter()
	// Static content
	r.PathPrefix("/public/").Handler(http.StripPrefix("/public/",
		http.FileServer(http.Dir(cfg.PublicDir))))

	// Root
	r.Path("/").Handler(handler(RootIndex)).Name("RootIndex").Methods("GET")
	r.Path("/status").Handler(handler(RootStatus)).Name("RootStatus").Methods("GET")
	r.Path("/mailbox").Handler(handler(MailboxIndex)).Name("MailboxIndex").Methods("GET")
	r.Path("/mailbox/list/{name}").Handler(handler(MailboxList)).Name("MailboxList").Methods("GET")
	r.Path("/mailbox/show/{name}/{id}").Handler(handler(MailboxShow)).Name("MailboxShow").Methods("GET")
	r.Path("/mailbox/html/{name}/{id}").Handler(handler(MailboxHtml)).Name("MailboxHtml").Methods("GET")
	r.Path("/mailbox/source/{name}/{id}").Handler(handler(MailboxSource)).Name("MailboxSource").Methods("GET")
	r.Path("/mailbox/delete/{name}/{id}").Handler(handler(MailboxDelete)).Name("MailboxDelete").Methods("POST")
	r.Path("/mailbox/dattach/{name}/{id}/{num}/{file}").Handler(handler(MailboxDownloadAttach)).Name("MailboxDownloadAttach").Methods("GET")
	r.Path("/mailbox/vattach/{name}/{id}/{num}/{file}").Handler(handler(MailboxViewAttach)).Name("MailboxViewAttach").Methods("GET")

	// Register w/ HTTP
	Router = r
	http.Handle("/", Router)
}

// Start() the web server
func Start() {
	cfg := config.GetWebConfig()
	setupRoutes(cfg)

	// TODO Make configurable
	sessionStore = sessions.NewCookieStore([]byte("something-very-secret"))

	addr := fmt.Sprintf("%v:%v", cfg.Ip4address, cfg.Ip4port)
	server := &http.Server{
		Addr:         addr,
		Handler:      nil,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	// We don't use ListenAndServe because it lacks a way to close the listener
	log.Info("HTTP listening on TCP4 %v", addr)
	var err error
	listener, err = net.Listen("tcp", addr)
	if err != nil {
		log.Error("HTTP failed to start TCP4 listener: %v", err)
		// TODO More graceful early-shutdown procedure
		panic(err)
	}

	err = server.Serve(listener)
	if shutdown {
		log.Trace("HTTP server shutting down on request")
	} else if err != nil {
		log.Error("HTTP server failed: %v", err)
	}
}

func Stop() {
	log.Trace("HTTP shutdown requested")
	shutdown = true
	if listener != nil {
		listener.Close()
	} else {
		log.Error("HTTP listener was nil during shutdown")
	}
}

// ServeHTTP builds the context and passes onto the real handler
func (h handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Create the context
	ctx, err := NewContext(req)
	if err != nil {
		log.Error("Failed to create context: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer ctx.Close()

	// Run the handler, grab the error, and report it
	buf := new(httpbuf.Buffer)
	log.Trace("Web: %v %v %v %v", req.RemoteAddr, req.Proto, req.Method, req.RequestURI)
	err = h(buf, req, ctx)
	if err != nil {
		log.Error("Error handling %v: %v", req.RequestURI, err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Save the session
	if err = ctx.Session.Save(req, buf); err != nil {
		log.Error("Failed to save session: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Apply the buffered response to the writer
	buf.Apply(w)
}
