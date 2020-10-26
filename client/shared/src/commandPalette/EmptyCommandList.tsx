import React from 'react'
import { SettingsCascadeOrError } from '../settings/settings'
import { onlyDefaultExtensionsAdded } from '../extensions/extensions'

interface Props {
    settingsCascade?: SettingsCascadeOrError
    sourcegraphURL: string
}

export const EmptyCommandList: React.FunctionComponent<Props> = ({ settingsCascade, sourcegraphURL }) => {
    // if no settings cascade, default to 'no active extensions'
    const onlyDefault = settingsCascade ? onlyDefaultExtensionsAdded(settingsCascade) : false

    return (
        <div className="empty-command-list">
            <p className="empty-command-list__title">
                {onlyDefault ? "You don't have any extensions enabled" : "You don't have any active actions"}
            </p>
            <p className="empty-command-list__text">
                {onlyDefault
                    ? 'Enable Sourcegraph extensions to get additional functionality, integrations, and make special actions available from this menu.'
                    : 'Commands from your installed extensions will be shown when you navigate to certain pages.'}
            </p>

            <a className="btn btn-primary" href={sourcegraphURL + '/extensions'}>
                Explore extensions
            </a>

            <PuzzleIllustration className="empty-command-list__illustration" />
        </div>
    )
}

const PuzzleIllustration = React.memo<{ className?: string }>(({ className }) => (
    <svg
        width="59"
        height="64"
        viewBox="0 0 59 64"
        fill="none"
        xmlns="http://www.w3.org/2000/svg"
        className={className}
    >
        <path
            d="M51.5825 43.9313C55.4271 45.582 57.2515 40.6516 57.8679 40.6516C58.4844 40.6516 58.9775 41.1397 58.9775 41.7497V60.3841C58.9775 62.3806 57.3411 64 55.3237 64H37.9287C37.7942 63.9778 37.6934 63.9556 37.6373 63.9335C35.9449 63.1459 40.2322 61.0779 40.1928 58.8201C40.1396 55.7739 38.0632 54.658 33.5688 54.658C29.0744 54.658 26.6802 55.4072 27.9131 59.6776C28.3325 61.1303 31.0022 63.1903 29.1192 63.9335C29.052 63.9556 28.8838 63.9778 28.6597 64H11.7803C9.78528 64 8.17132 62.4139 8.1377 60.4395V43.225C8.16011 43.0697 8.18253 42.9588 8.21615 42.9033C9.01192 41.2284 11.7803 44.8776 13.3831 45.4322C17.6197 46.8853 21.5201 43.1362 21.5201 38.6884C21.5201 34.2406 18.9535 27.9515 14.7168 29.4156C13.1141 29.9702 9.00072 34.8363 8.24978 32.9729C8.21615 32.9063 8.19374 32.7178 8.17132 32.4626L8.1377 16.9373C8.1377 16.9373 8.18253 16.3605 8.23857 16.0832C8.5636 13.9979 10.7492 13.1328 12.6209 13.1328L14.7168 13.1771C14.7168 13.1771 19.8725 13.2991 24.0644 13.3102H25.8016C27.6509 13.3102 29.0632 13.2659 29.3097 13.166C31.1927 12.4229 27.3147 9.63882 26.7543 8.05269C25.286 3.85997 29.0856 0 33.5688 0C38.052 0 41.8516 3.85997 40.3945 8.04159C39.8341 9.62773 36.1467 12.3785 37.8391 13.1549C37.996 13.2326 38.4219 13.277 39.0496 13.3102H46.7047C49.7533 13.2437 52.4096 13.1549 52.4096 13.1549L54.5055 13.1106C55.2228 13.1106 55.9962 13.2659 56.7023 13.5653C56.7247 13.5764 56.7583 13.5875 56.7919 13.5986C56.8592 13.6319 56.9152 13.6652 56.9825 13.6984C57.1394 13.7872 57.3075 13.8759 57.4532 13.9757C57.5877 14.0756 57.7222 14.1865 57.8455 14.2974C57.8791 14.3307 57.9127 14.3529 57.9464 14.3861C58.4283 14.852 58.787 15.451 58.9215 16.1941C58.9775 16.427 58.9999 16.6711 58.9999 16.9262V29.4156C58.9887 30.0257 58.4218 32.5682 57.9403 32.9484C55.2912 35.0404 54.2316 27.1052 50.5229 30.8565C47.5946 33.8183 47.7379 42.2805 51.5825 43.9313Z"
            fill="url(#paint0_linear)"
            fillOpacity="0.32"
        />
        <rect y="24.6855" width="12.9467" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect y="46.627" width="12.9467" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect y="52.1133" width="12.9467" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect y="19.1992" width="11.0972" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="11.0977" y="30.1699" width="12.9467" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="11.0977" y="41.1426" width="12.9467" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="11.0977" y="35.6562" width="15.7211" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="13.8711" y="24.6855" width="9.24768" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="13.8711" y="46.627" width="9.24768" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="13.8711" y="52.1133" width="5.54861" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="12.0225" y="19.1992" width="5.54861" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="24.9688" y="30.1699" width="4.62384" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="24.9688" y="41.1426" width="9.24767" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="30.5176" y="30.1699" width="7.39814" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="35.1416" y="41.1426" width="4.62384" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <rect x="27.7432" y="35.6562" width="7.39814" height="2.74286" rx="1.37143" fill="#ABC7EB" fillOpacity="0.5" />
        <defs>
            <linearGradient id="paint0_linear" x1="33.5688" y1="0" x2="33.5688" y2="64" gradientUnits="userSpaceOnUse">
                <stop stopColor="#95A5C6" stopOpacity="0.6" />
                <stop offset="1" stopColor="#95A5C6" />
            </linearGradient>
        </defs>
    </svg>
))
