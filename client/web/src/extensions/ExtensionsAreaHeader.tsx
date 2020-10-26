import * as React from 'react'
import { Link, RouteComponentProps } from 'react-router-dom'
import { ActionButtonDescriptor } from '../util/contributions'
import { ExtensionsAreaRouteContext } from './ExtensionsArea'
import { PageHeader } from '../components/PageHeader'
import PuzzleOutlineIcon from 'mdi-react/PuzzleOutlineIcon'

export interface ExtensionsAreaHeaderProps extends ExtensionsAreaRouteContext, RouteComponentProps<{}> {
    isPrimaryHeader: boolean
    actionButtons: readonly ExtensionsAreaHeaderActionButton[]
}

export interface ExtensionAreaHeaderContext {
    isPrimaryHeader: boolean
}

export interface ExtensionsAreaHeaderActionButton extends ActionButtonDescriptor<ExtensionAreaHeaderContext> {}

/**
 * Header for the extensions area.
 */
export const ExtensionsAreaHeader: React.FunctionComponent<ExtensionsAreaHeaderProps> = (
    props: ExtensionsAreaHeaderProps
) => (
    <div className="container">
        {props.isPrimaryHeader && (
            <div className="navbar navbar-expand px-0 pt-4 pb-2">
                <PageHeader title="Extensions" icon={PuzzleOutlineIcon} />
                <div className="spacer" />
                <ul className="navbar-nav nav">
                    {props.actionButtons.map(
                        ({ condition = () => true, to, icon: Icon, label, tooltip }) =>
                            condition(props) && (
                                <li className="nav-item" key={label}>
                                    <Link className="btn ml-2 btn-secondary" to={to(props)} data-tooltip={tooltip}>
                                        {Icon && <Icon className="icon-inline" />} {label}
                                    </Link>
                                </li>
                            )
                    )}
                </ul>
            </div>
        )}
    </div>
)
