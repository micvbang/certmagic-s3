package certmagic_s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/certmagic"
	"go.uber.org/zap"
)

type S3 struct {
	log *zap.SugaredLogger

	// S3
	S3     s3iface.S3API
	Bucket string `json:"bucket"`
	Prefix string `json:"prefix"`
}

func init() {
	caddy.RegisterModule(S3{})
}

func (s *S3) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		var value string

		key := strings.ToLower(d.Val())

		if !d.Args(&value) {
			continue
		}

		switch key {
		case "bucket":
			s.Bucket = value
		case "prefix":
			s.Prefix = value
		}

	}

	return nil
}

func (s *S3) Provision(ctx caddy.Context) error {
	s.log = ctx.Logger(s).Sugar()

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
	s.log.Infof("store %s (%d bytes)", key, len(value))

	length := int64(len(value))

	_, err := s.S3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		Bucket:        &s.Bucket,
		Key:           aws.String(s.keyPrefix(key)),
		Body:          bytes.NewReader(value),
		ContentLength: &length,
	})

	return err
}

func (s S3) Load(ctx context.Context, key string) ([]byte, error) {
	s.log.Infof("load %s", key)

	object, err := s.S3.GetObjectWithContext(ctx, &s3.GetObjectInput{
		Bucket: &s.Bucket,
		Key:    aws.String(s.keyPrefix(key)),
	})
	if err != nil {
		return nil, errors.Join(fs.ErrNotExist, err)
	}
	defer object.Body.Close()

	return io.ReadAll(object.Body)
}

func (s S3) Delete(ctx context.Context, key string) error {
	s.log.Infof("delete %s", key)

	key = s.keyPrefix(key)

	_, err := s.S3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: &s.Bucket,
		Key:    &key,
	})
	return err
}

func (s S3) Exists(ctx context.Context, key string) bool {
	s.log.Infof("exists %s", key)

	_, err := s.S3.GetObjectAttributes(&s3.GetObjectAttributesInput{
		Bucket: &s.Bucket,
		Key:    aws.String(key),
	})

	exists := err == nil
	return exists
}

func (s S3) List(ctx context.Context, prefix string, recursive bool) ([]string, error) {
	s.log.Infof("list %s", prefix)

	keys := make([]string, 0, 32)
	err := s.S3.ListObjectsPagesWithContext(ctx, &s3.ListObjectsInput{
		Bucket: aws.String(s.Bucket),
		Prefix: aws.String(s.Prefix + prefix),
	}, func(objects *s3.ListObjectsOutput, b bool) bool {
		// TODO: handle recursive
		for _, obj := range objects.Contents {
			if obj == nil || obj.Key == nil {
				continue
			}

			if strings.HasPrefix(*obj.Key, prefix) {
				keys = append(keys, *obj.Key)
			}

		}
		return true
	})

	return keys, err
}

func (s S3) Stat(ctx context.Context, key string) (certmagic.KeyInfo, error) {
	s.log.Infof("stat %s", key)

	keyWithPrefix := s.keyPrefix(key)
	attrs, err := s.S3.GetObjectAttributes(&s3.GetObjectAttributesInput{
		Bucket: &s.Bucket,
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
	return path.Join(s.Prefix, key)
}

func (s S3) String() string {
	return fmt.Sprintf("S3 Storage Bucket: %s, Prefix: %s", s.Bucket, s.Prefix)
}
