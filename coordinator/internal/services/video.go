package services

import (
	"context"
	"coordinator/internal/domain"
	st "coordinator/internal/storage"
	"coordinator/internal/utility"
	"coordinator/pkg/sse"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Errors
var (
	ErrVideoStorageUnavailable = utility.NewInternalError("video storage is not configured")
	ErrS3EndpointInvalid       = utility.NewInternalError("given S3 endpoint is invalid")
)

// Constants
const (
	uploadLinkTTL   = time.Hour
	downloadLinkTTL = 24 * time.Hour
)

// S3Storage issues presigned links for an S3-compatible storage
type S3Storage struct {
	endpoint  url.URL
	accessKey string
	secretKey string
	region    string
	bucket    string
}

// VideoService exposes downloaded videos
type VideoService struct {
	storage st.Storage
	s3      *S3Storage
}

// NewVideoService creates a new VideoService
func NewVideoService(_ context.Context, _ *sse.Hub, storage st.Storage, s3URL, s3Region, s3Bucket, s3AccessKey, s3SecretKey string) (*VideoService, error) {
	if storage == nil {
		return nil, ErrVideoStorageUnavailable
	}

	s3, err := NewS3Storage(s3URL, s3AccessKey, s3SecretKey, s3Region, s3Bucket)
	if err != nil {
		return nil, err
	}

	return &VideoService{
		storage: storage,
		s3:      s3,
	}, nil
}

// StorageKey returns the bucket and object key a resource is stored under
func (s *VideoService) StorageKey(resourceID int) (string, string) {
	return s.s3.bucket, fmt.Sprintf("%d", resourceID)
}

// UploadLink returns a presigned link a worker uploads a resource to
func (s *VideoService) UploadLink(resourceID int) (string, error) {
	bucket, objectKey := s.StorageKey(resourceID)
	return s.s3.presign("PUT", bucket, objectKey, uploadLinkTTL)
}

// GetVideoByID returns a video with the given id
func (s *VideoService) GetVideoByID(ctx context.Context, id int) (domain.Video, string, error) {
	video, err := s.storage.GetVideoByID(ctx, id)
	if err != nil {
		return video, "", err
	}

	streamURL, err := s.downloadLink(video)
	if err != nil {
		return video, "", err
	}

	return video, streamURL, nil
}

// downloadLink returns a link the video can be streamed from
func (s *VideoService) downloadLink(video domain.Video) (string, error) {
	bucket := video.StorageMetadata.BucketName
	objectKey := strings.TrimLeft(video.StorageMetadata.ObjectKey, "/")
	return s.s3.DownloadLink(bucket, objectKey)
}

// GetVideosByJobID returns every video belonging to a job
func (s *VideoService) GetVideosByJobID(ctx context.Context, jobID int) ([]domain.Video, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.storage.GetVideosByJobID(ctx, jobID)
}

// AddVideo stores a video
func (s *VideoService) AddVideo(ctx context.Context, video domain.Video) error {
	return s.storage.AddVideo(ctx, &video)
}

// Search returns videos matching the query
func (s *VideoService) Search(ctx context.Context, query string, limit, offset int) ([]domain.Video, error) {
	return s.storage.GetVideosByQuery(ctx, query, limit, offset)
}

// GetRecommended returns a page of recommended videos
func (s *VideoService) GetRecommended(ctx context.Context, limit, offset int) ([]domain.Video, error) {
	return s.storage.GetVideos(ctx, limit, offset)
}

// GetRecommendedForVideo returns videos related to the given video
func (s *VideoService) GetRecommendedForVideo(ctx context.Context, videoID, limit, offset int) ([]domain.Video, error) {
	return s.storage.GetVideos(ctx, limit, offset)
}

// NewS3Storage creates a new StorageService
func NewS3Storage(endpoint string, accessKey string, secretKey string, region string, bucket string) (*S3Storage, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Host == "" {
		return nil, ErrS3EndpointInvalid
	}

	return &S3Storage{
		endpoint:  *parsed,
		accessKey: accessKey,
		secretKey: secretKey,
		region:    region,
		bucket:    bucket,
	}, nil
}

// Bucket returns the default bucket videos are stored in
func (s *S3Storage) Bucket() string {
	return s.bucket
}

// UploadLink returns a presigned PUT link a resource is uploaded to
func (s *S3Storage) UploadLink(bucket string, objectKey string) (string, error) {
	return s.presign("PUT", bucket, objectKey, uploadLinkTTL)
}

// DownloadLink returns a presigned GET link a resource is streamed from
func (s *S3Storage) DownloadLink(bucket string, objectKey string) (string, error) {
	return s.presign("GET", bucket, objectKey, downloadLinkTTL)
}

// presign creates an AWS SigV4 query-signed link for the given object
func (s *S3Storage) presign(method string, bucket string, objectKey string, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	amzDate := now.Format("20060102T150405Z")
	date := now.Format("20060102")

	scope := strings.Join([]string{date, s.region, "s3", "aws4_request"}, "/")
	path := "/" + bucket + "/" + strings.TrimLeft(objectKey, "/")

	// Only the host header is signed, everything else is enforced by the query
	query := map[string]string{
		"X-Amz-Algorithm":     "AWS4-HMAC-SHA256",
		"X-Amz-Credential":    s.accessKey + "/" + scope,
		"X-Amz-Date":          amzDate,
		"X-Amz-Expires":       strconv.Itoa(int(ttl.Seconds())),
		"X-Amz-SignedHeaders": "host",
	}
	canonicalQuery := encodeQuery(query)

	canonicalRequest := strings.Join([]string{
		method,
		encodePath(path),
		canonicalQuery,
		"host:" + s.endpoint.Host + "\n",
		"host",
		"UNSIGNED-PAYLOAD",
	}, "\n")

	stringToSign := strings.Join([]string{
		"AWS4-HMAC-SHA256",
		amzDate,
		scope,
		hashHex(canonicalRequest),
	}, "\n")

	key := hmacSHA256([]byte("AWS4"+s.secretKey), date)
	key = hmacSHA256(key, s.region)
	key = hmacSHA256(key, "s3")
	key = hmacSHA256(key, "aws4_request")

	signature := hex.EncodeToString(hmacSHA256(key, stringToSign))

	return fmt.Sprintf(
		"%s%s?%s&X-Amz-Signature=%s",
		s.endpoint.String(),
		encodePath(path),
		canonicalQuery,
		signature,
	), nil
}

// uriEncode percent-encodes a value the way AWS SigV4 expects it
func uriEncode(value string) string {
	var encoded strings.Builder
	for _, c := range []byte(value) {
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9',
			c == '-', c == '.', c == '_', c == '~':
			encoded.WriteByte(c)
		default:
			fmt.Fprintf(&encoded, "%%%02X", c)
		}
	}
	return encoded.String()
}

// encodePath encodes every path segment while keeping the slashes
func encodePath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = uriEncode(segment)
	}
	return strings.Join(segments, "/")
}

// encodeQuery renders query parameters in the canonical sorted form
func encodeQuery(query map[string]string) string {
	keys := make([]string, 0, len(query))
	for key := range query {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(query))
	for _, key := range keys {
		pairs = append(pairs, uriEncode(key)+"="+uriEncode(query[key]))
	}
	return strings.Join(pairs, "&")
}

// hashHex returns the hex digest of the given data
func hashHex(data string) string {
	digest := sha256.Sum256([]byte(data))
	return hex.EncodeToString(digest[:])
}

// hmacSHA256 signs the given data with the given key
func hmacSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
