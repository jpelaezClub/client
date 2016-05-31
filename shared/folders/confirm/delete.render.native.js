// @flow

import React from 'react'
import type {Props} from './delete.render'
import {Confirm, Box, Text, Icon} from '../../common-adapters'
import {headerBoxStyle} from '../../common-adapters/confirm.native'
import {usernameText} from '../../common-adapters/usernames'
import {globalColors} from '../../styles/style-guide'

const Render = ({isPrivate, users, folderSize, onSubmit, onCancel}: Props) => {
  const theme = isPrivate ? 'private' : 'public'
  return (
    <Confirm theme={theme} danger submitLabel='Yes, delete it' onSubmit={onSubmit} onCancel={onCancel}>
      <Box style={headerBoxStyle}>
        <Icon type={iconThemed[theme]} />
      </Box>
      <Box style={{marginBottom: 16}}>
        <Text type='Header' style={{textAlign: 'center'}}>
          <Text type='Header' style={textColorThemed[theme]}>Delete files and clear history for {isPrivate ? 'private/' : 'public/'}</Text>
          {usernameText({type: 'Header', style: textColorThemed[theme], users})}
          <Text type='Header' style={textColorThemed[theme]}>? ({folderSize})</Text>
        </Text>
      </Box>
      <Text type='Body' style={{...textColorThemed[theme], textAlign: 'center'}}>Deletes everything in this folder, including the backup versions.</Text>
    </Confirm>
  )
}

const textColorThemed = {
  'public': {
    color: globalColors.black_75
  },
  'private': {
    color: globalColors.white
  }
}

const iconThemed = {
  'public': 'files-public-delete-64',
  'private': 'files-private-delete-64'
}

export default Render
