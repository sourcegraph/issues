# Deeper dive into Sourcegraph search modes

> NOTE: If you're new to code search in general, check out our blog post [The find-and-replace Odyssey, a programmer's guide](https://about.sourcegraph.com/blog/a-programmers-guide-to-find-and-replace) which takes you on a journey from using basic search and replace in your editor to the most advanced commandline tools, and of course, Sourcegraph.

Sourcegraph provides three search modes with different syntax:

- **Exact string matching (literal)**<br/>
The default and simplest search mode.<br/>

- **Regexp**<br/>
More complex but essential when you need to match based on patterns instead of being limited to an exact string.<br/>

- **Structural search**<br/>
A brand new type of language aware search that understands the syntactic structure of programming languages.

Learn more about each search mode and when to use them by watching the videos in each section below.

## Exact string search (literal search)

Learn when to use literal search and how the `content` filter handles searches that contain a colon such as [`content:"FROM python:3"`](https://sourcegraph.com/search?q=content:%22FROM+python:3%22&patternType=literal).

<div class="container my-4 video-embed embed-responsive embed-responsive-16by9">
    <iframe class="embed-responsive-item" src="https://www.youtube.com/embed/CX6F5oCjfoc?autoplay=0&amp;cc_load_policy=0&amp;start=0&amp;end=0&amp;loop=0&amp;controls=1&amp;modestbranding=0&amp;rel=0" allowfullscreen="" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" frameborder="0"></iframe>
</div>

## Regular expression search

Exact string searches are great for simple cases, but regexp queries and filters (such as `file` that accept regexp), amplify your ability to discover code by finding matches based on patterns.

<div class="container my-4 video-embed embed-responsive embed-responsive-16by9">
    <iframe class="embed-responsive-item" src="https://www.youtube.com/embed/J9k7l5W1qbk?autoplay=0&amp;cc_load_policy=0&amp;start=0&amp;end=0&amp;loop=0&amp;controls=1&amp;modestbranding=0&amp;rel=0" allowfullscreen="" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" frameborder="0"></iframe>
</div>

## Structural search

Sourcegraph structural search is language structure and syntax aware search syntax that lets you match nested expressions and whole code blocks that can be difficult or impossible using regexp.

Learn more by checking out the [structural search docs](../user/search/structural.md) and the [going beyond regexp with structural code search blog post](https://about.sourcegraph.com/blog/going-beyond-regular-expressions-with-structural-code-search/).

<div style="padding:56.25% 0 0 0;position:relative;">
    <iframe src="https://www.youtube.com/embed/Lg4cYEoSHeo" style="position:absolute;top:0;left:0;width:100%;height:100%;" frameborder="0" webkitallowfullscreen="" mozallowfullscreen="" allowfullscreen=""></iframe>
</div>

---

[**» Next: Universal Code Intelligence**](universal_code_intelligence.md)
