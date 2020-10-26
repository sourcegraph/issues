package graphqlbackend

type repositoryContributorResolver struct {
	name  string
	email string
	count int32

	repo *RepositoryResolver
	args repositoryContributorsArgs
}

func (r *repositoryContributorResolver) Person() *PersonResolver {
	return &PersonResolver{name: r.name, email: r.email}
}

func (r *repositoryContributorResolver) Count() int32 { return r.count }

func (r *repositoryContributorResolver) Repository() *RepositoryResolver { return r.repo }

func (r *repositoryContributorResolver) Commits(args *struct {
	First *int32
}) *gitCommitConnectionResolver {
	var revisionRange string
	if r.args.RevisionRange != nil {
		revisionRange = *r.args.RevisionRange
	}
	return &gitCommitConnectionResolver{
		revisionRange: revisionRange,
		path:          r.args.Path,
		author:        &r.email, // TODO(sqs): support when contributor resolves to user, and user has multiple emails
		after:         r.args.After,
		first:         args.First,
		repo:          r.repo,
	}
}
