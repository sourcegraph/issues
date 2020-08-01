# Installing Sourcegraph

Once Sourcegraph is deployed and running, the installation process takes only a few minutes. This screencast shows the typical steps involved in installing and configuring a private, self-hosted Sourcegraph instance, ready for sharing internally.

It uses GitHub as the code host, but the steps are largely the same for other code hosts, the only major difference being that GitLab and Bitbucket Server provide native code intelligence integrations.

<div class="container my-4 video-embed embed-responsive embed-responsive-16by9">
    <iframe class="embed-responsive-item" src="https://www.youtube.com/embed/iVTroSw9dhQ?autoplay=0&amp;cc_load_policy=0&amp;start=0&amp;end=0&amp;loop=0&amp;controls=1&amp;modestbranding=0&amp;rel=0" allowfullscreen="" allow="accelerometer; autoplay; encrypted-media; gyroscope; picture-in-picture" frameborder="0"></iframe>
</div>

---

Below is a similar configuration example to the one used in the screencast.

```javascript
// Note that Sourcegraph configuration supports JSONC (JSON with comments).
{
  "url": "https://github.com",
  "token": "<token>",

  "orgs": [
    "sourcegraph"
  ],
  
  // Specific include list, e.g, open source code used internally
  "repos": [
    "hashicorp/terraform",
    "terraform-providers/terraform-provider-aws",
    "jenkinsci/jenkins",
    "facebook/react",
    "twbs/bootstrap",
    "django/django"
  ],

  // Filter out archived and forked repos from repositories included by "orgs"
  "exclude": [
    {
      "archived": true
    },
    {
      "forks": true
    }
  ]
}
```

[**» Next: Connecting and integrating your code host**](connecting_integrating_your_code_host.md)
