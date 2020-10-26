import { TextField } from '../shared/textFields'
import { ViewResolver } from '../shared/views'

export const commentTextFieldResolver: ViewResolver<TextField> = {
    selector: '.comment-form-textarea',
    resolveView: element => {
        if (!(element instanceof HTMLTextAreaElement)) {
            return null
        }
        return { element }
    },
}
