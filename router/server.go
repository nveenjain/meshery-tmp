package router

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/layer5io/meshery/handlers"
	"github.com/layer5io/meshery/models"
)

// Router represents Meshery router
type Router struct {
	s    *mux.Router
	port int
}

// NewRouter returns a new ServeMux with app routes.
func NewRouter(ctx context.Context, h models.HandlerInterface, port int) *Router {
	router := mux.NewRouter()
	router.HandleFunc("/api/provider", h.ProviderHandler)
	router.HandleFunc("/api/providers", h.ProvidersHandler)
	router.PathPrefix("/provider/").HandlerFunc(h.ProviderUIHandler)

	router.Handle("/api/user", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.UserHandler))))
	router.Handle("/api/user/stats", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.AnonymousStatsHandler))))
	router.Handle("/api/config/sync", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.SessionSyncHandler))))

	router.Handle("/api/k8sconfig", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.K8SConfigHandler))))
	router.Handle("/api/k8sconfig/contexts", h.ProviderMiddleware(h.AuthMiddleware(http.HandlerFunc(h.GetContextsFromK8SConfig))))
	router.Handle("/api/k8sconfig/ping", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.KubernetesPingHandler))))
	router.Handle("/api/mesh/scan", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.InstalledMeshesHandler))))

	router.Handle("/api/load-test", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.LoadTestHandler))))
	router.Handle("/api/load-test-smps", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.LoadTestUsingSMPSHandler))))
	router.Handle("/api/load-test-prefs", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.LoadTestPrefencesHandler))))
	router.Handle("/api/results", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.FetchResultsHandler))))
	router.Handle("/api/result", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.GetResultHandler))))

	router.Handle("/api/mesh/manage", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.MeshAdapterConfigHandler))))
	router.Handle("/api/mesh/ops", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.MeshOpsHandler))))
	router.Handle("/api/mesh/adapters", h.ProviderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		providerI := req.Context().Value(models.ProviderCtxKey)
		provider, ok := providerI.(models.Provider)
		if !ok {
			http.Redirect(w, req, "/provider", http.StatusFound)
			return
		}
		h.GetAllAdaptersHandler(w, req, provider)
	})))
	router.Handle("/api/mesh/adapter/ping", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.AdapterPingHandler))))
	router.Handle("/api/events", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.EventStreamHandler))))

	router.Handle("/api/grafana/config", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.GrafanaConfigHandler))))
	router.Handle("/api/grafana/boards", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.GrafanaBoardsHandler))))
	router.Handle("/api/grafana/query", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.GrafanaQueryHandler))))
	router.Handle("/api/grafana/query_range", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.GrafanaQueryRangeHandler))))
	router.Handle("/api/grafana/ping", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.GrafanaPingHandler))))

	router.Handle("/api/prometheus/config", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.PrometheusConfigHandler))))
	router.Handle("/api/prometheus/board_import", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.GrafanaBoardImportForPrometheusHandler))))
	router.Handle("/api/prometheus/query", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.PrometheusQueryHandler))))
	router.Handle("/api/prometheus/query_range", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.PrometheusQueryRangeHandler))))
	router.Handle("/api/prometheus/ping", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.PrometheusPingHandler))))
	router.Handle("/api/prometheus/static_board", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.PrometheusStaticBoardHandler))))
	router.Handle("/api/prometheus/boards", h.ProviderMiddleware(h.AuthMiddleware(h.SessionInjectorMiddleware(h.SaveSelectedPrometheusBoardsHandler))))

	router.Handle("/logout", h.ProviderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		providerI := req.Context().Value(models.ProviderCtxKey)
		provider, ok := providerI.(models.Provider)
		if !ok {
			http.Redirect(w, req, "/provider", http.StatusFound)
			return
		}
		h.LogoutHandler(w, req, provider)
	})))
	router.Handle("/login", h.ProviderMiddleware(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		providerI := req.Context().Value(models.ProviderCtxKey)
		provider, ok := providerI.(models.Provider)
		if !ok {
			http.Redirect(w, req, "/provider", http.StatusFound)
			return
		}
		h.LoginHandler(w, req, provider, false)
	})))

	// TODO: have to change this too
	router.Handle("/favicon.ico", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=3600") // 1 hr
		http.ServeFile(w, r, "../ui/out/static/img/meshery-logo.png")
	}))

	router.PathPrefix("/").Handler(h.ProviderMiddleware(h.AuthMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlers.ServeUI(w, r, "", "../ui/out/")
	}))))

	return &Router{
		s:    router,
		port: port,
	}
}

// Run starts the http server
func (r *Router) Run() error {
	// s := &http.Server{
	// 	Addr:           fmt.Sprintf(":%d", r.port),
	// 	Handler:        r.s,
	// 	ReadTimeout:    5 * time.Second,
	// 	WriteTimeout:   2 * time.Minute,
	// 	MaxHeaderBytes: 1 << 20,
	// 	IdleTimeout:    0, //time.Second,
	// }
	// return s.ListenAndServe()
	return http.ListenAndServe(fmt.Sprintf(":%d", r.port), r.s)
}
