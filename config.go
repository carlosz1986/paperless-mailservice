package main

import (
	"fmt"
	"log"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Global config variable
var Config config

// Struct for config validation using go-playground/validator
type config struct {
	Paperless struct {
		InstanceURL      string `validate:"required,url"`
		InstanceToken    string `validate:"required"`
		AddQueueTagName  string `validate:"required"`
		ProcessedTagName string `validate:"required"`
		Rules            []struct {
			Name            string `validate:"required"`
			ReceiverAddress string `validate:"required,email"`
			MailBody        string
			MailHeader      string
			Tags            []string
		}
	}
	Email struct {
		SMTPAddress        string `validate:"required,email"`
		SMTPServer         string `validate:"required,hostname"`
		SMTPPort           string `validate:"required,min=1,max=65535"`
		SMTPConnectionType string `validate:"required,oneof=starttls tls"`
		SMTPUser           string `validate:"required"`
		SMTPPassword       string `validate:"required"`
		MailBody           string `validate:"required"`
		MailHeader         string `validate:"required"`
	}
	RunEveryXMinute int `validate:"required,min=-1,max=65535"`
}

// validateConfigKeys function to validate required keys
func validateConfigKeys() error {
	typeValidations := map[string]string{
		"Paperless.InstanceURL":      "string",
		"Paperless.InstanceToken":    "string",
		"Paperless.AddQueueTagName":  "string",
		"Paperless.ProcessedTagName": "string",
		// todo Rules
		"Email.SMTPAddress":        "string",
		"Email.SMTPServer":         "string",
		"Email.SMTPPort":           "int",
		"Email.SMTPConnectionType": "string",
		"Email.SMTPUser":           "string",
		"Email.SMTPPassword":       "string",
		"Email.MailBody":           "string",
		"Email.MailHeader":         "string",
		"RunEveryXMinute":          "int",
	}

	for key, expectedType := range typeValidations {
		val := viper.Get(key)

		if val == nil {
			return fmt.Errorf("missing required config key: %s", key)
		}

		// Check if the type matches
		actualType := reflect.TypeOf(val).String()
		if actualType != expectedType {
			return fmt.Errorf("invalid type for key '%s': expected '%s', got '%s'", key, expectedType, actualType)
		}
	}

	return nil
}

// Validate config using go-playground/validator
func validateWithPlayground(config config) error {
	validate := validator.New()

	err := validate.Struct(config)
	if err != nil {
		// If validation fails, print errors
		for _, err := range err.(validator.ValidationErrors) {
			fmt.Printf("Validation failed on field '%s', condition: '%s'\n", err.StructField(), err.Tag())
		}
		return err
	}

	return nil
}

// LoadConfig function to initialize config
func LoadConfig() {

	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("config/")

	// Attempt to read the config file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Fatalf("Error loading Config: %s", err)
		} else {
			log.Fatalf("Error reading Config: %s", err)
		}
	}

	// Validate required keys
	if err := validateConfigKeys(); err != nil {
		log.Fatalf("Config validation failed: %s", err)
	}

	// Populate struct for validator use
	err := viper.Unmarshal(&Config)
	if err != nil {
		log.Fatalf("Unable to unmarshal into struct, %v", err)
	}

	// Validate the struct using go-playground/validator
	if err := validateWithPlayground(Config); err != nil {
		log.Fatalf("Struct validation failed: %v", err)
	}
}

// PrintRules prints the current config to stdout
func PrintRules() {
	log.Printf("Documents with Tag %s at paperless will be marked for queuing", Config.Paperless.AddQueueTagName)

	for _, rule := range Config.Paperless.Rules {
		log.Printf("Found Rule \"%s\": Send Documents with Tag(s): \"%s\" to address %s", rule.Name, strings.Join(rule.Tags, ","), rule.ReceiverAddress)
	}

	log.Printf("All processed documents will be marked with Tag: %s at paperless", Config.Paperless.ProcessedTagName)

}
