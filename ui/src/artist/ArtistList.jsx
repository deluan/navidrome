import React, { useMemo } from 'react'
import { useHistory } from 'react-router-dom'
import {
  Datagrid,
  DatagridBody,
  DatagridRow,
  Filter,
  NumberField,
  SearchInput,
  SelectInput,
  TextField,
  useTranslate,
} from 'react-admin'
import { useMediaQuery, withWidth } from '@material-ui/core'
import FavoriteIcon from '@material-ui/icons/Favorite'
import FavoriteBorderIcon from '@material-ui/icons/FavoriteBorder'
import { makeStyles } from '@material-ui/core/styles'
import { useDrag } from 'react-dnd'
import {
  ArtistContextMenu,
  List,
  QuickFilter,
  useGetHandleArtistClick,
  ArtistSimpleList,
  RatingField,
  useSelectedFields,
  useResourceRefresh,
  SizeField,
} from '../common'
import config from '../config'
import ArtistListActions from './ArtistListActions'
import { DraggableTypes } from '../consts'
import en from '../i18n/en.json'

const useStyles = makeStyles({
  contextHeader: {
    marginLeft: '3px',
    marginTop: '-2px',
    verticalAlign: 'text-top',
  },
  row: {
    '&:hover': {
      '& $contextMenu': {
        visibility: 'visible',
      },
      '& $ratingField': {
        visibility: 'visible',
      },
    },
  },
  contextMenu: {
    visibility: 'hidden',
  },
  ratingField: {
    visibility: 'hidden',
  },
})

const ArtistFilter = (props) => {
  const translate = useTranslate()
  const rolesObj = en?.resources?.artist?.roles
  const roles = Object.keys(rolesObj).reduce((acc, role) => {
    acc.push({
      id: role,
      name: translate(`resources.artist.roles.${role}`, {
        smart_count: 2,
      }),
    })
    return acc
  }, [])
  return (
    <Filter {...props} variant={'outlined'}>
      <SearchInput id="search" source="name" alwaysOn />
      <SelectInput source="role" choices={roles} alwaysOn />
      {config.enableFavourites && (
        <QuickFilter
          source="starred"
          label={<FavoriteIcon fontSize={'small'} />}
          defaultValue={true}
        />
      )}
    </Filter>
  )
}

const ArtistDatagridRow = (props) => {
  const { record } = props
  const [, dragArtistRef] = useDrag(
    () => ({
      type: DraggableTypes.ARTIST,
      item: { artistIds: [record?.id] },
      options: { dropEffect: 'copy' },
    }),
    [record],
  )
  return <DatagridRow ref={dragArtistRef} {...props} />
}

const ArtistDatagridBody = (props) => (
  <DatagridBody {...props} row={<ArtistDatagridRow />} />
)

const ArtistDatagrid = (props) => (
  <Datagrid {...props} body={<ArtistDatagridBody />} />
)

const ArtistListView = ({ hasShow, hasEdit, hasList, width, ...rest }) => {
  const classes = useStyles()
  const handleArtistLink = useGetHandleArtistClick(width)
  const history = useHistory()
  const isXsmall = useMediaQuery((theme) => theme.breakpoints.down('xs'))
  useResourceRefresh('artist')

  const toggleableFields = useMemo(() => {
    return {
      albumCount: <NumberField source="albumCount" sortByOrder={'DESC'} />,
      songCount: <NumberField source="songCount" sortByOrder={'DESC'} />,
      size: !isXsmall && <SizeField source="size" />,
      playCount: <NumberField source="playCount" sortByOrder={'DESC'} />,
      rating: config.enableStarRating && (
        <RatingField
          source="rating"
          sortByOrder={'DESC'}
          resource={'artist'}
          className={classes.ratingField}
        />
      ),
    }
  }, [classes.ratingField, isXsmall])

  const columns = useSelectedFields(
    {
      resource: 'artist',
      columns: toggleableFields,
    },
    ['size'],
  )

  return isXsmall ? (
    <ArtistSimpleList
      linkType={(id) => history.push(handleArtistLink(id))}
      {...rest}
    />
  ) : (
    <ArtistDatagrid rowClick={handleArtistLink} classes={{ row: classes.row }}>
      <TextField source="name" />
      {columns}
      <ArtistContextMenu
        source={'starred_at'}
        sortByOrder={'DESC'}
        sortable={config.enableFavourites}
        className={classes.contextMenu}
        label={
          config.enableFavourites && (
            <FavoriteBorderIcon
              fontSize={'small'}
              className={classes.contextHeader}
            />
          )
        }
      />
    </ArtistDatagrid>
  )
}

const ArtistList = (props) => {
  return (
    <>
      <List
        {...props}
        sort={{ field: 'name', order: 'ASC' }}
        exporter={false}
        bulkActionButtons={false}
        filters={<ArtistFilter />}
        filterDefaultValues={{ role: 'albumartist' }}
        actions={<ArtistListActions />}
      >
        <ArtistListView {...props} />
      </List>
    </>
  )
}

const ArtistListWithWidth = withWidth()(ArtistList)

export default ArtistListWithWidth
