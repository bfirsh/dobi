package config

import (
	"fmt"
	"reflect"

	"github.com/dnephin/dobi/execenv"
	shlex "github.com/kballard/go-shellquote"
)

// JobConfig A **job** resource uses an `image`_ to run a job in a conatiner.
// A **job** resource that doesn't have an **artifact** is never considered
// up-to-date and will always run.  If a job resource has an **artifact**
// the last modified time of that file will be used as the modified time for the
// **job**.
//
// The `image`_ specified in **use** and any `mount`_ resources listed in
// **mounts** are automatically added as dependencies and will always be
// created first.
// name: job
// example: Run a container using the ``builder`` image to compile some source
// code to ``./dist/app-binary``.
//
// .. code-block:: yaml
//
//     job=compile:
//         use: builder
//         mounts: [source, dist]
//         artifact: dist/app-binary
//
type JobConfig struct {
	// Use The name of an `image`_ resource. The referenced image is used
	// to created the container for the **job**.
	Use string `config:"required"`
	// Artifact A host path to a file or directory that is the output of this
	// **job**. Paths are relative to the current working directory.
	Artifact string
	// Command The command to run in the container.
	// type: shell quoted string
	// example: ``"bash -c 'echo something'"``
	Command ShlexSlice
	// Entrypoint Override the image entrypoint
	// type: shell quoted string
	Entrypoint ShlexSlice
	// Sources A list of files or directories which are used to create the
	// artifact. The modified time of these files are compared to the modified time
	// of the artifact to determine if the **job** is stale. If the **sources**
	// list is defined the modified time of **mounts** and the **use** image are
	// ignored.
	// type: list of files or directories
	Sources []string
	// Mounts A list of `mount`_ resources to use when creating the container.
	// type: list of mount resources
	Mounts []string
	// Privileged Gives extended privileges to the container
	Privileged bool
	// Interactive Makes the container interative and enables a tty.
	Interactive bool
	// Depends The list of resources dependencies
	// type: list of resource names
	Depends []string
	// Env Environment variables to pass to the container. This field
	// supports :doc:`variables`.
	// type: list of ``key=value`` strings
	Env []string
	// ProvideDocker Exposes the docker engine to the container by either
	// mounting the unix socket or setting the **DOCKER_HOST** environment
	// variable.
	ProvideDocker bool
	// NetMode The network mode to use. This field supports :doc:`variables`.
	NetMode string
	// WorkingDir The directory to set as the active working directory in the
	// container. This field supports :doc:`variables`.
	WorkingDir string
}

// Dependencies returns the list of implicit and explicit dependencies
func (c *JobConfig) Dependencies() []string {
	return append([]string{c.Use}, append(c.Depends, c.Mounts...)...)
}

// Validate checks that all fields have acceptable values
func (c *JobConfig) Validate(path Path, config *Config) *PathError {
	if err := c.validateUse(config); err != nil {
		return PathErrorf(path.add("use"), err.Error())
	}
	if err := c.validateMounts(config); err != nil {
		return PathErrorf(path.add("mounts"), err.Error())
	}
	return nil
}

func (c *JobConfig) validateUse(config *Config) error {
	err := fmt.Errorf("%s is not an image resource", c.Use)

	res, ok := config.Resources[c.Use]
	if !ok {
		return err
	}

	switch res.(type) {
	case *ImageConfig:
	default:
		return err
	}

	return nil
}

func (c *JobConfig) validateMounts(config *Config) error {
	for _, mount := range c.Mounts {
		err := fmt.Errorf("%s is not a mount resource", mount)

		res, ok := config.Resources[mount]
		if !ok {
			return err
		}

		switch res.(type) {
		case *MountConfig:
		default:
			return err
		}
	}
	return nil
}

func (c *JobConfig) String() string {
	artifact, command := "", ""
	if c.Artifact != "" {
		artifact = fmt.Sprintf(" to create '%s'", c.Artifact)
	}
	// TODO: look for entrypoint as well as command
	if !c.Command.Empty() {
		command = fmt.Sprintf("'%s' using ", c.Command.String())
	}
	return fmt.Sprintf("Run %sthe '%s' image%s", command, c.Use, artifact)
}

// Resolve resolves variables in the resource
func (c *JobConfig) Resolve(env *execenv.ExecEnv) (Resource, error) {
	var err error
	c.Env, err = env.ResolveSlice(c.Env)
	if err != nil {
		return c, err
	}
	c.WorkingDir, err = env.Resolve(c.WorkingDir)
	if err != nil {
		return c, err
	}
	c.NetMode, err = env.Resolve(c.NetMode)
	return c, err
}

// ShlexSlice is a type used for config transforming a string into a []string
// using shelx.
type ShlexSlice struct {
	original string
	parsed   []string
}

func (s *ShlexSlice) String() string {
	return s.original
}

// Value returns the slice value
func (s *ShlexSlice) Value() []string {
	return s.parsed
}

// Empty returns true if the instance contains the zero value
func (s *ShlexSlice) Empty() bool {
	return s.original == ""
}

// TransformConfig is used to transform a string from a config file into a
// sliced value, using shlex.
func (s *ShlexSlice) TransformConfig(raw reflect.Value) error {
	var err error
	switch value := raw.Interface().(type) {
	case string:
		s.original = value
		s.parsed, err = shlex.Split(value)
		if err != nil {
			return fmt.Errorf("failed to parse command %q: %s", value, err)
		}
	default:
		return fmt.Errorf("must be a string, not %T", value)
	}
	return nil
}

func jobFromConfig(name string, values map[string]interface{}) (Resource, error) {
	cmd := &JobConfig{}
	return cmd, Transform(name, values, cmd)
}

func init() {
	RegisterResource("job", jobFromConfig)
	// Backwards compatibility for v0.4, remove in v0.6
	RegisterResource("run", jobFromConfig)
}
