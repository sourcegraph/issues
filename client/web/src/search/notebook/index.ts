import { Observable } from 'rxjs'
import * as uuid from 'uuid'

import { SearchPatternType } from '@sourcegraph/shared/src/graphql-operations'
import { aggregateStreamingSearch, AggregateStreamingSearchResults } from '@sourcegraph/shared/src/search/stream'
import { renderMarkdown } from '@sourcegraph/shared/src/util/markdown'

import { LATEST_VERSION } from '../results/StreamingSearchResults'

export type BlockType = 'md' | 'query'

interface BaseBlock<O> {
    id: string
    type: BlockType
    input: string
    output: O | null
}

export interface QueryBlock extends BaseBlock<Observable<AggregateStreamingSearchResults>> {
    type: 'query'
}

export interface MarkdownBlock extends BaseBlock<string> {
    type: 'md'
}

export type Block = QueryBlock | MarkdownBlock

export type BlockInitializer = Pick<Block, 'type' | 'input'>

export type BlockDirection = 'up' | 'down'

export interface BlockProps {
    isReadOnly: boolean
    isSelected: boolean
    isOtherBlockSelected: boolean
    onRunBlock(id: string): void
    onDeleteBlock(id: string): void
    onBlockInputChange(id: string, value: string): void
    onSelectBlock(id: string | null): void
    onMoveBlockSelection(id: string, direction: BlockDirection): void
    onMoveBlock(id: string, direction: BlockDirection): void
    onDuplicateBlock(id: string): void
}

export class Notebook {
    private blocks: Map<string, Block>
    private blockOrder: string[]

    constructor(initializerBlocks: BlockInitializer[]) {
        const blocks = initializerBlocks.map(block => ({ ...block, id: uuid.v4(), output: null }))

        this.blocks = new Map(blocks.map(block => [block.id, block]))
        this.blockOrder = blocks.map(block => block.id)

        this.renderMarkdownBlocks()
    }

    private renderMarkdownBlocks(): void {
        const blocks = this.blocks.values()
        for (const block of blocks) {
            if (block.type === 'md') {
                this.runBlockById(block.id)
            }
        }
    }

    public getBlocks(): Block[] {
        return this.blockOrder.map(blockId => {
            const block = this.blocks.get(blockId)
            if (!block) {
                throw new Error(`Block with id:${blockId} does not exist.`)
            }
            return block
        })
    }

    public setBlockInputById(id: string, value: string): void {
        const block = this.blocks.get(id)
        if (!block) {
            return
        }
        this.blocks.set(block.id, { ...block, input: value })
    }

    public runBlockById(id: string): void {
        const block = this.blocks.get(id)
        if (!block) {
            return
        }
        switch (block.type) {
            case 'md':
                this.blocks.set(block.id, { ...block, output: renderMarkdown(block.input) })
                break
            case 'query':
                this.blocks.set(block.id, {
                    ...block,
                    output: aggregateStreamingSearch({
                        // Removes comments
                        query: block.input.replace(/\/\/.*/g, ''),
                        version: LATEST_VERSION,
                        patternType: SearchPatternType.literal,
                        caseSensitive: false,
                        versionContext: undefined,
                        trace: undefined,
                    }),
                })
                break
        }
    }

    public deleteBlockById(id: string): void {
        const index = this.blockOrder.indexOf(id)
        if (index === -1) {
            return
        }
        this.blocks.delete(id)
        this.blockOrder.splice(index, 1)
    }

    public insertBlockAtIndex(index: number, type: BlockType, input: string): Block {
        const id = uuid.v4()
        const block = { id, type, input, output: null }
        // Insert block at the provided index
        this.blockOrder.splice(index, 0, id)
        this.blocks.set(id, block)
        return block
    }

    public getBlockById(id: string): Block | null {
        return this.blocks.get(id) ?? null
    }

    public moveBlockById(id: string, direction: BlockDirection): void {
        const index = this.blockOrder.indexOf(id)
        if ((direction === 'up' && index < 1) || (direction === 'down' && index === this.blockOrder.length - 1)) {
            return
        }
        const swapIndex = direction === 'up' ? index - 1 : index + 1
        const temporaryId = this.blockOrder[swapIndex]
        this.blockOrder[swapIndex] = id
        this.blockOrder[index] = temporaryId
    }

    public duplicateBlockById(id: string): Block | null {
        const block = this.blocks.get(id)
        if (!block) {
            return null
        }
        const index = this.blockOrder.indexOf(id)
        return this.insertBlockAtIndex(index + 1, block.type, block.input)
    }

    public getFirstBlockId(): string | null {
        return this.blockOrder.length > 0 ? this.blockOrder[0] : null
    }

    public getLastBlockId(): string | null {
        return this.blockOrder.length > 0 ? this.blockOrder[this.blockOrder.length - 1] : null
    }

    public getPreviousBlockId(id: string): string | null {
        const index = this.blockOrder.indexOf(id)
        return index >= 1 ? this.blockOrder[index - 1] : null
    }

    public getNextBlockId(id: string): string | null {
        const index = this.blockOrder.indexOf(id)
        return index >= 0 && index < this.blockOrder.length - 1 ? this.blockOrder[index + 1] : null
    }
}
