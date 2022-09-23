package main

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

type (
	MysqlCfg struct {
		Host      string `yaml:"host"`
		Port      int    `yaml:"port"`
		User      string `yaml:"user"`
		Password  string `yaml:"password"`
		DbName    string `yaml:"dbName"`
		Charset   string `yaml:"charset"`
		ParseTime bool   `yaml:"parseTime"`
		MaxIdle   int    `yaml:"maxIdle"`
		MaxOpen   int    `yaml:"maxOpen"`
		Debug     bool   `yaml:"debug"`
	}

	Config struct {
		Mysql  *MysqlCluster
		Logger *struct {
			Path string `yaml:"path"`
			Name string `yaml:"name"`
		} `yaml:"log"`
	}
	MysqlCluster struct {
		Default *MysqlCfg `yaml:"default"`
	}
)

var conf Config

func getConfig() {
	s, _ := os.Executable()
	path := filepath.Dir(s)
	//data, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", path, "config.yaml"))
	data, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatal("read config fail,err:" + err.Error() + ",filePath:" + path)
	}
	err = yaml.Unmarshal(data, &conf)
	if err != nil {
		log.Fatal("decode config fail,err:" + err.Error())
	}
}
