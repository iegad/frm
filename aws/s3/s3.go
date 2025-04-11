package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
	"github.com/gox/frm/log"
	"github.com/gox/frm/utils"
)

var (
	checkImageMap = map[string]bool{
		"png": true, "jpeg": true, "jpg": true,
	}
)

type Config struct {
	Timeout   int64  `yaml:"timeout(s)"`
	AccessID  string `yaml:"access_id"`
	AccessKey string `yaml:"access_key"`
	Region    string `yaml:"region"`
	Bucket    string `yaml:"bucket"`
}

type awss3 struct {
	c   *s3.Client
	cfg *Config
}

var Instance *awss3

func Init(conf *Config) error {
	if Instance != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(conf.Timeout)*time.Second)
	defer cancel()

	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(conf.Region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(conf.AccessID, conf.AccessKey, "")),
	)

	if err != nil {
		return err
	}

	stsc := sts.NewFromConfig(cfg)
	_, err = stsc.GetCallerIdentity(ctx, &sts.GetCallerIdentityInput{})
	if err != nil {
		return err
	}

	s3c := s3.NewFromConfig(cfg)

	Instance = &awss3{
		c:   s3c,
		cfg: conf,
	}

	return nil
}

func (this_ *awss3) DeleteObject(key, version string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(this_.cfg.Bucket),
		Key:    aws.String(key),
	}

	if len(version) > 0 {
		input.VersionId = aws.String(version)
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(this_.cfg.Timeout)*time.Second)
	defer cancel()
	_, err := this_.c.DeleteObject(ctx, input)
	if err != nil {
		return err
	}

	err = s3.NewObjectNotExistsWaiter(this_.c).Wait(ctx, &s3.HeadObjectInput{
		Bucket: input.Bucket,
		Key:    input.Key,
	}, time.Duration(this_.cfg.Timeout)*time.Second)

	return err
}

func (this_ *awss3) DeleteObjects(keys []string, version string) error {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(this_.cfg.Timeout)*time.Second)
	defer cancel()

	objects := []types.ObjectIdentifier{}

	for _, key := range keys {
		objects = append(objects, types.ObjectIdentifier{
			Key: aws.String(key),
		})
	}

	input := &s3.DeleteObjectsInput{
		Bucket: aws.String(this_.cfg.Bucket),
		Delete: &types.Delete{
			Objects: objects,
			Quiet:   aws.Bool(true),
		},
	}

	out, err := this_.c.DeleteObjects(ctx, input)
	if err != nil {
		return err
	}

	for _, do := range out.Deleted {
		err = s3.NewObjectNotExistsWaiter(this_.c).Wait(ctx, &s3.HeadObjectInput{
			Bucket: aws.String(this_.cfg.Bucket),
			Key:    do.Key,
		}, time.Minute)

		if err != nil {
			log.Error("Wait object delete failed: %v", err)
		}
	}

	return nil
}

func (this_ *awss3) PutObjectByFile(key string, file io.Reader) error {
	input := &s3.PutObjectInput{
		Bucket: aws.String(this_.cfg.Bucket),
		Key:    aws.String(key),
		Body:   file,
	}

	suffix := utils.GetFileSuffix(key)
	if checkImage(suffix) {
		input.ContentType = aws.String(fmt.Sprintf("image/%v", suffix))
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(this_.cfg.Timeout)*time.Second)
	defer cancel()

	_, err := this_.c.PutObject(ctx, input)
	if err != nil {
		return err
	}

	err = s3.NewObjectExistsWaiter(this_.c).Wait(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(this_.cfg.Bucket),
		Key:    aws.String(key),
	}, time.Duration(this_.cfg.Timeout)*time.Second)

	return err
}

func (this_ *awss3) PutObjectByData(key string, data []byte) error {
	input := &s3.PutObjectInput{
		Bucket:            aws.String(this_.cfg.Bucket),
		Key:               aws.String(key),
		Body:              bytes.NewReader(data),
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
	}

	suffix := utils.GetFileSuffix(key)
	if checkImage(suffix) {
		input.ContentType = aws.String(fmt.Sprintf("image/%v", suffix))
	}

	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(this_.cfg.Timeout)*time.Second)
	defer cancel()

	uper := manager.NewUploader(this_.c)
	_, err := uper.Upload(ctx, input)
	if err != nil {
		return err
	}

	err = s3.NewObjectExistsWaiter(this_.c).Wait(ctx, &s3.HeadObjectInput{
		Bucket: input.Bucket,
		Key:    input.Key,
	}, time.Duration(this_.cfg.Timeout)*time.Second)

	return err
}

func checkImage(suffix string) bool {
	return checkImageMap[suffix]
}
