import { storiesOf } from '@storybook/react'
import React, { useCallback, useState } from 'react'
import { LoaderInput } from './LoaderInput'
import { BrandedStory } from './BrandedStory'

const { add } = storiesOf('branded/LoaderInput', module).addDecorator(story => (
    <div className="container mt-3" style={{ width: 800 }}>
        {story()}
    </div>
))

add('Interactive', () => (
    <BrandedStory>
        {() => {
            const [loading, setLoading] = useState(true)
            const toggleLoading = useCallback(() => setLoading(loading => !loading), [])

            return (
                <>
                    <button type="button" className="btn btn-primary mb-2" onClick={toggleLoading}>
                        Toggle Loading
                    </button>
                    <p>
                        <LoaderInput loading={loading}>
                            <input type="text" placeholder="Loader input" className="form-control" />
                        </LoaderInput>
                    </p>
                </>
            )
        }}
    </BrandedStory>
))
