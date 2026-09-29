package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/Denio1337/goenv"
)

type Config struct {
	Port       int           `env:"PORT" required:"true"`
	Timeout    time.Duration `env:"TIMEOUT" default:"5s"`
	MaxRetries int           `env:"MAX_RETRIES" default:"3"`
	Debug      bool          `env:"DEBUG"`
	Database   struct {
		Host     string `env:"HOST" default:"localhost"`
		Password string `env:"PASSWORD" required:"true"`
	} `env-prefix:"DATABASE_"`
}

func main() {
	var cfg Config

	err := goenv.Load(&cfg, goenv.WithDotEnv(".env"))
	if err == nil {
		fmt.Println("Unexpected success! Config loaded.")
		return
	}

	fmt.Println("=== Configuration Schema Validation Failed ===")
	fmt.Println(err)

	// Programmatic inspection of specific field errors
	var valErr *goenv.ValidationError
	if errors.As(err, &valErr) {
		fmt.Printf("\nTotal field errors caught: %d\n", len(valErr.Errors))
		for i, fe := range valErr.Errors {
			fmt.Printf("Error #%d: Field=%s, Key=%s, Expected=%s\n",
				i+1, fe.Field, fe.Key, fe.TargetType)
		}
	}
}
