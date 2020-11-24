# Table "public.access_tokens"
```
     Column      |           Type           |                         Modifiers                          
-----------------+--------------------------+------------------------------------------------------------
 id              | bigint                   | not null default nextval('access_tokens_id_seq'::regclass)
 subject_user_id | integer                  | not null
 value_sha256    | bytea                    | not null
 note            | text                     | not null
 created_at      | timestamp with time zone | not null default now()
 last_used_at    | timestamp with time zone | 
 deleted_at      | timestamp with time zone | 
 creator_user_id | integer                  | not null
 scopes          | text[]                   | not null
Indexes:
    "access_tokens_pkey" PRIMARY KEY, btree (id)
    "access_tokens_value_sha256_key" UNIQUE CONSTRAINT, btree (value_sha256)
    "access_tokens_lookup" hash (value_sha256) WHERE deleted_at IS NULL
Foreign-key constraints:
    "access_tokens_creator_user_id_fkey" FOREIGN KEY (creator_user_id) REFERENCES users(id)
    "access_tokens_subject_user_id_fkey" FOREIGN KEY (subject_user_id) REFERENCES users(id)

```

# Table "public.campaign_specs"
```
      Column       |           Type           |                          Modifiers                          
-------------------+--------------------------+-------------------------------------------------------------
 id                | bigint                   | not null default nextval('campaign_specs_id_seq'::regclass)
 rand_id           | text                     | not null
 raw_spec          | text                     | not null
 spec              | jsonb                    | not null default '{}'::jsonb
 namespace_user_id | integer                  | 
 namespace_org_id  | integer                  | 
 user_id           | integer                  | 
 created_at        | timestamp with time zone | not null default now()
 updated_at        | timestamp with time zone | not null default now()
Indexes:
    "campaign_specs_pkey" PRIMARY KEY, btree (id)
    "campaign_specs_rand_id" btree (rand_id)
Check constraints:
    "campaign_specs_has_1_namespace" CHECK ((namespace_user_id IS NULL) <> (namespace_org_id IS NULL))
Foreign-key constraints:
    "campaign_specs_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL DEFERRABLE
Referenced by:
    TABLE "campaigns" CONSTRAINT "campaigns_campaign_spec_id_fkey" FOREIGN KEY (campaign_spec_id) REFERENCES campaign_specs(id) DEFERRABLE
    TABLE "changeset_specs" CONSTRAINT "changeset_specs_campaign_spec_id_fkey" FOREIGN KEY (campaign_spec_id) REFERENCES campaign_specs(id) DEFERRABLE

```

# Table "public.campaigns"
```
       Column       |           Type           |                       Modifiers                        
--------------------+--------------------------+--------------------------------------------------------
 id                 | bigint                   | not null default nextval('campaigns_id_seq'::regclass)
 name               | text                     | not null
 description        | text                     | 
 initial_applier_id | integer                  | 
 namespace_user_id  | integer                  | 
 namespace_org_id   | integer                  | 
 created_at         | timestamp with time zone | not null default now()
 updated_at         | timestamp with time zone | not null default now()
 changeset_ids      | jsonb                    | not null default '{}'::jsonb
 closed_at          | timestamp with time zone | 
 campaign_spec_id   | bigint                   | not null
 last_applier_id    | bigint                   | 
 last_applied_at    | timestamp with time zone | not null
Indexes:
    "campaigns_pkey" PRIMARY KEY, btree (id)
    "campaigns_changeset_ids_gin_idx" gin (changeset_ids)
    "campaigns_namespace_org_id" btree (namespace_org_id)
    "campaigns_namespace_user_id" btree (namespace_user_id)
Check constraints:
    "campaigns_changeset_ids_check" CHECK (jsonb_typeof(changeset_ids) = 'object'::text)
    "campaigns_has_1_namespace" CHECK ((namespace_user_id IS NULL) <> (namespace_org_id IS NULL))
    "campaigns_name_not_blank" CHECK (name <> ''::text)
Foreign-key constraints:
    "campaigns_campaign_spec_id_fkey" FOREIGN KEY (campaign_spec_id) REFERENCES campaign_specs(id) DEFERRABLE
    "campaigns_initial_applier_id_fkey" FOREIGN KEY (initial_applier_id) REFERENCES users(id) ON DELETE SET NULL DEFERRABLE
    "campaigns_last_applier_id_fkey" FOREIGN KEY (last_applier_id) REFERENCES users(id) ON DELETE SET NULL DEFERRABLE
    "campaigns_namespace_org_id_fkey" FOREIGN KEY (namespace_org_id) REFERENCES orgs(id) ON DELETE CASCADE DEFERRABLE
    "campaigns_namespace_user_id_fkey" FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE
Referenced by:
    TABLE "changesets" CONSTRAINT "changesets_owned_by_campaign_id_fkey" FOREIGN KEY (owned_by_campaign_id) REFERENCES campaigns(id) ON DELETE SET NULL DEFERRABLE
Triggers:
    trig_delete_campaign_reference_on_changesets AFTER DELETE ON campaigns FOR EACH ROW EXECUTE PROCEDURE delete_campaign_reference_on_changesets()

```

# Table "public.campaigns_old"
```
       Column       |           Type           | Modifiers 
--------------------+--------------------------+-----------
 id                 | bigint                   | 
 name               | text                     | 
 description        | text                     | 
 initial_applier_id | integer                  | 
 namespace_user_id  | integer                  | 
 namespace_org_id   | integer                  | 
 created_at         | timestamp with time zone | 
 updated_at         | timestamp with time zone | 
 changeset_ids      | jsonb                    | 
 closed_at          | timestamp with time zone | 
 campaign_spec_id   | bigint                   | 
 last_applier_id    | bigint                   | 
 last_applied_at    | timestamp with time zone | 

```

# Table "public.changeset_events"
```
    Column    |           Type           |                           Modifiers                           
--------------+--------------------------+---------------------------------------------------------------
 id           | bigint                   | not null default nextval('changeset_events_id_seq'::regclass)
 changeset_id | bigint                   | not null
 kind         | text                     | not null
 key          | text                     | not null
 created_at   | timestamp with time zone | not null default now()
 metadata     | jsonb                    | not null default '{}'::jsonb
 updated_at   | timestamp with time zone | not null default now()
Indexes:
    "changeset_events_pkey" PRIMARY KEY, btree (id)
    "changeset_events_changeset_id_kind_key_unique" UNIQUE CONSTRAINT, btree (changeset_id, kind, key)
Check constraints:
    "changeset_events_key_check" CHECK (key <> ''::text)
    "changeset_events_kind_check" CHECK (kind <> ''::text)
    "changeset_events_metadata_check" CHECK (jsonb_typeof(metadata) = 'object'::text)
Foreign-key constraints:
    "changeset_events_changeset_id_fkey" FOREIGN KEY (changeset_id) REFERENCES changesets(id) ON DELETE CASCADE DEFERRABLE

```

# Table "public.changeset_specs"
```
      Column       |           Type           |                          Modifiers                           
-------------------+--------------------------+--------------------------------------------------------------
 id                | bigint                   | not null default nextval('changeset_specs_id_seq'::regclass)
 rand_id           | text                     | not null
 raw_spec          | text                     | not null
 spec              | jsonb                    | not null default '{}'::jsonb
 campaign_spec_id  | bigint                   | 
 repo_id           | integer                  | not null
 user_id           | integer                  | 
 diff_stat_added   | integer                  | 
 diff_stat_changed | integer                  | 
 diff_stat_deleted | integer                  | 
 created_at        | timestamp with time zone | not null default now()
 updated_at        | timestamp with time zone | not null default now()
Indexes:
    "changeset_specs_pkey" PRIMARY KEY, btree (id)
    "changeset_specs_rand_id" btree (rand_id)
Foreign-key constraints:
    "changeset_specs_campaign_spec_id_fkey" FOREIGN KEY (campaign_spec_id) REFERENCES campaign_specs(id) DEFERRABLE
    "changeset_specs_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) DEFERRABLE
    "changeset_specs_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL DEFERRABLE
Referenced by:
    TABLE "changesets" CONSTRAINT "changesets_changeset_spec_id_fkey" FOREIGN KEY (current_spec_id) REFERENCES changeset_specs(id) DEFERRABLE
    TABLE "changesets" CONSTRAINT "changesets_previous_spec_id_fkey" FOREIGN KEY (previous_spec_id) REFERENCES changeset_specs(id) DEFERRABLE

```

# Table "public.changesets"
```
        Column         |           Type           |                        Modifiers                        
-----------------------+--------------------------+---------------------------------------------------------
 id                    | bigint                   | not null default nextval('changesets_id_seq'::regclass)
 campaign_ids          | jsonb                    | not null default '{}'::jsonb
 repo_id               | integer                  | not null
 created_at            | timestamp with time zone | not null default now()
 updated_at            | timestamp with time zone | not null default now()
 metadata              | jsonb                    | default '{}'::jsonb
 external_id           | text                     | 
 external_service_type | text                     | not null
 external_deleted_at   | timestamp with time zone | 
 external_branch       | text                     | 
 external_updated_at   | timestamp with time zone | 
 external_state        | text                     | 
 external_review_state | text                     | 
 external_check_state  | text                     | 
 added_to_campaign     | boolean                  | not null default false
 diff_stat_added       | integer                  | 
 diff_stat_changed     | integer                  | 
 diff_stat_deleted     | integer                  | 
 sync_state            | jsonb                    | not null default '{}'::jsonb
 current_spec_id       | bigint                   | 
 previous_spec_id      | bigint                   | 
 publication_state     | text                     | default 'UNPUBLISHED'::text
 owned_by_campaign_id  | bigint                   | 
 reconciler_state      | text                     | default 'queued'::text
 failure_message       | text                     | 
 started_at            | timestamp with time zone | 
 finished_at           | timestamp with time zone | 
 process_after         | timestamp with time zone | 
 num_resets            | integer                  | not null default 0
 unsynced              | boolean                  | not null default false
 closing               | boolean                  | not null default false
 num_failures          | integer                  | not null default 0
 log_contents          | text                     | 
 history               | jsonb                    | 
Indexes:
    "changesets_pkey" PRIMARY KEY, btree (id)
    "changesets_repo_external_id_unique" UNIQUE CONSTRAINT, btree (repo_id, external_id)
Check constraints:
    "changesets_campaign_ids_check" CHECK (jsonb_typeof(campaign_ids) = 'object'::text)
    "changesets_external_id_check" CHECK (external_id <> ''::text)
    "changesets_external_service_type_not_blank" CHECK (external_service_type <> ''::text)
    "changesets_metadata_check" CHECK (jsonb_typeof(metadata) = 'object'::text)
    "external_branch_ref_prefix" CHECK (external_branch ~~ 'refs/heads/%'::text)
Foreign-key constraints:
    "changesets_changeset_spec_id_fkey" FOREIGN KEY (current_spec_id) REFERENCES changeset_specs(id) DEFERRABLE
    "changesets_owned_by_campaign_id_fkey" FOREIGN KEY (owned_by_campaign_id) REFERENCES campaigns(id) ON DELETE SET NULL DEFERRABLE
    "changesets_previous_spec_id_fkey" FOREIGN KEY (previous_spec_id) REFERENCES changeset_specs(id) DEFERRABLE
    "changesets_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE DEFERRABLE
Referenced by:
    TABLE "changeset_events" CONSTRAINT "changeset_events_changeset_id_fkey" FOREIGN KEY (changeset_id) REFERENCES changesets(id) ON DELETE CASCADE DEFERRABLE
Triggers:
    trig_delete_changeset_reference_on_campaigns AFTER DELETE ON changesets FOR EACH ROW EXECUTE PROCEDURE delete_changeset_reference_on_campaigns()

```

# Table "public.changesets_old"
```
        Column         |           Type           | Modifiers 
-----------------------+--------------------------+-----------
 id                    | bigint                   | 
 campaign_ids          | jsonb                    | 
 repo_id               | integer                  | 
 created_at            | timestamp with time zone | 
 updated_at            | timestamp with time zone | 
 metadata              | jsonb                    | 
 external_id           | text                     | 
 external_service_type | text                     | 
 external_deleted_at   | timestamp with time zone | 
 external_branch       | text                     | 
 external_updated_at   | timestamp with time zone | 
 external_state        | text                     | 
 external_review_state | text                     | 
 external_check_state  | text                     | 
 created_by_campaign   | boolean                  | 
 added_to_campaign     | boolean                  | 
 diff_stat_added       | integer                  | 
 diff_stat_changed     | integer                  | 
 diff_stat_deleted     | integer                  | 
 sync_state            | jsonb                    | 
 current_spec_id       | bigint                   | 
 previous_spec_id      | bigint                   | 
 publication_state     | text                     | 
 owned_by_campaign_id  | bigint                   | 
 reconciler_state      | text                     | 
 failure_message       | text                     | 
 started_at            | timestamp with time zone | 
 finished_at           | timestamp with time zone | 
 process_after         | timestamp with time zone | 
 num_resets            | integer                  | 

```

# Table "public.cm_emails"
```
   Column   |           Type           |                       Modifiers                        
------------+--------------------------+--------------------------------------------------------
 id         | bigint                   | not null default nextval('cm_emails_id_seq'::regclass)
 monitor    | bigint                   | not null
 enabled    | boolean                  | not null
 priority   | cm_email_priority        | not null
 header     | text                     | not null
 created_by | integer                  | not null
 created_at | timestamp with time zone | not null default now()
 changed_by | integer                  | not null
 changed_at | timestamp with time zone | not null default now()
Indexes:
    "cm_emails_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "cm_emails_changed_by_fk" FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE CASCADE
    "cm_emails_created_by_fk" FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
    "cm_emails_monitor" FOREIGN KEY (monitor) REFERENCES cm_monitors(id) ON DELETE CASCADE
Referenced by:
    TABLE "cm_recipients" CONSTRAINT "cm_recipients_emails" FOREIGN KEY (email) REFERENCES cm_emails(id) ON DELETE CASCADE

```

# Table "public.cm_monitors"
```
      Column       |           Type           |                        Modifiers                         
-------------------+--------------------------+----------------------------------------------------------
 id                | bigint                   | not null default nextval('cm_monitors_id_seq'::regclass)
 created_by        | integer                  | not null
 created_at        | timestamp with time zone | not null default now()
 description       | text                     | not null
 changed_at        | timestamp with time zone | not null default now()
 changed_by        | integer                  | not null
 enabled           | boolean                  | not null default true
 namespace_user_id | integer                  | 
 namespace_org_id  | integer                  | 
Indexes:
    "cm_monitors_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "cm_monitors_changed_by_fk" FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE CASCADE
    "cm_monitors_created_by_fk" FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
    "cm_monitors_org_id_fk" FOREIGN KEY (namespace_org_id) REFERENCES orgs(id) ON DELETE CASCADE
    "cm_monitors_user_id_fk" FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE
Referenced by:
    TABLE "cm_emails" CONSTRAINT "cm_emails_monitor" FOREIGN KEY (monitor) REFERENCES cm_monitors(id) ON DELETE CASCADE
    TABLE "cm_queries" CONSTRAINT "cm_triggers_monitor" FOREIGN KEY (monitor) REFERENCES cm_monitors(id) ON DELETE CASCADE

```

# Table "public.cm_queries"
```
   Column   |           Type           |                        Modifiers                        
------------+--------------------------+---------------------------------------------------------
 id         | bigint                   | not null default nextval('cm_queries_id_seq'::regclass)
 monitor    | bigint                   | not null
 query      | text                     | not null
 created_by | integer                  | not null
 created_at | timestamp with time zone | not null default now()
 changed_by | integer                  | not null
 changed_at | timestamp with time zone | not null default now()
Indexes:
    "cm_queries_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "cm_triggers_changed_by_fk" FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE CASCADE
    "cm_triggers_created_by_fk" FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
    "cm_triggers_monitor" FOREIGN KEY (monitor) REFERENCES cm_monitors(id) ON DELETE CASCADE

```

# Table "public.cm_recipients"
```
      Column       |  Type   |                         Modifiers                          
-------------------+---------+------------------------------------------------------------
 id                | bigint  | not null default nextval('cm_recipients_id_seq'::regclass)
 email             | bigint  | not null
 namespace_user_id | integer | 
 namespace_org_id  | integer | 
Indexes:
    "cm_recipients_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "cm_recipients_emails" FOREIGN KEY (email) REFERENCES cm_emails(id) ON DELETE CASCADE
    "cm_recipients_org_id_fk" FOREIGN KEY (namespace_org_id) REFERENCES orgs(id) ON DELETE CASCADE
    "cm_recipients_user_id_fk" FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE

```

# Table "public.codeintel_schema_migrations"
```
 Column  |  Type   | Modifiers 
---------+---------+-----------
 version | bigint  | not null
 dirty   | boolean | not null
Indexes:
    "codeintel_schema_migrations_pkey" PRIMARY KEY, btree (version)

```

# Table "public.critical_and_site_config"
```
   Column   |           Type           |                               Modifiers                               
------------+--------------------------+-----------------------------------------------------------------------
 id         | integer                  | not null default nextval('critical_and_site_config_id_seq'::regclass)
 type       | critical_or_site         | not null
 contents   | text                     | not null
 created_at | timestamp with time zone | not null default now()
 updated_at | timestamp with time zone | not null default now()
Indexes:
    "critical_and_site_config_pkey" PRIMARY KEY, btree (id)
    "critical_and_site_config_unique" UNIQUE, btree (id, type)

```

# Table "public.default_repos"
```
 Column  |  Type   | Modifiers 
---------+---------+-----------
 repo_id | integer | not null
Indexes:
    "default_repos_pkey" PRIMARY KEY, btree (repo_id)
Foreign-key constraints:
    "default_repos_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE

```

# Table "public.discussion_comments"
```
     Column     |           Type           |                            Modifiers                             
----------------+--------------------------+------------------------------------------------------------------
 id             | bigint                   | not null default nextval('discussion_comments_id_seq'::regclass)
 thread_id      | bigint                   | not null
 author_user_id | integer                  | not null
 contents       | text                     | not null
 created_at     | timestamp with time zone | not null default now()
 updated_at     | timestamp with time zone | not null default now()
 deleted_at     | timestamp with time zone | 
 reports        | text[]                   | not null default '{}'::text[]
Indexes:
    "discussion_comments_pkey" PRIMARY KEY, btree (id)
    "discussion_comments_author_user_id_idx" btree (author_user_id)
    "discussion_comments_reports_array_length_idx" btree (array_length(reports, 1))
    "discussion_comments_thread_id_idx" btree (thread_id)
Foreign-key constraints:
    "discussion_comments_author_user_id_fkey" FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT
    "discussion_comments_thread_id_fkey" FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE

```

# Table "public.discussion_mail_reply_tokens"
```
   Column   |           Type           | Modifiers 
------------+--------------------------+-----------
 token      | text                     | not null
 user_id    | integer                  | not null
 thread_id  | bigint                   | not null
 deleted_at | timestamp with time zone | 
Indexes:
    "discussion_mail_reply_tokens_pkey" PRIMARY KEY, btree (token)
    "discussion_mail_reply_tokens_user_id_thread_id_idx" btree (user_id, thread_id)
Foreign-key constraints:
    "discussion_mail_reply_tokens_thread_id_fkey" FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE
    "discussion_mail_reply_tokens_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT

```

# Table "public.discussion_threads"
```
     Column     |           Type           |                            Modifiers                            
----------------+--------------------------+-----------------------------------------------------------------
 id             | bigint                   | not null default nextval('discussion_threads_id_seq'::regclass)
 author_user_id | integer                  | not null
 title          | text                     | 
 target_repo_id | bigint                   | 
 created_at     | timestamp with time zone | not null default now()
 archived_at    | timestamp with time zone | 
 updated_at     | timestamp with time zone | not null default now()
 deleted_at     | timestamp with time zone | 
Indexes:
    "discussion_threads_pkey" PRIMARY KEY, btree (id)
    "discussion_threads_author_user_id_idx" btree (author_user_id)
Foreign-key constraints:
    "discussion_threads_author_user_id_fkey" FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT
    "discussion_threads_target_repo_id_fk" FOREIGN KEY (target_repo_id) REFERENCES discussion_threads_target_repo(id) ON DELETE CASCADE
Referenced by:
    TABLE "discussion_comments" CONSTRAINT "discussion_comments_thread_id_fkey" FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE
    TABLE "discussion_mail_reply_tokens" CONSTRAINT "discussion_mail_reply_tokens_thread_id_fkey" FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE
    TABLE "discussion_threads_target_repo" CONSTRAINT "discussion_threads_target_repo_thread_id_fkey" FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE

```

# Table "public.discussion_threads_target_repo"
```
     Column      |  Type   |                                  Modifiers                                  
-----------------+---------+-----------------------------------------------------------------------------
 id              | bigint  | not null default nextval('discussion_threads_target_repo_id_seq'::regclass)
 thread_id       | bigint  | not null
 repo_id         | integer | not null
 path            | text    | 
 branch          | text    | 
 revision        | text    | 
 start_line      | integer | 
 end_line        | integer | 
 start_character | integer | 
 end_character   | integer | 
 lines_before    | text    | 
 lines           | text    | 
 lines_after     | text    | 
Indexes:
    "discussion_threads_target_repo_pkey" PRIMARY KEY, btree (id)
    "discussion_threads_target_repo_repo_id_path_idx" btree (repo_id, path)
Foreign-key constraints:
    "discussion_threads_target_repo_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE
    "discussion_threads_target_repo_thread_id_fkey" FOREIGN KEY (thread_id) REFERENCES discussion_threads(id) ON DELETE CASCADE
Referenced by:
    TABLE "discussion_threads" CONSTRAINT "discussion_threads_target_repo_id_fk" FOREIGN KEY (target_repo_id) REFERENCES discussion_threads_target_repo(id) ON DELETE CASCADE

```

# Table "public.event_logs"
```
      Column       |           Type           |                        Modifiers                        
-------------------+--------------------------+---------------------------------------------------------
 id                | bigint                   | not null default nextval('event_logs_id_seq'::regclass)
 name              | text                     | not null
 url               | text                     | not null
 user_id           | integer                  | not null
 anonymous_user_id | text                     | not null
 source            | text                     | not null
 argument          | jsonb                    | not null
 version           | text                     | not null
 timestamp         | timestamp with time zone | not null
Indexes:
    "event_logs_pkey" PRIMARY KEY, btree (id)
    "event_logs_anonymous_user_id" btree (anonymous_user_id)
    "event_logs_name" btree (name)
    "event_logs_source" btree (source)
    "event_logs_timestamp" btree ("timestamp")
    "event_logs_timestamp_at_utc" btree (date(timezone('UTC'::text, "timestamp")))
    "event_logs_user_id" btree (user_id)
Check constraints:
    "event_logs_check_has_user" CHECK (user_id = 0 AND anonymous_user_id <> ''::text OR user_id <> 0 AND anonymous_user_id = ''::text OR user_id <> 0 AND anonymous_user_id <> ''::text)
    "event_logs_check_name_not_empty" CHECK (name <> ''::text)
    "event_logs_check_source_not_empty" CHECK (source <> ''::text)
    "event_logs_check_version_not_empty" CHECK (version <> ''::text)

```

# Table "public.external_service_repos"
```
       Column        |  Type   | Modifiers 
---------------------+---------+-----------
 external_service_id | bigint  | not null
 repo_id             | integer | not null
 clone_url           | text    | not null
Indexes:
    "external_service_repos_external_service_id" btree (external_service_id)
    "external_service_repos_repo_id" btree (repo_id)
Foreign-key constraints:
    "external_service_repos_external_service_id_fkey" FOREIGN KEY (external_service_id) REFERENCES external_services(id) ON DELETE CASCADE DEFERRABLE
    "external_service_repos_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE DEFERRABLE
Triggers:
    trig_soft_delete_orphan_repo_by_external_service_repo AFTER DELETE ON external_service_repos FOR EACH ROW EXECUTE PROCEDURE soft_delete_orphan_repo_by_external_service_repos()

```

# Table "public.external_service_sync_jobs"
```
       Column        |           Type           |                                Modifiers                                
---------------------+--------------------------+-------------------------------------------------------------------------
 id                  | integer                  | not null default nextval('external_service_sync_jobs_id_seq'::regclass)
 state               | text                     | not null default 'queued'::text
 failure_message     | text                     | 
 started_at          | timestamp with time zone | 
 finished_at         | timestamp with time zone | 
 process_after       | timestamp with time zone | 
 num_resets          | integer                  | not null default 0
 external_service_id | bigint                   | 
 num_failures        | integer                  | not null default 0
 log_contents        | text                     | 
Indexes:
    "external_service_sync_jobs_state_idx" btree (state)
Foreign-key constraints:
    "external_services_id_fk" FOREIGN KEY (external_service_id) REFERENCES external_services(id)

```

# Table "public.external_services"
```
      Column       |           Type           |                           Modifiers                            
-------------------+--------------------------+----------------------------------------------------------------
 id                | bigint                   | not null default nextval('external_services_id_seq'::regclass)
 kind              | text                     | not null
 display_name      | text                     | not null
 config            | text                     | not null
 created_at        | timestamp with time zone | not null default now()
 updated_at        | timestamp with time zone | not null default now()
 deleted_at        | timestamp with time zone | 
 last_sync_at      | timestamp with time zone | 
 next_sync_at      | timestamp with time zone | 
 namespace_user_id | integer                  | 
 unrestricted      | boolean                  | not null default false
Indexes:
    "external_services_pkey" PRIMARY KEY, btree (id)
    "external_services_namespace_user_id_idx" btree (namespace_user_id)
Check constraints:
    "check_non_empty_config" CHECK (btrim(config) <> ''::text)
Foreign-key constraints:
    "external_services_namepspace_user_id_fkey" FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE
Referenced by:
    TABLE "external_service_repos" CONSTRAINT "external_service_repos_external_service_id_fkey" FOREIGN KEY (external_service_id) REFERENCES external_services(id) ON DELETE CASCADE DEFERRABLE
    TABLE "external_service_sync_jobs" CONSTRAINT "external_services_id_fk" FOREIGN KEY (external_service_id) REFERENCES external_services(id)
Triggers:
    trig_delete_external_service_ref_on_external_service_repos AFTER UPDATE OF deleted_at ON external_services FOR EACH ROW EXECUTE PROCEDURE delete_external_service_ref_on_external_service_repos()

```

# Table "public.global_state"
```
         Column          |  Type   |         Modifiers         
-------------------------+---------+---------------------------
 site_id                 | uuid    | not null
 initialized             | boolean | not null default false
 mgmt_password_plaintext | text    | not null default ''::text
 mgmt_password_bcrypt    | text    | not null default ''::text
Indexes:
    "global_state_pkey" PRIMARY KEY, btree (site_id)

```

# Table "public.lsif_data_definitions"
```
   Column   |  Type   | Modifiers 
------------+---------+-----------
 dump_id    | integer | not null
 scheme     | text    | not null
 identifier | text    | not null
 data       | bytea   | 
Indexes:
    "lsif_data_definitions_pkey" PRIMARY KEY, btree (dump_id, scheme, identifier)

```

# Table "public.lsif_data_documents"
```
 Column  |  Type   | Modifiers 
---------+---------+-----------
 dump_id | integer | not null
 path    | text    | not null
 data    | bytea   | 
Indexes:
    "lsif_data_documents_pkey" PRIMARY KEY, btree (dump_id, path)

```

# Table "public.lsif_data_metadata"
```
      Column       |  Type   | Modifiers 
-------------------+---------+-----------
 dump_id           | integer | not null
 num_result_chunks | integer | 
Indexes:
    "lsif_data_metadata_pkey" PRIMARY KEY, btree (dump_id)

```

# Table "public.lsif_data_references"
```
   Column   |  Type   | Modifiers 
------------+---------+-----------
 dump_id    | integer | not null
 scheme     | text    | not null
 identifier | text    | not null
 data       | bytea   | 
Indexes:
    "lsif_data_references_pkey" PRIMARY KEY, btree (dump_id, scheme, identifier)

```

# Table "public.lsif_data_result_chunks"
```
 Column  |  Type   | Modifiers 
---------+---------+-----------
 dump_id | integer | not null
 idx     | integer | not null
 data    | bytea   | 
Indexes:
    "lsif_data_result_chunks_pkey" PRIMARY KEY, btree (dump_id, idx)

```

# Table "public.lsif_dirty_repositories"
```
    Column     |  Type   | Modifiers 
---------------+---------+-----------
 repository_id | integer | not null
 dirty_token   | integer | not null
 update_token  | integer | not null
Indexes:
    "lsif_dirty_repositories_pkey" PRIMARY KEY, btree (repository_id)

```

# Table "public.lsif_index_configuration"
```
    Column     |  Type   |                               Modifiers                               
---------------+---------+-----------------------------------------------------------------------
 id            | bigint  | not null default nextval('lsif_index_configuration_id_seq'::regclass)
 repository_id | integer | not null
 data          | bytea   | not null
Indexes:
    "lsif_index_configuration_pkey" PRIMARY KEY, btree (id)
    "lsif_index_configuration_repository_id_key" UNIQUE CONSTRAINT, btree (repository_id)
Foreign-key constraints:
    "lsif_index_configuration_repository_id_fkey" FOREIGN KEY (repository_id) REFERENCES repo(id) ON DELETE CASCADE

```

# Table "public.lsif_indexable_repositories"
```
         Column         |           Type           |                                Modifiers                                 
------------------------+--------------------------+--------------------------------------------------------------------------
 id                     | integer                  | not null default nextval('lsif_indexable_repositories_id_seq'::regclass)
 repository_id          | integer                  | not null
 search_count           | integer                  | not null default 0
 precise_count          | integer                  | not null default 0
 last_index_enqueued_at | timestamp with time zone | 
 last_updated_at        | timestamp with time zone | not null default now()
 enabled                | boolean                  | 
Indexes:
    "lsif_indexable_repositories_pkey" PRIMARY KEY, btree (id)
    "lsif_indexable_repositories_repository_id_key" UNIQUE CONSTRAINT, btree (repository_id)

```

# Table "public.lsif_indexes"
```
     Column      |           Type           |                         Modifiers                         
-----------------+--------------------------+-----------------------------------------------------------
 id              | bigint                   | not null default nextval('lsif_indexes_id_seq'::regclass)
 commit          | text                     | not null
 queued_at       | timestamp with time zone | not null default now()
 state           | lsif_index_state         | not null default 'queued'::lsif_index_state
 failure_message | text                     | 
 started_at      | timestamp with time zone | 
 finished_at     | timestamp with time zone | 
 repository_id   | integer                  | not null
 process_after   | timestamp with time zone | 
 num_resets      | integer                  | not null default 0
 num_failures    | integer                  | not null default 0
 docker_steps    | jsonb[]                  | not null
 root            | text                     | not null
 indexer         | text                     | not null
 indexer_args    | text[]                   | not null
 outfile         | text                     | not null
 log_contents    | text                     | 
Indexes:
    "lsif_indexes_pkey" PRIMARY KEY, btree (id)
Check constraints:
    "lsif_uploads_commit_valid_chars" CHECK (commit ~ '^[a-z0-9]{40}$'::text)

```

# Table "public.lsif_nearest_uploads"
```
      Column      |  Type   | Modifiers 
------------------+---------+-----------
 repository_id    | integer | not null
 upload_id        | integer | not null
 distance         | integer | not null
 ancestor_visible | boolean | not null
 overwritten      | boolean | not null
 commit_bytea     | bytea   | not null
Indexes:
    "lsif_nearest_uploads_repository_id_commit_bytea" btree (repository_id, commit_bytea)

```

# Table "public.lsif_packages"
```
 Column  |  Type   |                         Modifiers                          
---------+---------+------------------------------------------------------------
 id      | integer | not null default nextval('lsif_packages_id_seq'::regclass)
 scheme  | text    | not null
 name    | text    | not null
 version | text    | 
 dump_id | integer | not null
Indexes:
    "lsif_packages_pkey" PRIMARY KEY, btree (id)
    "lsif_packages_scheme_name_version" btree (scheme, name, version)
Foreign-key constraints:
    "lsif_packages_dump_id_fkey" FOREIGN KEY (dump_id) REFERENCES lsif_uploads(id) ON DELETE CASCADE

```

# Table "public.lsif_references"
```
 Column  |  Type   |                          Modifiers                           
---------+---------+--------------------------------------------------------------
 id      | integer | not null default nextval('lsif_references_id_seq'::regclass)
 scheme  | text    | not null
 name    | text    | not null
 version | text    | 
 filter  | bytea   | not null
 dump_id | integer | not null
Indexes:
    "lsif_references_pkey" PRIMARY KEY, btree (id)
    "lsif_references_package" btree (scheme, name, version)
Foreign-key constraints:
    "lsif_references_dump_id_fkey" FOREIGN KEY (dump_id) REFERENCES lsif_uploads(id) ON DELETE CASCADE

```

# Table "public.lsif_uploads"
```
     Column      |           Type           |                        Modifiers                        
-----------------+--------------------------+---------------------------------------------------------
 id              | integer                  | not null default nextval('lsif_dumps_id_seq'::regclass)
 commit          | text                     | not null
 root            | text                     | not null default ''::text
 uploaded_at     | timestamp with time zone | not null default now()
 state           | lsif_upload_state        | not null default 'queued'::lsif_upload_state
 failure_message | text                     | 
 started_at      | timestamp with time zone | 
 finished_at     | timestamp with time zone | 
 repository_id   | integer                  | not null
 indexer         | text                     | not null
 num_parts       | integer                  | not null
 uploaded_parts  | integer[]                | not null
 process_after   | timestamp with time zone | 
 num_resets      | integer                  | not null default 0
 upload_size     | bigint                   | 
 num_failures    | integer                  | not null default 0
Indexes:
    "lsif_uploads_pkey" PRIMARY KEY, btree (id)
    "lsif_uploads_repository_id_commit_root_indexer" UNIQUE, btree (repository_id, commit, root, indexer) WHERE state = 'completed'::lsif_upload_state
    "lsif_uploads_state" btree (state)
    "lsif_uploads_uploaded_at" btree (uploaded_at)
Check constraints:
    "lsif_uploads_commit_valid_chars" CHECK (commit ~ '^[a-z0-9]{40}$'::text)
Referenced by:
    TABLE "lsif_packages" CONSTRAINT "lsif_packages_dump_id_fkey" FOREIGN KEY (dump_id) REFERENCES lsif_uploads(id) ON DELETE CASCADE
    TABLE "lsif_references" CONSTRAINT "lsif_references_dump_id_fkey" FOREIGN KEY (dump_id) REFERENCES lsif_uploads(id) ON DELETE CASCADE

```

# Table "public.lsif_uploads_visible_at_tip"
```
    Column     |  Type   | Modifiers 
---------------+---------+-----------
 repository_id | integer | not null
 upload_id     | integer | not null
Indexes:
    "lsif_uploads_visible_at_tip_repository_id" btree (repository_id)

```

# Table "public.names"
```
 Column  |  Type   | Modifiers 
---------+---------+-----------
 name    | citext  | not null
 user_id | integer | 
 org_id  | integer | 
Indexes:
    "names_pkey" PRIMARY KEY, btree (name)
Check constraints:
    "names_check" CHECK (user_id IS NOT NULL OR org_id IS NOT NULL)
Foreign-key constraints:
    "names_org_id_fkey" FOREIGN KEY (org_id) REFERENCES orgs(id) ON UPDATE CASCADE ON DELETE CASCADE
    "names_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE

```

# Table "public.org_invitations"
```
      Column       |           Type           |                          Modifiers                           
-------------------+--------------------------+--------------------------------------------------------------
 id                | bigint                   | not null default nextval('org_invitations_id_seq'::regclass)
 org_id            | integer                  | not null
 sender_user_id    | integer                  | not null
 recipient_user_id | integer                  | not null
 created_at        | timestamp with time zone | not null default now()
 notified_at       | timestamp with time zone | 
 responded_at      | timestamp with time zone | 
 response_type     | boolean                  | 
 revoked_at        | timestamp with time zone | 
 deleted_at        | timestamp with time zone | 
Indexes:
    "org_invitations_pkey" PRIMARY KEY, btree (id)
    "org_invitations_singleflight" UNIQUE, btree (org_id, recipient_user_id) WHERE responded_at IS NULL AND revoked_at IS NULL AND deleted_at IS NULL
    "org_invitations_org_id" btree (org_id) WHERE deleted_at IS NULL
    "org_invitations_recipient_user_id" btree (recipient_user_id) WHERE deleted_at IS NULL
Check constraints:
    "check_atomic_response" CHECK ((responded_at IS NULL) = (response_type IS NULL))
    "check_single_use" CHECK (responded_at IS NULL AND response_type IS NULL OR revoked_at IS NULL)
Foreign-key constraints:
    "org_invitations_org_id_fkey" FOREIGN KEY (org_id) REFERENCES orgs(id)
    "org_invitations_recipient_user_id_fkey" FOREIGN KEY (recipient_user_id) REFERENCES users(id)
    "org_invitations_sender_user_id_fkey" FOREIGN KEY (sender_user_id) REFERENCES users(id)

```

# Table "public.org_members"
```
   Column   |           Type           |                        Modifiers                         
------------+--------------------------+----------------------------------------------------------
 id         | integer                  | not null default nextval('org_members_id_seq'::regclass)
 org_id     | integer                  | not null
 created_at | timestamp with time zone | not null default now()
 updated_at | timestamp with time zone | not null default now()
 user_id    | integer                  | not null
Indexes:
    "org_members_pkey" PRIMARY KEY, btree (id)
    "org_members_org_id_user_id_key" UNIQUE CONSTRAINT, btree (org_id, user_id)
Foreign-key constraints:
    "org_members_references_orgs" FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE RESTRICT
    "org_members_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT

```

# Table "public.org_members_bkup_1514536731"
```
   Column    |           Type           | Modifiers 
-------------+--------------------------+-----------
 id          | integer                  | 
 org_id      | integer                  | 
 user_id_old | text                     | 
 created_at  | timestamp with time zone | 
 updated_at  | timestamp with time zone | 
 user_id     | integer                  | 

```

# Table "public.orgs"
```
      Column       |           Type           |                     Modifiers                     
-------------------+--------------------------+---------------------------------------------------
 id                | integer                  | not null default nextval('orgs_id_seq'::regclass)
 name              | citext                   | not null
 created_at        | timestamp with time zone | not null default now()
 updated_at        | timestamp with time zone | not null default now()
 display_name      | text                     | 
 slack_webhook_url | text                     | 
 deleted_at        | timestamp with time zone | 
Indexes:
    "orgs_pkey" PRIMARY KEY, btree (id)
    "orgs_name" UNIQUE, btree (name) WHERE deleted_at IS NULL
Check constraints:
    "orgs_display_name_max_length" CHECK (char_length(display_name) <= 255)
    "orgs_name_max_length" CHECK (char_length(name::text) <= 255)
    "orgs_name_valid_chars" CHECK (name ~ '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[-.](?=[a-zA-Z0-9]))*-?$'::citext)
Referenced by:
    TABLE "campaigns" CONSTRAINT "campaigns_namespace_org_id_fkey" FOREIGN KEY (namespace_org_id) REFERENCES orgs(id) ON DELETE CASCADE DEFERRABLE
    TABLE "cm_monitors" CONSTRAINT "cm_monitors_org_id_fk" FOREIGN KEY (namespace_org_id) REFERENCES orgs(id) ON DELETE CASCADE
    TABLE "cm_recipients" CONSTRAINT "cm_recipients_org_id_fk" FOREIGN KEY (namespace_org_id) REFERENCES orgs(id) ON DELETE CASCADE
    TABLE "names" CONSTRAINT "names_org_id_fkey" FOREIGN KEY (org_id) REFERENCES orgs(id) ON UPDATE CASCADE ON DELETE CASCADE
    TABLE "org_invitations" CONSTRAINT "org_invitations_org_id_fkey" FOREIGN KEY (org_id) REFERENCES orgs(id)
    TABLE "org_members" CONSTRAINT "org_members_references_orgs" FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE RESTRICT
    TABLE "registry_extensions" CONSTRAINT "registry_extensions_publisher_org_id_fkey" FOREIGN KEY (publisher_org_id) REFERENCES orgs(id)
    TABLE "saved_searches" CONSTRAINT "saved_searches_org_id_fkey" FOREIGN KEY (org_id) REFERENCES orgs(id)
    TABLE "settings" CONSTRAINT "settings_references_orgs" FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE RESTRICT

```

# Table "public.phabricator_repos"
```
   Column   |           Type           |                           Modifiers                            
------------+--------------------------+----------------------------------------------------------------
 id         | integer                  | not null default nextval('phabricator_repos_id_seq'::regclass)
 callsign   | citext                   | not null
 repo_name  | citext                   | not null
 created_at | timestamp with time zone | not null default now()
 updated_at | timestamp with time zone | not null default now()
 deleted_at | timestamp with time zone | 
 url        | text                     | not null default ''::text
Indexes:
    "phabricator_repos_pkey" PRIMARY KEY, btree (id)
    "phabricator_repos_repo_name_key" UNIQUE CONSTRAINT, btree (repo_name)

```

# Table "public.product_licenses"
```
         Column          |           Type           |       Modifiers        
-------------------------+--------------------------+------------------------
 id                      | uuid                     | not null
 product_subscription_id | uuid                     | not null
 license_key             | text                     | not null
 created_at              | timestamp with time zone | not null default now()
Indexes:
    "product_licenses_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "product_licenses_product_subscription_id_fkey" FOREIGN KEY (product_subscription_id) REFERENCES product_subscriptions(id)

```

# Table "public.product_subscriptions"
```
         Column          |           Type           |       Modifiers        
-------------------------+--------------------------+------------------------
 id                      | uuid                     | not null
 user_id                 | integer                  | not null
 billing_subscription_id | text                     | 
 created_at              | timestamp with time zone | not null default now()
 updated_at              | timestamp with time zone | not null default now()
 archived_at             | timestamp with time zone | 
Indexes:
    "product_subscriptions_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "product_subscriptions_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
Referenced by:
    TABLE "product_licenses" CONSTRAINT "product_licenses_product_subscription_id_fkey" FOREIGN KEY (product_subscription_id) REFERENCES product_subscriptions(id)

```

# Table "public.query_runner_state"
```
      Column      |           Type           | Modifiers 
------------------+--------------------------+-----------
 query            | text                     | 
 last_executed    | timestamp with time zone | 
 latest_result    | timestamp with time zone | 
 exec_duration_ns | bigint                   | 

```

# Table "public.registry_extension_releases"
```
        Column         |           Type           |                                Modifiers                                 
-----------------------+--------------------------+--------------------------------------------------------------------------
 id                    | bigint                   | not null default nextval('registry_extension_releases_id_seq'::regclass)
 registry_extension_id | integer                  | not null
 creator_user_id       | integer                  | not null
 release_version       | citext                   | 
 release_tag           | citext                   | not null
 manifest              | jsonb                    | not null
 bundle                | text                     | 
 created_at            | timestamp with time zone | not null default now()
 deleted_at            | timestamp with time zone | 
 source_map            | text                     | 
Indexes:
    "registry_extension_releases_pkey" PRIMARY KEY, btree (id)
    "registry_extension_releases_version" UNIQUE, btree (registry_extension_id, release_version) WHERE release_version IS NOT NULL
    "registry_extension_releases_registry_extension_id" btree (registry_extension_id, release_tag, created_at DESC) WHERE deleted_at IS NULL
Foreign-key constraints:
    "registry_extension_releases_creator_user_id_fkey" FOREIGN KEY (creator_user_id) REFERENCES users(id)
    "registry_extension_releases_registry_extension_id_fkey" FOREIGN KEY (registry_extension_id) REFERENCES registry_extensions(id) ON UPDATE CASCADE ON DELETE CASCADE

```

# Table "public.registry_extensions"
```
      Column       |           Type           |                            Modifiers                             
-------------------+--------------------------+------------------------------------------------------------------
 id                | integer                  | not null default nextval('registry_extensions_id_seq'::regclass)
 uuid              | uuid                     | not null
 publisher_user_id | integer                  | 
 publisher_org_id  | integer                  | 
 name              | citext                   | not null
 manifest          | text                     | 
 created_at        | timestamp with time zone | not null default now()
 updated_at        | timestamp with time zone | not null default now()
 deleted_at        | timestamp with time zone | 
Indexes:
    "registry_extensions_pkey" PRIMARY KEY, btree (id)
    "registry_extensions_publisher_name" UNIQUE, btree ((COALESCE(publisher_user_id, 0)), (COALESCE(publisher_org_id, 0)), name) WHERE deleted_at IS NULL
    "registry_extensions_uuid" UNIQUE, btree (uuid)
Check constraints:
    "registry_extensions_name_length" CHECK (char_length(name::text) > 0 AND char_length(name::text) <= 128)
    "registry_extensions_name_valid_chars" CHECK (name ~ '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[_.-](?=[a-zA-Z0-9]))*$'::citext)
    "registry_extensions_single_publisher" CHECK ((publisher_user_id IS NULL) <> (publisher_org_id IS NULL))
Foreign-key constraints:
    "registry_extensions_publisher_org_id_fkey" FOREIGN KEY (publisher_org_id) REFERENCES orgs(id)
    "registry_extensions_publisher_user_id_fkey" FOREIGN KEY (publisher_user_id) REFERENCES users(id)
Referenced by:
    TABLE "registry_extension_releases" CONSTRAINT "registry_extension_releases_registry_extension_id_fkey" FOREIGN KEY (registry_extension_id) REFERENCES registry_extensions(id) ON UPDATE CASCADE ON DELETE CASCADE

```

# Table "public.repo"
```
        Column         |           Type           |                     Modifiers                     
-----------------------+--------------------------+---------------------------------------------------
 id                    | integer                  | not null default nextval('repo_id_seq'::regclass)
 name                  | citext                   | not null
 description           | text                     | 
 fork                  | boolean                  | 
 created_at            | timestamp with time zone | not null default now()
 updated_at            | timestamp with time zone | 
 external_id           | text                     | 
 external_service_type | text                     | 
 external_service_id   | text                     | 
 archived              | boolean                  | not null default false
 uri                   | citext                   | 
 deleted_at            | timestamp with time zone | 
 metadata              | jsonb                    | not null default '{}'::jsonb
 private               | boolean                  | not null default false
 cloned                | boolean                  | not null default false
Indexes:
    "repo_pkey" PRIMARY KEY, btree (id)
    "repo_external_unique_idx" UNIQUE, btree (external_service_type, external_service_id, external_id)
    "repo_name_unique" UNIQUE CONSTRAINT, btree (name) DEFERRABLE
    "repo_archived" btree (archived)
    "repo_cloned" btree (cloned)
    "repo_created_at" btree (created_at)
    "repo_fork" btree (fork)
    "repo_metadata_gin_idx" gin (metadata)
    "repo_name_idx" btree (lower(name::text) COLLATE "C")
    "repo_name_trgm" gin (lower(name::text) gin_trgm_ops)
    "repo_private" btree (private)
    "repo_uri_idx" btree (uri)
Check constraints:
    "check_name_nonempty" CHECK (name <> ''::citext)
    "repo_metadata_check" CHECK (jsonb_typeof(metadata) = 'object'::text)
Referenced by:
    TABLE "changeset_specs" CONSTRAINT "changeset_specs_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) DEFERRABLE
    TABLE "changesets" CONSTRAINT "changesets_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE DEFERRABLE
    TABLE "default_repos" CONSTRAINT "default_repos_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE
    TABLE "discussion_threads_target_repo" CONSTRAINT "discussion_threads_target_repo_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE
    TABLE "external_service_repos" CONSTRAINT "external_service_repos_repo_id_fkey" FOREIGN KEY (repo_id) REFERENCES repo(id) ON DELETE CASCADE DEFERRABLE
    TABLE "lsif_index_configuration" CONSTRAINT "lsif_index_configuration_repository_id_fkey" FOREIGN KEY (repository_id) REFERENCES repo(id) ON DELETE CASCADE
Triggers:
    trig_delete_repo_ref_on_external_service_repos AFTER UPDATE OF deleted_at ON repo FOR EACH ROW EXECUTE PROCEDURE delete_repo_ref_on_external_service_repos()

```

# Table "public.repo_pending_permissions"
```
    Column     |           Type           |            Modifiers             
---------------+--------------------------+----------------------------------
 repo_id       | integer                  | not null
 permission    | text                     | not null
 user_ids      | bytea                    | not null default '\x'::bytea
 updated_at    | timestamp with time zone | not null
 user_ids_ints | integer[]                | not null default '{}'::integer[]
Indexes:
    "repo_pending_permissions_perm_unique" UNIQUE CONSTRAINT, btree (repo_id, permission)

```

# Table "public.repo_permissions"
```
    Column     |           Type           |            Modifiers             
---------------+--------------------------+----------------------------------
 repo_id       | integer                  | not null
 permission    | text                     | not null
 user_ids      | bytea                    | not null default '\x'::bytea
 updated_at    | timestamp with time zone | not null
 synced_at     | timestamp with time zone | 
 user_ids_ints | integer[]                | not null default '{}'::integer[]
Indexes:
    "repo_permissions_perm_unique" UNIQUE CONSTRAINT, btree (repo_id, permission)

```

# Table "public.saved_searches"
```
      Column       |           Type           |                          Modifiers                          
-------------------+--------------------------+-------------------------------------------------------------
 id                | integer                  | not null default nextval('saved_searches_id_seq'::regclass)
 description       | text                     | not null
 query             | text                     | not null
 created_at        | timestamp with time zone | not null default now()
 updated_at        | timestamp with time zone | not null default now()
 notify_owner      | boolean                  | not null
 notify_slack      | boolean                  | not null
 user_id           | integer                  | 
 org_id            | integer                  | 
 slack_webhook_url | text                     | 
Indexes:
    "saved_searches_pkey" PRIMARY KEY, btree (id)
Check constraints:
    "user_or_org_id_not_null" CHECK (user_id IS NOT NULL AND org_id IS NULL OR org_id IS NOT NULL AND user_id IS NULL)
Foreign-key constraints:
    "saved_searches_org_id_fkey" FOREIGN KEY (org_id) REFERENCES orgs(id)
    "saved_searches_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)

```

# Table "public.schema_migrations"
```
 Column  |  Type   | Modifiers 
---------+---------+-----------
 version | bigint  | not null
 dirty   | boolean | not null
Indexes:
    "schema_migrations_pkey" PRIMARY KEY, btree (version)

```

# Table "public.secrets"
```
   Column    |          Type          |                      Modifiers                       
-------------+------------------------+------------------------------------------------------
 id          | bigint                 | not null default nextval('secrets_id_seq'::regclass)
 source_type | character varying(50)  | 
 source_id   | bigint                 | 
 key_name    | character varying(100) | 
 value       | text                   | not null
Indexes:
    "secrets_pkey" PRIMARY KEY, btree (id)
    "secret_key_idx" UNIQUE, btree (key_name)
    "secret_sourcetype_idx" UNIQUE, btree (source_type, source_id)

```

# Table "public.settings"
```
     Column     |           Type           |                       Modifiers                       
----------------+--------------------------+-------------------------------------------------------
 id             | integer                  | not null default nextval('settings_id_seq'::regclass)
 org_id         | integer                  | 
 contents       | text                     | 
 created_at     | timestamp with time zone | not null default now()
 user_id        | integer                  | 
 author_user_id | integer                  | 
Indexes:
    "settings_pkey" PRIMARY KEY, btree (id)
    "settings_org_id_idx" btree (org_id)
Foreign-key constraints:
    "settings_author_user_id_fkey" FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT
    "settings_references_orgs" FOREIGN KEY (org_id) REFERENCES orgs(id) ON DELETE RESTRICT
    "settings_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT

```

# Table "public.settings_bkup_1514702776"
```
       Column       |           Type           | Modifiers 
--------------------+--------------------------+-----------
 id                 | integer                  | 
 org_id             | integer                  | 
 author_user_id_old | text                     | 
 contents           | text                     | 
 created_at         | timestamp with time zone | 
 user_id            | integer                  | 
 author_user_id     | integer                  | 

```

# Table "public.survey_responses"
```
   Column   |           Type           |                           Modifiers                           
------------+--------------------------+---------------------------------------------------------------
 id         | bigint                   | not null default nextval('survey_responses_id_seq'::regclass)
 user_id    | integer                  | 
 email      | text                     | 
 score      | integer                  | not null
 reason     | text                     | 
 better     | text                     | 
 created_at | timestamp with time zone | not null default now()
Indexes:
    "survey_responses_pkey" PRIMARY KEY, btree (id)
Foreign-key constraints:
    "survey_responses_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)

```

# Table "public.user_credentials"
```
        Column         |           Type           |                           Modifiers                           
-----------------------+--------------------------+---------------------------------------------------------------
 id                    | bigint                   | not null default nextval('user_credentials_id_seq'::regclass)
 domain                | text                     | not null
 user_id               | integer                  | not null
 external_service_type | text                     | not null
 external_service_id   | text                     | not null
 credential            | text                     | not null
 created_at            | timestamp with time zone | not null default now()
 updated_at            | timestamp with time zone | not null default now()
Indexes:
    "user_credentials_pkey" PRIMARY KEY, btree (id)
    "user_credentials_domain_user_id_external_service_type_exter_key" UNIQUE CONSTRAINT, btree (domain, user_id, external_service_type, external_service_id)
Foreign-key constraints:
    "user_credentials_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE

```

# Table "public.user_emails"
```
          Column           |           Type           |       Modifiers        
---------------------------+--------------------------+------------------------
 user_id                   | integer                  | not null
 email                     | citext                   | not null
 created_at                | timestamp with time zone | not null default now()
 verification_code         | text                     | 
 verified_at               | timestamp with time zone | 
 last_verification_sent_at | timestamp with time zone | 
 is_primary                | boolean                  | not null default false
Indexes:
    "user_emails_no_duplicates_per_user" UNIQUE CONSTRAINT, btree (user_id, email)
    "user_emails_user_id_is_primary_idx" UNIQUE, btree (user_id, is_primary) WHERE is_primary = true
    "user_emails_unique_verified_email" EXCLUDE USING btree (email WITH =) WHERE (verified_at IS NOT NULL)
Foreign-key constraints:
    "user_emails_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)

```

# Table "public.user_external_accounts"
```
    Column     |           Type           |                              Modifiers                              
---------------+--------------------------+---------------------------------------------------------------------
 id            | integer                  | not null default nextval('user_external_accounts_id_seq'::regclass)
 user_id       | integer                  | not null
 service_type  | text                     | not null
 service_id    | text                     | not null
 account_id    | text                     | not null
 auth_data     | text                     | 
 account_data  | text                     | 
 created_at    | timestamp with time zone | not null default now()
 updated_at    | timestamp with time zone | not null default now()
 deleted_at    | timestamp with time zone | 
 client_id     | text                     | not null
 expired_at    | timestamp with time zone | 
 last_valid_at | timestamp with time zone | 
Indexes:
    "user_external_accounts_pkey" PRIMARY KEY, btree (id)
    "user_external_accounts_account" UNIQUE, btree (service_type, service_id, client_id, account_id) WHERE deleted_at IS NULL
    "user_external_accounts_user_id" btree (user_id) WHERE deleted_at IS NULL
Foreign-key constraints:
    "user_external_accounts_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)

```

# Table "public.user_pending_permissions"
```
     Column      |           Type           |                               Modifiers                               
-----------------+--------------------------+-----------------------------------------------------------------------
 id              | integer                  | not null default nextval('user_pending_permissions_id_seq'::regclass)
 bind_id         | text                     | not null
 permission      | text                     | not null
 object_type     | text                     | not null
 object_ids      | bytea                    | not null default '\x'::bytea
 updated_at      | timestamp with time zone | not null
 service_type    | text                     | not null
 service_id      | text                     | not null
 object_ids_ints | integer[]                | not null default '{}'::integer[]
Indexes:
    "user_pending_permissions_service_perm_object_unique" UNIQUE CONSTRAINT, btree (service_type, service_id, permission, object_type, bind_id)

```

# Table "public.user_permissions"
```
     Column      |           Type           |            Modifiers             
-----------------+--------------------------+----------------------------------
 user_id         | integer                  | not null
 permission      | text                     | not null
 object_type     | text                     | not null
 object_ids      | bytea                    | not null default '\x'::bytea
 updated_at      | timestamp with time zone | not null
 synced_at       | timestamp with time zone | 
 object_ids_ints | integer[]                | not null default '{}'::integer[]
Indexes:
    "user_permissions_perm_object_unique" UNIQUE CONSTRAINT, btree (user_id, permission, object_type)

```

# Table "public.users"
```
         Column          |           Type           |                     Modifiers                      
-------------------------+--------------------------+----------------------------------------------------
 id                      | integer                  | not null default nextval('users_id_seq'::regclass)
 username                | citext                   | not null
 display_name            | text                     | 
 avatar_url              | text                     | 
 created_at              | timestamp with time zone | not null default now()
 updated_at              | timestamp with time zone | not null default now()
 deleted_at              | timestamp with time zone | 
 invite_quota            | integer                  | not null default 15
 passwd                  | text                     | 
 passwd_reset_code       | text                     | 
 passwd_reset_time       | timestamp with time zone | 
 site_admin              | boolean                  | not null default false
 page_views              | integer                  | not null default 0
 search_queries          | integer                  | not null default 0
 tags                    | text[]                   | default '{}'::text[]
 billing_customer_id     | text                     | 
 invalidated_sessions_at | timestamp with time zone | not null default now()
Indexes:
    "users_pkey" PRIMARY KEY, btree (id)
    "users_billing_customer_id" UNIQUE, btree (billing_customer_id) WHERE deleted_at IS NULL
    "users_username" UNIQUE, btree (username) WHERE deleted_at IS NULL
    "users_created_at_idx" btree (created_at)
Check constraints:
    "users_display_name_max_length" CHECK (char_length(display_name) <= 255)
    "users_username_max_length" CHECK (char_length(username::text) <= 255)
    "users_username_valid_chars" CHECK (username ~ '^[a-zA-Z0-9](?:[a-zA-Z0-9]|[-.](?=[a-zA-Z0-9]))*-?$'::citext)
Referenced by:
    TABLE "access_tokens" CONSTRAINT "access_tokens_creator_user_id_fkey" FOREIGN KEY (creator_user_id) REFERENCES users(id)
    TABLE "access_tokens" CONSTRAINT "access_tokens_subject_user_id_fkey" FOREIGN KEY (subject_user_id) REFERENCES users(id)
    TABLE "campaign_specs" CONSTRAINT "campaign_specs_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL DEFERRABLE
    TABLE "campaigns" CONSTRAINT "campaigns_initial_applier_id_fkey" FOREIGN KEY (initial_applier_id) REFERENCES users(id) ON DELETE SET NULL DEFERRABLE
    TABLE "campaigns" CONSTRAINT "campaigns_last_applier_id_fkey" FOREIGN KEY (last_applier_id) REFERENCES users(id) ON DELETE SET NULL DEFERRABLE
    TABLE "campaigns" CONSTRAINT "campaigns_namespace_user_id_fkey" FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE
    TABLE "changeset_specs" CONSTRAINT "changeset_specs_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL DEFERRABLE
    TABLE "cm_emails" CONSTRAINT "cm_emails_changed_by_fk" FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE CASCADE
    TABLE "cm_emails" CONSTRAINT "cm_emails_created_by_fk" FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
    TABLE "cm_monitors" CONSTRAINT "cm_monitors_changed_by_fk" FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE CASCADE
    TABLE "cm_monitors" CONSTRAINT "cm_monitors_created_by_fk" FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
    TABLE "cm_monitors" CONSTRAINT "cm_monitors_user_id_fk" FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE
    TABLE "cm_recipients" CONSTRAINT "cm_recipients_user_id_fk" FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE
    TABLE "cm_queries" CONSTRAINT "cm_triggers_changed_by_fk" FOREIGN KEY (changed_by) REFERENCES users(id) ON DELETE CASCADE
    TABLE "cm_queries" CONSTRAINT "cm_triggers_created_by_fk" FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE CASCADE
    TABLE "discussion_comments" CONSTRAINT "discussion_comments_author_user_id_fkey" FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "discussion_mail_reply_tokens" CONSTRAINT "discussion_mail_reply_tokens_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "discussion_threads" CONSTRAINT "discussion_threads_author_user_id_fkey" FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "external_services" CONSTRAINT "external_services_namepspace_user_id_fkey" FOREIGN KEY (namespace_user_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE
    TABLE "names" CONSTRAINT "names_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON UPDATE CASCADE ON DELETE CASCADE
    TABLE "org_invitations" CONSTRAINT "org_invitations_recipient_user_id_fkey" FOREIGN KEY (recipient_user_id) REFERENCES users(id)
    TABLE "org_invitations" CONSTRAINT "org_invitations_sender_user_id_fkey" FOREIGN KEY (sender_user_id) REFERENCES users(id)
    TABLE "org_members" CONSTRAINT "org_members_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "product_subscriptions" CONSTRAINT "product_subscriptions_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
    TABLE "registry_extension_releases" CONSTRAINT "registry_extension_releases_creator_user_id_fkey" FOREIGN KEY (creator_user_id) REFERENCES users(id)
    TABLE "registry_extensions" CONSTRAINT "registry_extensions_publisher_user_id_fkey" FOREIGN KEY (publisher_user_id) REFERENCES users(id)
    TABLE "saved_searches" CONSTRAINT "saved_searches_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
    TABLE "settings" CONSTRAINT "settings_author_user_id_fkey" FOREIGN KEY (author_user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "settings" CONSTRAINT "settings_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE RESTRICT
    TABLE "survey_responses" CONSTRAINT "survey_responses_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
    TABLE "user_credentials" CONSTRAINT "user_credentials_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE DEFERRABLE
    TABLE "user_emails" CONSTRAINT "user_emails_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
    TABLE "user_external_accounts" CONSTRAINT "user_external_accounts_user_id_fkey" FOREIGN KEY (user_id) REFERENCES users(id)
Triggers:
    trig_invalidate_session_on_password_change BEFORE UPDATE OF passwd ON users FOR EACH ROW EXECUTE PROCEDURE invalidate_session_for_userid_on_password_change()
    trig_soft_delete_user_reference_on_external_service AFTER UPDATE OF deleted_at ON users FOR EACH ROW EXECUTE PROCEDURE soft_delete_user_reference_on_external_service()

```

# Table "public.versions"
```
   Column   |           Type           |       Modifiers        
------------+--------------------------+------------------------
 service    | text                     | not null
 version    | text                     | not null
 updated_at | timestamp with time zone | not null default now()
Indexes:
    "versions_pkey" PRIMARY KEY, btree (service)

```
