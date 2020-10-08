---
ignoreDisconnectedPageCheck: true
---

# Sourcegraph @ dotGo EU 2019

We loved meeting so many awesome and talented engineers at dotGo EU 2019. The Go community certainly is passionate about developer tools and productivity, and we'll be back next year for sure! - [Quinn](https://twitter.com/sqs), [Loic (left)](https://twitter.com/lguychard) and [Ryan (right)](https://twitter.com/ryan_blunden)

<p class="text-center">
  <img src="https://user-images.githubusercontent.com/133014/55120082-ba819a00-50b1-11e9-8823-6a1fead17a42.jpg" style="width:60%;" />
</p>

> NOTE: If you weren't able to chat with us at the conference and want guidance for setting up Sourcegraph for your team, [tweet (@srcgraph)](https://twitter.com/srcgraph) or [email](mailto:hi@sourcegraph.com?subject=dotGo%202019).

The demos you saw at the conference are below:

## 👉 IDE-like code intelligence on GitHub PRs

1. [**Start reviewing this GitHub PR**](https://github.com/sourcegraph/go-diff/pull/31/files#diff-334200dfc76f817e050f8dc5d9745843R19)
1. Hover on `PrintMultiFileDiff(`
2. Explore from there (who calls it?), using **Go to definition** and **Find references**

You can see a demo of this below.

<p class="container">
  <div style="padding:56.25% 0 0 0;position:relative;">
    <iframe src="https://player.vimeo.com/video/326271362?color=0CB6F4&title=0&byline=0&portrait=0" style="position:absolute;top:0;left:0;width:100%;height:100%;" frameborder="0" webkitallowfullscreen mozallowfullscreen allowfullscreen></iframe>
  </div>
</p>

<p class="text-center">[🎬 View on Vimeo.com](https://vimeo.com/326271362)<p>

## 👉 Reuse existing code and find examples

Save time when you're (for example) writing a new HTTP transport:

1. [**Search for `http.(RoundTripper|Transport)`**](https://sourcegraph.com/search?q=http.%28Transport%7CRoundTripper%29)
1. Add [`type:diff author:Keegan`](https://sourcegraph.com/search?q=http.%28Transport%7CRoundTripper%29+type:diff+author:keegan) to the query to find recent code from your smart teammate
1. Explore from there!
