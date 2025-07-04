package ses

import (
	"context"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/sesv2"
	"github.com/aws/aws-sdk-go-v2/service/sesv2/types"
	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const (
	CHAR_SET = "UTF-8"
)

type Config struct {
	Timeout    int64    `yaml:"timeout(s)"`  // 超时时间，单位秒
	AccessID   string   `yaml:"access_id"`   // AccessKey ID
	AccessKey  string   `yaml:"access_key"`  // AccessKey Secret
	Region     string   `yaml:"region"`      // 区域
	SenderList []string `yaml:"sender_list"` // 发件人地址列表
}

type awsses struct {
	c   *sesv2.Client
	cfg *Config
}

var Instance *awsses

func Init(conf *Config) error {
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

	sesc := sesv2.NewFromConfig(cfg)

	Instance = &awsses{
		c:   sesc,
		cfg: conf,
	}

	return nil
}

func (this_ *awsses) SendEmail(recver, sender, title, content string) (string, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), time.Duration(this_.cfg.Timeout)*time.Second)
	defer cancel()

	input := &sesv2.SendEmailInput{
		Destination: &types.Destination{
			CcAddresses: []string{},
			ToAddresses: []string{
				recver,
			},
		},

		Content: &types.EmailContent{
			Simple: &types.Message{
				Subject: &types.Content{
					Data:    aws.String(title),
					Charset: aws.String(CHAR_SET),
				},
				Body: &types.Body{
					Text: &types.Content{
						Data:    aws.String(content),
						Charset: aws.String(CHAR_SET),
					},
				},
			},
		},

		FromEmailAddress: aws.String(sender),
	}

	out, err := this_.c.SendEmail(ctx, input)
	if err != nil {
		return "", err
	}

	return *out.MessageId, nil
}
