# Provisioning a Kubernetes cluster

<div class="alert alert-info">

**Security note:** If you intend to set this up as a production instance, we recommend you create the cluster in a VPC
or other secure network that restricts unauthenticated access from the public Internet. You can later expose the
necessary ports via an
[Internet Gateway](http://docs.aws.amazon.com/AmazonVPC/latest/UserGuide/VPC_Internet_Gateway.html) or equivalent
mechanism. Take care to secure your cluster in a manner that meets your organization's security requirements.

</div>

Follow the instructions linked in the table below to provision a Kubernetes cluster for the
infrastructure provider of your choice, using the recommended node and list types in the
table.

> Note: Sourcegraph can run on any Kubernetes cluster, so if your infrastructure provider is not
> listed, see the "Other" row. Pull requests to add rows for more infrastructure providers are
> welcome!

|Provider|Node type|Boot/ephemeral disk size|
|--- |--- |--- |
|Compute nodes| | |
|[Amazon EKS (better than plain EC2)](eks.md)|m5.4xlarge|N/A|
|[AWS EC2](https://kubernetes.io/docs/getting-started-guides/aws/)|m5.4xlarge|N/A|
|[Google Kubernetes Engine (GKE)](https://cloud.google.com/kubernetes-engine/docs/quickstart)|n1-standard-16|100 GB (default)|
|[Azure](azure.md)|D16 v3|100 GB (SSD preferred)|
|[Other](https://kubernetes.io/docs/setup/pick-right-solution/)|16 vCPU, 60 GiB memory per node|100 GB (SSD preferred)|
