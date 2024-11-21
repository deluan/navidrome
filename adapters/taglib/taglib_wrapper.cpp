#include <stdlib.h>
#include <string.h>
#include <typeinfo>

#define TAGLIB_STATIC
#include <aifffile.h>
#include <asffile.h>
#include <fileref.h>
#include <flacfile.h>
#include <id3v2tag.h>
#include <unsynchronizedlyricsframe.h>
#include <synchronizedlyricsframe.h>
#include <mp4file.h>
#include <mpegfile.h>
#include <opusfile.h>
#include <tpropertymap.h>
#include <vorbisfile.h>
#include <wavfile.h>

#include "taglib_wrapper.h"

char has_cover(const TagLib::FileRef f);

static char TAGLIB_VERSION[16];

char* taglib_version() {
    snprintf((char *)TAGLIB_VERSION, 16, "%d.%d.%d", TAGLIB_MAJOR_VERSION, TAGLIB_MINOR_VERSION, TAGLIB_PATCH_VERSION);
    return (char *)TAGLIB_VERSION;
}

int taglib_read(const FILENAME_CHAR_T *filename, unsigned long id) {
  TagLib::FileRef f(filename, true, TagLib::AudioProperties::Fast);

  if (f.isNull()) {
    return TAGLIB_ERR_PARSE;
  }

  if (!f.audioProperties()) {
    return TAGLIB_ERR_AUDIO_PROPS;
  }

  // Add audio properties to the tags
  const TagLib::AudioProperties *props(f.audioProperties());
  goPutInt(id, (char *)"_lengthinmilliseconds", props->lengthInMilliseconds());
  goPutInt(id, (char *)"_bitrate", props->bitrate());
  goPutInt(id, (char *)"_channels", props->channels());
  goPutInt(id, (char *)"_samplerate", props->sampleRate());

  // Send all properties to the Go map
  TagLib::PropertyMap tags = f.file()->properties();
  for (TagLib::PropertyMap::ConstIterator i = tags.begin(); i != tags.end();
       ++i) {
    char *key = (char *)i->first.toCString(true);
    for (TagLib::StringList::ConstIterator j = i->second.begin();
         j != i->second.end(); ++j) {
      char *val = (char *)(*j).toCString(true);
      goPutStr(id, key, val);
    }
  }

  // M4A may have some iTunes specific tags not captured by the PropertyMap interface
  TagLib::MP4::File *m4afile(dynamic_cast<TagLib::MP4::File *>(f.file()));
  if (m4afile != NULL) {
    const auto itemListMap = m4afile->tag()->itemMap();
    for (const auto item: itemListMap) {
      char *key = (char *)item.first.toCString(true);
      for (const auto value: item.second.toStringList()) {
        char *val = (char *)value.toCString(true);
        goPutStr(id, key, val);
      }
    }
  }

  // WMA/ASF files may have additional tags not captured by the PropertyMap interface
  TagLib::ASF::File *asfFile(dynamic_cast<TagLib::ASF::File *>(f.file()));
  if (asfFile != NULL) {
    const TagLib::ASF::Tag *asfTags{asfFile->tag()};
    const auto itemListMap = asfTags->attributeListMap();
    for (const auto item : itemListMap) {
      char *key = (char *)item.first.toCString(true);
      char *val = (char *)item.second.front().toString().toCString(true);
      goPutStr(id, key, val);
    }
  }

  // Cover art has to be handled separately
  if (has_cover(f)) {
    goPutStr(id, (char *)"has_picture", (char *)"true");
  }

  return 0;
}

// Detect if the file has cover art. Returns 1 if the file has cover art, 0 otherwise.
char has_cover(const TagLib::FileRef f) {
  char hasCover = 0;
  // ----- MP3
  if (TagLib::MPEG::File *
      mp3File{dynamic_cast<TagLib::MPEG::File *>(f.file())}) {
    if (mp3File->ID3v2Tag()) {
      const auto &frameListMap{mp3File->ID3v2Tag()->frameListMap()};
      hasCover = !frameListMap["APIC"].isEmpty();
    }
  }
  // ----- FLAC
  else if (TagLib::FLAC::File *
           flacFile{dynamic_cast<TagLib::FLAC::File *>(f.file())}) {
    hasCover = !flacFile->pictureList().isEmpty();
  }
  // ----- MP4
  else if (TagLib::MP4::File *
           mp4File{dynamic_cast<TagLib::MP4::File *>(f.file())}) {
    auto &coverItem{mp4File->tag()->itemMap()["covr"]};
    TagLib::MP4::CoverArtList coverArtList{coverItem.toCoverArtList()};
    hasCover = !coverArtList.isEmpty();
  }
  // ----- Ogg
  else if (TagLib::Ogg::Vorbis::File *
           vorbisFile{dynamic_cast<TagLib::Ogg::Vorbis::File *>(f.file())}) {
    hasCover = !vorbisFile->tag()->pictureList().isEmpty();
  }
  // ----- Opus
  else if (TagLib::Ogg::Opus::File *
           opusFile{dynamic_cast<TagLib::Ogg::Opus::File *>(f.file())}) {
    hasCover = !opusFile->tag()->pictureList().isEmpty();
  }
  // ----- WMA
  if (TagLib::ASF::File *
      asfFile{dynamic_cast<TagLib::ASF::File *>(f.file())}) {
    const TagLib::ASF::Tag *tag{asfFile->tag()};
    hasCover = tag && tag->attributeListMap().contains("WM/Picture");
  }

  return hasCover;
}
