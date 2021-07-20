package uploadstore

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/cockroachdb/errors"
	"github.com/inconshreveable/log15"
	"github.com/opentracing/opentracing-go/log"

	"github.com/sourcegraph/sourcegraph/internal/env"
	"github.com/sourcegraph/sourcegraph/internal/goroutine"
	"github.com/sourcegraph/sourcegraph/internal/observation"
)

type s3Store struct {
	bucket                       string
	manageBucket                 bool
	client                       s3API
	uploader                     s3Uploader
	bucketLifecycleConfiguration *s3types.BucketLifecycleConfiguration
	operations                   *operations
}

var _ Store = &s3Store{}

type S3Config struct {
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
}

func (c *S3Config) load(parent *env.BaseConfig) {
	c.Region = parent.Get("PRECISE_CODE_INTEL_UPLOAD_AWS_REGION", "us-east-1", "The target AWS region.")
	c.Endpoint = parent.Get("PRECISE_CODE_INTEL_UPLOAD_AWS_ENDPOINT", "http://minio:9000", "The target AWS endpoint.")
	c.AccessKeyID = parent.Get("PRECISE_CODE_INTEL_UPLOAD_AWS_ACCESS_KEY_ID", "AKIAIOSFODNN7EXAMPLE", "An AWS access key associated with a user with access to S3.")
	c.SecretAccessKey = parent.Get("PRECISE_CODE_INTEL_UPLOAD_AWS_SECRET_ACCESS_KEY", "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY", "An AWS secret key associated with a user with access to S3.")
	c.SessionToken = parent.GetOptional("PRECISE_CODE_INTEL_UPLOAD_AWS_SESSION_TOKEN", "An optional AWS session token associated with a user with access to S3.")
}

// newS3FromConfig creates a new store backed by AWS Simple Storage Service.
func newS3FromConfig(ctx context.Context, config *Config, operations *operations) (Store, error) {
	cfg, err := s3ClientConfig(ctx, config.S3)
	if err != nil {
		return nil, err
	}

	s3Client := s3.NewFromConfig(cfg, s3ClientOptions(config.Backend, config.S3))
	api := &s3APIShim{s3Client}
	uploader := &s3UploaderShim{manager.NewUploader(s3Client)}
	return newS3WithClients(api, uploader, config.Bucket, config.ManageBucket, s3BucketLifecycleConfiguration(config.Backend, config.TTL), operations), nil
}

func newS3WithClients(client s3API, uploader s3Uploader, bucket string, manageBucket bool, lifecycleConfiguration *s3types.BucketLifecycleConfiguration, operations *operations) *s3Store {
	return &s3Store{
		bucket:                       bucket,
		manageBucket:                 manageBucket,
		client:                       client,
		uploader:                     uploader,
		operations:                   operations,
		bucketLifecycleConfiguration: lifecycleConfiguration,
	}
}

func (s *s3Store) Init(ctx context.Context) error {
	if !s.manageBucket {
		return nil
	}

	if err := s.create(ctx); err != nil {
		return errors.Wrap(err, "failed to create bucket")
	}

	if err := s.update(ctx); err != nil {
		return errors.Wrap(err, "failed to update bucket attributes")
	}

	return nil
}

// maxZeroReads is the maximum number of no-progress iterations (due to connection reset errors)
// in Get that can occur in a row before returning an error.
const maxZeroReads = 3

// errNoDownloadProgress is returned from Get after multiple connection reset errors occur
// in a row.
var errNoDownloadProgress = errors.New("no download progress")

func (s *s3Store) Get(ctx context.Context, key string) (_ io.ReadCloser, err error) {
	ctx, endObservation := s.operations.get.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.String("key", key),
	}})
	defer endObservation(1, observation.Args{})

	reader := writeToPipe(func(w io.Writer) error {
		zeroReads := 0
		byteOffset := int64(0)

		for {
			n, err := s.readObjectInto(ctx, w, key, byteOffset)
			if err == nil || !isConnectionResetError(err) {
				return err
			}

			byteOffset += n
			log15.Warn("Transient error while reading payload", "key", key, "error", err)

			if n == 0 {
				zeroReads++

				if zeroReads > maxZeroReads {
					return errNoDownloadProgress
				}
			} else {
				zeroReads = 0
			}
		}
	})

	return io.NopCloser(reader), nil
}

// ioCopyHook is a pointer to io.Copy. This function is replaced in unit tests so that we can
// easily inject errors when reading from the backing S3 store.
var ioCopyHook = io.Copy

// readObjectInto reads the content of the given key starting at the given byte offset into the
// given writer. The number of bytes read is returned. On successful read, the error value is nil.
func (s *s3Store) readObjectInto(ctx context.Context, w io.Writer, key string, byteOffset int64) (int64, error) {
	var bytesRange *string
	if byteOffset > 0 {
		bytesRange = aws.String(fmt.Sprintf("bytes=%d-", byteOffset))
	}

	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Range:  bytesRange,
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to get object")
	}
	defer resp.Body.Close()

	return ioCopyHook(w, resp.Body)
}

func (s *s3Store) Upload(ctx context.Context, key string, r io.Reader) (_ int64, err error) {
	ctx, endObservation := s.operations.upload.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.String("key", key),
	}})
	defer endObservation(1, observation.Args{})

	cr := &countingReader{r: r}

	if err := s.uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
		Body:   cr,
	}); err != nil {
		return 0, errors.Wrap(err, "failed to upload object")
	}

	return int64(cr.n), nil
}

func (s *s3Store) Compose(ctx context.Context, destination string, sources ...string) (_ int64, err error) {
	ctx, endObservation := s.operations.compose.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.String("destination", destination),
		log.String("sources", strings.Join(sources, ", ")),
	}})
	defer endObservation(1, observation.Args{})

	multipartUpload, err := s.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(destination),
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to create multipart upload")
	}

	defer func() {
		if err == nil {
			// Delete sources on success
			if err := s.deleteSources(ctx, *multipartUpload.Bucket, sources); err != nil {
				log15.Error("Failed to delete source objects", "error", err)
			}
		} else {
			// On failure, try to clean up copied then orphaned parts
			if _, err := s.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
				Bucket:   multipartUpload.Bucket,
				Key:      multipartUpload.Key,
				UploadId: multipartUpload.UploadId,
			}); err != nil {
				log15.Error("Failed to abort multipart upload", "error", err)
			}
		}
	}()

	var m sync.Mutex
	etags := map[int]*string{}

	if err := goroutine.RunWorkersOverStrings(sources, func(index int, source string) error {
		partNumber := index + 1

		copyResult, err := s.client.UploadPartCopy(ctx, &s3.UploadPartCopyInput{
			Bucket:     multipartUpload.Bucket,
			Key:        multipartUpload.Key,
			UploadId:   multipartUpload.UploadId,
			PartNumber: int32(partNumber),
			CopySource: aws.String(fmt.Sprintf("%s/%s", s.bucket, source)),
		})
		if err != nil {
			return errors.Wrap(err, "failed to upload part")
		}

		m.Lock()
		etags[partNumber] = copyResult.CopyPartResult.ETag
		m.Unlock()

		return nil
	}); err != nil {
		return 0, err
	}

	var parts []s3types.CompletedPart
	for i := 0; i < len(sources); i++ {
		partNumber := i + 1

		parts = append(parts, s3types.CompletedPart{
			ETag:       etags[partNumber],
			PartNumber: int32(partNumber),
		})
	}

	if _, err := s.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:          multipartUpload.Bucket,
		Key:             multipartUpload.Key,
		UploadId:        multipartUpload.UploadId,
		MultipartUpload: &s3types.CompletedMultipartUpload{Parts: parts},
	}); err != nil {
		return 0, errors.Wrap(err, "failed to complete multipart upload")
	}

	obj, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: multipartUpload.Bucket,
		Key:    multipartUpload.Key,
	})
	if err != nil {
		return 0, errors.Wrap(err, "failed to stat composed object")
	}

	return obj.ContentLength, nil
}

func (s *s3Store) Delete(ctx context.Context, key string) (err error) {
	ctx, endObservation := s.operations.delete.With(ctx, &err, observation.Args{LogFields: []log.Field{
		log.String("key", key),
	}})
	defer endObservation(1, observation.Args{})

	_, err = s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})

	return errors.Wrap(err, "failed to delete object")
}

func (s *s3Store) create(ctx context.Context) error {
	_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(s.bucket),
	})

	if errors.HasType(err, &s3types.BucketAlreadyExists{}) || errors.HasType(err, &s3types.BucketAlreadyOwnedByYou{}) {
		return nil
	}

	return err
}

func (s *s3Store) update(ctx context.Context) error {
	configureRequest := &s3.PutBucketLifecycleConfigurationInput{
		Bucket:                 aws.String(s.bucket),
		LifecycleConfiguration: s.bucketLifecycleConfiguration,
	}

	_, err := s.client.PutBucketLifecycleConfiguration(ctx, configureRequest)
	return err
}

func (s *s3Store) deleteSources(ctx context.Context, bucket string, sources []string) error {
	return goroutine.RunWorkersOverStrings(sources, func(index int, source string) error {
		if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(source),
		}); err != nil {
			return errors.Wrap(err, "failed to delete source object")
		}

		return nil
	})
}

// countingReader is an io.Reader that counts the number of bytes sent
// back to the caller.
type countingReader struct {
	r io.Reader
	n int
}

func (r *countingReader) Read(p []byte) (n int, err error) {
	n, err = r.r.Read(p)
	r.n += n
	return n, err
}

func s3ClientConfig(ctx context.Context, s3config S3Config) (aws.Config, error) {
	optFns := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(s3config.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			s3config.AccessKeyID,
			s3config.SecretAccessKey,
			s3config.SessionToken,
		)),
	}

	return awsconfig.LoadDefaultConfig(ctx, optFns...)
}

func s3ClientOptions(backend string, config S3Config) func(o *s3.Options) {
	return func(o *s3.Options) {
		if backend == "minio" {
			o.EndpointResolver = s3.EndpointResolverFromURL(config.Endpoint)
			o.UsePathStyle = true
		}
	}
}

// writeToPipe invokes the given function with a pipe writer in a goroutine
// and returns the associated pipe reader.
func writeToPipe(fn func(w io.Writer) error) io.Reader {
	pr, pw := io.Pipe()
	go func() { _ = pw.CloseWithError(fn(pw)) }()
	return pr
}

func isConnectionResetError(err error) bool {
	return err != nil && strings.Contains(err.Error(), "read: connection reset by peer")
}

func s3BucketLifecycleConfiguration(backend string, ttl time.Duration) *s3types.BucketLifecycleConfiguration {
	days := int32(ttl / (time.Hour * 24))

	rules := []s3types.LifecycleRule{
		{
			ID:         aws.String("Expiration Rule"),
			Status:     s3types.ExpirationStatusEnabled,
			Filter:     &s3types.LifecycleRuleFilterMemberPrefix{Value: ""},
			Expiration: &s3types.LifecycleExpiration{Days: days},
		},
	}

	if backend != "minio" {
		rules = append(rules, s3types.LifecycleRule{
			ID:                             aws.String("Abort Incomplete Multipart Upload Rule"),
			Status:                         s3types.ExpirationStatusEnabled,
			Filter:                         &s3types.LifecycleRuleFilterMemberPrefix{Value: ""},
			AbortIncompleteMultipartUpload: &s3types.AbortIncompleteMultipartUpload{DaysAfterInitiation: days},
		})
	}

	return &s3types.BucketLifecycleConfiguration{Rules: rules}
}
