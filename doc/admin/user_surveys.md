# User surveys

## Submit a survey

After using Sourcegraph for a few days, users are presented with a request to answer "How likely is it that you would recommend Sourcegraph to a friend?", and to provide some qualitative product feedback.

Users can also always submit (or re-submit) this survey by visiting `https://sourcegraph.example.com/survey`.

## View survey responses

Responses to this survey are visible to all site admins on the Sourcegraph instance. View all responses on the **Site admin > User surveys** page. (The URL is `https://sourcegraph.example.com/site-admin/surveys`.)

On this page, admins can view individual satisfaction scores (from 0–10) and individual qualitative feedback, along with aggregate historical scores.

Survey responses are also always sent to Sourcegraph.com.

## Restart feedback survey

By default, users are only presented with the feedback survey once. Site admins may restart the feedback survey for all users (regardless of whether they have already responded). To restart the feedback survey, use the [site configuration's `htmlBodyBottom` property](../admin/config/site_config.md#reference):

```json
{
  ...,
  "htmlBodyBottom": "<script>if (localStorage.getItem('reset-survey-000') === null) { localStorage.removeItem('has-dismissed-survey-toast'); localStorage.setItem('days-active-count', 3); localStorage.setItem('reset-survey-000', true); }</script>",
  ...
}
```

This is a temporary solution that injects a JavaScript code snippet that resets the "has dismissed survey" flag for each user. Users will be prompted for feedback in the same way the next time they view a repository, directory, or file page.

To restart the feedback survey for a second (or subsequent) time, change both instances of `-000` to `-001` (or `-002`, etc.). Follow and participate in [issue #666](https://github.com/sourcegraph/sourcegraph/issues/666) if you need a more complete solution.


## See also 

- [Usage statistics](usage_statistics.md)
