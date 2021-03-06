package config

import "golang.org/x/exp/slices"

// Config represents a full configuration.
type Config struct {
	Project  ProjectConfig  `toml:"project"`
	Build    BuildConfig    `toml:"build"`
	Upload   UploadConfig   `toml:"upload"`
	Template TemplateConfig `toml:"template"`
	Stacks   []StackConfig  `toml:"stacks"`
}

// FindStack searches for the stack with the given name. If no stack is defined
// with the provided name, FindStack returns ok == false.
func (c *Config) FindStack(name string) (stack StackConfig, ok bool) {
	i := slices.IndexFunc(c.Stacks, func(s StackConfig) bool { return s.Name == name })
	if i < 0 {
		return StackConfig{}, false
	}
	return c.Stacks[i], true

}

// ProjectConfig represents the configuration for this project, which is
// expected to be common across all possible deployments.
type ProjectConfig struct {
	Name string `toml:"name"`
}

// BuildConfig represents the configuration for building a deployable Go binary.
type BuildConfig struct {
	Path string `toml:"path"`
}

// UploadConfig represents the configuration for uploading a Go binary in a
// Lambda .zip archive to an Amazon S3 bucket.
type UploadConfig struct {
	Bucket string `toml:"bucket"`
	Prefix string `toml:"prefix"`
}

// TemplateConfig represents the configuration of the AWS CloudFormation
// template associated with the deployment.
type TemplateConfig struct {
	Path         string   `toml:"path"`
	Capabilities []string `toml:"capabilities"`
}

// StackConfig represents the configuration of an AWS CloudFormation stack, a
// specific deployment of the CloudFormation template with a unique set of
// parameters.
type StackConfig struct {
	Name       string            `toml:"name"`
	Parameters map[string]string `toml:"parameters"`
}
