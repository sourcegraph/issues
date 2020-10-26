import { uniqueId } from 'lodash'
import { TextDocumentDecorationType } from 'sourcegraph'

export const createDecorationType = (): TextDocumentDecorationType => ({ key: uniqueId('TextDocumentDecorationType') })
