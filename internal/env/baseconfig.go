package env

import (
	"strconv"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/hashicorp/go-multierror"
)

type Config interface {
	// Load is called prior to env.Lock an application startup. This method should
	// read the values from the environment and store errors to be reported later.
	Load()

	// Validate performs non-trivial validation and returns any resulting errors.
	// This method should also return errors that occurred while reading values from
	// the environment in Load. This method is called after the environment has been
	// locked, so all environment variable reads must happen in Load.
	Validate() error
}

// BaseConfig is a base struct for configuration objects. The following is a minimal
// example of declaring, loading, and validating configuration from the environment.
//
//     type Config struct {
//         env.BaseConfig
//
//         Name   string
//         Weight int
//         Age    time.Duration
//     }
//
//     func (c *Config) Load() {
//         c.Name = c.Get("SRC_NAME", "test", "The service's name (wat).")
//         c.Weight = c.GetInt("SRC_WEIGHT", "1m", "The service's weight (wat).")
//         c.Age = c.GetInterval("SRC_AGE", "10s", "The service's age (wat).")
//     }
//
//     func applicationInit() {
//         config := &Config{}
//         config.Load()
//
//         env.Lock()
//         env.HandleHelpFlag()
//
//         if err := config.Validate(); err != nil{
//             // handle me
//         }
//     }
type BaseConfig struct {
	errs []error

	// getter is used to mock the environment in tests
	getter GetterFunc
}

type GetterFunc func(name, defaultValue, description string) string

// Validate returns any errors constructed from a Get* method after the values have
// been loaded from the environment.
func (c *BaseConfig) Validate() error {
	if len(c.errs) == 0 {
		return nil
	}

	err := c.errs[0]
	for i := 1; i < len(c.errs); i++ {
		err = multierror.Append(err, c.errs[i])
	}

	return err
}

// Get returns the value with the given name. If no value was supplied in the
// environment, the given default is used in its place. If no value is available,
// an error is added to the validation errors list.
func (c *BaseConfig) Get(name, defaultValue, description string) string {
	rawValue := c.get(name, defaultValue, description)
	if rawValue == "" {
		c.AddError(errors.Errorf("invalid value %q for %s: no value supplied", rawValue, name))
		return ""
	}

	return rawValue
}

// GetOptional returns the value with the given name.
func (c *BaseConfig) GetOptional(name, description string) string {
	return c.get(name, "", description)
}

// GetInt returns the value with the given name interpreted as an integer. If no
// value was supplied in the environment, the given default is used in its place.
// If no value is available, or if the given value or default cannot be converted
// to an integer, an error is added to the validation errors list.
func (c *BaseConfig) GetInt(name, defaultValue, description string) int {
	rawValue := c.get(name, defaultValue, description)
	i, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		c.AddError(errors.Errorf("invalid int %q for %s: %s", rawValue, name, err))
		return 0
	}

	return int(i)
}

// GetPercent returns the value with the given name interpreted as an integer between
// 0 and 100. If no value was supplied in the environment, the given default is used
// in its place. If no value is available, if the given value or default cannot be
// converted to an integer, or if the value is out of the expected range, an error
// is added to the validation errors list.
func (c *BaseConfig) GetPercent(name, defaultValue, description string) int {
	value := c.GetInt(name, defaultValue, description)
	if value < 0 || value > 100 {
		c.AddError(errors.Errorf("invalid percent %q for %s: must be 0 <= p <= 100", value, name))
		return 0
	}

	return value
}

func (c *BaseConfig) GetInterval(name, defaultValue, description string) time.Duration {
	rawValue := c.get(name, defaultValue, description)
	d, err := time.ParseDuration(rawValue)
	if err != nil {
		c.AddError(errors.Errorf("invalid duration %q for %s: %s", rawValue, name, err))
		return 0
	}

	return d
}

// GetBool returns the value with the given name interpreted as a boolean. If no value was
// supplied in the environment, the given default is used in its place. If no value is available,
// or if the given value or default cannot be converted to a boolean, an error is added to the
// validation errors list.
func (c *BaseConfig) GetBool(name, defaultValue, description string) bool {
	rawValue := c.get(name, defaultValue, description)
	v, err := strconv.ParseBool(rawValue)
	if err != nil {
		c.AddError(errors.Errorf("invalid bool %q for %s: %s", rawValue, name, err))
		return false
	}

	return v
}

// AddError adds a validation error to the configuration object. This should be
// called from within the Load method of a decorated configuration object to have
// any effect.
func (c *BaseConfig) AddError(err error) {
	c.errs = append(c.errs, err)
}

func (c *BaseConfig) get(name, defaultValue, description string) string {
	if c.getter != nil {
		return c.getter(name, defaultValue, description)
	}

	return Get(name, defaultValue, description)
}

// SetMockGetter sets mock to use in place of this packge's Get function.
func (c *BaseConfig) SetMockGetter(getter GetterFunc) {
	c.getter = getter
}
