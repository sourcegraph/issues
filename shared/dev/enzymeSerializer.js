import { createSerializer } from 'enzyme-to-json'
import { replaceVerboseObjects } from '../src/util/enzymeSnapshotModifiers'

module.exports = createSerializer({ map: replaceVerboseObjects })
