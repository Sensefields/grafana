package jsonexecutor 

import (
        "bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"golang.org/x/net/context/ctxhttp"

	"github.com/grafana/grafana/pkg/log"
	"github.com/grafana/grafana/pkg/models"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/tsdb"
)


type QueryTarget struct {

  Target string `json:"target"`
  RefId string  `json:"refId"`
  Type string   `json:"type"`
}

type QueryRange struct {
  From string `json:"from"`
  To string   `json:"to"`
}

type SimpleJsonQuery struct {
  Range QueryRange  `json:"range"`
  Interval string   `json:"interval"`
  Format string     `json:"format"`
  MaxDataPoints int `json:"maxDataPoints"`
  Targets []QueryTarget  `json:"targets"`
}

type JsonExecutor struct {
	*models.DataSource
	HttpClient *http.Client
}

func NewJsonExecutor(datasource *models.DataSource) (tsdb.Executor, error) {
	httpClient, err := datasource.GetHttpClient()

	if err != nil {
		return nil, err
	}

	return &JsonExecutor{
		DataSource: datasource,
		HttpClient: httpClient,
	}, nil
}

var (
	glog log.Logger
)

func init() {
	glog = log.New("tsdb.grafana-simple-json-datasource")
	glog.Info("Registering Json tsdb")
	tsdb.RegisterExecutor("grafana-simple-json-datasource", NewJsonExecutor)
}

func (e *JsonExecutor) Execute(ctx context.Context, queries tsdb.QuerySlice, context *tsdb.QueryContext) *tsdb.BatchResult {
	result := &tsdb.BatchResult{}

        var url string

        var jsonQuery SimpleJsonQuery
        jsonQuery.Range.From = context.TimeRange.From
        jsonQuery.Range.To = context.TimeRange.To
        jsonQuery.Format = "json"
        jsonQuery.MaxDataPoints = 0 

	for _, query := range queries {
                var target QueryTarget
                target.Target = query.Model.Get("target").MustString()
                target.RefId  = query.Model.Get("refId").MustString()
                target.Type   = query.Model.Get("type").MustString()

                jsonQuery.Targets = append(jsonQuery.Targets, target) 

                //FIXME: Only url for last query will take effect. Assumed all are the same
                url = query.DataSource.Url

 
			
	}

	req, err := e.createRequest(url, jsonQuery)
	if err != nil {
		result.Error = err
		return result
	}

	res, err := ctxhttp.Do(ctx, e.HttpClient, req)
	if err != nil {
		result.Error = err
		return result
	}

	data, err := e.parseResponse(res)
	if err != nil {
		result.Error = err
		return result
	}

	result.QueryResults = make(map[string]*tsdb.QueryResult)
	queryRes := tsdb.NewQueryResult()

	for _, series := range data {
		queryRes.Series = append(queryRes.Series, &tsdb.TimeSeries{
			Name:   series.Target,
			Points: series.DataPoints,
		})

		if setting.Env == setting.DEV {
			glog.Debug("Json response", "target", series.Target, "datapoints", len(series.DataPoints))
		}
	}

	result.QueryResults["A"] = queryRes
	return result
}

func (e *JsonExecutor) parseResponse(res *http.Response) ([]TargetResponseDTO, error) {
	body, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()
	if err != nil {
		return nil, err
	}

	if res.StatusCode/100 != 2 {
		glog.Info("Request failed", "status", res.Status, "body", string(body))
		return nil, fmt.Errorf("Request failed status: %v", res.Status)
	}

	var data []TargetResponseDTO
	err = json.Unmarshal(body, &data)
	if err != nil {
		glog.Info("Failed to unmarshal response", "error", err, "status", res.Status, "body", string(body))
		return nil, err
	}

	return data, nil
}

func (e *JsonExecutor) createRequest(dsUrl string, query SimpleJsonQuery) (*http.Request, error) {

        url := dsUrl + "/query"

        var body,_ = json.Marshal(query)

	req, err := http.NewRequest(http.MethodPost,  url, bytes.NewBuffer(body))
	if err != nil {
		glog.Info("Failed to create request", "error", err)
		return nil, fmt.Errorf("Failed to create request. error: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.BasicAuth {
		req.SetBasicAuth(e.BasicAuthUser, e.BasicAuthPassword)
	}

	return req, err
}

func formatTimeRange(input string) string {
	if input == "now" {
		return input
	}
	return strings.Replace(strings.Replace(input, "m", "min", -1), "M", "mon", -1)
}

func fixIntervalFormat(target string) string {
	rMinute := regexp.MustCompile(`'(\d+)m'`)
	rMin := regexp.MustCompile("m")
	target = rMinute.ReplaceAllStringFunc(target, func(m string) string {
		return rMin.ReplaceAllString(m, "min")
	})
	rMonth := regexp.MustCompile(`'(\d+)M'`)
	rMon := regexp.MustCompile("M")
	target = rMonth.ReplaceAllStringFunc(target, func(M string) string {
		return rMon.ReplaceAllString(M, "mon")
	})
	return target
}
