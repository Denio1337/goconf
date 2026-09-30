package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/Denio1337/goconf"
)

type Config struct {
	Port       int           `required:"true"`
	Timeout    time.Duration `default:"5s"`
	MaxRetries int           `default:"3"`
	Debug      bool
	Database   struct {
		Host     string `default:"localhost"`
		Password string `required:"true"`
	} `prefix:"DATABASE_"`
}

func main() {
	var cfg Config

	err := goconf.Load(&cfg, goconf.WithDotEnv(".env"))
	if err == nil {
		fmt.Println("Unexpected success! Config loaded.")
		return
	}

	fmt.Println("=== Configuration Schema Validation Failed ===")
	fmt.Println(err)

	// Programmatic inspection of specific field errors
	var valErr *goconf.ValidationError
	if errors.As(err, &valErr) {
		fmt.Printf("\nTotal field errors caught: %d\n", len(valErr.Errors))
		for i, fe := range valErr.Errors {
			fmt.Printf("Error #%d: Field=%s, Key=%s, Expected=%s\n",
				i+1, fe.Field, fe.Key, fe.TargetType)
		}
	}
}
