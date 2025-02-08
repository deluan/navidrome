//go:build go1.21

package dlna

import (
	"encoding/xml"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/anacrolix/dms/dlna"
	"github.com/anacrolix/dms/upnp"
	"github.com/navidrome/navidrome/conf"
	"github.com/navidrome/navidrome/dlna/upnpav"
	"github.com/navidrome/navidrome/log"
	"github.com/navidrome/navidrome/model"
	"github.com/oriser/regroup"
)

type contentDirectoryService struct {
	*DLNAServer
	upnp.Eventing
}

var filesRegex *regroup.ReGroup
var artistRegex *regroup.ReGroup
var albumRegex *regroup.ReGroup
var genresRegex *regroup.ReGroup
var recentRegex *regroup.ReGroup
var playlistRegex *regroup.ReGroup

func init() {
	filesRegex = regroup.MustCompile("\\/Music\\/Files[\\/]?((?P<Path>.+))?")
	artistRegex = regroup.MustCompile("\\/Music\\/Artists[\\/]?(?P<Artist>[^\\/]+)?[\\/]?(?<ArtistAlbum>[^\\/]+)?[\\/]?")
	albumRegex = regroup.MustCompile("\\/Music\\/Albums[\\/]?(?P<AlbumTitle>[^\\/]+)?[\\/]?")
	genresRegex = regroup.MustCompile("\\/Music\\/Genres[\\/]?(?P<Genre>[^\\/]+)?[\\/]?(?P<GenreArtist>[^/]+)?[\\/]?")
	recentRegex = regroup.MustCompile("\\/Music\\/Recently Added[\\/]?(?P<RecentAlbum>[^\\/]+)?")
	playlistRegex = regroup.MustCompile("\\/Music\\/Playlists[\\/]?(?P<Playlist>[^\\/]+)?[\\/]?")
}

func (cds *contentDirectoryService) updateIDString() string {
	return fmt.Sprintf("%d", uint32(os.Getpid()))
}

// Turns the given entry and DMS host into a UPnP object. A nil object is
// returned if the entry is not of interest.
func (cds *contentDirectoryService) cdsObjectToUpnpavObject(cdsObject object, isContainer bool, host string) (ret interface{}) {
	obj := upnpav.Object{
		ID:         cdsObject.ID(),
		Restricted: 1,
		ParentID:   cdsObject.ParentID(),
		Title:      filepath.Base(cdsObject.Path),
	}

	if isContainer {
		defaultChildCount := 1
		obj.Class = "object.container.storageFolder"
		return upnpav.Container{
			Object:     obj,
			ChildCount: &defaultChildCount,
		}
	}
	// Read the mime type from the fs.Object if possible,
	// otherwise fall back to working out what it is from the file path.
	var mimeType = "audio/mp3" //TODO

	obj.Class = "object.item.audioItem.musicTrack"
	obj.Date = upnpav.Timestamp{Time: time.Now()}

	item := upnpav.Item{
		Object: obj,
		Res:    make([]upnpav.Resource, 0, 1),
	}

	item.Res = append(item.Res, upnpav.Resource{
		URL: (&url.URL{
			Scheme: "http",
			Host:   host,
			Path:   path.Join(resourcePath, resourceFilePath, cdsObject.Path),
		}).String(),
		ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", mimeType, dlna.ContentFeatures{
			SupportRange: true,
		}.String()),
		Size: uint64(1048576), //TODO
	})

	ret = item
	return ret
}

// Returns all the upnpav objects in a directory.
func (cds *contentDirectoryService) readContainer(o object, host string) (ret []interface{}, err error) {
	log.Debug(fmt.Sprintf("ReadContainer called '%s'", o))

	if o.Path == "/" || o.Path == "" {
		log.Debug("ReadContainer default route")
		newObject := object{Path: "/Music"}
		ret = append(ret, cds.cdsObjectToUpnpavObject(newObject, true, host))
		return ret, nil
	}

	if o.Path == "/Music" {
		return handleDefault(ret, cds, o, host)
	} else if _, err := filesRegex.Groups(o.Path); err == nil {
		return cds.doFiles(ret, o.Path, host)
	} else if matchResults, err := artistRegex.Groups(o.Path); err == nil {
		return handleArtist(matchResults, ret, cds, o, host)
	} else if matchResults, err := albumRegex.Groups(o.Path); err == nil {
		return handleAlbum(matchResults, ret, cds, o, host)
	} else if matchResults, err := genresRegex.Groups(o.Path); err == nil {
		return handleGenre(matchResults, ret, cds, o, host)
	} else if matchResults, err := recentRegex.Groups(o.Path); err == nil {
		return handleRecent(matchResults, ret, cds, o, host)
	} else if matchResults, err := playlistRegex.Groups(o.Path); err == nil {
		return handlePlaylists(matchResults, ret, cds, o, host)
	}
	return ret, nil
}

func handleDefault(ret []interface{}, cds *contentDirectoryService, o object, host string) ([]interface{}, error) {
	ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Files"}, true, host))
	ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Artists"}, true, host))
	ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Albums"}, true, host))
	ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Genres"}, true, host))
	ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Recently Added"}, true, host))
	ret = append(ret, cds.cdsObjectToUpnpavObject(object{Path: "/Music/Playlists"}, true, host))
	return ret, nil
}

func handleArtist(matchResults map[string]string, ret []interface{}, cds *contentDirectoryService, o object, host string) ([]interface{}, error) {
	if matchResults["ArtistAlbum"] != "" {
		tracks, _ := cds.ds.MediaFile(cds.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": matchResults["ArtistAlbum"]}})
		return cds.doMediaFiles(tracks, o.Path, ret, host)
	} else if matchResults["Artist"] != "" {
		allAlbumsForThisArtist, _ := cds.ds.Album(cds.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_artist_id": matchResults["Artist"]}})
		return cds.doAlbums(allAlbumsForThisArtist, o.Path, ret, host)
	} else {
		indexes, err := cds.ds.Artist(cds.ctx).GetIndex()
		if err != nil {
			fmt.Printf("Error retrieving Indexes: %+v", err)
			return nil, err
		}
		for letterIndex := range indexes {
			for artist := range indexes[letterIndex].Artists {
				artistId := indexes[letterIndex].Artists[artist].ID
				child := object{
					Path: path.Join(o.Path, indexes[letterIndex].Artists[artist].Name),
					Id:   path.Join(o.Path, artistId),
				}
				ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
			}
		}
		return ret, nil
	}
	return ret, nil
}

func handleAlbum(matchResults map[string]string, ret []interface{}, cds *contentDirectoryService, o object, host string) ([]interface{}, error) {
	if matchResults["AlbumTitle"] != "" {
		tracks, _ := cds.ds.MediaFile(cds.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": matchResults["AlbumTitle"]}})
		return cds.doMediaFiles(tracks, o.Path, ret, host)
	}
	indexes, err := cds.ds.Album(cds.ctx).GetAllWithoutGenres()
	if err != nil {
		fmt.Printf("Error retrieving Indexes: %+v", err)
		return nil, err
	}
	for indexItem := range indexes {
		child := object{
			Path: path.Join(o.Path, indexes[indexItem].Name),
			Id:   path.Join(o.Path, indexes[indexItem].ID),
		}
		ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
	}
	return ret, nil
}

func handleGenre(matchResults map[string]string, ret []interface{}, cds *contentDirectoryService, o object, host string) ([]interface{}, error) {
	if matchResults["GenreArtist"] != "" {
		tracks, err := cds.ds.MediaFile(cds.ctx).GetAll(model.QueryOptions{Filters: squirrel.And{
			squirrel.Eq{"genre.id": matchResults["Genre"]},
			squirrel.Eq{"artist_id": matchResults["GenreArtist"]},
		},
		})
		if err != nil {
			fmt.Printf("Error retrieving tracks for artist and genre: %+v", err)
			return nil, err
		}
		return cds.doMediaFiles(tracks, o.Path, ret, host)
	} else if matchResults["Genre"] != "" {
		if matchResults["GenreArtist"] == "" {
			artists, err := cds.ds.Artist(cds.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"genre.id": matchResults["Genre"]}})
			if err != nil {
				fmt.Printf("Error retrieving artists for genre: %+v", err)
				return nil, err
			}
			for artistIndex := range artists {
				child := object{
					Path: path.Join(o.Path, artists[artistIndex].Name),
					Id:   path.Join(o.Path, artists[artistIndex].ID),
				}
				ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
			}
		}
	}
	indexes, err := cds.ds.Genre(cds.ctx).GetAll()
	if err != nil {
		fmt.Printf("Error retrieving Indexes: %+v", err)
		return nil, err
	}
	for indexItem := range indexes {
		child := object{
			Path: path.Join(o.Path, indexes[indexItem].Name),
			Id:   path.Join(o.Path, indexes[indexItem].ID),
		}
		ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
	}
	return ret, nil
}

func handleRecent(matchResults map[string]string, ret []interface{}, cds *contentDirectoryService, o object, host string) ([]interface{}, error) {
	if matchResults["RecentAlbum"] != "" {
		tracks, _ := cds.ds.MediaFile(cds.ctx).GetAll(model.QueryOptions{Filters: squirrel.Eq{"album_id": matchResults["RecentAlbum"]}})
		return cds.doMediaFiles(tracks, o.Path, ret, host)
	}
	indexes, err := cds.ds.Album(cds.ctx).GetAllWithoutGenres(model.QueryOptions{Sort: "recently_added", Order: "desc", Max: 25})
	if err != nil {
		fmt.Printf("Error retrieving Indexes: %+v", err)
		return nil, err
	}
	for indexItem := range indexes {
		child := object{
			Path: path.Join(o.Path, indexes[indexItem].Name),
			Id:   path.Join(o.Path, indexes[indexItem].ID),
		}
		ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
	}
	return ret, nil
}

func handlePlaylists(matchResults map[string]string, ret []interface{}, cds *contentDirectoryService, o object, host string) ([]interface{}, error) {
	if matchResults["Playlist"] != "" {
		x, err := cds.ds.Playlist(cds.ctx).GetWithTracks(matchResults["Playlist"], false)
		if err != nil {
			log.Error("Error fetching playlist", "playlist", matchResults["Playlist"], err)
			return ret, nil
		}
		return cds.doMediaFiles(x.MediaFiles(), o.Path, ret, host)
	}
	indexes, err := cds.ds.Playlist(cds.ctx).GetAll()
	if err != nil {
		fmt.Printf("Error retrieving Indexes: %+v", err)
		return nil, err
	}
	for indexItem := range indexes {
		child := object{
			Path: path.Join(o.Path, indexes[indexItem].Name),
			Id:   path.Join(o.Path, indexes[indexItem].ID),
		}
		ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
	}
	return ret, nil
}

func (cds *contentDirectoryService) doMediaFiles(tracks model.MediaFiles, basePath string, ret []interface{}, host string) ([]interface{}, error) {
	for _, track := range tracks {
		trackDateAsTimeObject, _ := time.Parse(time.DateOnly, track.Date)

		obj := upnpav.Object{
			ID:                  path.Join(basePath, track.ID),
			Restricted:          1,
			ParentID:            basePath,
			Title:               track.Title,
			Class:               "object.item.audioItem.musicTrack",
			Artist:              track.Artist,
			Album:               track.Album,
			Genre:               track.Genre,
			OriginalTrackNumber: track.TrackNumber,
			Date:                upnpav.Timestamp{Time: trackDateAsTimeObject},
		}

		if track.HasCoverArt {
			obj.AlbumArtURI = (&url.URL{
				Scheme: "http",
				Host:   host,
				Path:   path.Join(resourcePath, resourceArtPath, track.CoverArtID().String()),
			}).String()
		}

		//TODO figure out how this fits with transcoding etc
		var mimeType = "audio/mp3"

		item := upnpav.Item{
			Object: obj,
			Res:    make([]upnpav.Resource, 0, 1),
		}

		streamAccessPath := path.Join(resourcePath, resourceStreamPath, track.ID)
		item.Res = append(item.Res, upnpav.Resource{
			URL: (&url.URL{
				Scheme: "http",
				Host:   host,
				Path:   streamAccessPath,
			}).String(),
			ProtocolInfo: fmt.Sprintf("http-get:*:%s:%s", mimeType, dlna.ContentFeatures{
				SupportRange: false,
			}.String()),
			Size:     uint64(track.Size),
			Duration: floatToDurationString(track.Duration),
		})
		ret = append(ret, item)
	}

	return ret, nil
}

func floatToDurationString(totalSeconds32 float32) string {
	totalSeconds := float64(totalSeconds32)
	secondsInAnHour := float64(60 * 60)
	secondsInAMinute := float64(60)

	hours := int(math.Floor(totalSeconds / secondsInAnHour))
	minutes := int(math.Floor(math.Mod(totalSeconds, secondsInAnHour) / secondsInAMinute))
	seconds := int(math.Floor(math.Mod(totalSeconds, secondsInAMinute)))
	ms := int(math.Floor(math.Mod(totalSeconds, 1) * 1000))

	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, seconds, ms)
}

func (cds *contentDirectoryService) doAlbums(albums model.Albums, basepath string, ret []interface{}, host string) ([]interface{}, error) {
	for _, album := range albums {
		child := object{
			Path: path.Join(basepath, album.Name),
			Id:   path.Join(basepath, album.ID),
		}
		ret = append(ret, cds.cdsObjectToUpnpavObject(child, true, host))
	}
	return ret, nil
}

func (cds *contentDirectoryService) doFiles(ret []interface{}, oPath string, host string) ([]interface{}, error) {
	pathComponents := strings.Split(strings.TrimPrefix(oPath, "/Music/Files"), "/")
	if slices.Contains(pathComponents, "..") || slices.Contains(pathComponents, ".") {
		log.Error("Attempt to use .. or . detected", oPath, host)
		return ret, nil
	}
	totalPathArrayBits := append([]string{conf.Server.MusicFolder}, pathComponents...)
	localFilePath := filepath.Join(totalPathArrayBits...)

	files, _ := os.ReadDir(localFilePath)
	for _, file := range files {
		child := object{
			Path: path.Join(oPath, file.Name()),
			Id:   path.Join(oPath, file.Name()),
		}
		ret = append(ret, cds.cdsObjectToUpnpavObject(child, file.IsDir(), host))
	}
	return ret, nil
}

type browse struct {
	ObjectID       string
	BrowseFlag     string
	Filter         string
	StartingIndex  int
	RequestedCount int
}

// ContentDirectory object from ObjectID.
func (cds *contentDirectoryService) objectFromID(id string) (o object, err error) {
	log.Debug("objectFromID called", "id", id)
	o.Path, err = url.QueryUnescape(id)
	if err != nil {
		return
	}
	if o.Path == "0" {
		o.Path = "/"
	}
	o.Path = path.Clean(o.Path)
	if !path.IsAbs(o.Path) {
		err = fmt.Errorf("bad ObjectID %v", o.Path)
		return
	}
	return
}

func (cds *contentDirectoryService) Handle(action string, argsXML []byte, r *http.Request) (map[string]string, error) {
	host := r.Host
	log.Info(fmt.Sprintf("Handle called with action: %s", action))

	switch action {
	case "GetSystemUpdateID":
		return map[string]string{
			"Id": cds.updateIDString(),
		}, nil
	case "GetSortCapabilities":
		return map[string]string{
			"SortCaps": "dc:title",
		}, nil
	case "Browse":
		var browse browse
		if err := xml.Unmarshal(argsXML, &browse); err != nil {
			return nil, err
		}
		obj, err := cds.objectFromID(browse.ObjectID)
		if err != nil {
			return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, "%s", err.Error())
		}
		switch browse.BrowseFlag {
		case "BrowseDirectChildren":
			objs, err := cds.readContainer(obj, host)
			if err != nil {
				return nil, upnp.Errorf(upnpav.NoSuchObjectErrorCode, "%s", err.Error())
			}
			totalMatches := len(objs)
			objs = objs[func() (low int) {
				low = browse.StartingIndex
				if low > len(objs) {
					low = len(objs)
				}
				return
			}():]
			if browse.RequestedCount != 0 && browse.RequestedCount < len(objs) {
				objs = objs[:browse.RequestedCount]
			}
			result, err := xml.Marshal(objs)
			log.Debug(fmt.Sprintf("XMLResponse: '%s'", result))
			if err != nil {
				return nil, err
			}
			return map[string]string{
				"TotalMatches":   fmt.Sprint(totalMatches),
				"NumberReturned": fmt.Sprint(len(objs)),
				"Result":         didlLite(string(result)),
				"UpdateID":       cds.updateIDString(),
			}, nil
		case "BrowseMetadata":
			//TODO
			return map[string]string{
				"Result": didlLite(string("result")),
			}, nil
		default:
			return nil, upnp.Errorf(upnp.ArgumentValueInvalidErrorCode, "unhandled browse flag: %v", browse.BrowseFlag)
		}
	case "GetSearchCapabilities":
		return map[string]string{
			"SearchCaps": "",
		}, nil
	// Samsung Extensions
	case "X_GetFeatureList":
		return map[string]string{
			"FeatureList": `<Features xmlns="urn:schemas-upnp-org:av:avs" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="urn:schemas-upnp-org:av:avs http://www.upnp.org/schemas/av/avs.xsd">
	<Feature name="samsung.com_BASICVIEW" version="1">
		<container id="0" type="object.item.imageItem"/>
		<container id="0" type="object.item.audioItem"/>
		<container id="0" type="object.item.videoItem"/>
	</Feature>
</Features>`}, nil
	case "X_SetBookmark":
		// just ignore
		return map[string]string{}, nil
	default:
		return nil, upnp.InvalidActionError
	}
}

// Represents a ContentDirectory object.
type object struct {
	Path string // The cleaned, absolute path for the object relative to the server.
	Id   string
}

// Returns the actual local filesystem path for the object.
func (o *object) FilePath() string {
	return filepath.FromSlash(o.Path)
}

// Returns the ObjectID for the object. This is used in various ContentDirectory actions.
func (o object) ID() string {
	if o.Id != "" {
		return o.Id
	}
	if !path.IsAbs(o.Path) {
		log.Fatal("Relative object path used", "path", o.Path)
		return "-1"
	}
	if len(o.Path) == 1 {
		return "0"
	}
	return url.QueryEscape(o.Path)
}

func (o *object) IsRoot() bool {
	return o.Path == "/"
}

// Returns the object's parent ObjectID. Fortunately it can be deduced from the
// ObjectID (for now).
func (o object) ParentID() string {
	if o.IsRoot() {
		return "-1"
	}
	o.Path = path.Dir(o.Path)
	return o.ID()
}
