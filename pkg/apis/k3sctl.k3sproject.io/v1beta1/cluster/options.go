package cluster

import (
	"fmt"
	"strings"
)

type Options struct {
	Drain       *DrainOption       `yaml:"drain,omitempty"`
	Concurrency *ConcurrencyOption `yaml:"concurrency,omitempty"`
	EvictTaint  *EvictTaintOption  `yaml:"evictTaint,omitempty"`
}

type DrainOption struct {
	Enabled          bool   `yaml:"enabled" default:"true"`
	Timeout          string `yaml:"timeout" default:"5m"`
	GracePeriod      int    `yaml:"gracePeriod" default:"-1"`
	DeleteEmptyDir   bool   `yaml:"deleteEmptyDirData" default:"true"`
	IgnoreDaemonSets bool   `yaml:"ignoreDaemonSets" default:"true"`
	DisableEviction  bool   `yaml:"disableEviction" default:"false"`
}

type ConcurrencyOption struct {
	Limit   int `yaml:"limit" default:"30"`
	Uploads int `yaml:"uploads" default:"5"`
}

type EvictTaintOption struct {
	Enabled bool `yaml:"enabled" default:"false"`
}

func (o *Options) GetDrainOptions() *DrainOption {
	if o == nil || o.Drain == nil {
		return &DrainOption{
			Enabled:          true,
			Timeout:          "5m",
			GracePeriod:      -1,
			DeleteEmptyDir:   true,
			IgnoreDaemonSets: true,
		}
	}
	return o.Drain
}

// UnmarshalYAML applies defaults before decoding so that partially specified
// drain options keep the documented default behavior.
func (d *DrainOption) UnmarshalYAML(unmarshal func(any) error) error {
	tmp := DrainOption{
		Enabled:          true,
		Timeout:          "5m",
		GracePeriod:      -1,
		DeleteEmptyDir:   true,
		IgnoreDaemonSets: true,
	}
	type plain DrainOption
	if err := unmarshal((*plain)(&tmp)); err != nil {
		return err
	}
	*d = tmp
	return nil
}

// ToKubectlArgs renders the options as kubectl drain arguments.
func (d *DrainOption) ToKubectlArgs() string {
	if d == nil {
		d = (&Options{}).GetDrainOptions()
	}
	args := []string{"--force"}
	if d.GracePeriod >= 0 {
		args = append(args, fmt.Sprintf("--grace-period=%d", d.GracePeriod))
	}
	if d.Timeout != "" {
		args = append(args, "--timeout="+d.Timeout)
	}
	if d.DeleteEmptyDir {
		args = append(args, "--delete-emptydir-data")
	}
	if d.IgnoreDaemonSets {
		args = append(args, "--ignore-daemonsets")
	}
	if d.DisableEviction {
		args = append(args, "--disable-eviction")
	}
	return strings.Join(args, " ")
}

func (o *Options) GetConcurrency() int {
	if o == nil || o.Concurrency == nil {
		return 30
	}
	return o.Concurrency.Limit
}

func (o *Options) GetConcurrentUploads() int {
	if o == nil || o.Concurrency == nil {
		return 5
	}
	return o.Concurrency.Uploads
}

func (o *Options) WorkerDisruptionPercent() int {
	return 10
}
