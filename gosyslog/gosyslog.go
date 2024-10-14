package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/mcuadros/go-syslog.v2"
	"gopkg.in/mcuadros/go-syslog.v2/format"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig `yaml:"server"`
	LogFolder string       `yaml:"log_folder"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

func loadConfigFromYAML(configFilePath string) (*Config, error) {
	content, err := os.ReadFile(configFilePath)
	if err != nil {
		return nil, errors.New("error reading config file: " + err.Error())
	}

	config := &Config{}
	err = yaml.Unmarshal(content, config)
	if err != nil {
		return nil, errors.New("error parsing config file: " + err.Error())
	}
	return config, nil
}

func mkFullFilename(logFolder, hostname string) (string, error) {
	logFolder = strings.Replace(logFolder, "%HOSTNAME%", hostname, 1)
	if _, err := os.Stat(logFolder); os.IsNotExist(err) {
		err = os.MkdirAll(logFolder, os.ModePerm)
		if err != nil {
			return "", errors.New("failed to create directory: " + err.Error())
		}
	} else if err != nil {
		return "", errors.New("error creating directory: " + err.Error())
	}
	filename := filepath.Join(logFolder, "syslog.log")
	return filename, nil
}

func writeLogToFile(logFolder string, logParts format.LogParts) error {
	filename, err := mkFullFilename(logFolder, logParts["hostname"].(string))
	if err != nil {
		return errors.New("error making full filename: " + err.Error())
	}

	file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return errors.New("failed to open log file: " + err.Error())
	}
	defer file.Close()

	_, err = fmt.Fprintln(file, logParts)
	if err != nil {
		return errors.New("failed to write log: " + err.Error())
	}
	return nil
}

func main() {
	configFilePath := "config.yml"
	/*
			Пример содержимого конфигурационного файла config.yml:

			server:
		  		port: 514
			log_folder: /var/log/%HOSTNAME%/
	*/

	cfg, err := loadConfigFromYAML(configFilePath)
	if err != nil {
		fmt.Println("Error loading configuration:", err)
		os.Exit(1)
	}

	logFolder := cfg.LogFolder
	port := cfg.Server.Port

	channel := make(syslog.LogPartsChannel)
	handler := syslog.NewChannelHandler(channel)

	server := syslog.NewServer()
	server.SetFormat(syslog.Automatic)
	server.SetHandler(handler)
	server.ListenUDP(fmt.Sprintf("0.0.0.0:%d", port))

	server.Boot()

	go func(channel syslog.LogPartsChannel) {
		for logParts := range channel {
			err := writeLogToFile(logFolder, logParts)
			if err != nil {
				fmt.Println("Error writing log:", err)
				os.Exit(1)
			}
		}
	}(channel)

	server.Wait()
}
