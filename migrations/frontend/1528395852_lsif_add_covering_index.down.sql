BEGIN;

CREATE INDEX lsif_packages_scheme_name_version ON lsif_packages(scheme, name, version);
DROP INDEX lsif_packages_scheme_name_version_dump_id;

CREATE INDEX lsif_references_package ON lsif_references(scheme, name, version);
DROP INDEX lsif_references_scheme_name_version_dump_id;

COMMIT;
