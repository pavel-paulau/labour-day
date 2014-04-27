package main

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/couchbaselabs/go-couchbase"
)

var ddocs = map[string]string{
	"default": `{
		"views": {
			"by_build": {
				"map": "function (doc, meta) {emit([doc.build, doc.priority, doc.os], [doc.failCount, doc.totalCount]);}"
			}
		}
	}`,
}

type DataSource struct {
	CouchbaseAddress string
}

func (ds *DataSource) GetBucket(bucket string) *couchbase.Bucket {
	uri := fmt.Sprintf("http://%s:%s@%s/", bucket, "", ds.CouchbaseAddress)

	client, _ := couchbase.Connect(uri)
	pool, _ := client.GetPool("default")

	b, err := pool.GetBucket(bucket)
	if err != nil {
		log.Fatalf("Error reading bucket:  %v", err)
	}
	return b
}

func (ds *DataSource) QueryView(b *couchbase.Bucket, ddoc, view string,
	params map[string]interface{}) []couchbase.ViewRow {
	params["stale"] = "false"
	vr, err := b.View(ddoc, view, params)
	if err != nil {
		ds.installDDoc(ddoc)
	}
	return vr.Rows
}

func (ds *DataSource) installDDoc(ddoc string) {
	b := ds.GetBucket(ddoc) // bucket name == ddoc name
	err := b.PutDDoc(ddoc, ddocs[ddoc])
	if err != nil {
		log.Fatalf("%v", err)
	}
}

func (ds *DataSource) GetTimeline(abs bool) []byte {
	b := ds.GetBucket("default")
	rows := ds.QueryView(b, "default", "by_build", map[string]interface{}{})

	failed := map[string]float64{}
	total := map[string]float64{}
	builds := []string{}
	for i := range rows {
		build := rows[i].Key.([]interface{})[0].(string)
		failCount := rows[i].Value.([]interface{})[0].(float64)
		totalCount := rows[i].Value.([]interface{})[1].(float64)
		if _, ok := failed[build]; ok {
			failed[build] += failCount
		} else {
			failed[build] = failCount
		}
		if _, ok := total[build]; ok {
			total[build] += totalCount
		} else {
			total[build] = totalCount
		}
		builds = append(builds, build)
	}

	passedValues := []interface{}{}
	failedValues := []interface{}{}
	sort.Strings(builds)
	if abs {
		for _, build := range builds {
			passedValues = append(passedValues, []interface{}{
				build,
				total[build] - failed[build],
			})
			failedValues = append(failedValues, []interface{}{
				build,
				-failed[build],
			})
		}
	} else {
		for _, build := range builds {
			passedValues = append(passedValues, []interface{}{
				build,
				100.0 * (total[build] - failed[build]) / total[build],
			})
			failedValues = append(failedValues, []interface{}{
				build,
				100.0 * failed[build] / total[build],
			})
		}
	}

	timeline := []map[string]interface{}{
		map[string]interface{}{"key": "Passed", "values": passedValues},
		map[string]interface{}{"key": "Failed", "values": failedValues},
	}
	j, _ := json.Marshal(timeline)
	return j
}

func appendIfUnique(slice []string, s string) []string {
	for i := range slice {
		if slice[i] == s {
			return slice
		}
	}
	return append(slice, s)
}

func (ds *DataSource) GetBreakdown(build string, by_platform bool) []byte {
	b := ds.GetBucket("default")
	params := map[string]interface{}{"startkey": []string{build}}
	rows := ds.QueryView(b, "default", "by_build", params)

	keys := []string{}
	failed := map[string]float64{}
	total := map[string]float64{}
	for i := range rows {
		var key string
		if by_platform {
			key = rows[i].Key.([]interface{})[2].(string)
		} else {
			key = rows[i].Key.([]interface{})[1].(string)
		}
		if key == "N/A" {
			continue
		}

		failCount := rows[i].Value.([]interface{})[0].(float64)
		totalCount := rows[i].Value.([]interface{})[1].(float64)
		if _, ok := failed[key]; ok {
			failed[key] += failCount
		} else {
			failed[key] = failCount
		}
		if _, ok := total[key]; ok {
			total[key] += totalCount
		} else {
			total[key] = totalCount
		}
		keys = appendIfUnique(keys, key)
	}

	sort.Strings(keys)
	data := map[string]interface{}{}
	for _, key := range keys {
		title := strings.Title(strings.ToLower(key))
		data[title] = []map[string]interface{}{
			{"key": "Passed", "value": total[key] - failed[key]},
			{"key": "Failed", "value": failed[key]},
		}
	}

	j, _ := json.Marshal(data)
	return j
}