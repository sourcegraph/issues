import { min, noop } from 'lodash'
import { Observable, of, Subject, Subscription } from 'rxjs'
import { delay } from 'rxjs/operators'
import * as sinon from 'sinon'
import {
    createValidationPipeline,
    InputValidationState,
    ValidationOptions,
    VALIDATION_DEBOUNCE_TIME,
} from './useInputValidation'

describe('useInputValidation()', () => {
    let clock: sinon.SinonFakeTimers
    let subscriptions: Subscription

    let setupValidationPipelineTest: (
        validationOptions: ValidationOptions
    ) => (inputScript: (string | number)[]) => InputValidationState[]

    beforeAll(() => {
        clock = sinon.useFakeTimers()
    })

    afterAll(() => {
        clock.restore()
    })

    beforeEach(() => {
        subscriptions = new Subscription()

        setupValidationPipelineTest = (
            validationOptions
        ): ((inputScript: (string | number)[]) => InputValidationState[]) => {
            const inputElement = createEmailInputElement()
            const inputReference = { current: inputElement }

            const inputValidationStates: InputValidationState[] = []

            const validationPipeline = createValidationPipeline(
                validationOptions,
                inputReference,
                // We want to test the values that this callback is called with,
                // not the emissions of the returned observable. Therefore, we will
                // push these values to an array whose values we will assert.
                inputValidationStates.push.bind(inputValidationStates)
            )

            // Creating this type instead of a generic util because TS doesn't support higher-kinded types
            type ObservableEmission<T> = T extends Observable<infer V> ? V : never
            const changeEvents = new Subject<ObservableEmission<Parameters<typeof validationPipeline>[0]>>()

            // We don't care about this observable.
            // Here, we simply set up the pipeline:
            // change event -> validation pipeline -> validation results
            //                      |
            //                      v
            //                 validation states (this is what we're testing)
            subscriptions.add(validationPipeline(changeEvents).subscribe(noop))

            // Simulate user input: change input value, then dispatch change event
            function userInput(value: string): void {
                inputElement.changeValue(value)
                changeEvents.next({
                    preventDefault: noop,
                    target: inputReference.current,
                })
            }

            // "Scripting" user interaction: strings are new input values, numbers are delays before next input in ms.
            return function executeUserInputScript(inputScript) {
                for (const input of inputScript) {
                    if (typeof input === 'string') {
                        userInput(input)
                    } else {
                        clock.tick(input)
                    }
                }
                // Wait for debounceTime after final input
                clock.tick(500)

                return inputValidationStates
            }
        }
    })

    afterEach(() => {
        subscriptions.unsubscribe()
    })

    /**
     * Creates a mock input element for emails. Only checks for '@'. Acts as
     * if `novalidate` is true.
     *
     * Implements the `HTMLInputElement` properties needed by `useInputValidation`,
     * along with a `changeValue` method to simulate input and change the internal value
     */
    function createEmailInputElement(): Pick<
        HTMLInputElement,
        'checkValidity' | 'validationMessage' | 'setCustomValidity' | 'value'
    > & { changeValue: (newValue: string) => void } {
        const internalStrings = {
            value: '',
            validationMessage: '',
        }

        return {
            get value() {
                return internalStrings.value
            },
            get validationMessage() {
                return internalStrings.validationMessage
            },
            checkValidity() {
                // Check if custom validity was set
                if (internalStrings.validationMessage.length > 0) {
                    return false
                }
                // Built-in rules
                if (!internalStrings.value.includes('@')) {
                    internalStrings.validationMessage = "Email must include '@'"
                    return false
                }
                // Clear built-in messages
                internalStrings.validationMessage = ''
                return true
            },
            setCustomValidity(error) {
                internalStrings.validationMessage = error
            },
            changeValue(newValue) {
                internalStrings.value = newValue
            },
        }
    }

    /**
     * Shared test validators
     */
    function isDotCo(email: string): string | undefined {
        if (email.endsWith('.co')) {
            return undefined
        }

        return "Email must end with '.co'"
    }

    it('works without initial value', () => {
        const executeUserInputScript = setupValidationPipelineTest({
            synchronousValidators: [isDotCo],
        })

        const inputs: (string | number)[] = [
            'source',
            'sourcegraph',
            // Advance less than `VALIDATION_DEBOUNCE_TIME` to ensure value isn't validated in this case
            min([VALIDATION_DEBOUNCE_TIME - 100, 200]) ?? 200,
            'sourcegraph@',
            VALIDATION_DEBOUNCE_TIME,
            'sourcegraph@sg.co',
        ]

        const expectedStates: InputValidationState[] = [
            {
                kind: 'LOADING',
                value: 'source',
            },
            {
                kind: 'LOADING',
                value: 'sourcegraph',
            },
            {
                kind: 'LOADING',
                value: 'sourcegraph@',
            },
            {
                kind: 'INVALID',
                value: 'sourcegraph@',
                reason: "Email must end with '.co'",
            },
            {
                kind: 'LOADING',
                value: 'sourcegraph@sg.co',
            },
            {
                kind: 'VALID',
                value: 'sourcegraph@sg.co',
            },
        ]

        expect(executeUserInputScript(inputs)).toStrictEqual(expectedStates)
    })

    it('works with initial value', () => {
        const executeUserInputScript = setupValidationPipelineTest({
            synchronousValidators: [isDotCo],
            initialValue: 'so',
        })

        const inputs: (string | number)[] = [
            VALIDATION_DEBOUNCE_TIME,
            'source',
            'sourcegraph',
            // Advance less than `VALIDATION_DEBOUNCE_TIME` to ensure value isn't validated in this case
            min([VALIDATION_DEBOUNCE_TIME - 100, 200]) ?? 200,
            'sourcegraph@',
            VALIDATION_DEBOUNCE_TIME,
            'sourcegraph@sg.co',
        ]

        const expectedStates: InputValidationState[] = [
            {
                kind: 'LOADING',
                value: 'so',
            },
            {
                kind: 'INVALID',
                reason: "Email must include '@'",
                value: 'so',
            },
            {
                kind: 'LOADING',
                value: 'source',
            },
            {
                kind: 'LOADING',
                value: 'sourcegraph',
            },
            {
                kind: 'LOADING',
                value: 'sourcegraph@',
            },
            {
                kind: 'INVALID',
                value: 'sourcegraph@',
                reason: "Email must end with '.co'",
            },
            {
                kind: 'LOADING',
                value: 'sourcegraph@sg.co',
            },
            {
                kind: 'VALID',
                value: 'sourcegraph@sg.co',
            },
        ]

        expect(executeUserInputScript(inputs)).toStrictEqual(expectedStates)
    })

    it('works with async validators', () => {
        function isEmailUnique(email: string): Observable<string | undefined> {
            return of(email === 'test@sg.co' ? 'Email is taken' : undefined).pipe(delay(200))
        }

        const executeUserInputScript = setupValidationPipelineTest({
            synchronousValidators: [isDotCo],
            asynchronousValidators: [isEmailUnique],
        })

        const inputs: (string | number)[] = [
            'test',
            VALIDATION_DEBOUNCE_TIME,
            'test@sg.c',
            'test@sg.co',
            // Advance 200 more ms due to delay from `isEmailUnique`
            VALIDATION_DEBOUNCE_TIME + 200,
            '@sg.co',
            // Advance less than `VALIDATION_DEBOUNCE_TIME` to ensure value isn't validated in this case
            min([VALIDATION_DEBOUNCE_TIME - 100, 200]) ?? 200,
            'sourcegraph@sg.co',
            // Advance 200 more ms due to delay from `isEmailUnique`
            200,
        ]

        const expectedStates: InputValidationState[] = [
            {
                kind: 'LOADING',
                value: 'test',
            },
            {
                kind: 'INVALID',
                reason: "Email must include '@'",
                value: 'test',
            },
            {
                kind: 'LOADING',
                value: 'test@sg.c',
            },
            {
                kind: 'LOADING',
                value: 'test@sg.co',
            },
            {
                kind: 'INVALID',
                value: 'test@sg.co',
                reason: 'Email is taken',
            },
            {
                kind: 'LOADING',
                value: '@sg.co',
            },
            {
                kind: 'LOADING',
                value: 'sourcegraph@sg.co',
            },
            {
                kind: 'VALID',
                value: 'sourcegraph@sg.co',
            },
        ]

        expect(executeUserInputScript(inputs)).toStrictEqual(expectedStates)
    })
})
