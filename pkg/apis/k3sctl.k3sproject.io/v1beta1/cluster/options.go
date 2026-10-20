package cluster

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
