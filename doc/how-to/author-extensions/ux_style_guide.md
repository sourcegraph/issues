# Extension UX style guide

In addition to what features your extension has, the way your extension behaves from the user's perspective has a big impact on whether or not they'll decide to keep your extension enabled.

Your extension has to compete against alternative methods of achieving the same goal. For example, users can find the definition of a function in the same file in 2 seconds by double-clicking the variable, pressing <kbd>Ctrl</kbd>-<kbd>C</kbd> <kbd>Ctrl</kbd>-<kbd>F</kbd> <kbd>Ctrl</kbd>-<kbd>V</kbd> and hitting <kbd>Enter</kbd> a few times, or in a different file in 10 seconds by running a Sourcegraph search. Users often times have finely-tuned development environments and can get there in 15 seconds by running `git stash && git checkout <branch> && code src/main.go`. If your extension does not respond in less than that time on every action, users will learn to avoid your extension and probably disable it.

Your extension must provide clear value to the user, rather than clutter their experience with information that could be obtained through simpler and potentially faster methods, for instance a simple text search in the current document to find references to a function.

**Guide the user**

If your extension requires a few settings, gently inform them that it won't work until they provide more information. A file header icon that triggers a prompt usually does the trick nicely.

Try to replicate the experience of a new user activating your extension for the first time. Can they set up your extension properly and intuitively without reading the README? If your extension requires external setup to function properly (eg. a server component), does the README provide clear, precise and reproducible steps to setting up these components?

**Treat UI like real estate**

Every pixel is precious space, and if your extension is taking up more than most, it will be seen as obtrusive and is more likely to get disabled. Consider the macOS menu bar: nothing in the API prevents the networking app from spelling its name out "Wireless Network Connections" as opposed to showing the icon by itself, but you probably couldn't count to ten before people would try to hide it if it did.

Space in the file header is particularly tight, so consider providing more information when the user hovers over your button rather than having it always present in the label.

**Avoid overcommunicating what your extension is doing**

Most of the time the user just opened a pull request and are busy reading the code. Displaying UI elements to them at that time takes attention away from their task at hand. Defer that communication until the user is most likely to want that information (e.g. once they hover over a token).

When faced with a tradeoff between reassuring the user that progress is being made and respecting their attention, err on the quiet side and try to reduce the response time instead of making the progress updates more detailed and granular.

**Write actionable and descriptive errors**

Most users aren't experts in the subject of your extension. If the user did something wrong, be sure to mention in the error messaging what your extension noticed, what it expected, and what the user can do to fix or work around the problem.
