BEGIN;

CREATE INDEX lsif_packages_scheme_name_version_dump_id ON lsif_packages(scheme, name, version, dump_id);
DROP INDEX lsif_packages_scheme_name_version;

CREATE INDEX lsif_references_scheme_name_version_dump_id ON lsif_references(scheme, name, version, dump_id);
DROP INDEX lsif_references_package;

COMMIT;
