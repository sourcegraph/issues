package main

import (
	"regexp"
	"sort"
	"strings"
)

// Emojis returns a string of emojis that should be displayed with an issue or a pull request.
// Additional emojis can be supplied and will overwrite any emoji with the same category.
func Emojis(labels []string, repository, body string, additional map[string]string) string {
	categories := map[string]string{}
	for _, categorizer := range categorizers {
		categorizer(labels, repository, body, categories)
	}
	for k, v := range additional {
		categories[k] = v
	}

	var sorted []string
	for _, emoji := range categories {
		sorted = append(sorted, emoji)
	}
	sort.Strings(sorted)

	return strings.Join(sorted, "")
}

var categorizers = []func(labels []string, repository, body string, categories map[string]string){
	categorizeSecurityIssue,
	categorizeCustomerIssue,
	categorizeLabels,
}

// categorizeSecurityIssue adds a security emoji if the repository matches sourcegraph/security-issues.
func categorizeSecurityIssue(labels []string, repository, body string, categories map[string]string) {
	if repository == "sourcegraph/security-issues" {
		categories["security"] = emojis["security"]
	}
}

var customerMatcher = regexp.MustCompile(`https://app\.hubspot\.com/contacts/2762526/company/\d+`)

// categorizeCustomerIssue adds a customer emoji if the repository matches sourcegraph/customer or if
// the issue contains a hubspot URL.
func categorizeCustomerIssue(labels []string, repository, body string, categories map[string]string) {
	if repository == "sourcegraph/customer" || contains(labels, "customer") {
		if customer := customerMatcher.FindString(body); customer != "" {
			categories["customer"] = "[👩](" + customer + ")"
		} else {
			categories["customer"] = "👩"
		}
	}
}

var emojis = map[string]string{
	"bug":             "🐛",
	"debt":            "🧶",
	"quality-of-life": "🎩",
	"roadmap":         "🛠️",
	"security":        "🔒",
	"spike":           "🕵️",
}

// categorizeLabels adds emojis based on the issue labels.
func categorizeLabels(labels []string, repository, body string, categories map[string]string) {
	for _, label := range labels {
		if emoji, ok := emojis[label]; ok {
			categories[label] = emoji
		}
	}
}
