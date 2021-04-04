import React from 'react'
import { useDispatch } from 'react-redux'
import {
  Button,
  sanitizeListRestProps,
  TopToolbar,
  useTranslate,
  useDataProvider,
} from 'react-admin'
import PlayArrowIcon from '@material-ui/icons/PlayArrow'
import ShuffleIcon from '@material-ui/icons/Shuffle'
import CloudDownloadOutlinedIcon from '@material-ui/icons/CloudDownloadOutlined'
import { RiPlayListAddFill, RiPlayList2Fill } from 'react-icons/ri'
import QueueMusicIcon from '@material-ui/icons/QueueMusic'
import { httpClient } from '../dataProvider'
import { playNext, addTracks, playTracks, shuffleTracks } from '../actions'
import { M3U_MIME_TYPE, REST_URL } from '../consts'
import subsonic from '../subsonic'
import PropTypes from 'prop-types'
import { formatBytes } from '../utils'
import { useMediaQuery } from '@material-ui/core'
import config from '../config'

const PlaylistActions = ({ className, ids, data, record, ...rest }) => {
  const dispatch = useDispatch()
  const translate = useTranslate()
  const dataProvider = useDataProvider()
  const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
  const { id, songCount } = record

  const getAllSongs = React.useCallback(
    (type) => {
      dataProvider
        .getList('playlistTrack', {
          pagination: { page: 1, perPage: 0 },
          sort: { field: 'id', order: 'ASC' },
          filter: { playlist_id: id },
        })
        .then((res) => {
          const data = res.data.reduce(
            (acc, curr) => ({ ...acc, [curr.id]: curr }),
            {}
          )
          const ids = Object.keys(data)
          type === 'play'
            ? dispatch(playTracks(data, ids))
            : dispatch(shuffleTracks(data, ids))
        })
    },
    [dataProvider, dispatch, id]
  )

  const handlePlay = React.useCallback(() => {
    if (ids.length === songCount) {
      return dispatch(playTracks(data, ids))
    }
    getAllSongs('play')
  }, [dispatch, data, ids, getAllSongs, songCount])

  const handlePlayNext = React.useCallback(() => {
    dispatch(playNext(data, ids))
  }, [dispatch, data, ids])

  const handlePlayLater = React.useCallback(() => {
    dispatch(addTracks(data, ids))
  }, [dispatch, data, ids])

  const handleShuffle = React.useCallback(() => {
    if (ids.length === songCount) {
      return dispatch(shuffleTracks(data, ids))
    }
    getAllSongs('shuffle')
  }, [dispatch, data, ids, getAllSongs, songCount])

  const handleDownload = React.useCallback(() => {
    subsonic.download(record.id)
  }, [record])

  const handleExport = React.useCallback(
    () =>
      httpClient(`${REST_URL}/playlist/${record.id}/tracks`, {
        headers: new Headers({ Accept: M3U_MIME_TYPE }),
      }).then((res) => {
        const blob = new Blob([res.body], { type: M3U_MIME_TYPE })
        const url = window.URL.createObjectURL(blob)
        const link = document.createElement('a')
        link.href = url
        link.download = `${record.name}.m3u`
        document.body.appendChild(link)
        link.click()
        link.parentNode.removeChild(link)
      }),
    [record]
  )

  return (
    <TopToolbar className={className} {...sanitizeListRestProps(rest)}>
      <Button
        onClick={handlePlay}
        label={translate('resources.album.actions.playAll')}
      >
        <PlayArrowIcon />
      </Button>
      <Button
        onClick={handleShuffle}
        label={translate('resources.album.actions.shuffle')}
      >
        <ShuffleIcon />
      </Button>
      <Button
        onClick={handlePlayNext}
        label={translate('resources.album.actions.playNext')}
      >
        <RiPlayList2Fill />
      </Button>
      <Button
        onClick={handlePlayLater}
        label={translate('resources.album.actions.addToQueue')}
      >
        <RiPlayListAddFill />
      </Button>
      {config.enableDownloads && (
        <Button
          onClick={handleDownload}
          label={
            translate('resources.album.actions.download') +
            (isDesktop ? ` (${formatBytes(record.size)})` : '')
          }
        >
          <CloudDownloadOutlinedIcon />
        </Button>
      )}
      <Button
        onClick={handleExport}
        label={translate('resources.playlist.actions.export')}
      >
        <QueueMusicIcon />
      </Button>
    </TopToolbar>
  )
}

PlaylistActions.propTypes = {
  record: PropTypes.object.isRequired,
  selectedIds: PropTypes.arrayOf(PropTypes.number),
}

PlaylistActions.defaultProps = {
  record: {},
  selectedIds: [],
  onUnselectItems: () => null,
}

export default PlaylistActions
