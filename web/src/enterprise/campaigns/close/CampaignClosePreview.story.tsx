import * as H from 'history'
import { storiesOf } from '@storybook/react'
import { radios } from '@storybook/addon-knobs'
import React from 'react'
import webStyles from '../../../enterprise.scss'
import { Tooltip } from '../../../components/tooltip/Tooltip'
import { CampaignClosePreview } from './CampaignClosePreview'
import { NOOP_TELEMETRY_SERVICE } from '../../../../../shared/src/telemetry/telemetryService'

const { add } = storiesOf('web/campaigns/CampaignClosePreview', module).addDecorator(story => {
    const theme = radios('Theme', { Light: 'light', Dark: 'dark' }, 'light')
    document.body.classList.toggle('theme-light', theme === 'light')
    document.body.classList.toggle('theme-dark', theme === 'dark')
    return (
        <>
            <Tooltip />
            <style>{webStyles}</style>
            <div className="p-3 container">{story()}</div>
        </>
    )
})

add('Header', () => {
    const history = H.createMemoryHistory()
    return (
        <CampaignClosePreview
            campaignID="123"
            history={history}
            location={history.location}
            isLightTheme={true}
            telemetryService={NOOP_TELEMETRY_SERVICE}
        />
    )
})
