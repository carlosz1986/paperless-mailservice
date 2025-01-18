package main

import (
	"log"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

// Global config variable
var Config config

// Struct for config validation using go-playground/validator
type config struct {
	Paperless Paperless `validate:"required"`
	Email     struct {
		SMTPAddress        string `validate:"required,email"`
		SMTPServer         string `validate:"required,hostname"`
		SMTPPort           string `validate:"required,min=1,max=65535"`
		SMTPConnectionType string `validate:"required,oneof=starttls tls"`
		SMTPUser           string `validate:"required"`
		SMTPPassword       string `validate:"required"`
		MailBody           string
		MailHeader         string
	}
	RunEveryXMinute int `validate:"required,min=-1,max=65535"`
}

type Paperless struct {
	InstanceURL      string `validate:"required,url"`
	InstanceToken    string `validate:"required"`
	AddQueueTagName  string `validate:"required"`
	ProcessedTagName string `validate:"required"`
	Rules            []rule `validate:"required,dive,required"`
}

type rule struct {
	Name              string   `validate:"required"`
	ReceiverAddresses []string `validate:"required,dive,required,email"`
	BCCAddresses      []string `validate:"dive,required,email"`
	MailBody          string
	MailHeader        string
	Tags              []string
	Type              string
	Correspondent     string
}

// Validate config using go-playground/validator
func validateWithPlayground(config config) error {
	validate := validator.New()

	//custom validator to check if rule or paperless has at least a mailbody or header
	validate.RegisterStructValidation(RuleValidation, rule{})

	err := validate.Struct(config)
	if err != nil {
		// If validation fails, print errors
		for _, err := range err.(validator.ValidationErrors) {
			log.Printf("Validation failed on field '%s', condition: '%s'\n", err.StructField(), err.Tag())
		}
		return err
	}

	return nil
}

// ruleValidation custom validator to check details of rule
func RuleValidation(sl validator.StructLevel) {

	// rule to validate
	r := sl.Current().Interface().(rule)
	// entire config struct
	p := sl.Top().Interface().(config)

	// email body and header must be set in rule or config
	if len(r.MailBody) == 0 && len(p.Email.MailBody) == 0 {
		sl.ReportError(r.MailBody, "MailBody", "MailBody", "`MailBody` of rule or at least `Mailbody` of `Config.Email `must be set", "")
	}

	if len(r.MailHeader) == 0 && len(p.Email.MailHeader) == 0 {
		sl.ReportError(r.MailHeader, "MailHeader", "MailHeader", "`MailHeader` of rule or at least `MailHeader` of `Config.Email` must be set", "")
	}

	// atleast tags, correspondent or type must be set in the rule
	if len(r.Tags) == 0 && len(r.Correspondent) == 0 && len(r.Type) == 0 {
		sl.ReportError(r, "", "rule", "At least one of `Tags`, `Correspondent` or `Type` must be set in the rule", "")
	}

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
		var l string
		var details []string

		if len(rule.Tags) > 0 {
			details = append(details, "Tag(s): \""+strings.Join(rule.Tags, ",")+"\"")
		}
		if len(rule.Correspondent) > 0 {
			details = append(details, "Correspondent: \""+rule.Correspondent+"\"")
		}
		if len(rule.Type) > 0 {
			details = append(details, "Type: \""+rule.Type+"\"")
		}
		l += strings.Join(details, ", ")
		l += " to Address(es): \"" + strings.Join(rule.ReceiverAddresses, ",") + "\" "
		if len(rule.BCCAddresses) > 0 {
			l += "and Bcc to: \"" + strings.Join(rule.BCCAddresses, ",") + "\""
		}

		log.Printf("Found Rule \"%s\": Send Documents with %s", rule.Name, l)
	}

	log.Printf("All processed documents will be marked with Tag: %s at paperless", Config.Paperless.ProcessedTagName)

}
