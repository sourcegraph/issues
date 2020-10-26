import React from 'react'

interface OptionsPageAdvancedSettingsProps {
    optionFlags: { key: string; label: string; value: boolean }[]
    onChangeOptionFlag: (key: string, value: boolean) => void
}

export const OptionsPageAdvancedSettings: React.FunctionComponent<OptionsPageAdvancedSettingsProps> = ({
    optionFlags,
    onChangeOptionFlag,
}) => (
    <section className="mt-3">
        <h6>
            <small>Configuration</small>
        </h6>
        <div>
            {optionFlags.map(({ label, key, value }) => (
                <div className="form-check" key={key}>
                    <label className="form-check-label">
                        <input
                            id={key}
                            onChange={event => onChangeOptionFlag(key, event.target.checked)}
                            className="form-check-input"
                            type="checkbox"
                            checked={value}
                        />{' '}
                        {label}
                    </label>
                </div>
            ))}
        </div>
    </section>
)
