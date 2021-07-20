import { isArray } from 'lodash'
import CheckIcon from 'mdi-react/CheckIcon'
import ErrorIcon from 'mdi-react/ErrorIcon'
import ProgressClockIcon from 'mdi-react/ProgressClockIcon'
import TimerSandIcon from 'mdi-react/TimerSandIcon'
import React, { FunctionComponent, useMemo } from 'react'

import { isDefined } from '@sourcegraph/shared/src/util/types'

import { ExecutionLogEntry } from '../../../components/ExecutionLogEntry'
import { Timeline, TimelineStage } from '../../../components/Timeline'
import { LsifIndexFields, LSIFIndexState } from '../../../graphql-operations'

export interface CodeIntelIndexTimelineProps {
    index: LsifIndexFields
    now?: () => Date
    className?: string
}

export const CodeIntelIndexTimeline: FunctionComponent<CodeIntelIndexTimelineProps> = ({ index, now, className }) => {
    const stages = useMemo(
        () => [
            { icon: <TimerSandIcon />, text: 'Queued', date: index.queuedAt, className: 'bg-success' },
            { icon: <ProgressClockIcon />, text: 'Began processing', date: index.startedAt, className: 'bg-success' },

            indexSetupStage(index, now),
            indexPreIndexStage(index, now),
            indexIndexStage(index, now),
            indexUploadStage(index, now),
            indexTeardownStage(index, now),

            index.state === LSIFIndexState.COMPLETED
                ? { icon: <CheckIcon />, text: 'Finished', date: index.finishedAt, className: 'bg-success' }
                : { icon: <ErrorIcon />, text: 'Failed', date: index.finishedAt, className: 'bg-danger' },
        ],
        [index, now]
    )

    return <Timeline stages={stages.filter(isDefined)} now={now} className={className} />
}

const indexSetupStage = (index: LsifIndexFields, now?: () => Date): TimelineStage | undefined =>
    index.steps.setup.length === 0
        ? undefined
        : {
              text: 'Setup',
              details: index.steps.setup.map(logEntry => (
                  <ExecutionLogEntry key={logEntry.key} logEntry={logEntry} now={now} />
              )),
              ...genericStage(index.steps.setup),
          }

const indexPreIndexStage = (index: LsifIndexFields, now?: () => Date): TimelineStage | undefined => {
    const logEntries = index.steps.preIndex.map(step => step.logEntry).filter(isDefined)

    return logEntries.length === 0
        ? undefined
        : {
              text: 'Pre Index',
              details: index.steps.preIndex.map(
                  step =>
                      step.logEntry && (
                          <div key={`${step.image}${step.root}${step.commands.join(' ')}}`}>
                              <ExecutionLogEntry logEntry={step.logEntry} now={now}>
                                  <ExecutionMetaInformation
                                      {...{
                                          image: step.image,
                                          commands: step.commands,
                                          root: step.root,
                                      }}
                                  />
                              </ExecutionLogEntry>
                          </div>
                      )
              ),
              ...genericStage(logEntries),
          }
}

const indexIndexStage = (index: LsifIndexFields, now?: () => Date): TimelineStage | undefined =>
    !index.steps.index.logEntry
        ? undefined
        : {
              text: 'Index',
              details: (
                  <>
                      <ExecutionLogEntry logEntry={index.steps.index.logEntry} now={now}>
                          <ExecutionMetaInformation
                              {...{
                                  image: index.inputIndexer,
                                  commands: index.steps.index.indexerArgs,
                                  root: index.inputRoot,
                              }}
                          />
                      </ExecutionLogEntry>
                  </>
              ),
              ...genericStage(index.steps.index.logEntry),
          }

const indexUploadStage = (index: LsifIndexFields, now?: () => Date): TimelineStage | undefined =>
    !index.steps.upload
        ? undefined
        : {
              text: 'Upload',
              details: <ExecutionLogEntry logEntry={index.steps.upload} now={now} />,
              ...genericStage(index.steps.upload),
          }

const indexTeardownStage = (index: LsifIndexFields, now?: () => Date): TimelineStage | undefined =>
    index.steps.teardown.length === 0
        ? undefined
        : {
              text: 'Teardown',
              details: index.steps.teardown.map(logEntry => (
                  <ExecutionLogEntry key={logEntry.key} logEntry={logEntry} now={now} />
              )),
              ...genericStage(index.steps.teardown),
          }

const genericStage = <E extends { startTime: string; exitCode: number }>(
    value: E | E[]
): Pick<TimelineStage, 'icon' | 'date' | 'className' | 'expanded'> => {
    const success = isArray(value) ? value.every(logEntry => logEntry.exitCode === 0) : value.exitCode === 0

    return {
        icon: success ? <CheckIcon /> : <ErrorIcon />,
        date: isArray(value) ? value[0].startTime : value.startTime,
        className: success ? 'bg-success' : 'bg-danger',
        expanded: !success,
    }
}

const ExecutionMetaInformation: React.FunctionComponent<{ image: string; commands: string[]; root: string }> = ({
    image,
    commands,
    root,
}) => (
    <div className="pt-3">
        <div className="docker-command-spec py-2 border-top pl-2">
            <strong className="docker-command-spec__header">Image</strong>
            <div>{image}</div>
        </div>
        <div className="docker-command-spec py-2 border-top pl-2">
            <strong className="docker-command-spec__header">Commands</strong>
            <div>
                <code>{commands.join(' ')}</code>
            </div>
        </div>
        <div className="docker-command-spec py-2 border-top pl-2">
            <strong className="docker-command-spec__header">Root</strong>
            <div>/{root}</div>
        </div>
    </div>
)
