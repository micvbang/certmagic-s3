package certmagic_s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/certmagic"
)

type S3 struct {

	// S3
	S3       s3iface.S3API
	bucket   string `json:"bucket"`
	prefix   string `json:"prefix"`
	Insecure bool   `json:"insecure"`
}

func init() {
	caddy.RegisterModule(S3{})
}

func (s *S3) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		var value string

		key := d.Val()

		if !d.Args(&value) {
			continue
		}

		switch key {
		case "bucket":
			s.bucket = value
		case "prefix":
			s.prefix = value
		case "insecure":
			insecure, err := strconv.ParseBool(value)
			if err != nil {
				return d.Err("Invalid usage of insecure in s3-storage config: " + err.Error())
			}
			s.Insecure = insecure
		}

	}

	return nil
}

func (s *S3) Provision(ctx caddy.Context) error {
	awsSession, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("creating s3 session: %w", err)
	}

	s.S3 = s3.New(awsSession)

	return nil
}

func (S3) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "caddy.storage.s3",
		New: func() caddy.Module {
			return new(S3)
		},
	}
}

func (s S3) CertMagicStorage() (certmagic.Storage, error) {
	return s, nil
}

func (s S3) Lock(ctx context.Context, key string) error {
	return nil
}

func (s S3) Unlock(ctx context.Context, key string) error {
	return nil
}

func (s S3) Store(ctx context.Context, key string, value []byte) error {
	length := int64(len(value))

	_, err := s.S3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           aws.String(s.keyPrefix(key)),
		Body:          bytes.NewReader(value),
		ContentLength: &length,
	})

	return err
}

func (s S3) Load(ctx context.Context, key string) ([]byte, error) {
	if !s.Exists(ctx, key) {
		return nil, fs.ErrNotExist
	}

	object, err := s.S3.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &s.bucket,
		Key:    aws.String(s.keyPrefix(key)),
	})
	if err != nil {
		return nil, err
	}
	defer object.Body.Close()

	return io.ReadAll(object.Body)
}

func (s S3) Delete(ctx context.Context, key string) error {
	key = s.keyPrefix(key)

	_, err := s.S3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &s.bucket,
		Key:    &key,
	})
	return err
}

func (s S3) Exists(ctx context.Context, key string) bool {
	_, err := s.Stat(ctx, key)

	exists := err == nil
	return exists
}

func (s S3) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {
	keys := make([]string, 0, 32)
	err := s.S3.ListObjectsPagesWithContext(ctx, &s3.ListObjectsInput{
		Bucket: aws.String(s.bucket),
		Prefix: &s.prefix,
	}, func(objects *s3.ListObjectsOutput, b bool) bool {
		for _, obj := range objects.Contents {
			if obj == nil || obj.Key == nil {
				continue
			}

			keys = append(keys, *obj.Key)
		}
		return true
	})

	return keys, err
}

func (s S3) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
	keyWithPrefix := s.keyPrefix(key)
	attrs, err := s.S3.GetObjectAttributes(&s3.GetObjectAttributesInput{
		Bucket: &s.bucket,
		Key:    aws.String(keyWithPrefix),
	})
	if err != nil {
		return certmagic.KeyInfo{}, nil
	}

	return certmagic.KeyInfo{
		Key:        keyWithPrefix,
		Modified:   *attrs.LastModified,
		Size:       *attrs.ObjectSize,
		IsTerminal: strings.HasSuffix(keyWithPrefix, "/"),
	}, nil
}

func (s S3) keyPrefix(key string) string {
	return path.Join(s.prefix, key)
}

func (s S3) String() string {
	return fmt.Sprintf("S3 Storage Bucket: %s, Prefix: %s", s.bucket, s.prefix)
}
