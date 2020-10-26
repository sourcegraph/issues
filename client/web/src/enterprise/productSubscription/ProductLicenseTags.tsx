import React from 'react'

export const ProductLicenseTags: React.FunctionComponent<{
    tags: string[]
}> = ({ tags }) => (
    <>
        {tags.map(tag => (
            <div className="mr-1 badge badge-secondary" key={tag}>
                {tag}
            </div>
        ))}
    </>
)
