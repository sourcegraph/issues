# Install Sourcegraph with Docker Compose on AWS

This tutorial shows you how to deploy Sourcegraph via [Docker Compose](https://docs.docker.com/compose/) to a single EC2 instance on AWS.

> NOTE: Trying to decide how to deploy Sourcegraph? See [our recommendations](../index.md) for how to chose a deployment type that suits your needs.

---

## (optional, recommended) Create a fork for customizations

We **strongly** recommend that you create your own fork of [sourcegraph/deploy-sourcegraph-docker](https://github.com/sourcegraph/deploy-sourcegraph-docker/) to track customizations to the [Sourcegraph Docker Compose yaml](https://github.com/sourcegraph/deploy-sourcegraph-docker/blob/master/docker-compose/docker-compose.yaml). This will make upgrades far easier.

See ["Store customizations in a fork"](./index.md#optional-recommended-store-customizations-in-a-fork) for full instructions.

## Deploy to EC2

* Click **Launch Instance** from your [EC2 dashboard](https://console.aws.amazon.com/ec2/v2/home).
* Select the **Amazon Linux 2 AMI (HVM), SSD Volume Type**.
* Select an appropriate instance size (use the [resource estimator](../resource_estimator.md) to find a good starting point for your deployment), then **Next: Configure Instance Details.**
* Ensure the **Auto-assign Public IP** option is "Enable". This ensures your instance is accessible to the Internet.
* Add the following user data (as text) in the **Advanced Details** section:
  * (optional) If you [created a fork as recommended above](#optional-recommended-create-a-fork-for-customizations), update the following environment variables in the script below:
    * `DEPLOY_SOURCEGRAPH_DOCKER_FORK_CLONE_URL`: Your fork's git clone URL
    * `DEPLOY_SOURCEGRAPH_DOCKER_FORK_REVISION`: The git revision containing your fork's customizations to the base Sourcegraph Docker Compose yaml. Most likely, `DEPLOY_SOURCEGRAPH_DOCKER_FORK_REVISION='release'` if you followed our branching recommendations in ["Store customizations in a fork"](./index.md#optional-recommended-store-customizations-in-a-fork).

```bash
#!/usr/bin/env bash

set -euxo pipefail

EBS_VOLUME_DEVICE_NAME='/dev/sdb'
DOCKER_DATA_ROOT='/mnt/docker-data'

DOCKER_COMPOSE_VERSION='1.25.3'
DEPLOY_SOURCEGRAPH_DOCKER_CHECKOUT='/home/ec2-user/deploy-sourcegraph-docker'

# 🚨 Update these variables with the correct values from your fork!
DEPLOY_SOURCEGRAPH_DOCKER_FORK_CLONE_URL='https://github.com/sourcegraph/deploy-sourcegraph-docker.git'
DEPLOY_SOURCEGRAPH_DOCKER_FORK_REVISION='v3.17.2'

# Install git
yum update -y
yum install git -y

# Clone Docker Compose definition
git clone "${DEPLOY_SOURCEGRAPH_DOCKER_FORK_CLONE_URL}" "${DEPLOY_SOURCEGRAPH_DOCKER_CHECKOUT}"
cd "${DEPLOY_SOURCEGRAPH_DOCKER_CHECKOUT}"
git checkout "${DEPLOY_SOURCEGRAPH_DOCKER_FORK_REVISION}"

# Format (if necessary) and mount EBS volume
device_fs=$(lsblk "${EBS_VOLUME_DEVICE_NAME}" --noheadings --output fsType)
if [ "${device_fs}" == "" ] ## only format the volume if it isn't already formatted
then
  mkfs -t xfs "${EBS_VOLUME_DEVICE_NAME}"
fi
mkdir -p "${DOCKER_DATA_ROOT}"
mount "${EBS_VOLUME_DEVICE_NAME}" "${DOCKER_DATA_ROOT}"

# Mount EBS volume on reboots
EBS_UUID=$(blkid -s UUID -o value "${EBS_VOLUME_DEVICE_NAME}")
echo "UUID=${EBS_UUID}  ${DOCKER_DATA_ROOT}  xfs  defaults,nofail  0  2" >> '/etc/fstab'
umount "${DOCKER_DATA_ROOT}"
mount -a

# Install, configure, and enable Docker
yum update -y
amazon-linux-extras install docker
systemctl enable --now docker
sed -i -e 's/1024/262144/g' /etc/sysconfig/docker
sed -i -e 's/4096/262144/g' /etc/sysconfig/docker
usermod -a -G docker ec2-user

# Install jq for scripting
yum install -y jq

# Edit Docker storage directory to mounted volume
DOCKER_DAEMON_CONFIG_FILE='/etc/docker/daemon.json'

## initialize the config file with empty json if it doesn't exist
if [ ! -f "${DOCKER_DAEMON_CONFIG_FILE}" ]
then
  mkdir -p $(dirname "${DOCKER_DAEMON_CONFIG_FILE}")
  echo '{}' > "${DOCKER_DAEMON_CONFIG_FILE}"
fi

## update Docker's 'data-root' to point to our mounted disk
tmp_config=$(mktemp)
trap "rm -f ${tmp_config}" EXIT
cat "${DOCKER_DAEMON_CONFIG_FILE}" | jq --arg DATA_ROOT "${DOCKER_DATA_ROOT}" '.["data-root"]=$DATA_ROOT' > "${tmp_config}"
cat "${tmp_config}" > "${DOCKER_DAEMON_CONFIG_FILE}"

# Configure inotify limits for indexed search.
echo -e "\nfs.inotify.max_user_watches = 128000\n" >> /etc/sysctl.d/local.conf

## finally, restart Docker daemon to pick up our changes
systemctl restart --now docker

# Install Docker Compose
curl -L "https://github.com/docker/compose/releases/download/${DOCKER_COMPOSE_VERSION}/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
chmod +x /usr/local/bin/docker-compose
curl -L "https://raw.githubusercontent.com/docker/compose/${DOCKER_COMPOSE_VERSION}/contrib/completion/bash/docker-compose" -o /etc/bash_completion.d/docker-compose

# Run Sourcegraph. Restart the containers upon reboot.
cd "${DEPLOY_SOURCEGRAPH_DOCKER_CHECKOUT}"/docker-compose
docker-compose up -d
```

* Select **Next: Add Storage**
* Click "Add New Volume" and add an additional volume (for storing Docker data) with the following settings:

  * **Volume Type** (left-most column): EBS
  * **IMPORTANT: Device**: `/dev/sdb` 
  * **Size (GiB)**: `250` GB minimum *(As a rule of thumb, Sourcegraph needs at least as much space as all your repositories combined take up. Allocating as much disk space as you can upfront helps you avoid [resizing your volume](https://aws.amazon.com/premiumsupport/knowledge-center/expand-root-ebs-linux/) later on.)*
  * **Volume Type**: General Purpose SSD (gp2)
  * **Delete on Termination**: Leave this setting unchecked

* Select **Next: ...** until you get to the **Configure Security Group** page. Then add the following rules:

  * Default **HTTP** rule: port range `80`, source `0.0.0.0/0, ::/0`
  * Default **HTTPS** rule: port range `443`, source `0.0.0.0/0, ::/0`<br>(NOTE: additional work will be required later on to [configure SSL in the Docker Compose deployment](../../../admin/http_https_configuration.md#sourcegraph-via-docker-compose-caddy-2))

* Launch your instance, then navigate to its public IP in your browser. (This can be found by navigating to the instance page on EC2 and looking in the "Description" panel for the "IPv4 Public IP" value.) You may have to wait a minute or two for the instance to finish initializing before Sourcegraph becomes accessible. You can monitor the status by SSHing into the instance and using the following diagnostic commands:

```bash
# Follow the status of the user data script you provided earlier
tail -f /var/log/cloud-init-output.log

# (Once the user data script completes) monitor the health of the "sourcegraph-frontend" container
docker ps --filter="name=sourcegraph-frontend-0"
```

---

## Update your Sourcegraph version

To update to the most recent version of Sourcegraph (X.Y.Z), SSH into your instance and run the following:

```bash
cd /home/ec2-user/deploy-sourcegraph-docker/docker-compose
git pull
git checkout vX.Y.Z
docker-compose up -d
```

---

## Storage and Backups

The [Sourcegraph Docker Compose definition](https://github.com/sourcegraph/deploy-sourcegraph-docker/blob/master/docker-compose/docker-compose.yaml) uses [Docker volumes](https://docs.docker.com/storage/volumes/) to store its data. The script above [configures Docker](https://docs.docker.com/engine/reference/commandline/dockerd/#daemon-configuration-file) to store all Docker data on the additional EBS volume that was attached to the instance (mounted at `/mnt/docker-data` - the volumes themselves are stored under `/mnt/docker-data/volumes`) There are a few different ways to backup this data:

* (**recommended**) The most straightfoward method to backup this data is to [snapshot the entire `/mnt/docker-data` EBS disk](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ebs-creating-snapshot.html) on an [automatic, scheduled basis](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/snapshot-lifecycle.html).

* Using an external Postgres instance (see below) lets a service such as [AWS RDS for PostgreSQL](https://aws.amazon.com/rds/) take care of backing up all of Sourcegraph's user data for you. If the EC2 instance running Sourcegraph ever dies or is destroyed, creating a fresh instance that's connected to that external Postgres will leave Sourcegraph in the same state that it was before.

---

## Using an external database for persistence

The Docker Compose configuration has its own internal PostgreSQL and Redis databases. To preserve this data when you kill and recreate the containers, you can [use external databases](../../external_database.md) for persistence, such as [AWS RDS for PostgreSQL](https://aws.amazon.com/rds/) and [Amazon ElastiCache](https://aws.amazon.com/elasticache/redis/).

> NOTE: Use of external databases requires [Sourcegraph Enterprise](https://about.sourcegraph.com/pricing).
