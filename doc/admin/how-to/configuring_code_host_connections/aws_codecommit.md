# AWS CodeCommit

Site admins can sync Git repositories hosted on [AWS CodeCommit](https://aws.amazon.com/codecommit/) with Sourcegraph so that users can search and navigate the repositories.

To connect AWS CodeCommit to Sourcegraph:

1. Go to **Site admin > Manage repositories > Add repositories**
1. Select **AWS CodeCommit repositories**.
1. Configure the connection to AWS CodeCommit using the action buttons above the text field, and additional fields can be added using <kbd>Cmd/Ctrl+Space</kbd> for auto-completion. See the [configuration documentation below](#configuration).
1. Press **Add repositories**.

## AWS CodeCommit Git credentials

Since version **3.4** of Sourcegraph, the AWS CodeCommit service **requires** [Git credentials](https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_ssh-keys.html#git-credentials-code-commit) in order to clone repositories via HTTPS. Git credentials consist of a username and a password that you can create in AWS IAM. 

For detailed instructions on how to create the credentials in IAM, see: [Setup for HTTPS Users Using Git Credentials](https://docs.aws.amazon.com/codecommit/latest/userguide/setting-up-gc.html)

## Configuration

AWS CodeCommit connections support the following configuration options, which are specified in the JSON editor in the site admin "Manage repositories" area.

<div markdown-func=jsonschemadoc jsonschemadoc:path="admin/external_service/aws_codecommit.schema.json">[View page on docs.sourcegraph.com](https://docs.sourcegraph.com/admin/external_service/aws_codecommit) to see rendered content.</div>
