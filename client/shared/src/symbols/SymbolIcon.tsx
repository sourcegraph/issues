import { kebabCase } from 'lodash'
import { MdiReactIconComponentType } from 'mdi-react'
import CodeArrayIcon from 'mdi-react/CodeArrayIcon'
import CodeBracesIcon from 'mdi-react/CodeBracesIcon'
import CodeNotEqualIcon from 'mdi-react/CodeNotEqualIcon'
import CodeStringIcon from 'mdi-react/CodeStringIcon'
import CubeIcon from 'mdi-react/CubeIcon'
import CubeOutlineIcon from 'mdi-react/CubeOutlineIcon'
import DrawingBoxIcon from 'mdi-react/DrawingBoxIcon'
import FileDocumentIcon from 'mdi-react/FileDocumentIcon'
import FunctionIcon from 'mdi-react/FunctionIcon'
import KeyIcon from 'mdi-react/KeyIcon'
import LinkIcon from 'mdi-react/LinkIcon'
import MatrixIcon from 'mdi-react/MatrixIcon'
import NullIcon from 'mdi-react/NullIcon'
import NumericIcon from 'mdi-react/NumericIcon'
import PackageIcon from 'mdi-react/PackageIcon'
import PiBoxIcon from 'mdi-react/PiBoxIcon'
import PillarIcon from 'mdi-react/PillarIcon'
import PoundIcon from 'mdi-react/PoundIcon'
import ShapeIcon from 'mdi-react/ShapeIcon'
import SitemapIcon from 'mdi-react/SitemapIcon'
import TextBoxIcon from 'mdi-react/TextBoxIcon'
import TimetableIcon from 'mdi-react/TimetableIcon'
import WebIcon from 'mdi-react/WebIcon'
import WrenchIcon from 'mdi-react/WrenchIcon'
import * as React from 'react'
import { SymbolKind } from '../graphql-operations'

/**
 * Returns the icon component for a given symbol kind
 */
const getSymbolIconComponent = (kind: SymbolKind): MdiReactIconComponentType => {
    switch (kind) {
        case 'FILE':
            return FileDocumentIcon
        case 'MODULE':
            return CodeBracesIcon
        case 'NAMESPACE':
            return WebIcon
        case 'PACKAGE':
            return PackageIcon
        case 'CLASS':
            return SitemapIcon
        case 'METHOD':
            return CubeOutlineIcon
        case 'PROPERTY':
            return WrenchIcon
        case 'FIELD':
            return TextBoxIcon
        case 'CONSTRUCTOR':
            return CubeOutlineIcon
        case 'ENUM':
            return NumericIcon
        case 'INTERFACE':
            return LinkIcon
        case 'FUNCTION':
            return FunctionIcon
        case 'VARIABLE':
            return CubeIcon
        case 'CONSTANT':
            return PiBoxIcon
        case 'STRING':
            return CodeStringIcon
        case 'NUMBER':
            return PoundIcon
        case 'BOOLEAN':
            return MatrixIcon
        case 'ARRAY':
            return CodeArrayIcon
        case 'OBJECT':
            return DrawingBoxIcon
        case 'KEY':
            return KeyIcon
        case 'NULL':
            return NullIcon
        case 'ENUMMEMBER':
            return NumericIcon
        case 'STRUCT':
            return PillarIcon
        case 'EVENT':
            return TimetableIcon
        case 'OPERATOR':
            return CodeNotEqualIcon
        case 'TYPEPARAMETER':
            return CubeIcon
        case 'UNKNOWN':
        default:
            return ShapeIcon
    }
}

interface SymbolIconProps {
    kind: SymbolKind
    className?: string
}

/**
 * Renders an Icon for a given symbol kind
 */
export const SymbolIcon: React.FunctionComponent<SymbolIconProps> = ({ kind, className = '' }) => {
    const Icon = getSymbolIconComponent(kind)
    return (
        <Icon
            className={`symbol-icon symbol-icon--kind-${kebabCase(kind)} ${className}`}
            data-tooltip={kind.toLowerCase()}
        />
    )
}
