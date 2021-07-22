import { getCollisions, Position } from '@reach/popover';

const DEFAULT_VERTICAL_OFFSET = 4

/**
 * Custom popover position calculator. Returns position objects (top,left,right,bottom) styles
 * with values such that the target and the popover element have the same right borders.
 *
 * ------------ | Target | --------
 * ----|*****************| --------
 * ----|*****************| --------
 * ----|*** Popover *****| --------
 * ----|*****************| --------
 * ----|*****************| --------
 *
 * @param targetRectangle - bounding client rect of the target element
 * @param popoverRectangle - bounding client rect of the pop-over element. All calculation props
 * that are returned from this function will be applied to this element.
 */
export const positionRight: Position = (targetRectangle, popoverRectangle) => {
    if (!targetRectangle || !popoverRectangle) {
        return {};
    }

    const { directionUp } = getCollisions(targetRectangle, popoverRectangle);

    return {
        left: `${targetRectangle.right - popoverRectangle.width + window.pageXOffset}px`,
        top: directionUp
            ? `${targetRectangle.top - popoverRectangle.height + window.pageYOffset - DEFAULT_VERTICAL_OFFSET}px`
            : `${targetRectangle.top + targetRectangle.height + window.pageYOffset + DEFAULT_VERTICAL_OFFSET}px`,
    };
};
