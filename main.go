package main

import (
	"fmt"
	"log"
	"reflect"
	"strconv"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ssm"
)

func getSSMClient() *ssm.SSM {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
	}))

	client := ssm.New(sess)
	return client
}

var (
	ssmPath = "/Env/Application/"
	client  = getSSMClient()
)

// Name a person's name
type Name struct {
	FirstName string `ssm:"Name/FirstName"`
	LastName  string `ssm:"Name/LastName"`
}

// Contact a person's contact info
type Contact struct {
	Email  string `ssm:"Contact/Email"`
	Number string `ssm:"Contact/Number"`
}

// Person a person
type Person struct {
	Name                       Name
	Contact                    Contact
	FavoriteNumber             int     `ssm:"FavoriteNumber"`
	FavoriteInconvenientNumber float64 `ssm:"FavoriteInconvenientNumber"`
}

func getSSMParameter(name *string) (*string, error) {
	withDecryption := true
	input := ssm.GetParameterInput{
		Name:           name,
		WithDecryption: &withDecryption,
	}
	value, err := client.GetParameter(&input)
	if err != nil {
		return nil, err
	}
	return value.Parameter.Value, nil
}

func handleSSMUpdate(field *reflect.Value, fieldType *reflect.StructField, ssmPath *string, ssm *string) error {
	parameterPath := fmt.Sprintf("%s%s", *ssmPath, *ssm)
	switch field.Kind() {
	case reflect.String:
		updatedValue, err := getSSMParameter(&parameterPath)
		if err != nil {
			return err
		}
		field.SetString(*updatedValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		updatedValue, err := getSSMParameter(&parameterPath)
		if err != nil {
			return err
		}
		ssmInt, err := strconv.ParseInt(*updatedValue, 0, 64)
		if err != nil {
			return err
		}
		field.SetInt(ssmInt)
	case reflect.Float32, reflect.Float64:
		updatedValue, err := getSSMParameter(&parameterPath)
		if err != nil {
			return err
		}
		ssmFloat, err := strconv.ParseFloat(*updatedValue, 64)
		if err != nil {
			return err
		}
		field.SetFloat(ssmFloat)
	case reflect.Bool:
		updatedValue, err := getSSMParameter(&parameterPath)
		if err != nil {
			return err
		}
		ssmBool, err := strconv.ParseBool(*updatedValue)
		if err != nil {
			return err
		}
		field.SetBool(ssmBool)
	default:
		return fmt.Errorf("Field %s is of a type that cannot be set", fieldType.Name)
	}
	return nil
}

// UpdateBySSM updates struct by SSM tag
func UpdateBySSM(generic interface{}, ssmPath *string) error {
	valueOfGeneric := reflect.ValueOf(generic).Elem()
	typeOfGeneric := valueOfGeneric.Type()

	for i := 0; i < valueOfGeneric.NumField(); i++ {
		field := valueOfGeneric.Field(i)
		fieldType := typeOfGeneric.Field(i)
		if field.Kind() == reflect.Struct {
			UpdateBySSM(field.Addr().Interface(), ssmPath)
		} else if ssm, ok := fieldType.Tag.Lookup("ssm"); ok {
			handleSSMUpdate(&field, &fieldType, ssmPath, &ssm)
		}
	}
	return nil
}

func main() {
	person := Person{}
	err := UpdateBySSM(&person, &ssmPath)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Updated person", person)
}
