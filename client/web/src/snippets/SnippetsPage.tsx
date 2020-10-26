import * as H from 'history'
import { uniqueId } from 'lodash'
import React, { createRef, useEffect, useLayoutEffect, useState } from 'react'
import { map } from 'rxjs/operators'
import { ViewerId, observeEditorAndModel } from '../../../shared/src/api/client/services/viewerService'
import { TextModel } from '../../../shared/src/api/client/services/modelService'
import { PanelViewWithComponent } from '../../../shared/src/api/client/services/panelViews'
import { SNIPPET_URI_SCHEME } from '../../../shared/src/api/client/types/textDocument'
import { ContributableViewContainer } from '../../../shared/src/api/protocol'
import { EditorTextField } from '../../../shared/src/components/editorTextField/EditorTextField'
import { WithLinkPreviews } from '../../../shared/src/components/linkPreviews/WithLinkPreviews'
import { Markdown } from '../../../shared/src/components/Markdown'
import { ExtensionsControllerProps } from '../../../shared/src/extensions/controller'
import { renderMarkdown } from '../../../shared/src/util/markdown'
import { LINK_PREVIEW_CLASS } from '../components/linkPreviews/styles'
import { WebEditorCompletionWidget } from '../components/shared'
import { setElementTooltip } from '../../../branded/src/components/tooltip/Tooltip'

interface Props extends ExtensionsControllerProps {
    location: H.Location
    history: H.History
}

/**
 * Shows a text field for a snippet. This functionality is currently incomplete and is intended only
 * to allow experimentation with extensions that listen for changes in documents and display
 * Markdown-formatted text.
 */
export const SnippetsPage: React.FunctionComponent<Props> = ({ location, history, extensionsController }) => {
    const [textArea, setTextArea] = useState<HTMLTextAreaElement | null>(null)
    const textAreaReference = createRef<HTMLTextAreaElement>()
    useLayoutEffect(() => setTextArea(textAreaReference.current), [textAreaReference])

    const [viewerId, setViewerId] = useState<ViewerId | null>(null)
    const [modelUri, setModelUri] = useState<string | null>(null)

    const urlQuery = new URLSearchParams(location.search)

    const textAreaClassName = urlQuery.has('mono') ? 'text-monospace' : ''

    const initialModelUriScheme = urlQuery.get('scheme') || SNIPPET_URI_SCHEME
    const initialModelLanguageId = urlQuery.get('languageId') || 'plaintext'
    const initialModelText = urlQuery.get('text') || ''
    useEffect(() => {
        const model: TextModel = {
            uri: uniqueId(`${initialModelUriScheme}://`),
            languageId: initialModelLanguageId,
            text: initialModelText,
        }
        extensionsController.services.model.addModel(model)
        setModelUri(model.uri)
        const editor = extensionsController.services.viewer.addViewer({
            type: 'CodeEditor',
            resource: model.uri,
            selections: [],
            isActive: true,
        })
        setViewerId(editor)
        return () => {
            extensionsController.services.viewer.removeViewer(editor)
        }
    }, [
        initialModelUriScheme,
        initialModelLanguageId,
        initialModelText,
        extensionsController.services.model,
        extensionsController.services.viewer,
    ])

    const [panelViews, setPanelViews] = useState<PanelViewWithComponent[] | null>(null)
    useEffect(() => {
        const subscription = extensionsController.services.panelViews
            .getPanelViews(ContributableViewContainer.Panel)
            .subscribe(views => setPanelViews(views))
        return () => subscription.unsubscribe()
    }, [extensionsController.services.panelViews])

    // Add Markdown panel for Markdown snippets.
    const [modelText, setModelText] = useState<string | null>(null)
    useEffect(() => {
        if (!viewerId) {
            return () => undefined
        }
        const subscription = observeEditorAndModel(
            viewerId,
            extensionsController.services.viewer,
            extensionsController.services.model
        )
            .pipe(map(editor => editor.model.text))
            .subscribe(text => setModelText(text || null))
        return () => subscription.unsubscribe()
    }, [viewerId, initialModelLanguageId, extensionsController.services.viewer, extensionsController.services.model])
    const allPanelViews: PanelViewWithComponent[] | null =
        initialModelLanguageId === 'markdown' && modelText !== null
            ? [...(panelViews || []), { title: 'Preview', content: modelText, priority: 0 }]
            : panelViews

    return (
        <div className="container mt-3">
            <h1>
                Snippet editor <span className="badge badge-warning text-uppercase">Experimental</span>
            </h1>
            {viewerId && modelUri && (
                <>
                    {textArea && (
                        <WebEditorCompletionWidget
                            textArea={textArea}
                            viewerId={viewerId.viewerId}
                            extensionsController={extensionsController}
                        />
                    )}
                    <EditorTextField
                        className={`form-control ${textAreaClassName || ''}`}
                        placeholder="Type a snippet"
                        viewerId={viewerId.viewerId}
                        modelUri={modelUri}
                        autoFocus={true}
                        spellCheck={false}
                        rows={12}
                        textAreaRef={textAreaReference}
                        extensionsController={extensionsController}
                    />
                </>
            )}
            {allPanelViews &&
                allPanelViews.length > 0 &&
                allPanelViews.map((view, index) => (
                    <div key={index} className="mt-3 card">
                        <h3 className="card-header">{view.title}</h3>
                        <div className="card-body">
                            <WithLinkPreviews
                                dangerousInnerHTML={renderMarkdown(view.content)}
                                extensionsController={extensionsController}
                                setElementTooltip={setElementTooltip}
                                linkPreviewContentClass={LINK_PREVIEW_CLASS}
                            >
                                {props => <Markdown {...props} history={history} />}
                            </WithLinkPreviews>
                        </div>
                    </div>
                ))}
        </div>
    )
}
