# Back up or migrate Sourcegraph configuration

Backing up this data requires copy-pasting the text from the [configuration JSON files](../background-information/data_storage.md#configuration-json) on the old Sourcegraph instance into the new one.

## 1. Copy the site configuration

- Navigate to **Site admin** -> **Site configuration** on your old Sourcegraph instance.
- Copy the JSON text from the editor.
- Navigate to **Site admin** -> **Site configuration** on your new Sourcegraph instance.
- Completely replace the configuration JSON on the new Sourcegraph instance with the copied text from the old one. You do not need to combine the two configuraitons.

## 2. Copy the global configuration

- Navigate to **Site admin** -> **Global settings** on your old Sourcegraph instance.
- Copy the JSON text from the editor.
- Navigate to **Site admin** -> **Global settings** on your new Sourcegraph instance.
- Completely replace the configuration JSON on the new Sourcegraph instance with the copied text from the old one. You do not need to combine the two configuraitons.

## 3. Copy all code host connections

- Navigate to **Site admin** -> **Manage repositories** on your old Sourcegraph instance.
- For each of the code hosts listed on this page, .
- You will need to re-create each of the code host connections listed. For each code host:
  - Click 'Edit' for the code host.
  - Note the type of code host connection listed at the top of the page. For example "Generic Git host", "GitHub", or Bitbucket Server".
  - Navigate to **Site admin** -> **Manage repositories** on your new Sourcegraph instance.
  - Click 'Add repositories'
  - Select the code host connection type that matches the one you noted above.
  - Copy the 'Display name' from the old configuration into the new connection you are creating.
  - Copy the entire JSON text from the old configuration into the new connection.
  - On the new connection, click 'Add repositories' to save the configuration.
  - Sourcegraph will immediately start to clone the configured repositories from the newly created code host connection.

## 4. Restart the new Sourcegraph instance

- After all settings have been copied into your new Sourcegraph instance, restart the new instance for all changes to take effect.
