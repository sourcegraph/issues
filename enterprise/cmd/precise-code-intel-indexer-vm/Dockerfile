FROM scratch

# NOTE: We use this container only for distribution of this binary, not for a containerized
# way to _run_ the application. This binary is extracted from the container during the init
# process of the compute node that runs the indexer, as it need direct access to the host's
# ignite binary and firecracker state.
#
# See https://github.com/sourcegraph/infrastructure/blob/master/code-intel/startup-script.sh.
COPY precise-code-intel-indexer-vm /usr/local/bin/

# Required for docker create to work correctly. Without this docker create responds with
# the folowing error:
#
# Error response from daemon: No command specified
CMD ["/usr/local/bin/precise-code-intel-indexer-vm"]
