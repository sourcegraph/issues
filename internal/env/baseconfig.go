package env

import (
	"fmt"
	"strconv"
	"time"

	"github.com/hashicorp/go-multierror"
)

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
}

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
	rawValue := Get(name, defaultValue, description)
	if rawValue == "" {
		c.errs = append(c.errs, fmt.Errorf("invalid value %q for %s: no value supplied", rawValue, name))
		return ""
	}

	return rawValue
}

// GetInt returns the value with the given name interpreted as an integer. If no
// value was supplied in the environment, the given default is used in its place.
// If no value is available, or if the given value or default cannot be converted
// to an integer, an error is added to the validation errors list.
func (c *BaseConfig) GetInt(name, defaultValue, description string) int {
	rawValue := Get(name, defaultValue, description)
	i, err := strconv.ParseInt(rawValue, 10, 64)
	if err != nil {
		c.errs = append(c.errs, fmt.Errorf("invalid int %q for %s: %s", rawValue, name, err))
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
		c.errs = append(c.errs, fmt.Errorf("invalid percent %q for %s: must be 0 <= p <= 100", value, name))
		return 0
	}

	return value
}

func (c *BaseConfig) GetInterval(name, defaultValue, description string) time.Duration {
	rawValue := Get(name, defaultValue, description)
	d, err := time.ParseDuration(rawValue)
	if err != nil {
		c.errs = append(c.errs, fmt.Errorf("invalid duration %q for %s: %s", rawValue, name, err))
		return 0
	}

	return d
}

// GetBool returns the value with the given name interpreted as a boolean. If no value was
// supplied in the environment, the given default is used in its place. If no value is available,
// or if the given value or default cannot be converted to a boolean, an error is added to the
// validation errors list.
func (c *BaseConfig) GetBool(name, defaultValue, description string) bool {
	rawValue := Get(name, defaultValue, description)
	v, err := strconv.ParseBool(rawValue)
	if err != nil {
		c.errs = append(c.errs, fmt.Errorf("invalid bool %q for %s: %s", rawValue, name, err))
		return false
	}

	return v
}
