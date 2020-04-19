package main

import (
	"fmt"
	"io/ioutil"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/imZack/gogregate/internal/jq"
	"github.com/imroc/req"
	"github.com/tidwall/sjson"
	"gopkg.in/yaml.v2"
)

type configuration struct {
	APIs map[string]api `yaml:"apis"`
}

type api struct {
	Name      string           `yaml:"name"`
	Endpoints []endpointDetail `yaml:"endpoints"`
	JqFilter  string           `yaml:"jq"`
	result    string
}

type endpointDetail struct {
	Name     string            `yaml:"name"`
	Endpoint string            `yaml:"endpoint"`
	Headers  map[string]string `yaml:"headers"`
	result   string
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func execAPI(a api) <-chan api {
	var results = make(chan endpointDetail)
	var apiResult = make(chan api)
	var wg sync.WaitGroup

	vm, err := jq.Compile(a.JqFilter)
	check(err)
	tojson, err := jq.Compile("to_entries | map( {(.key): (.value | fromjson) }) | add")
	check(err)

	for _, endpoint := range a.Endpoints {
		wg.Add(1)
		go func(endpoint endpointDetail) {
			defer wg.Done()
			r, err := req.Get(endpoint.Endpoint, req.Header(endpoint.Headers))
			check(err)
			endpoint.result, err = r.ToString()
			check(err)
			results <- endpoint
		}(endpoint)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	go func() {
		var res string
		for entry := range results {
			res, _ = sjson.Set(res, entry.Name, entry.result)
		}

		defer tojson.Close()
		jsonStr, err := tojson.Apply2(res)
		check(err)
		fmt.Println(jsonStr)

		defer vm.Close()
		a.result, err = vm.Apply2(jsonStr)
		check(err)

		apiResult <- a
		close(apiResult)
	}()

	return apiResult
}

func main() {
	// load routing information
	data, err := ioutil.ReadFile("./config.yml")
	check(err)

	configuration := configuration{}
	err = yaml.Unmarshal([]byte(data), &configuration)
	check(err)

	// start web server
	r := gin.Default()
	r.GET("/:apiName", func(c *gin.Context) {
		apiName := c.Param("apiName")
		if api, ok := configuration.APIs[apiName]; ok {
			for apiResult := range execAPI(api) {
				fmt.Println(apiResult.Name, "done")
				fmt.Println(apiResult.result)
				c.Header("Content-TYpe", "application/json; charset=utf-8")
				c.String(200, apiResult.result)
				return
			}
		}

		c.Status(404)
	})

	// By default it serves on :8080 unless a
	// PORT environment variable was defined.
	r.Run()
}
