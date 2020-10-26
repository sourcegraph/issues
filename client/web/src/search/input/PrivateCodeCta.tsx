import * as React from 'react'
import { CtaBanner } from '../../components/CtaBanner'

interface Props {
    className?: string
}

export const PrivateCodeCta: React.FunctionComponent<Props> = ({ className }) => (
    <CtaBanner
        className={className}
        icon={<DownloadIllustration />}
        title="Search your private code"
        bodyText="Set up a self-hosted Sourcegraph instance to search your private repositories on GitHub, GitLab, Bitbucket and local installations of Git, Perforce, Subversion and other code repositories."
        linkText="Install now"
        href="https://docs.sourcegraph.com/"
        googleAnalytics={true}
    />
)

const DownloadIllustration = React.memo(() => (
    <svg width="68" height="115" viewBox="0 0 68 115" fill="none" xmlns="http://www.w3.org/2000/svg">
        <path
            d="M32.2917 65.8333C32.2917 65.2811 32.7394 64.8333 33.2917 64.8333H35.7084C36.2606 64.8333 36.7084 65.2811 36.7084 65.8333V88.9191C36.7084 89.81 37.7855 90.2562 38.4155 89.6262L48.1471 79.8946C48.5376 79.5041 49.1708 79.5041 49.5613 79.8946L51.2388 81.5721C51.6293 81.9626 51.6293 82.5958 51.2388 82.9863L35.2071 99.0179C34.8166 99.4084 34.1834 99.4084 33.7929 99.0179L17.7613 82.9863C17.3708 82.5958 17.3708 81.9626 17.7613 81.5721L19.4388 79.8946C19.8293 79.5041 20.4624 79.5041 20.853 79.8946L30.5846 89.6262C31.2146 90.2562 32.2917 89.81 32.2917 88.9191V65.8333Z"
            fill="url(#paint0_linear)"
            fillOpacity="0.23"
        />
        <path
            fillRule="evenodd"
            clipRule="evenodd"
            d="M22.1705 13.9612L33.9756 57.1843C34.7705 60.0883 37.7634 61.7988 40.6628 61.0062C43.5623 60.209 45.2682 57.2093 44.4756 54.3053L32.6682 11.08C31.8733 8.17597 28.8805 6.46546 25.981 7.26036C23.0839 8.05526 21.3756 11.0549 22.1705 13.9612Z"
            fill="#F96216"
        />
        <path
            fillRule="evenodd"
            clipRule="evenodd"
            d="M44.0325 13.7972L14.4685 47.2309C12.4755 49.4858 12.6828 52.9341 14.9331 54.9316C17.1834 56.9291 20.6227 56.7196 22.6157 54.4647L52.1796 21.031C54.1726 18.7761 53.9653 15.33 51.715 13.3325C49.4647 11.335 46.0277 11.5423 44.0325 13.7972Z"
            fill="#B200F8"
        />
        <path
            fillRule="evenodd"
            clipRule="evenodd"
            d="M10.5347 32.3262L52.6963 46.2928C55.5502 47.2403 58.6274 45.687 59.5726 42.8285C60.5178 39.9678 58.9667 36.8838 56.1151 35.9409L13.9511 21.9675C11.0972 21.0245 8.02013 22.5733 7.07718 25.4341C6.13195 28.2948 7.68076 31.3787 10.5347 32.3262Z"
            fill="#00B4F2"
        />
        <rect x="12" y="102" width="43" height="3" rx="1.5" fill="#95A5C6" fillOpacity="0.3" />
        <defs>
            <linearGradient id="paint0_linear" x1="34.5" y1="101.5" x2="35" y2="48" gradientUnits="userSpaceOnUse">
                <stop stopColor="#95A5C6" />
                <stop offset="1" stopColor="#95A5C6" stopOpacity="0" />
            </linearGradient>
        </defs>
    </svg>
))
