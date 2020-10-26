# Alerts

Alerts can be configured to notify site admins when there is something wrong or noteworthy on the Sourcegraph instance.

## Understanding alerts

See [alert solutions](alert_solutions.md) for possible solutions when alerts are firing, and learn more about alert labels and metrics in our [metrics guide](metrics_guide.md).

## Setting up alerting

Visit your site configuration (e.g. `https://sourcegraph.example.com/site-admin/configuration`) to configure alerts using the `observability.alerts` field. As always, you can use `Ctrl+Space` at any time to get hints about allowed fields as well as relevant documentation inside the configuration editor.

Once configured, Sourcegraph alerts will automatically be routed to the appropriate notification channels by severity level.

### Example notifiers

#### Slack

```json
"observability.alerts": [
  {
    "level": "critical",
    "notifier": {
      "type": "slack",
      // Slack incoming webhook URL.
      "url": "https://hooks.slack.com/services/xxxxxxxxx/xxxxxxxxxxx/xxxxxxxxxxxxxxxxxxxxxxxx",
    }
  }
]
```

#### PagerDuty

```json
"observability.alerts": [
  {
    "level": "critical",
    "notifier": {
      "type": "pagerduty",
      // Routing key for the PagerDuty Events API v2
      "routingKey": "XXXXXXXX"
    }
  }
]
```

#### Webhook

```json
"observability.alerts": [
  {
    "level": "critical",
    "notifier": {
      "type": "webhook",
      // Webhook URL.
      "url": "https://my.webhook.url"
    }
  }
]
```

Webhook events provide the following fields relevant for Sourcegraph alerts that we recommend you leverage:

<!-- Refer to `commonLabels` on receivers.go for labels we can guarantee will be provided in commonLabels -->

```json
{
  "status": "<resolved|firing>",
  "commonLabels": {
    "level": "<critical|warning>",
    // Use the service name and alert name to find solutions in https://docs.sourcegraph.com/admin/observability/alert_solutions
    "service_name": "<string>",
    "name": "<string>",
    "description": "<string>",
    // This field can be provided to Sourcegraph to help direct support.
    "owner": "<string>"
  },
}
```

For the complete set of fields, please refer to the [Alertmanager webhook documentation](https://prometheus.io/docs/alerting/latest/configuration/#webhook_config).

#### Email

Note that to receive email notifications, the [`email.address`](../config/site_config.md#email-address) and [`email.smtp`](../config/site_config.md#email-smtp) fields must be configured in site configuration.

```json
"observability.alerts": [
  {
    "level": "critical",
    "notifier": {
      "type": "email",
      // Address where alerts will be sent
      "address": "sourcegraph@company.com"
    }
  }
]
```

### Testing alerts

Configured alerts can be tested using the Sourcegraph GraphQL API. Visit your API Console (e.g. `https://sourcegraph.example.com/api/console`) and use the following mutation to trigger an alert:

```gql
mutation {
  triggerObservabilityTestAlert(
    level: "critical"
  ) { alwaysNil }
}
```

The test alert may take up to a minute to fire. The triggered alert will automatically resolve itself as well.

### Silencing alerts

If there is an alert you are aware of and you wish to silence notifications for it, add an entry to the `observability.silenceAlerts` field. For example:

```json
{
  "observability.silenceAlerts": [
    "warning_gitserver_disk_space_remaining"
  ]
}
```

You can find the appropriate identifier for each alert in [alert solutions](./alert_solutions.md).

## Setting up alerting: before Sourcegraph 3.17

### Configure alert channels in Grafana

Before configuring specific alerts in Grafana, you must set up alert channels. Each channel
corresponds to an external service to which Grafana will push alerts.

1. Access Grafana directly as a Grafana admin:
  1. Follow [these instructions](metrics.md#accessing-grafana-directly) to access Grafana directly (instead of going through the Sourcegraph Site Admin Monitoring page as usual).
  1. Navigate to the Grafana `/login` URL (e.g., `http://localhost:3070/-/debug/grafana/login` or append `/login` to your Grafana direct access URL if different). If you are doing this for the first time, user the username and password `admin` and `admin`.
1. In the left sidebar, click the bell icon 🔔 and select `Notification channels`.
1. Click `New channel` and then specify the settings of your channel. The `Type` field selects the type of external service (e.g., PagerDuty, Slack, Email). Some service types will require additional configuration in the service itself. Here are some examples:
  1. Slack
     1. Go to `https://api.slack.com/apps` to create a new Slack App.
     1. Click `Create an App`, give the app a name, and click `Create App`.
     1. Click `Incoming Webhooks` and toggle on `Activate Incoming Webhooks`.
     1. Click `Add New Webhook to Workspace`.
     1. Pick the channel to which this Slack App will post.
     1. Back on the Grafana New Notification Channel page, copy the webhook URL to the `Url` field.
  1. PagerDuty
     1. Go to `https://app.pagerduty.com/developer/apps`.
     1. Click `Create New App`. Give the app a name and decription, and set the category to
        "Application Performance Management". For "Would you like to publish the app for all
        PagerDuty users?", select "No". Click `Save`.
     1. On the Configure App page, under Functionality > Events Integration, click `Add`.
     1. On the Event Integration page, under Events Integration Test > Create a Test Service, enter a name and click `Create`.
     1. Copy the Integration Key. Click `Save` and then `Save` again on the Configure App page.
     1. Back in the Grafana New Notification Channel page, paste the Integration Key into the `Integration Key` field.
1. After you have specified the settings on the Grafana New Notification Channel page, click `Send Test` to send a test notification. You should receive a notification from Grafana via your specified channel. If this worked, click `Save`.

> NOTE: Alerts have a link back to the relevant Grafana panel. In order for these links to work properly Grafana needs
> to know under which external URL it is running (note: this is usually different from the direct access URL you used
> earlier). Set the environment variable `GF_SERVER_ROOT_URL` to your Sourcegraph instance external URL followed
> by the path `/-/debug/grafana`.

### Set up an individual alert

After adding the appropriate notification channels, configure individual alerts to notify those channels.

1. Navigate to the dashboard with the panel and metric for which you'd like to configure an
   alert.
   1. Make sure the dashboard is not read-only (the default Sourcegraph-provided dashboards are
      read-only, because they are provisioned from disk). If the dashboard is read-only, go to the
      dashboard settings (the gear icon in the upper right) and click `Save As..` to create a
      writeable copy.
1. The panel title has a small dropdown next to it. Click the dropdown icon and select `Edit`.
1. In the left sidebar, choose the bell icon 🔔 for Alert.
1. Fill out the fields for the alert rule and select a notification channel.
1. Verify your rule by clicking `Test Rule` or viewing `State History`.
1. Return to the dashboard page by clicking the left arrow in the upper left. Save the dashboard by
   clicking the save icon in the upper right.
