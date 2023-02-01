package nativeapi

import (
	"context"
	"net/http"

	"github.com/deluan/rest"
	"github.com/go-chi/chi/v5"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/core"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/server"
	"github.com/navidrome/navidrome/server/events"
)

type Router struct {
	http.Handler
	ds     model.DataStore
	broker events.Broker
	share  core.Share
}

func New(ds model.DataStore, broker events.Broker, share core.Share) *Router {
	r := &Router{ds: ds, broker: broker, share: share}
	r.Handler = r.routes()
	return r
}

func (n *Router) routes() http.Handler {
	r := chi.NewRouter()

	// Public
	n.RX(r, "/translation", newTranslationRepository, false)

	// Protected
	r.Group(func(r chi.Router) {
		r.Use(server.Authenticator(n.ds))
		r.Use(server.JWTRefresher)
		n.R(r, "/user", model.User{}, true)
		n.R(r, "/song", model.MediaFile{}, false)
		n.R(r, "/album", model.Album{}, false)
		n.R(r, "/artist", model.Artist{}, false)
		n.R(r, "/genre", model.Genre{}, false)
		n.R(r, "/player", model.Player{}, true)
		n.R(r, "/playlist", model.Playlist{}, true)
		n.R(r, "/transcoding", model.Transcoding{}, conf.Server.EnableTranscodingConfig)
		n.R(r, "/radio", model.Radio{}, true)
		n.R(r, "/radioInfo", model.RadioInfo{}, false)
		n.RX(r, "/share", n.share.NewRepository, true)

		n.addPlaylistTrackRoute(r)

		if conf.Server.RadioBrowser.SendClicks {
			n.submitStation(r)
		}

		// Keepalive endpoint to be used to keep the session valid (ex: while playing songs)
		r.Get("/keepalive/*", func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`{"response":"ok", "id":"keepalive"}`))
		})

		if conf.Server.DevActivityPanel {
			r.Handle("/events", n.broker)
		}
	})

	return r
}

func (n *Router) R(r chi.Router, pathPrefix string, model interface{}, persistable bool) {
	constructor := func(ctx context.Context) rest.Repository {
		return n.ds.Resource(ctx, model)
	}
	n.RX(r, pathPrefix, constructor, persistable)
}

func (n *Router) RX(r chi.Router, pathPrefix string, constructor rest.RepositoryConstructor, persistable bool) {
	r.Route(pathPrefix, func(r chi.Router) {
		r.Get("/", rest.GetAll(constructor))
		if persistable {
			r.Post("/", rest.Post(constructor))
		}
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Get("/", rest.Get(constructor))
			if persistable {
				r.Put("/", rest.Put(constructor))
				r.Delete("/", rest.Delete(constructor))
			}
		})
	})
}

func (n *Router) addPlaylistTrackRoute(r chi.Router) {
	r.Route("/playlist/{playlistId}/tracks", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			getPlaylist(n.ds)(w, r)
		})
		r.With(server.URLParamsMiddleware).Route("/", func(r chi.Router) {
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				deleteFromPlaylist(n.ds)(w, r)
			})
			r.Post("/", func(w http.ResponseWriter, r *http.Request) {
				addToPlaylist(n.ds)(w, r)
			})
		})
		r.Route("/{id}", func(r chi.Router) {
			r.Use(server.URLParamsMiddleware)
			r.Put("/", func(w http.ResponseWriter, r *http.Request) {
				reorderItem(n.ds)(w, r)
			})
			r.Delete("/", func(w http.ResponseWriter, r *http.Request) {
				deleteFromPlaylist(n.ds)(w, r)
			})
		})
	})
}

func (n *Router) submitStation(r chi.Router) {
	r.Route("/radio/{radioId}", func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			submitClick(n.ds)
		})
	})
}
