package artwork

import (
	"context"
	"io"
	"time"

	"github.com/navidrome/navidrome/model"
)

type mediafileArtworkReader struct {
	cacheKey
	a         *artwork
	mediafile model.MediaFile
	album     model.Album
}

func newMediafileArtworkReader(ctx context.Context, artwork *artwork, artID model.ArtworkID) (*mediafileArtworkReader, error) {
	mf, err := artwork.ds.MediaFile(ctx).Get(artID.ID)
	if err != nil {
		return nil, err
	}
	al, err := artwork.ds.Album(ctx).Get(mf.AlbumID)
	if err != nil {
		return nil, err
	}
	a := &mediafileArtworkReader{
		a:         artwork,
		mediafile: *mf,
		album:     *al,
	}
	a.cacheKey.artID = artID
	if al.UpdatedAt.After(mf.UpdatedAt) {
		a.cacheKey.lastUpdate = al.UpdatedAt
	} else {
		a.cacheKey.lastUpdate = mf.UpdatedAt
	}
	return a, nil
}

func (a *mediafileArtworkReader) LastUpdated() time.Time {
	return a.lastUpdate
}

func (a *mediafileArtworkReader) Reader(ctx context.Context) (io.ReadCloser, string, error) {
	var ff []sourceFunc
	if a.mediafile.CoverArtID().Kind == model.KindMediaFileArtwork {
		ff = []sourceFunc{
			fromTag(a.mediafile.Path),
			fromFFmpegTag(ctx, a.a.ffmpeg, a.mediafile.Path),
		}
	}
	ff = append(ff, fromAlbum(ctx, a.a, a.mediafile.AlbumCoverArtID()))
	r, source := extractImage(ctx, a.artID, ff...)
	return r, source, nil
}

func fromAlbum(ctx context.Context, a *artwork, id model.ArtworkID) sourceFunc {
	return func() (io.ReadCloser, string, error) {
		r, _, err := a.Get(ctx, id.String(), 0)
		if err != nil {
			return nil, "", err
		}
		return r, id.String(), nil
	}
}
