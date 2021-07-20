module progress-bot

go 1.16

require (
	cloud.google.com/go/storage v1.14.0
	github.com/cockroachdb/errors v1.8.5
	github.com/drexedam/gravatar v0.0.0-20170403222345-e4917c5607c3
	github.com/ozankasikci/go-image-merge v0.2.2
	github.com/slack-go/slack v0.8.1
	github.com/yuin/goldmark v1.3.5
)

replace github.com/ozankasikci/go-image-merge => github.com/sourcegraph/go-image-merge v0.2.3-0.20210226214948-f91742c8193e
