package server

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/consts"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/navidrome/navidrome/utils"
)

func Index(ds model.DataStore, fs fs.FS) http.HandlerFunc {
	return serveIndex(ds, fs, nil)
}

func IndexWithShare(ds model.DataStore, fs fs.FS, shareInfo *model.Share) http.HandlerFunc {
	return serveIndex(ds, fs, shareInfo)
}

// Injects the config in the `index.html` template
func serveIndex(ds model.DataStore, fs fs.FS, shareInfo *model.Share) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := ds.User(r.Context()).CountAll()
		firstTime := c == 0 && err == nil

		t, err := getIndexTemplate(r, fs)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		appConfig := map[string]interface{}{
			"version":                   consts.Version,
			"firstTime":                 firstTime,
			"variousArtistsId":          consts.VariousArtistsID,
			"baseURL":                   utils.SanitizeText(strings.TrimSuffix(conf.Server.BaseURL, "/")),
			"loginBackgroundURL":        utils.SanitizeText(conf.Server.UILoginBackgroundURL),
			"welcomeMessage":            utils.SanitizeText(conf.Server.UIWelcomeMessage),
			"enableTranscodingConfig":   conf.Server.EnableTranscodingConfig,
			"enableDownloads":           conf.Server.EnableDownloads,
			"enableFavourites":          conf.Server.EnableFavourites,
			"enableStarRating":          conf.Server.EnableStarRating,
			"defaultTheme":              conf.Server.DefaultTheme,
			"defaultLanguage":           conf.Server.DefaultLanguage,
			"defaultUIVolume":           conf.Server.DefaultUIVolume,
			"enableCoverAnimation":      conf.Server.EnableCoverAnimation,
			"gaTrackingId":              conf.Server.GATrackingID,
			"losslessFormats":           strings.ToUpper(strings.Join(consts.LosslessFormats, ",")),
			"devActivityPanel":          conf.Server.DevActivityPanel,
			"enableUserEditing":         conf.Server.EnableUserEditing,
			"devEnableShare":            conf.Server.DevEnableShare,
			"devSidebarPlaylists":       conf.Server.DevSidebarPlaylists,
			"lastFMEnabled":             conf.Server.LastFM.Enabled,
			"lastFMApiKey":              conf.Server.LastFM.ApiKey,
			"devShowArtistPage":         conf.Server.DevShowArtistPage,
			"listenBrainzEnabled":       conf.Server.ListenBrainz.Enabled,
			"enableReplayGain":          conf.Server.EnableReplayGain,
			"defaultDownsamplingFormat": conf.Server.DefaultDownsamplingFormat,
		}
		if strings.HasPrefix(conf.Server.UILoginBackgroundURL, "/") {
			appConfig["loginBackgroundURL"] = path.Join(conf.Server.BaseURL, conf.Server.UILoginBackgroundURL)
		}
		auth := handleLoginFromHeaders(ds, r)
		if auth != nil {
			appConfig["auth"] = auth
		}
		appConfigJson, err := json.Marshal(appConfig)
		if err != nil {
			log.Error(r, "Error converting config to JSON", "config", appConfig, err)
		} else {
			log.Trace(r, "Injecting config in index.html", "config", string(appConfigJson))
		}

		shareInfoJson := marshalShareData(r.Context(), shareInfo)

		log.Debug("UI configuration", "appConfig", appConfig)
		version := consts.Version
		if version != "dev" {
			version = "v" + version
		}
		data := map[string]interface{}{
			"AppConfig": string(appConfigJson),
			"ShareInfo": string(shareInfoJson),
			"Version":   version,
		}
		w.Header().Set("Content-Type", "text/html")
		err = t.Execute(w, data)
		if err != nil {
			log.Error(r, "Could not execute `index.html` template", err)
		}
	}
}

func getIndexTemplate(r *http.Request, fs fs.FS) (*template.Template, error) {
	t := template.New("initial state")
	indexHtml, err := fs.Open("index.html")
	if err != nil {
		log.Error(r, "Could not find `index.html` template", err)
		return nil, err
	}
	indexStr, err := io.ReadAll(indexHtml)
	if err != nil {
		log.Error(r, "Could not read from `index.html`", err)
		return nil, err
	}
	t, err = t.Parse(string(indexStr))
	if err != nil {
		log.Error(r, "Error parsing `index.html`", err)
		return nil, err
	}
	return t, nil
}

type shareData struct {
	Description string             `json:"description"`
	Tracks      []model.ShareTrack `json:"tracks"`
}

func marshalShareData(ctx context.Context, shareInfo *model.Share) []byte {
	if shareInfo == nil {
		return nil
	}
	data := shareData{
		Description: shareInfo.Description,
		Tracks:      shareInfo.Tracks,
	}
	shareInfoJson, err := json.Marshal(data)
	if err != nil {
		log.Error(ctx, "Error converting shareInfo to JSON", "config", shareInfo, err)
	} else {
		log.Trace(ctx, "Injecting shareInfo in index.html", "config", string(shareInfoJson))
	}
	return shareInfoJson
}
