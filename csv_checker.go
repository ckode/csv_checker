package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/BurntSushi/toml"
)

// Struct for configuration file
type tomlConfig struct {
	WorkingDirectory    string
	InputCSVFile        string
	OutputCSVFile       string
	ErrorCSVFile        string
	ErrorLogFile        string
	SpecifiedFieldCheck string
}

// Check for only printable characters and a white space
// if you find anything that isn't, immediately return false
func OnlyPrintable(str string, noblank string) bool {
	runes := []rune(str)
	if str == "" {
		if noblank == "NO_BLANK" {
			return false
		} else {
			return true
		}
	}
	for _, c := range runes {
		if !unicode.IsPrint(c) {
			return false
		}
	}
	return true
}

// check that only numbers in the field.
// Return false if you find a non-numeric value
func OnlyDigits(str string, noblank string) bool {
	if str == "" {
		if noblank == "NO_BLANK" {
			return false
		} else {
			return true
		}
	}
	runes := []rune(str)
	for _, c := range runes {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

// Is the string a valid float?
func IsFloat(str string, noblank string) bool {
	if str == "" {
		if noblank == "NO_BLANK" {
			return false
		} else {
			return true
		}
	}
	_, err := strconv.ParseFloat(str, 32)
	if err == nil {
		return true
	}
	return false
}

func main() {
	conf := flag.String("conf", "filecheck.cfg", "Default configuration file")
	flag.Parse()

	var config tomlConfig
	_, err := toml.DecodeFile(*conf, &config)
	if err != nil {
		fmt.Printf("ERROR opening configuration file: %v", err)
		return
	}

	files, err := filepath.Glob(filepath.Join(config.WorkingDirectory + "*.csv"))
	if err != nil {
		log.Fatal(err)
	}
	if len(files) != 1 {
		if len(files) == 0 {
			fmt.Printf("There are no csv files in the working folder: %v", config.WorkingDirectory)
		} else if len(files) > 1 {
			fmt.Printf("There are too many csv files in the current working directory: %v\n", config.WorkingDirectory)
			fmt.Println("Please remove all csv files except the one you intend to work with.")
		}
		time.Sleep(15 * time.Second)
		os.Exit(1)
	}
	//inputfile := files[0]
	inputfile := files[0]
	nameonly := strings.TrimSuffix(inputfile, ".csv")
	logfilename := nameonly + ".log"
	goodfile := nameonly + "-GOOD.csv"
	badfile := nameonly + "-BAD.csv"

	// Open a logfile and prepare it for logging via the golang logger library.
	// It will create a new logfile, or truncate an existing one at start up.
	logfile, err := os.OpenFile(logfilename, os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln(err)
	}
	log.SetOutput(logfile)
	defer logfile.Close() //Close the logfile when main() exits

	newcsvfile, err := os.OpenFile(goodfile, os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln("Unable to open new CSV file.")
		os.Exit(1)
	}
	defer newcsvfile.Close()
	newcsv := csv.NewWriter(newcsvfile)

	errcsvfile, err := os.OpenFile(badfile, os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		log.Fatalln("Cannot open to error csv file.")
		os.Exit(1)
	}
	defer errcsvfile.Close()
	errcsv := csv.NewWriter(errcsvfile)

	index := config.SpecifiedFieldCheck
	csvfile, err := os.Open(inputfile)
	if err != nil {
		log.Fatalln("Couldn't open file: ", err)
	}
	r := csv.NewReader(csvfile)
	i := 0
	for {
		var bad_record = false
		record, err := r.Read()

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		for x, v := range record {
			index_ch := string(index[x])
			switch index_ch {
			case "S":
				if !OnlyPrintable(v, "BLANK_ALLOWED") {
					log.Println(fmt.Sprintf("Row %v, Field %v is not a string", i+1, x+1))
					bad_record = true
				}
				break
			case "F":
				if !IsFloat(v, "BLANK_ALLOWED") {
					log.Println(fmt.Sprintf("Row %v, Field %v is not a floating point number", i+1, x+1))
					bad_record = true
				}
				break

			case "N":
				if !OnlyDigits(v, "BLANK_ALLOW") {
					log.Println(fmt.Sprintf("Row %v, Field %v is not an integer", i+1, x+1))
					bad_record = true
				}
				break

			case "B":
				if !OnlyPrintable(v, "NO_BLANK") {
					log.Println(fmt.Sprintf("Row %v, Field %v is not a string or is blank", i+1, x+1))
					bad_record = true
				}
				break

			case "Q":
				if !IsFloat(v, "NO_BLANK") {
					log.Println(fmt.Sprintf("Row %v, Field %v is not a floating point number or is blank", i+1, x+1))
					bad_record = true
				}
				break

			case "P":
				if !OnlyDigits(v, "NO_BLANK") {
					log.Println(fmt.Sprintf("Row %v, Field %v is not an integer or is blank", i+1, x+1))
					bad_record = true
				}
				break

			default:
				log.Println(fmt.Sprintf("ERROR: CSV Field is beyond defined definition list."))
				bad_record = true

			}
		}
		if bad_record == true {
			if err := errcsv.Write(record); err != nil {
				log.Println(fmt.Sprintf("Error writing to error csv file: %v", i))
			}
			errcsv.Flush()
		} else if bad_record == false {
			if err := newcsv.Write(record); err != nil {
				log.Println(fmt.Sprintf("Error writing Row %v to new csv file", i))
			}
			newcsv.Flush()
			i++
		}
	}
}
