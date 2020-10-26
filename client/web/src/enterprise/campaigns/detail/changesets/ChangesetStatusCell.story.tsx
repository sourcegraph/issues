import { storiesOf } from '@storybook/react'
import React from 'react'
import { EnterpriseWebStory } from '../../../components/EnterpriseWebStory'
import {
    ChangesetStatusUnpublished,
    ChangesetStatusProcessing,
    ChangesetStatusClosed,
    ChangesetStatusMerged,
    ChangesetStatusOpen,
    ChangesetStatusDeleted,
    ChangesetStatusError,
    ChangesetStatusDraft,
} from './ChangesetStatusCell'

const { add } = storiesOf('web/campaigns/ChangesetStatusCell', module).addDecorator(story => (
    <div className="p-3 container web-content">{story()}</div>
))

add('Unpublished', () => (
    <EnterpriseWebStory>{() => <ChangesetStatusUnpublished className="d-flex" />}</EnterpriseWebStory>
))
add('Closed', () => <EnterpriseWebStory>{() => <ChangesetStatusClosed className="d-flex" />}</EnterpriseWebStory>)
add('Merged', () => <EnterpriseWebStory>{() => <ChangesetStatusMerged className="d-flex" />}</EnterpriseWebStory>)
add('Open', () => <EnterpriseWebStory>{() => <ChangesetStatusOpen className="d-flex" />}</EnterpriseWebStory>)
add('Draft', () => <EnterpriseWebStory>{() => <ChangesetStatusDraft className="d-flex" />}</EnterpriseWebStory>)
add('Deleted', () => <EnterpriseWebStory>{() => <ChangesetStatusDeleted className="d-flex" />}</EnterpriseWebStory>)
add('Error', () => <EnterpriseWebStory>{() => <ChangesetStatusError className="d-flex" />}</EnterpriseWebStory>)
add('Processing', () => (
    <EnterpriseWebStory>{() => <ChangesetStatusProcessing className="d-flex" />}</EnterpriseWebStory>
))
