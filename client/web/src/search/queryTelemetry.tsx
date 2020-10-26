import { count } from '../../../shared/src/util/strings'
import { parseSearchQuery, ParserResult, Sequence } from '../../../shared/src/search/parser/parser'
import { resolveFilter } from '../../../shared/src/search/parser/filters'

// eslint-disable-next-line @typescript-eslint/explicit-function-return-type
export function queryTelemetryData(query: string, caseSensitive: boolean) {
    return {
        // 🚨 PRIVACY: never provide any private data in { code_search: { query_data: { query } } }.
        query: query ? queryStringTelemetryData(query, caseSensitive) : undefined,
        combined: query,
        empty: !query,
    }
}
function filterExistsInQuery(parsedQuery: ParserResult<Sequence>, filterToMatch: string): boolean {
    if (parsedQuery.type === 'success') {
        const members = parsedQuery.token.members
        for (const member of members) {
            if (member.token.type === 'filter') {
                const resolvedFilter = resolveFilter(member.token.filterType.token.value)
                if (resolvedFilter !== undefined && resolvedFilter.type === filterToMatch) {
                    return true
                }
            }
        }
    }
    return false
}
// eslint-disable-next-line @typescript-eslint/explicit-function-return-type
function queryStringTelemetryData(query: string, caseSensitive: boolean) {
    // 🚨 PRIVACY: never provide any private data in this function's return value.
    // This only takes ~1.7ms per call, so it does not need to be optimized.
    const parsedQuery = parseSearchQuery(query)
    return {
        field_archived: filterExistsInQuery(parsedQuery, 'archived')
            ? {
                  count: count(query, /(^|\s)archived:/g),
              }
            : undefined,
        field_after: filterExistsInQuery(parsedQuery, 'after')
            ? {
                  count: count(query, /(^|\s)after:/g),
                  value_sec: count(query, /(^|\s)after:"[^"]*sec/g),
                  value_min: count(query, /(^|\s)after:"[^"]*min/g),
                  value_hour: count(query, /(^|\s)after:"[^"]*h(ou)?r/g),
                  value_day: count(query, /(^|\s)after:"[^"]*day/g),
                  value_week: count(query, /(^|\s)after:"[^"]*w(ee)?k/g),
                  value_month: count(query, /(^|\s)after:"[^"]*m(on|th)/g),
                  value_year: count(query, /(^|\s)after:"[^"]*y(ea)?r/g),
              }
            : undefined,
        field_author: filterExistsInQuery(parsedQuery, 'author')
            ? {
                  count: count(query, /(^|\s)author:/g),
                  count_negated: count(query, /(^|\s)-author:/g),
                  value_at_sign: count(query, /(^|\s)-?author:[^s]*@/g),
              }
            : undefined,
        field_before: filterExistsInQuery(parsedQuery, 'before')
            ? {
                  count: count(query, /(^|\s)before:/g),
                  value_sec: count(query, /(^|\s)before:"[^"]*sec/g),
                  value_min: count(query, /(^|\s)before:"[^"]*min/g),
                  value_hour: count(query, /(^|\s)before:"[^"]*h(ou)?r/g),
                  value_day: count(query, /(^|\s)before:"[^"]*day/g),
                  value_week: count(query, /(^|\s)before:"[^"]*w(ee)?k/g),
                  value_month: count(query, /(^|\s)before:"[^"]*m(on|th)/g),
                  value_year: count(query, /(^|\s)before:"[^"]*y(ea)?r/g),
              }
            : undefined,
        field_case: caseSensitive
            ? {
                  count: 1,
              }
            : undefined,
        field_committer: filterExistsInQuery(parsedQuery, 'committer')
            ? {
                  count: count(query, /(^|\s)committer:/g),
                  count_negated: count(query, /(^|\s)-committer:/g),
                  value_at_sign: count(query, /(^|\s)-?committer:[^s]*@/g),
              }
            : undefined,
        field_content: filterExistsInQuery(parsedQuery, 'content')
            ? {
                  count: count(query, /(^|\s)content:/g),
              }
            : undefined,
        field_count: filterExistsInQuery(parsedQuery, 'count')
            ? {
                  count: count(query, /(^|\s)count:/g),
              }
            : undefined,
        field_file: filterExistsInQuery(parsedQuery, 'file')
            ? {
                  count: count(query, /(^|\s)f(ile)?:/g),
                  count_negated: count(query, /(^|\s)-f(ile?):/g),
                  count_alias: count(query, /(^|\s)-?f:/g),

                  // likely regexp char
                  value_regexp: count(query, /(^|\s)-?f(ile)?:\S*[$()*?[\]^|]/g),

                  // likely regexp matching a file ext
                  value_regexp_file_ext: count(query, /(^|\s)-?f(ile)?:\S*\.\w+\$\b/g),

                  // oops! user tried to use a (likely) glob
                  value_glob: count(query, /(^|\s)-?f(ile)?:\S*(\*\.|\.{[A-Za-z]|\*\*|\/\*)/g),
              }
            : undefined,
        field_fork: filterExistsInQuery(parsedQuery, 'fork')
            ? {
                  count: count(query, /(^|\s)fork:/g),
              }
            : undefined,
        field_index: filterExistsInQuery(parsedQuery, 'index')
            ? {
                  count: count(query, /(^|\s)index:/g),
              }
            : undefined,
        field_lang: filterExistsInQuery(parsedQuery, 'lang')
            ? {
                  count: count(query, /(^|\s)l(ang)?:/g),
                  count_negated: count(query, /(^|\s)-l(ang)?:/g),
                  count_alias: count(query, /(^|\s)-?l:/g),
              }
            : undefined,
        field_message: filterExistsInQuery(parsedQuery, 'message')
            ? {
                  count: count(query, /(^|\s)m(essage)?:/g),
                  count_negated: count(query, /(^|\s)-m(essage)?:/g),
                  count_alias: count(query, /(^|\s)-?m:/g),
              }
            : undefined,
        field_patterntype: filterExistsInQuery(parsedQuery, 'patterntype')
            ? {
                  count: count(query, /(^|\s)patterntype:/gi),
              }
            : undefined,
        field_repo: filterExistsInQuery(parsedQuery, 'repo')
            ? {
                  count: count(query, /(^|\s)r(epo)?:/g),
                  count_negated: count(query, /(^|\s)-r(epo)?:/g),
                  count_alias: count(query, /(^|\s)-?r:/g),
                  value_at_sign: count(query, /(^|\s)-?r(epo)?:[^s]*@/g),
                  value_pipe: count(query, /(^|\s)-?r(epo)?:[^s]*\|/g),
                  value_rev_star: count(query, /(^|\s)-?r(epo)?:[^s]*@[^s]*\*/g),
                  value_rev_colon: count(query, /(^|\s)-?r(epo)?:[^s]*@[^s]*:/g),
                  value_rev_caret: count(query, /(^|\s)-?r(epo)?:[^s]*@[^s]*\^/g),

                  // likely regexp char
                  value_regexp: count(query, /(^|\s)-?r(epo)?:\S*[$()*?[\]^|]/g),

                  // oops! user tried to use a (likely) glob
                  value_glob: count(query, /(^|\s)-?r(epo)?:\S*(\*\.|\.{[A-Za-z]|\*\*|\/\*)/g),
              }
            : undefined,
        field_repogroup: filterExistsInQuery(parsedQuery, 'repogroup')
            ? {
                  count: count(query, /(^|\s)(repogroup|g):/g),
                  count_negated: count(query, /(^|\s)-(repogroup|g):/g),
                  count_alias: count(query, /(^|\s)-?g:/g),
                  value_active: count(query, /(^|\s)-?(repogroup|g):active/g),
                  value_inactive: count(query, /(^|\s)-?(repogroup|g):active/g),
              }
            : undefined,
        field_repohascommitafter: filterExistsInQuery(parsedQuery, 'repohascommitafter')
            ? {
                  count: count(query, /(^|\s)(repohascommitafter):/g),
              }
            : undefined,
        field_repohasfile: filterExistsInQuery(parsedQuery, 'repohasfile')
            ? {
                  count: count(query, /(^|\s)(repohasfile):/g),
              }
            : undefined,
        field_stable: filterExistsInQuery(parsedQuery, 'stable')
            ? {
                  count: count(query, /(^|\s)(stable):/g),
              }
            : undefined,
        field_timeout: filterExistsInQuery(parsedQuery, 'timeout')
            ? {
                  count: count(query, /(^|\s)timeout:/g),
              }
            : undefined,
        field_type: filterExistsInQuery(parsedQuery, 'type')
            ? {
                  count: count(query, /(^|\s)type:/g),
                  value_file: count(query, /(^|\s)type:file(\s|$)/g),
                  value_diff: count(query, /(^|\s)type:diff(\s|$)/g),
                  value_commit: count(query, /(^|\s)type:commit(\s|$)/g),
                  value_symbol: count(query, /(^|\s)type:symbol(\s|$)/g),
              }
            : undefined,
        field_default: defaultQueryFieldTelemetryData(query),
        fields: {
            count: count(query, /(^|\s)(\w+:)?([^\s"'/:]+|"[^"]*"|'[^']*'|\/[^/]*\/)/g),
            count_non_default: count(query, /(^|\s)\w+:([^\s"'/:]+|"[^"]*"|'[^']*')/g),
        },
        chars: {
            count: query.length,
            non_ascii: count(query, /[^\t\n\r -~]/g),
            space: count(query, /\s/g),
            double_quote: count(query, /"/g),
            single_quote: count(query, /'/g),
        },
    }
}

function defaultQueryFieldTelemetryData(query: string): { [key: string]: any } {
    // 🚨 PRIVACY: never provide any private data in this function's return value.

    // Strip non-default fields. Does not account for backslashes.
    query = query.replace(/(^|\s)\w+:([^\s"'/:]+|"[^"]*"|'[^']*')/g, ' ')

    return {
        count: count(query, /(^|\s)([^\s"'/:]+|"[^"]*"|'[^']*'|\/[^/]*\/)/g),
        count_literal: count(query, /(^|\s)[^\s"'/:]+/g),
        count_double_quote: count(query, /(^|\s)"[^"]*"/g),
        count_single_quote: count(query, /(^|\s)'[^']*'/g),
        count_pattern: count(query, /(^|\s)\/[^/]*\//g),
        count_regexp: count(query, /(^|\s)[$()*?[\]^|]/g),
    }
}
