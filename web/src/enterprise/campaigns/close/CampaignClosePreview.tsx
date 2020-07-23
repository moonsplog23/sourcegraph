import * as H from 'history'
import React from 'react'
import classNames from 'classnames'
import * as GQL from '../../../../../shared/src/graphql/schema'
import { CampaignHeader } from '../detail/CampaignHeader'
import { CampaignChangesets } from '../detail/changesets/CampaignChangesets'
import { ThemeProps } from '../../../../../shared/src/theme'
import { TelemetryProps } from '../../../../../shared/src/telemetry/telemetryService'
import { PlatformContextProps } from '../../../../../shared/src/platform/context'
import { ExtensionsControllerProps } from '../../../../../shared/src/extensions/controller'
import { Subject, Observable, of } from 'rxjs'
import CloseIcon from 'mdi-react/CloseIcon'
import { Connection } from '../../../components/FilteredConnection'
import { CampaignActionsBar } from '../detail/CampaignActionsBar'

interface Props extends ThemeProps, TelemetryProps, PlatformContextProps, ExtensionsControllerProps {
    campaignID: GQL.ID
    history: H.History
    location: H.Location
    className?: string
}

const queryFn: () => Observable<Connection<GQL.Changeset>> = () =>
    of({
        totalCount: 0,
        nodes: [],
    })

/**
 * Page that displays the actions to be taken when closing the campaign.
 */
export const CampaignClosePreview: React.FunctionComponent<Props> = ({
    campaignID,
    className,
    extensionsController,
    isLightTheme,
    platformContext,
    telemetryService,
    history,
    location,
}) => (
    <div className={classNames(className)}>
        <CampaignActionsBar
            campaign={{
                name: 'Awesome campaign',
                closedAt: null,
                viewerCanAdminister: false,
                changesets: { totalCount: 10, stats: { closed: 0, merged: 8, total: 10 } },
            }}
        />
        <div className="alert alert-warning">
            By closing this campaign, it will be read-only and no new campaign specs can be applied to this campaign.
        </div>
        <CampaignChangesets
            campaignID={campaignID}
            viewerCanAdminister={false}
            changesetUpdates={new Subject()}
            campaignUpdates={new Subject()}
            history={history}
            location={location}
            isLightTheme={isLightTheme}
            extensionsController={extensionsController}
            platformContext={platformContext}
            telemetryService={telemetryService}
            onlyOpen={true}
            hideFilters={true}
            queryChangesets={queryFn}
        />
        <div>
            <div className="alert alert-warning">
                <p>By default, all changesets remain untouched.</p>
                <div className="form-group mb-0">
                    <input id="checkbox-1" type="checkbox" />
                    <label htmlFor="checkbox-1" className="ml-2">
                        Also close open changesets on code hosts.
                    </label>
                </div>
            </div>
            <button type="button" className="btn btn-danger">
                <CloseIcon className="icon-inline" /> Close campaign
            </button>
        </div>
    </div>
)
