package aws

type Config struct {
	Timeout   int64  `yaml:"timeout(s)"`
	AccessID  string `yaml:"access_id"`
	AccessKey string `yaml:"access_key"`
	Region    string `yaml:"region"`
}
