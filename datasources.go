package main

import (
	"encoding/json"
	"log"
	"sort"
	"strings"

	"github.com/couchbaselabs/go-couchbase"
)

var ddocs = map[string]string{
	"jenkins": `{
		"views": {
			"data_by_build": {
				"map": "function (doc, meta) {emit(doc.build, [doc.failCount, doc.totalCount, doc.os, doc.priority]);}"
			}
		}
	}`,
}

type DataSource struct {
	CouchbaseAddress string
}

func (ds *DataSource) GetBucket(bucket string) *couchbase.Bucket {
	client, _ := couchbase.Connect(ds.CouchbaseAddress)
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
	b := ds.GetBucket("jenkins")
	rows := ds.QueryView(b, "jenkins", "data_by_build", map[string]interface{}{})

	failed := map[string]float64{}
	total := map[string]float64{}
	builds := []string{}
	for _, row := range rows {
		build := row.Key.(string)
		failCount, ok := row.Value.([]interface{})[0].(float64)
		if !ok {
			continue
		}
		totalCount, ok := row.Value.([]interface{})[1].(float64)
		if !ok {
			continue
		}
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
		builds = appendIfUnique(builds, build)
	}

	timeline := []map[string]interface{}{}
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
		timeline = append(timeline,
			map[string]interface{}{"key": "Passed", "values": passedValues})
		timeline = append(timeline,
			map[string]interface{}{"key": "Failed", "values": failedValues})
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
		timeline = append(timeline,
			map[string]interface{}{"key": "Passed, %", "values": passedValues})
		timeline = append(timeline,
			map[string]interface{}{"key": "Failed, %", "values": failedValues})
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
	b := ds.GetBucket("jenkins")
	params := map[string]interface{}{"key": build}
	rows := ds.QueryView(b, "jenkins", "data_by_build", params)

	keys := []string{}
	failed := map[string]float64{}
	total := map[string]float64{}
	for _, row := range rows {
		var key string
		if by_platform {
			key = row.Value.([]interface{})[2].(string)
		} else {
			key = row.Value.([]interface{})[3].(string)
		}
		if key == "N/A" {
			continue
		}

		failCount, ok := row.Value.([]interface{})[0].(float64)
		if !ok {
			continue
		}
		totalCount, ok := row.Value.([]interface{})[1].(float64)
		if !ok {
			continue
		}
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
