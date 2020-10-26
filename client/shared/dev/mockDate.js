// @ts-check

// Mock the global Date during tests for determinism.
const MockDate = require('mockdate')
MockDate.set(1136239445000, 0) // Mon Jan 2 15:04:05 MST 2006 (arbitrary)
