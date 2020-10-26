// Adapted from
// https://github.com/Microsoft/vscode/blob/7a07992127a6ff176f0ed073d0698d81d09fc4bb/src/vs/editor/common/viewModel/prefixSumComputer.ts.
// Copyright (c) Microsoft Corporation. All rights reserved. Licensed under the MIT License.

export class PrefixSumIndexOfResult {
    public _prefixSumIndexOfResultBrand: void

    public index: number
    public remainder: number

    constructor(index: number, remainder: number) {
        this.index = index
        this.remainder = remainder
    }
}

export class PrefixSumComputer {
    /**
     * values[i] is the value at index i
     */
    private values: Uint32Array

    /**
     * prefixSum[i] = SUM(heights[j]), 0 <= j <= i
     */
    private prefixSum: Uint32Array

    /**
     * prefixSum[i], 0 <= i <= prefixSumValidIndex can be trusted
     */
    private readonly prefixSumValidIndex: Int32Array

    constructor(values: Uint32Array) {
        this.values = values
        this.prefixSum = new Uint32Array(values.length)
        this.prefixSumValidIndex = new Int32Array(1)
        this.prefixSumValidIndex[0] = -1
    }

    public getCount(): number {
        return this.values.length
    }

    public insertValues(insertIndex: number, insertValues: Uint32Array): boolean {
        insertIndex = toUint32(insertIndex)
        const oldValues = this.values
        const oldPrefixSum = this.prefixSum
        const insertValuesLength = insertValues.length

        if (insertValuesLength === 0) {
            return false
        }

        this.values = new Uint32Array(oldValues.length + insertValuesLength)
        this.values.set(oldValues.subarray(0, insertIndex), 0)
        this.values.set(oldValues.subarray(insertIndex), insertIndex + insertValuesLength)
        this.values.set(insertValues, insertIndex)

        if (insertIndex - 1 < this.prefixSumValidIndex[0]) {
            this.prefixSumValidIndex[0] = insertIndex - 1
        }

        this.prefixSum = new Uint32Array(this.values.length)
        if (this.prefixSumValidIndex[0] >= 0) {
            this.prefixSum.set(oldPrefixSum.subarray(0, this.prefixSumValidIndex[0] + 1))
        }
        return true
    }

    public changeValue(index: number, value: number): boolean {
        index = toUint32(index)
        value = toUint32(value)

        if (this.values[index] === value) {
            return false
        }
        this.values[index] = value
        if (index - 1 < this.prefixSumValidIndex[0]) {
            this.prefixSumValidIndex[0] = index - 1
        }
        return true
    }

    public removeValues(startIndex: number, cnt: number): boolean {
        startIndex = toUint32(startIndex)
        cnt = toUint32(cnt)

        const oldValues = this.values
        const oldPrefixSum = this.prefixSum

        if (startIndex >= oldValues.length) {
            return false
        }

        const maxCnt = oldValues.length - startIndex
        if (cnt >= maxCnt) {
            cnt = maxCnt
        }

        if (cnt === 0) {
            return false
        }

        this.values = new Uint32Array(oldValues.length - cnt)
        this.values.set(oldValues.subarray(0, startIndex), 0)
        this.values.set(oldValues.subarray(startIndex + cnt), startIndex)

        this.prefixSum = new Uint32Array(this.values.length)
        if (startIndex - 1 < this.prefixSumValidIndex[0]) {
            this.prefixSumValidIndex[0] = startIndex - 1
        }
        if (this.prefixSumValidIndex[0] >= 0) {
            this.prefixSum.set(oldPrefixSum.subarray(0, this.prefixSumValidIndex[0] + 1))
        }
        return true
    }

    public getTotalValue(): number {
        if (this.values.length === 0) {
            return 0
        }
        return this._getAccumulatedValue(this.values.length - 1)
    }

    public getAccumulatedValue(index: number): number {
        if (index < 0) {
            return 0
        }

        index = toUint32(index)
        return this._getAccumulatedValue(index)
    }

    private _getAccumulatedValue(valueIndex: number): number {
        if (valueIndex <= this.prefixSumValidIndex[0]) {
            return this.prefixSum[valueIndex]
        }

        let startIndex = this.prefixSumValidIndex[0] + 1
        if (startIndex === 0) {
            this.prefixSum[0] = this.values[0]
            startIndex++
        }

        if (valueIndex >= this.values.length) {
            valueIndex = this.values.length - 1
        }

        for (let index = startIndex; index <= valueIndex; index++) {
            this.prefixSum[index] = this.prefixSum[index - 1] + this.values[index]
        }
        this.prefixSumValidIndex[0] = Math.max(this.prefixSumValidIndex[0], valueIndex)
        return this.prefixSum[valueIndex]
    }

    public getIndexOf(accumulatedValue: number): PrefixSumIndexOfResult {
        accumulatedValue = Math.floor(accumulatedValue) // @perf

        // Compute all sums (to get a fully valid prefixSum)
        this.getTotalValue()

        let low = 0
        let high = this.values.length - 1
        let mid = 0
        let midStop = 0
        let midStart = 0

        while (low <= high) {
            mid = (low + (high - low) / 2) | 0

            midStop = this.prefixSum[mid]
            midStart = midStop - this.values[mid]

            if (accumulatedValue < midStart) {
                high = mid - 1
            } else if (accumulatedValue >= midStop) {
                low = mid + 1
            } else {
                break
            }
        }

        return new PrefixSumIndexOfResult(mid, accumulatedValue - midStart)
    }
}

/** Max unsigned 32-bit integer. */
const MAX_UINT_32 = 4294967295 // 2^32 - 1

export function toUint32(number: number): number {
    if (number < 0) {
        return 0
    }
    if (number > MAX_UINT_32) {
        return MAX_UINT_32
    }
    return number | 0
}
