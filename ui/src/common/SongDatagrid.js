import React, { isValidElement, useMemo, useCallback, forwardRef } from 'react'
import { useDispatch } from 'react-redux'
import { Datagrid, PureDatagridBody, PureDatagridRow, useTranslate} from 'react-admin'
import {
  TableCell,
  TableRow,
  Typography,
  useMediaQuery,
} from '@material-ui/core'
import PropTypes from 'prop-types'
import { makeStyles } from '@material-ui/core/styles'
import AlbumIcon from '@material-ui/icons/Album'
import clsx from 'clsx'
import { useDrag } from 'react-dnd'
import { playTracks } from '../actions'
import { AlbumContextMenu,
  FormatFullDate,
} from '../common'
import { DraggableTypes } from '../consts'

const useStyles = makeStyles({
  subtitle: {
    whiteSpace: 'nowrap',
    overflow: 'hidden',
    textOverflow: 'ellipsis',
    verticalAlign: 'middle',
  },
  discIcon: {
    verticalAlign: 'text-top',
    marginRight: '4px',
  },
  row: {
    cursor: 'pointer',
    '&:hover': {
      '& $contextMenu': {
        visibility: 'visible',
      },
    },
  },
  headerStyle: {
    '& thead': {
      boxShadow: '0px 3px 3px rgba(0, 0, 0, 0.15)',
    },
    '& th': {
      fontWeight: 'bold',
      padding: '15px',
    },
  },
  contextMenu: {
    visibility: (props) => (props.isDesktop ? 'hidden' : 'visible'),
  },
})

const EditionRow = forwardRef(
  ({ record, onClick, colSpan, contextAlwaysVisible}, ref) => {
    const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
    const classes = useStyles({ isDesktop })
    const translate = useTranslate()
    const handlePlaySubset = (releaseDate) => () => {
      onClick(releaseDate)
    }

    let editionTitle = []
    if (record.releaseDate) {
      editionTitle.push(translate('resources.album.fields.released'))
      editionTitle.push(FormatFullDate(record.releaseDate))
      if (record.catalogNum && isDesktop) {
        editionTitle.push("· Cat #")
        editionTitle.push(record.catalogNum)
      }
    }

    return (
      <TableRow
        hover
        ref={ref}
        onClick={handlePlaySubset(record.releaseDate)}
        className={classes.row}
      >
        <TableCell colSpan={colSpan}>
          <Typography variant="h6" className={classes.subtitle}>
            {editionTitle.join(' ')}
          </Typography>
        </TableCell>
        <TableCell>
          <AlbumContextMenu
            record={{ id: record.albumId }}
            releaseDate={record.releaseDate}
            showLove={false}
            className={classes.contextMenu}
            visible={contextAlwaysVisible}
          />
        </TableCell>
      </TableRow>
    )
  }
)

const DiscSubtitleRow = forwardRef(
  ({ record, onClick, colSpan, contextAlwaysVisible}, ref) => {
    const isDesktop = useMediaQuery((theme) => theme.breakpoints.up('md'))
    const classes = useStyles({ isDesktop })
    const handlePlaySubset = (releaseDate, discNumber) => () => {
      onClick(releaseDate, discNumber)
    }

    let subtitle = []
    if (record.discNumber > 0) {
      subtitle.push(record.discNumber)
    }
    if (record.discSubtitle) {
      subtitle.push(record.discSubtitle)
    }

    return (
      <TableRow
        hover
        ref={ref}
        onClick={handlePlaySubset(record.releaseDate, record.discNumber)}
        className={classes.row}
      >
        <TableCell colSpan={colSpan}>
          <Typography variant="h6" className={classes.subtitle}>
            <AlbumIcon className={classes.discIcon} fontSize={'small'} />
            {subtitle.join(': ')}
          </Typography>
        </TableCell>
        <TableCell>
          <AlbumContextMenu
            record={{ id: record.albumId }}
            discNumber={record.discNumber}
            releaseDate={record.releaseDate}
            showLove={false}
            className={classes.contextMenu}
            visible={contextAlwaysVisible}
          />
        </TableCell>
      </TableRow>
    )
  }
)

export const SongDatagridRow = ({
  record,
  children,
  firstTracksOfDiscs,
  firstTracksOfEditions,
  contextAlwaysVisible,
  onClickSubset,
  className,
  ...rest
}) => {
  const classes = useStyles()
  const fields = React.Children.toArray(children).filter((c) =>
    isValidElement(c)
  )

  const [, dragDiscRef] = useDrag(
    () => ({
      type: DraggableTypes.DISC,
      item: {
        discs: [{ albumId: record?.albumId, releaseDate: record?.releaseDate, discNumber: record?.discNumber }],
      },
      options: { dropEffect: 'copy' },
    }),
    [record]
  )

  const [, dragSongRef] = useDrag(
    () => ({
      type: DraggableTypes.SONG,
      item: { ids: [record?.mediaFileId || record?.id] },
      options: { dropEffect: 'copy' },
    }),
    [record]
  )

  if (!record || !record.title) {
    return null
  }

  const childCount = fields.length
  return (
    <>      
    {firstTracksOfEditions.has(record.id) && (
        
      <EditionRow
        ref={dragDiscRef}
        record={record}
        onClick={onClickSubset}
        contextAlwaysVisible={contextAlwaysVisible}
        colSpan={childCount + (rest.expand ? 1 : 0)}
      />
    )}
      {firstTracksOfDiscs.has(record.id) && (
        
        <DiscSubtitleRow
          ref={dragDiscRef}
          record={record}
          onClick={onClickSubset}
          contextAlwaysVisible={contextAlwaysVisible}
          colSpan={childCount + (rest.expand ? 1 : 0)}
        />
      )}
      <PureDatagridRow
        ref={dragSongRef}
        record={record}
        {...rest}
        className={clsx(className, classes.row)}
      >
        {fields}
      </PureDatagridRow>
    </>
  )
}

SongDatagridRow.propTypes = {
  record: PropTypes.object,
  children: PropTypes.node,
  firstTracksOfDiscs: PropTypes.instanceOf(Set),
  firstTracksOfEditions: PropTypes.instanceOf(Set),
  contextAlwaysVisible: PropTypes.bool,
  onClickSubset: PropTypes.func,
}

SongDatagridRow.defaultProps = {
  onClickSubset: () => {},
}

const SongDatagridBody = ({
  contextAlwaysVisible,
  showDiscSubtitles,
  ...rest
}) => {
  const dispatch = useDispatch()
  const { ids, data } = rest

  const playSubset = useCallback(
    (releaseDate, discNumber) => {
      let idsToPlay = []
      if (discNumber != undefined) {
        idsToPlay = ids.filter((id) => (data[id].releaseDate === releaseDate && data[id].discNumber === discNumber))
      } else {
        idsToPlay = ids.filter((id) => (data[id].releaseDate === releaseDate))
      }
      dispatch(playTracks(data, idsToPlay))
    },
    [dispatch, data, ids]
  )

  const firstTracksOfDiscs = useMemo(() => {
    if (!ids) {
      return new Set()
    }
    let foundSubtitle = false
    const set = new Set(
      ids
        .filter((i) => data[i])
        .reduce((acc, id) => {
          const last = acc && acc[acc.length - 1]
          foundSubtitle = foundSubtitle || data[id].discSubtitle
          if (
            acc.length === 0 ||
            (last && data[id].discNumber !== data[last].discNumber) ||
            (last && data[id].releaseDate !== data[last].releaseDate)
          ) {
            acc.push(id)
          }
          return acc
        }, [])
    )
    if (!showDiscSubtitles || (set.size < 2 && !foundSubtitle)) {
      set.clear()
    }
    return set
  }, [ids, data, showDiscSubtitles])

  const firstTracksOfEditions = useMemo(() => {
    if (!ids) {
      return new Set()
    }
    const set = new Set(
      ids
        .filter((i) => data[i])
        .reduce((acc, id) => {
          const last = acc && acc[acc.length - 1]
          if (
            acc.length === 0 ||
            (last && data[id].releaseDate !== data[last].releaseDate)
          ) {
            acc.push(id)
          }
          return acc
        }, [])
    )
    if (set.size < 2) {
      set.clear()
    }
    return set
  }, [ids, data])

  return (
    <PureDatagridBody
      {...rest}
      row={
        <SongDatagridRow
          firstTracksOfDiscs={firstTracksOfDiscs}
          firstTracksOfEditions={firstTracksOfEditions}
          contextAlwaysVisible={contextAlwaysVisible}
          onClickSubset={playSubset}
        />
      }
    />
  )
}

export const SongDatagrid = ({
  contextAlwaysVisible,
  showDiscSubtitles,
  ...rest
}) => {
  const classes = useStyles()
  return (
    <Datagrid
      className={classes.headerStyle}
      {...rest}
      body={
        <SongDatagridBody
          contextAlwaysVisible={contextAlwaysVisible}
          showDiscSubtitles={showDiscSubtitles}
        />
      }
    />
  )
}

SongDatagrid.propTypes = {
  contextAlwaysVisible: PropTypes.bool,
  showDiscSubtitles: PropTypes.bool,
  classes: PropTypes.object,
}
