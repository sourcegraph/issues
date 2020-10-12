# Kubernetes on Amazon EKS

[Amazon EKS](https://aws.amazon.com/eks/) is Amazon's managed Kubernetes offering, similar to how Google Cloud offers managed Kubernetes clusters (GKE).

If your preferred cloud provider is Amazon, we strongly recommend using EKS instead of plain EC2. By using EKS, you will not need to manage your own Kubernetes control plane (complex). Instead, Amazon will provide it for you and you will only be responsible for managing Sourcegraph, which runs on the Kubernetes cluster.

## Create the Amazon EKS Service Role

Follow the [EKS Getting Started guide](https://docs.aws.amazon.com/eks/latest/userguide/getting-started.html#eks-prereqs) to create the IAM EKS service role:

1. Open the [**IAM console**](https://console.aws.amazon.com/iam/).
2. Click **Roles** -> **Create role**.
3. Choose **EKS**, accept the defaults and **Next: Permissions**.
4. Click **Next: Review**.
5. Under **Role name**, enter `eksServiceRoleSourcegraph`, then **Create role**.

## Create the Amazon EKS Cluster VPC

1. Open the [**AWS CloudFormation console**](https://console.aws.amazon.com/cloudformation/).
2. Ensure the region in the top right navigation bar is an EKS-supported region (many regions do not support EKS yet - see [this list](https://docs.aws.amazon.com/general/latest/gr/eks.html) for supported regions).
3. Click **Create stack**, and select **"with new resources"**.
4. When prompted to specify a template, select "Amazon S3 URL" as your **Template Source** and enter:
   ```
   https://amazon-eks.s3-us-west-2.amazonaws.com/cloudformation/2020-04-21/amazon-eks-vpc-sample.yaml
   ```
5. Under **Stack name**, enter `eks-vpc-sourcegraph`.
6. Click **Next** through the following pages until you get the option to **Create stack**. Review the configuration and click **Create stack**.

For more details on these steps, refer to [Amazon EKS prerequisites: Create your Amazon EKS cluster VPC](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-console.html#vpc-create).

## Create the Amazon EKS Cluster

1. Open the [**EKS console**](https://console.aws.amazon.com/eks/home#/clusters).
2. Click **Create cluster**.
3. Under **Cluster name**, enter `sourcegraph`.
4. Under **Cluster Service Role**, select `eksServiceRoleSourcegraph`.
5. Under **VPC**, select `eks-vpc-sourcegraph`.
6. Under **Security groups**, select the one prefixed `eks-vpc-sourcegraph-ControlPlaneSecurityGroup-`. (Do NOT select `NodeSecurityGroup`.)
7. Accept all other values as default and click **Create**.
8. Wait for the cluster to finish **CREATING**. This will take around 10 minutes to complete, so grab some ☕.

For more details on these steps, refer to [Amazon EKS prerequisites: Create your Amazon EKS cluster](https://docs.aws.amazon.com/eks/latest/userguide/getting-started-console.html#eks-create-cluster).

## Create Kubernetes cluster worker nodes

1. Open the [**AWS CloudFormation console**](https://console.aws.amazon.com/cloudformation/).
2. Click **Create stack**
3. Select the very last **Specify an Amazon S3 template URL** option and enter  
   ```
   https://amazon-eks.s3.us-west-2.amazonaws.com/cloudformation/2020-04-21/amazon-eks-nodegroup.yaml
   ```
4. Under **Stack name**, enter `sourcegraph-worker-nodes`.
5. Under **ClusterName**, enter the exact cluster name you used (`sourcegraph`).
6. Under **ClusterControlPlaneSecurityGroup**, scroll down or begin typing and select the option prefixed `eks-vpc-sourcegraph-ControlPlaneSecurityGroup-` (Do NOT select the `NodeSecurityGroup`.)
7. Under **NodeGroupName**, enter `sourcegraph-node-group`.
8. Choose **NodeAutoScalingGroupMinSize** and **NodeAutoScalingGroupMaxSize** and **NodeInstanceType** based on the following chart:

| Users        | Instance type | Min nodes | Max nodes | Cost est.  | Attached Storage | Root Storage |
| ------------ | ------------- | --------- | --------- | ---------- | ---------------- | ------------ |
| 10-500        | m5.4xlarge    | 3         | 6         | $59-118/day | 500 GB           | 100 GB        |
| 500-2000       | m5.4xlarge   | 6         | 10         | $118-195/day | 500 GB           | 100 GB        |

> **Note:** You can always come back here later and modify these values to scale up/down the number of worker nodes. To do so, just visit the console page again, select **Actions**, **Create Change Set For Current Stack**, enter the same template URL mentioned above, modify the values and hit "next" until reviewing final changes, and finally **Execute**.

9.  Under **KeyName**, choose a valid key name so that you can SSH into worker nodes if needed in the future.
10.   Under **VpcId**, select `eks-vpc-sourcegraph-VPC`.
11.   Under **Subnets**, search for and select *all* `eks-vpc-sourcegraph` subnets.
12.   Click **Next** through the following pages until you get the option to **Create stack**. Review the configuration and click **Create stack**.

For more details on these steps, refer to [Worker Nodes: Amazon EKS-optimized Linux AMI](https://docs.aws.amazon.com/eks/latest/userguide/eks-optimized-ami.html).

## Install `kubectl` and configure access to the cluster

On your dev machine:

1. Install the `aws` CLI tool: [bundled installer](https://docs.aws.amazon.com/cli/latest/userguide/awscli-install-bundle.html), [other installation methods](https://docs.aws.amazon.com/cli/latest/userguide/installing.html).
2. Follow [these instructions](https://docs.aws.amazon.com/cli/latest/userguide/cli-chap-getting-started.html) to create an access key and `aws configure` the CLI to use it.
3. Install `kubectl` and `aws-iam-authenticator` by following [these steps](https://docs.aws.amazon.com/eks/latest/userguide/configure-kubectl.html).
4. [Configure `kubectl` to interact with your cluster](https://docs.aws.amazon.com/eks/latest/userguide/create-kubeconfig.html):
   ```
   aws eks update-kubeconfig --name ${cluster_name}
   ```

**Important**: If `kubectl` commands prompt you for username/password, be sure that `kubectl version` reports a client version of v1.10+. Older versions of kubectl do not work with the authentication configuration provided by Amazon EKS.

At this point, `kubectl get svc` should show something like:

```
$ kubectl get svc
NAME         TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
kubernetes   ClusterIP   172.20.0.1   <none>        443/TCP   4m
```

## Enable worker nodes to join the Kubernetes cluster

Now it is time to enable the worker nodes created by CloudFormation to actually join the Kubernetes cluster:

1. Download, edit, and save [this configuration map file](https://docs.aws.amazon.com/eks/latest/userguide/add-user-role.html):
   ```
   curl -O curl -o aws-auth-cm.yaml https://amazon-eks.s3.us-west-2.amazonaws.com/cloudformation/2020-04-21/aws-auth-cm.yaml
   ```
2. Replace `rolearn` in the file (_do not_ modify the file otherwise) with the correct value. To find this value:
   - Open the [**AWS CloudFormation console**](https://console.aws.amazon.com/cloudformation/).
   - Locate and select the `sourcegraph-worker-nodes` row.
   - Click the **Output** tab, and copy the **NodeInstanceRole** value.
3. Run `kubectl apply -f aws-auth-cm.yaml`
4. Watch `kubectl get nodes --watch` until all nodes appear with status `Ready` (this will take a few minutes).

## Create the default storage class

EKS does not have a default Kubernetes storage class out of the box, but one is needed.

Follow [these short steps](https://docs.aws.amazon.com/eks/latest/userguide/storage-classes.html) to create it. (Simply copy and paste the suggested file and run all suggested `kubectl` commands. You do not need to modify the file.)

## Deploy the Kubernetes Web UI Dashboard (optional)

See [Tutorial: Deploy the Kubernetes Dashboard](https://docs.aws.amazon.com/eks/latest/userguide/dashboard-tutorial.html).

## Deploy Sourcegraph! 🎉

Your Kubernetes cluster is now all set up and running!

Luckily, deploying Sourcegraph on your cluster is much easier and quicker than the above steps. :)

Follow our [installation documentation](index.md) to continue.
