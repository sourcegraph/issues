# Table "public.codeintel_schema_migrations"
```
 Column  |  Type   | Collation | Nullable | Default 
---------+---------+-----------+----------+---------
 version | bigint  |           | not null | 
 dirty   | boolean |           | not null | 
Indexes:
    "codeintel_schema_migrations_pkey" PRIMARY KEY, btree (version)

```

# Table "public.lsif_data_definitions"
```
     Column     |  Type   | Collation | Nullable | Default 
----------------+---------+-----------+----------+---------
 dump_id        | integer |           | not null | 
 scheme         | text    |           | not null | 
 identifier     | text    |           | not null | 
 data           | bytea   |           |          | 
 schema_version | integer |           | not null | 
 num_locations  | integer |           | not null | 
Indexes:
    "lsif_data_definitions_pkey" PRIMARY KEY, btree (dump_id, scheme, identifier)
    "lsif_data_definitions_dump_id_schema_version" btree (dump_id, schema_version)
Triggers:
    lsif_data_definitions_schema_versions_insert AFTER INSERT ON lsif_data_definitions REFERENCING NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION update_lsif_data_definitions_schema_versions_insert()

```

Associates (document, range) pairs with the import monikers attached to the range.

**data**: A gob-encoded payload conforming to an array of [LocationData](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L106:6) types.

**dump_id**: The identifier of the associated dump in the lsif_uploads table (state=completed).

**identifier**: The moniker identifier.

**num_locations**: The number of locations stored in the data field.

**schema_version**: The schema version of this row - used to determine presence and encoding of data.

**scheme**: The moniker scheme.

# Table "public.lsif_data_definitions_schema_versions"
```
       Column       |  Type   | Collation | Nullable | Default 
--------------------+---------+-----------+----------+---------
 dump_id            | integer |           | not null | 
 min_schema_version | integer |           |          | 
 max_schema_version | integer |           |          | 
Indexes:
    "lsif_data_definitions_schema_versions_pkey" PRIMARY KEY, btree (dump_id)
    "lsif_data_definitions_schema_versions_dump_id_schema_version_bo" btree (dump_id, min_schema_version, max_schema_version)

```

Tracks the range of schema_versions for each upload in the lsif_data_definitions table.

**dump_id**: The identifier of the associated dump in the lsif_uploads table.

**max_schema_version**: An upper-bound on the `lsif_data_definitions.schema_version` where `lsif_data_definitions.dump_id = dump_id`.

**min_schema_version**: A lower-bound on the `lsif_data_definitions.schema_version` where `lsif_data_definitions.dump_id = dump_id`.

# Table "public.lsif_data_documentation_mappings"
```
  Column   |  Type   | Collation | Nullable | Default 
-----------+---------+-----------+----------+---------
 dump_id   | integer |           | not null | 
 path_id   | text    |           | not null | 
 result_id | integer |           | not null | 
Indexes:
    "lsif_data_documentation_mappings_pkey" PRIMARY KEY, btree (dump_id, path_id)
    "lsif_data_documentation_mappings_inverse_unique_idx" UNIQUE, btree (dump_id, result_id)

```

Maps documentation path IDs to their corresponding integral documentationResult vertex IDs, which are unique within a dump.

**dump_id**: The identifier of the associated dump in the lsif_uploads table (state=completed).

**path_id**: The documentation page path ID, see see GraphQL codeintel.schema:documentationPage for what this is.

**result_id**: The documentationResult vertex ID.

# Table "public.lsif_data_documentation_pages"
```
 Column  |  Type   | Collation | Nullable | Default 
---------+---------+-----------+----------+---------
 dump_id | integer |           | not null | 
 path_id | text    |           | not null | 
 data    | bytea   |           |          | 
Indexes:
    "lsif_data_documentation_pages_pkey" PRIMARY KEY, btree (dump_id, path_id)

```

Associates documentation pathIDs to their documentation page hierarchy chunk.

**data**: A gob-encoded payload conforming to a `type DocumentationPageData struct` pointer (lib/codeintel/semantic/types.go)

**dump_id**: The identifier of the associated dump in the lsif_uploads table (state=completed).

**path_id**: The documentation page path ID, see see GraphQL codeintel.schema:documentationPage for what this is.

# Table "public.lsif_data_documentation_path_info"
```
 Column  |  Type   | Collation | Nullable | Default 
---------+---------+-----------+----------+---------
 dump_id | integer |           | not null | 
 path_id | text    |           | not null | 
 data    | bytea   |           |          | 
Indexes:
    "lsif_data_documentation_path_info_pkey" PRIMARY KEY, btree (dump_id, path_id)

```

Associates documentation page pathIDs to information about what is at that pathID, its immediate children, etc.

**data**: A gob-encoded payload conforming to a `type DocumentationPathInoData struct` pointer (lib/codeintel/semantic/types.go)

**dump_id**: The identifier of the associated dump in the lsif_uploads table (state=completed).

**path_id**: The documentation page path ID, see see GraphQL codeintel.schema:documentationPage for what this is.

# Table "public.lsif_data_documents"
```
     Column      |  Type   | Collation | Nullable | Default 
-----------------+---------+-----------+----------+---------
 dump_id         | integer |           | not null | 
 path            | text    |           | not null | 
 data            | bytea   |           |          | 
 schema_version  | integer |           | not null | 
 num_diagnostics | integer |           | not null | 
 ranges          | bytea   |           |          | 
 hovers          | bytea   |           |          | 
 monikers        | bytea   |           |          | 
 packages        | bytea   |           |          | 
 diagnostics     | bytea   |           |          | 
Indexes:
    "lsif_data_documents_pkey" PRIMARY KEY, btree (dump_id, path)
    "lsif_data_documents_dump_id_schema_version" btree (dump_id, schema_version)
Triggers:
    lsif_data_documents_schema_versions_insert AFTER INSERT ON lsif_data_documents REFERENCING NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION update_lsif_data_documents_schema_versions_insert()

```

Stores reference, hover text, moniker, and diagnostic data about a particular text document witin a dump.

**data**: A gob-encoded payload conforming to the [DocumentData](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L13:6) type. This field is being migrated across ranges, hovers, monikers, packages, and diagnostics columns and will be removed in a future release of Sourcegraph.

**diagnostics**: A gob-encoded payload conforming to the [Diagnostics](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L18:2) field of the DocumentDatatype.

**dump_id**: The identifier of the associated dump in the lsif_uploads table (state=completed).

**hovers**: A gob-encoded payload conforming to the [HoversResults](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L15:2) field of the DocumentDatatype.

**monikers**: A gob-encoded payload conforming to the [Monikers](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L16:2) field of the DocumentDatatype.

**num_diagnostics**: The number of diagnostics stored in the data field.

**packages**: A gob-encoded payload conforming to the [PackageInformation](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L17:2) field of the DocumentDatatype.

**path**: The path of the text document relative to the associated dump root.

**ranges**: A gob-encoded payload conforming to the [Ranges](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L14:2) field of the DocumentDatatype.

**schema_version**: The schema version of this row - used to determine presence and encoding of data.

# Table "public.lsif_data_documents_schema_versions"
```
       Column       |  Type   | Collation | Nullable | Default 
--------------------+---------+-----------+----------+---------
 dump_id            | integer |           | not null | 
 min_schema_version | integer |           |          | 
 max_schema_version | integer |           |          | 
Indexes:
    "lsif_data_documents_schema_versions_pkey" PRIMARY KEY, btree (dump_id)
    "lsif_data_documents_schema_versions_dump_id_schema_version_boun" btree (dump_id, min_schema_version, max_schema_version)

```

Tracks the range of schema_versions for each upload in the lsif_data_documents table.

**dump_id**: The identifier of the associated dump in the lsif_uploads table.

**max_schema_version**: An upper-bound on the `lsif_data_documents.schema_version` where `lsif_data_documents.dump_id = dump_id`.

**min_schema_version**: A lower-bound on the `lsif_data_documents.schema_version` where `lsif_data_documents.dump_id = dump_id`.

# Table "public.lsif_data_metadata"
```
      Column       |  Type   | Collation | Nullable | Default 
-------------------+---------+-----------+----------+---------
 dump_id           | integer |           | not null | 
 num_result_chunks | integer |           |          | 
Indexes:
    "lsif_data_metadata_pkey" PRIMARY KEY, btree (dump_id)

```

Stores the number of result chunks associated with a dump.

**dump_id**: The identifier of the associated dump in the lsif_uploads table (state=completed).

**num_result_chunks**: A bound of populated indexes in the lsif_data_result_chunks table for the associated dump. This value is used to hash identifiers into the result chunk index to which they belong.

# Table "public.lsif_data_references"
```
     Column     |  Type   | Collation | Nullable | Default 
----------------+---------+-----------+----------+---------
 dump_id        | integer |           | not null | 
 scheme         | text    |           | not null | 
 identifier     | text    |           | not null | 
 data           | bytea   |           |          | 
 schema_version | integer |           | not null | 
 num_locations  | integer |           | not null | 
Indexes:
    "lsif_data_references_pkey" PRIMARY KEY, btree (dump_id, scheme, identifier)
    "lsif_data_references_dump_id_schema_version" btree (dump_id, schema_version)
Triggers:
    lsif_data_references_schema_versions_insert AFTER INSERT ON lsif_data_references REFERENCING NEW TABLE AS newtab FOR EACH STATEMENT EXECUTE FUNCTION update_lsif_data_references_schema_versions_insert()

```

Associates (document, range) pairs with the export monikers attached to the range.

**data**: A gob-encoded payload conforming to an array of [LocationData](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L106:6) types.

**dump_id**: The identifier of the associated dump in the lsif_uploads table (state=completed).

**identifier**: The moniker identifier.

**num_locations**: The number of locations stored in the data field.

**schema_version**: The schema version of this row - used to determine presence and encoding of data.

**scheme**: The moniker scheme.

# Table "public.lsif_data_references_schema_versions"
```
       Column       |  Type   | Collation | Nullable | Default 
--------------------+---------+-----------+----------+---------
 dump_id            | integer |           | not null | 
 min_schema_version | integer |           |          | 
 max_schema_version | integer |           |          | 
Indexes:
    "lsif_data_references_schema_versions_pkey" PRIMARY KEY, btree (dump_id)
    "lsif_data_references_schema_versions_dump_id_schema_version_bou" btree (dump_id, min_schema_version, max_schema_version)

```

Tracks the range of schema_versions for each upload in the lsif_data_references table.

**dump_id**: The identifier of the associated dump in the lsif_uploads table.

**max_schema_version**: An upper-bound on the `lsif_data_references.schema_version` where `lsif_data_references.dump_id = dump_id`.

**min_schema_version**: A lower-bound on the `lsif_data_references.schema_version` where `lsif_data_references.dump_id = dump_id`.

# Table "public.lsif_data_result_chunks"
```
 Column  |  Type   | Collation | Nullable | Default 
---------+---------+-----------+----------+---------
 dump_id | integer |           | not null | 
 idx     | integer |           | not null | 
 data    | bytea   |           |          | 
Indexes:
    "lsif_data_result_chunks_pkey" PRIMARY KEY, btree (dump_id, idx)

```

Associates result set identifiers with the (document path, range identifier) pairs that compose the set.

**data**: A gob-encoded payload conforming to the [ResultChunkData](https://sourcegraph.com/github.com/sourcegraph/sourcegraph@3.26/-/blob/enterprise/lib/codeintel/semantic/types.go#L76:6) type.

**dump_id**: The identifier of the associated dump in the lsif_uploads table (state=completed).

**idx**: The unique result chunk index within the associated dump. Every result set identifier present should hash to this index (modulo lsif_data_metadata.num_result_chunks).
